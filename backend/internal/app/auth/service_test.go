package auth_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appauth "github.com/g-trinh/job-tendencies/internal/app/auth"
	domainauth "github.com/g-trinh/job-tendencies/internal/domain/auth"
)

// testCookieKey is a 32-byte key used for all service tests.
var testCookieKey = bytes.Repeat([]byte("k"), 32)

// --- Test doubles ---

type fakeIDPClient struct {
	signInResult  *domainauth.SignInResult
	signInErr     error
	refreshResult *domainauth.RefreshResult
	refreshErr    error
}

func (f *fakeIDPClient) SignIn(_ context.Context, _, _ string) (*domainauth.SignInResult, error) {
	return f.signInResult, f.signInErr
}

func (f *fakeIDPClient) RefreshIDToken(_ context.Context, _ string) (*domainauth.RefreshResult, error) {
	return f.refreshResult, f.refreshErr
}

type fakeTokenVerifier struct {
	user *domainauth.User
	err  error
}

func (f *fakeTokenVerifier) VerifyIDToken(_ context.Context, _ string) (*domainauth.User, error) {
	return f.user, f.err
}

// --- Login tests ---

// AC: Successful login sets an httpOnly cookie value and returns the user (no token).

func TestService_Login(t *testing.T) {
	t.Parallel()

	const (
		uid   = "uid-123"
		email = "user@example.com"
	)
	validSignInResult := &domainauth.SignInResult{
		User:         domainauth.User{UID: uid, Email: email},
		IDToken:      "id-token-value",
		RefreshToken: "refresh-token-value",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	cases := []struct {
		name       string
		idpResult  *domainauth.SignInResult
		idpErr     error
		wantUID    string
		wantEmail  string
		wantCookie bool // true = non-empty cookie value expected
		wantErr    error
	}{
		{
			name:       "valid credentials return user and encrypted cookie",
			idpResult:  validSignInResult,
			wantUID:    uid,
			wantEmail:  email,
			wantCookie: true,
		},
		{
			name:    "invalid credentials return ErrInvalidCredentials",
			idpErr:  domainauth.ErrInvalidCredentials,
			wantErr: domainauth.ErrInvalidCredentials,
		},
		{
			name:    "IDP transport error is propagated",
			idpErr:  errors.New("connection refused"),
			wantErr: errors.New("connection refused"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			idp := &fakeIDPClient{signInResult: tc.idpResult, signInErr: tc.idpErr}
			verifier := &fakeTokenVerifier{}
			svc, err := appauth.New(idp, verifier, testCookieKey)
			require.NoError(t, err)

			result, err := svc.Login(context.Background(), "user@example.com", "password")

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.wantUID, result.User.UID)
			assert.Equal(t, tc.wantEmail, result.User.Email)
			if tc.wantCookie {
				assert.NotEmpty(t, result.CookieValue, "cookie value must not be empty on success")
			}
		})
	}
}

// --- Me tests ---

// AC: GET /api/auth/me returns the user when the cookie is valid, 401 otherwise.

func TestService_Me_ValidCookie(t *testing.T) {
	t.Parallel()

	const (
		uid   = "uid-abc"
		email = "user@example.com"
	)

	// First perform a login to obtain a real encrypted cookie.
	idp := &fakeIDPClient{signInResult: &domainauth.SignInResult{
		User:         domainauth.User{UID: uid, Email: email},
		IDToken:      "fresh-id-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour), // not near expiry → no refresh
	}}
	verifier := &fakeTokenVerifier{user: &domainauth.User{UID: uid, Email: email}}
	svc, err := appauth.New(idp, verifier, testCookieKey)
	require.NoError(t, err)

	loginResult, err := svc.Login(context.Background(), email, "pass")
	require.NoError(t, err)

	// Me should resolve the same user.
	meResult, err := svc.Me(context.Background(), loginResult.CookieValue)
	require.NoError(t, err)
	require.NotNil(t, meResult)
	assert.Equal(t, uid, meResult.User.UID)
	assert.Equal(t, email, meResult.User.Email)
	assert.Empty(t, meResult.UpdatedCookieValue, "no refresh needed when token is fresh")
}

