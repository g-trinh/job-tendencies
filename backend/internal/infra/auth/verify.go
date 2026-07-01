package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	domainauth "github.com/g-trinh/job-tendencies/internal/domain/auth"
)

// defaultCertsURL is the Google endpoint that returns the public X.509 certificates
// used to verify Firebase ID token signatures. Overridable via WithCertsURL for testing.
const defaultCertsURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

// jwksVerifier fetches and caches Google's X.509 public certificates, then uses them
// to verify Firebase ID token RS256 signatures and standard claims.
type jwksVerifier struct {
	projectID  string
	certsURL   string
	httpClient *http.Client

	mu       sync.RWMutex
	certs    map[string]*rsa.PublicKey // kid → RSA public key
	cacheExp time.Time
}

func newJWKSVerifier(projectID string, hc *http.Client) *jwksVerifier {
	return &jwksVerifier{
		projectID:  projectID,
		certsURL:   defaultCertsURL,
		httpClient: hc,
		certs:      make(map[string]*rsa.PublicKey),
	}
}

// jwtHeader contains the fields we need from the JWT header.
type jwtHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

// jwtClaims contains the fields we verify from the JWT payload.
type jwtClaims struct {
	Issuer   string `json:"iss"`
	Audience string `json:"aud"`
	Subject  string `json:"sub"`
	Email    string `json:"email"`
	Exp      int64  `json:"exp"`
	IAT      int64  `json:"iat"`
}

// verify parses and validates a Firebase ID token. It checks:
//  1. JWT structure (three base64url-encoded parts)
//  2. Algorithm is RS256
//  3. Claims: iss, aud, sub, exp
//  4. RS256 signature against the key identified by the header kid
//
// Returns [domainauth.ErrTokenExpired] for expired tokens, [domainauth.ErrTokenInvalid]
// for all other validation failures.
func (v *jwksVerifier) verify(ctx context.Context, idToken string) (*domainauth.User, error) {
	parts := strings.SplitN(idToken, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: malformed JWT (expected 3 parts, got %d)", domainauth.ErrTokenInvalid, len(parts))
	}

	// Decode and parse header.
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: decoding JWT header", domainauth.ErrTokenInvalid)
	}
	var header jwtHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("%w: parsing JWT header", domainauth.ErrTokenInvalid)
	}
	if header.Algorithm != "RS256" {
		return nil, fmt.Errorf("%w: unexpected algorithm %q (want RS256)", domainauth.ErrTokenInvalid, header.Algorithm)
	}

	// Decode and parse payload.
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: decoding JWT payload", domainauth.ErrTokenInvalid)
	}
	var claims jwtClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("%w: parsing JWT payload", domainauth.ErrTokenInvalid)
	}

	// Validate claims before fetching keys (fail fast on structural errors).
	if err := v.validateClaims(claims); err != nil {
		return nil, err
	}

	// Fetch the public key matching the token's kid.
	pubKey, err := v.publicKeyForKID(ctx, header.KeyID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domainauth.ErrTokenInvalid, err)
	}

	// Verify the RS256 signature over header.payload.
	if err := verifyRS256(parts[0]+"."+parts[1], parts[2], pubKey); err != nil {
		return nil, fmt.Errorf("%w: signature verification failed", domainauth.ErrTokenInvalid)
	}

	return &domainauth.User{UID: claims.Subject, Email: claims.Email}, nil
}

// validateClaims checks the standard claims required for a Firebase ID token.
func (v *jwksVerifier) validateClaims(c jwtClaims) error {
	if c.Exp <= time.Now().Unix() {
		return domainauth.ErrTokenExpired
	}
	expectedIss := "https://securetoken.google.com/" + v.projectID
	if c.Issuer != expectedIss {
		return fmt.Errorf("%w: unexpected issuer %q", domainauth.ErrTokenInvalid, c.Issuer)
	}
	if c.Audience != v.projectID {
		return fmt.Errorf("%w: unexpected audience %q", domainauth.ErrTokenInvalid, c.Audience)
	}
	if c.Subject == "" {
		return fmt.Errorf("%w: missing subject claim", domainauth.ErrTokenInvalid)
	}
	return nil
}

// publicKeyForKID returns the RSA public key for the given key ID. It serves
// from the in-memory cache when valid; otherwise it refreshes from the certs endpoint.
func (v *jwksVerifier) publicKeyForKID(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Fast path: read lock.
	v.mu.RLock()
	key, ok := v.certs[kid]
	cacheExp := v.cacheExp
	v.mu.RUnlock()

	if ok && time.Now().Before(cacheExp) {
		return key, nil
	}

	// Slow path: refresh the certificate set.
	if err := v.fetchCerts(ctx); err != nil {
		// If the cache has a stale entry, prefer it over a hard failure.
		v.mu.RLock()
		key, ok = v.certs[kid]
		v.mu.RUnlock()
		if ok {
			return key, nil
		}
		return nil, fmt.Errorf("fetching JWKS certs: %w", err)
	}

	v.mu.RLock()
	key, ok = v.certs[kid]
	v.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown key ID %q", kid)
	}
	return key, nil
}

// fetchCerts fetches the Google X.509 certificate set, parses the RSA public keys,
// and updates the in-memory cache with a TTL derived from the Cache-Control header.
func (v *jwksVerifier) fetchCerts(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.certsURL, nil)
	if err != nil {
		return fmt.Errorf("building certs request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching certs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("certs endpoint returned status %d", resp.StatusCode)
	}

	ttl := parseCacheControlMaxAge(resp.Header.Get("Cache-Control"))
	if ttl <= 0 {
		ttl = 3600 // default 1 hour when Cache-Control is absent or unparseable
	}

	// The certs endpoint returns {"<kid>": "<PEM certificate>", ...}
	var pemMap map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&pemMap); err != nil {
		return fmt.Errorf("decoding certs response: %w", err)
	}

	certs := make(map[string]*rsa.PublicKey, len(pemMap))
	for kid, pemStr := range pemMap {
		pub, err := parseRSAPublicKeyFromPEM([]byte(pemStr))
		if err != nil {
			return fmt.Errorf("parsing cert for kid %q: %w", kid, err)
		}
		certs[kid] = pub
	}

	v.mu.Lock()
	v.certs = certs
	v.cacheExp = time.Now().Add(time.Duration(ttl) * time.Second)
	v.mu.Unlock()

	return nil
}

// verifyRS256 verifies an RS256 JWT signature. signingInput is "header.payload"
// (the raw base64url parts joined by "."), signatureB64URL is the base64url-encoded
// signature, and pubKey is the RSA public key to verify against.
func verifyRS256(signingInput, signatureB64URL string, pubKey *rsa.PublicKey) error {
	sig, err := base64.RawURLEncoding.DecodeString(signatureB64URL)
	if err != nil {
		return fmt.Errorf("decoding signature: %w", err)
	}
	digest := sha256.Sum256([]byte(signingInput))
	return rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, digest[:], sig)
}

// parseRSAPublicKeyFromPEM parses an RSA public key from a PEM-encoded X.509 certificate.
func parseRSAPublicKeyFromPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing X.509 certificate: %w", err)
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("certificate does not contain an RSA public key")
	}
	return pub, nil
}

// parseCacheControlMaxAge extracts the max-age value (in seconds) from a Cache-Control
// header value. Returns 0 when the header is absent or max-age is not present.
func parseCacheControlMaxAge(header string) int {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "max-age=") {
			v, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
			if err == nil && v > 0 {
				return v
			}
		}
	}
	return 0
}
