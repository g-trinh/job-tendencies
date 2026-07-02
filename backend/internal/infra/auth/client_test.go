package auth_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainauth "github.com/g-trinh/job-tendencies/internal/domain/auth"
	infraauth "github.com/g-trinh/job-tendencies/internal/infra/auth"
)

// AC: Token verification rejects wrong aud/iss/expired/forged sig (unit-tested).
// AC: Invalid credentials/expired tokens are surfaced as typed errors (no panic).

func TestClient_VerifyIDToken(t *testing.T) {
	t.Parallel()

	const (
		projectID = "test-project"
		kid       = "test-key-id"
		uid       = "test-uid-abc"
		email     = "user@example.com"
	)

	// Generate the key pair used by the JWKS server (the "correct" key).
	correctKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "generating correct RSA key")

	// Generate a second key pair to produce a forged signature.
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "generating wrong RSA key")

	certPEM, err := selfSignedCertPEM(correctKey)
	require.NoError(t, err, "generating self-signed cert")

	// Fake JWKS server: returns correctKey's certificate indexed by kid.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{kid: string(certPEM)})
	}))
	t.Cleanup(srv.Close)

	validExp := time.Now().Add(time.Hour)
	expiredExp := time.Now().Add(-time.Minute)

	cases := []struct {
		name    string
		token   string
		wantUID string
		wantErr error
	}{
		{
			name:    "valid token returns authenticated user",
			token:   buildRS256JWT(t, correctKey, kid, projectID, uid, email, validExp),
			wantUID: uid,
		},
		{
			name:    "expired token returns ErrTokenExpired",
			token:   buildRS256JWT(t, correctKey, kid, projectID, uid, email, expiredExp),
			wantErr: domainauth.ErrTokenExpired,
		},
		{
			name:    "wrong audience returns ErrTokenInvalid",
			token:   buildRS256JWTWithAud(t, correctKey, kid, projectID, uid, email, validExp, "wrong-project"),
			wantErr: domainauth.ErrTokenInvalid,
		},
		{
			name:    "wrong issuer returns ErrTokenInvalid",
			token:   buildRS256JWTWithIss(t, correctKey, kid, projectID, uid, email, validExp, "https://evil.com/"+projectID),
			wantErr: domainauth.ErrTokenInvalid,
		},
		{
			name:    "forged signature (different key) returns ErrTokenInvalid",
			token:   buildRS256JWT(t, wrongKey, kid, projectID, uid, email, validExp),
			wantErr: domainauth.ErrTokenInvalid,
		},
		{
			name:    "only two JWT parts returns ErrTokenInvalid",
			token:   "header.payload",
			wantErr: domainauth.ErrTokenInvalid,
		},
		{
			name:    "malformed header returns ErrTokenInvalid",
			token:   "notbase64!.payload.signature",
			wantErr: domainauth.ErrTokenInvalid,
		},
		{
			name:    "empty sub (missing subject) returns ErrTokenInvalid",
			token:   buildRS256JWTWithSub(t, correctKey, kid, projectID, email, validExp, ""),
			wantErr: domainauth.ErrTokenInvalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, err := infraauth.NewClient("fake-api-key", projectID,
				infraauth.WithCertsURL(srv.URL),
				infraauth.WithHTTPClient(srv.Client()),
			)
			require.NoError(t, err)

			user, err := client.VerifyIDToken(context.Background(), tc.token)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "expected error type")
				assert.Nil(t, user, "user must be nil on error")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, user)
			assert.Equal(t, tc.wantUID, user.UID, "user UID")
			assert.Equal(t, email, user.Email, "user email")
		})
	}
}

func TestNewClient_RequiresAPIKeyAndProjectID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		apiKey    string
		projectID string
		wantErr   bool
	}{
		{name: "valid params creates client", apiKey: "key", projectID: "proj", wantErr: false},
		{name: "empty apiKey returns error", apiKey: "", projectID: "proj", wantErr: true},
		{name: "empty projectID returns error", apiKey: "key", projectID: "", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, err := infraauth.NewClient(tc.apiKey, tc.projectID)
			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// --- JWT construction helpers ---

// buildRS256JWT creates a signed Firebase-compatible JWT with standard claims.
func buildRS256JWT(t *testing.T, key *rsa.PrivateKey, kid, projectID, uid, email string, expiresAt time.Time) string {
	t.Helper()
	return signJWT(t, key, kid, map[string]interface{}{
		"iss":   "https://securetoken.google.com/" + projectID,
		"aud":   projectID,
		"sub":   uid,
		"email": email,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
	})
}

// buildRS256JWTWithAud creates a JWT with a custom audience (for wrong-aud tests).
func buildRS256JWTWithAud(t *testing.T, key *rsa.PrivateKey, kid, projectID, uid, email string, expiresAt time.Time, aud string) string {
	t.Helper()
	return signJWT(t, key, kid, map[string]interface{}{
		"iss":   "https://securetoken.google.com/" + projectID,
		"aud":   aud,
		"sub":   uid,
		"email": email,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
	})
}

// buildRS256JWTWithIss creates a JWT with a custom issuer (for wrong-iss tests).
func buildRS256JWTWithIss(t *testing.T, key *rsa.PrivateKey, kid, projectID, uid, email string, expiresAt time.Time, iss string) string {
	t.Helper()
	return signJWT(t, key, kid, map[string]interface{}{
		"iss":   iss,
		"aud":   projectID,
		"sub":   uid,
		"email": email,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
	})
}

// buildRS256JWTWithSub creates a JWT with a custom subject (for missing-sub tests).
func buildRS256JWTWithSub(t *testing.T, key *rsa.PrivateKey, kid, projectID, email string, expiresAt time.Time, sub string) string {
	t.Helper()
	return signJWT(t, key, kid, map[string]interface{}{
		"iss":   "https://securetoken.google.com/" + projectID,
		"aud":   projectID,
		"sub":   sub,
		"email": email,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
	})
}

// signJWT constructs and signs a JWT using RS256.
func signJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]interface{}) string {
	t.Helper()

	header := map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)

	payloadJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(payloadJSON)

	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// selfSignedCertPEM creates a self-signed X.509 certificate for the given RSA key
// and returns it PEM-encoded. Used to simulate the Google JWKS cert endpoint.
func selfSignedCertPEM(key *rsa.PrivateKey) ([]byte, error) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("creating certificate: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), nil
}
