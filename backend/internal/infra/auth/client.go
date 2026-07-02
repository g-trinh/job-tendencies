// Package auth provides the Identity Platform REST client that implements the
// domain/auth port interfaces ([domainauth.IDPClient] and [domainauth.TokenVerifier]).
// It wraps the Firebase Identity Platform REST API: sign-in with email/password,
// ID token refresh, and RS256 signature verification against Google's public JWKS.
//
// All identity traffic flows through this package; the browser never calls Firebase
// directly (see ADR-001 and the Phase-4 tech breakdown).
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	domainauth "github.com/g-trinh/job-tendencies/internal/domain/auth"
)

const (
	signInURL  = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword"
	refreshURL = "https://securetoken.googleapis.com/v1/token"
)

// Client wraps the Identity Platform REST API and implements both
// [domainauth.IDPClient] and [domainauth.TokenVerifier]. Use [NewClient] to construct.
//
// Example:
//
//	client, err := auth.NewClient(apiKey, projectID)
//	result, err := client.SignIn(ctx, email, password)
//	user, err := client.VerifyIDToken(ctx, result.IDToken)
type Client struct {
	apiKey     string
	projectID  string
	httpClient *http.Client
	verifier   *jwksVerifier
}

// Option is a functional option for [NewClient].
type Option func(*Client)

// WithHTTPClient replaces the default http.Client used for all Identity Platform
// and JWKS requests. Useful in tests that need a controlled transport.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
		cl.verifier.httpClient = c
	}
}

// WithCertsURL overrides the Google X.509 certificate URL used by the JWKS verifier.
// Useful in tests that serve a fake certificate endpoint via httptest.Server.
func WithCertsURL(url string) Option {
	return func(cl *Client) {
		cl.verifier.certsURL = url
	}
}

// NewClient constructs an Identity Platform [Client] with the given API key and GCP
// project ID. Returns an error when either field is empty. Options are applied after
// the default configuration so they can override defaults (e.g. HTTP client, cert URL).
func NewClient(apiKey, projectID string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("auth: apiKey is required")
	}
	if projectID == "" {
		return nil, fmt.Errorf("auth: projectID is required")
	}

	hc := &http.Client{Timeout: 10 * time.Second}
	c := &Client{
		apiKey:     apiKey,
		projectID:  projectID,
		httpClient: hc,
		verifier:   newJWKSVerifier(projectID, hc),
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// SignIn authenticates email and password against Identity Platform and returns the
// resulting tokens. Returns [domainauth.ErrInvalidCredentials] on wrong credentials.
func (c *Client) SignIn(ctx context.Context, email, password string) (*domainauth.SignInResult, error) {
	body, err := json.Marshal(signInRequest{Email: email, Password: password, ReturnSecureToken: true})
	if err != nil {
		return nil, fmt.Errorf("auth: marshalling sign-in request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, signInURL+"?key="+c.apiKey, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("auth: building sign-in request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth: sign-in: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, parseIDPError(resp.Body)
	}

	var sr signInResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("auth: decoding sign-in response: %w", err)
	}

	return &domainauth.SignInResult{
		User:         domainauth.User{UID: sr.LocalID, Email: sr.Email},
		IDToken:      sr.IDToken,
		RefreshToken: sr.RefreshToken,
		ExpiresAt:    expiresAtFromSeconds(sr.ExpiresIn),
	}, nil
}

// RefreshIDToken exchanges a refresh token for a new ID token. Returns
// [domainauth.ErrInvalidCredentials] when the refresh token is invalid or revoked.
func (c *Client) RefreshIDToken(ctx context.Context, refreshToken string) (*domainauth.RefreshResult, error) {
	body, err := json.Marshal(refreshRequest{GrantType: "refresh_token", RefreshToken: refreshToken})
	if err != nil {
		return nil, fmt.Errorf("auth: marshalling refresh request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshURL+"?key="+c.apiKey, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("auth: building refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth: refresh: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, parseIDPError(resp.Body)
	}

	var rr refreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return nil, fmt.Errorf("auth: decoding refresh response: %w", err)
	}

	return &domainauth.RefreshResult{
		User:         domainauth.User{UID: rr.UserID},
		IDToken:      rr.IDToken,
		RefreshToken: rr.RefreshToken,
		ExpiresAt:    expiresAtFromSeconds(rr.ExpiresIn),
	}, nil
}

// VerifyIDToken validates an ID token's RS256 signature against Google's JWKS and
// verifies the audience, issuer, and expiry claims. Delegates to the internal JWKS
// verifier which caches certificates according to their Cache-Control TTL.
func (c *Client) VerifyIDToken(ctx context.Context, idToken string) (*domainauth.User, error) {
	return c.verifier.verify(ctx, idToken)
}

// --- JSON shapes for Identity Platform REST ---

// signInRequest is the body for the signInWithPassword endpoint.
type signInRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

// signInResponse is the success body from the signInWithPassword endpoint.
type signInResponse struct {
	LocalID      string `json:"localId"`
	Email        string `json:"email"`
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"` // seconds as string
}

// refreshRequest is the body for the token refresh endpoint.
type refreshRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
}

// refreshResponse is the success body from the token refresh endpoint.
type refreshResponse struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	ExpiresIn    string `json:"expires_in"` // seconds as string
}

// idpErrorResponse is the error body returned by Identity Platform on 4xx/5xx.
type idpErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// parseIDPError reads the IDP error body and maps known error messages to domain errors.
func parseIDPError(body io.Reader) error {
	var e idpErrorResponse
	if err := json.NewDecoder(body).Decode(&e); err != nil {
		return domainauth.ErrInvalidCredentials
	}
	switch e.Error.Message {
	case "INVALID_PASSWORD", "EMAIL_NOT_FOUND", "INVALID_LOGIN_CREDENTIALS", "USER_DISABLED",
		"INVALID_REFRESH_TOKEN", "TOKEN_EXPIRED":
		return domainauth.ErrInvalidCredentials
	default:
		return fmt.Errorf("auth: IDP error %d: %s: %w", e.Error.Code, e.Error.Message, domainauth.ErrInvalidCredentials)
	}
}

// expiresAtFromSeconds parses the "expiresIn" field (seconds as a string) and returns
// the absolute expiry time relative to now. Defaults to 1 hour when unparseable.
func expiresAtFromSeconds(s string) time.Time {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		n = 3600
	}
	return time.Now().Add(time.Duration(n) * time.Second)
}