func TestService_Me_InvalidCookie(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		cookieValue string
		wantErr     error
	}{
		{
			name:        "empty cookie value returns ErrTokenInvalid",
			cookieValue: "",
			wantErr:     domainauth.ErrTokenInvalid,
		},
		{
			name:        "garbage cookie value returns ErrTokenInvalid",
			cookieValue: "not-a-valid-encrypted-cookie",
			wantErr:     domainauth.ErrTokenInvalid,
		},
		{
			name:        "base64 cookie with wrong key returns ErrTokenInvalid",
			cookieValue: "dGhpcyBpcyBub3QgZW5jcnlwdGVkIGNvcnJlY3RseQ", // valid base64url, wrong content
			wantErr:     domainauth.ErrTokenInvalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			idp := &fakeIDPClient{}
			verifier := &fakeTokenVerifier{}
			svc, err := appauth.New(idp, verifier, testCookieKey)
			require.NoError(t, err)

			result, err := svc.Me(context.Background(), tc.cookieValue)

			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, result)
		})
	}
}

func TestService_Me_TransparentRefreshNearExpiry(t *testing.T) {
	t.Parallel()

	const uid, email = "uid-refresh", "refresh@example.com"

	// Login with a near-expiry token (within 5 min window).
	idp := &fakeIDPClient{
		signInResult: &domainauth.SignInResult{
			User:         domainauth.User{UID: uid, Email: email},
			IDToken:      "expiring-id-token",
			RefreshToken: "old-refresh-token",
			ExpiresAt:    time.Now().Add(2 * time.Minute), // inside the 5-min window
		},
		refreshResult: &domainauth.RefreshResult{
			User:         domainauth.User{UID: uid},
			IDToken:      "new-id-token",
			RefreshToken: "new-refresh-token",
			ExpiresAt:    time.Now().Add(time.Hour),
		},
	}
	verifier := &fakeTokenVerifier{user: &domainauth.User{UID: uid, Email: email}}
	svc, err := appauth.New(idp, verifier, testCookieKey)
	require.NoError(t, err)

	loginResult, err := svc.Login(context.Background(), email, "pass")
	require.NoError(t, err)

	meResult, err := svc.Me(context.Background(), loginResult.CookieValue)
	require.NoError(t, err)
	require.NotNil(t, meResult)
	assert.Equal(t, uid, meResult.User.UID)
	assert.NotEmpty(t, meResult.UpdatedCookieValue, "refreshed session must produce an updated cookie value")
	assert.NotEqual(t, loginResult.CookieValue, meResult.UpdatedCookieValue,
		"updated cookie value must differ from original")
}

func TestService_Me_VerifierError_Returns401(t *testing.T) {
	t.Parallel()

	const uid, email = "uid-bad-token", "bad@example.com"

	idp := &fakeIDPClient{signInResult: &domainauth.SignInResult{
		User:         domainauth.User{UID: uid, Email: email},
		IDToken:      "invalid-id-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}}
	// Verifier rejects the ID token (e.g. forged signature).
	verifier := &fakeTokenVerifier{err: domainauth.ErrTokenInvalid}
	svc, err := appauth.New(idp, verifier, testCookieKey)
	require.NoError(t, err)

	loginResult, err := svc.Login(context.Background(), email, "pass")
	require.NoError(t, err)

	meResult, err := svc.Me(context.Background(), loginResult.CookieValue)

	require.Error(t, err)
	assert.Nil(t, meResult)
}

// --- Constructor tests ---

func TestNew_RequiresValidKey(t *testing.T) {
	t.Parallel()

	idp := &fakeIDPClient{}
	verifier := &fakeTokenVerifier{}

	cases := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{name: "32-byte key is valid", key: bytes.Repeat([]byte("k"), 32), wantErr: false},
		{name: "31-byte key returns error", key: bytes.Repeat([]byte("k"), 31), wantErr: true},
		{name: "33-byte key returns error", key: bytes.Repeat([]byte("k"), 33), wantErr: true},
		{name: "empty key returns error", key: nil, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, err := appauth.New(idp, verifier, tc.key)
			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, svc)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, svc)
			}
		})
	}
}
