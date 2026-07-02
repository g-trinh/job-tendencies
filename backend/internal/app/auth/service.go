// Package auth provides the authentication application service that orchestrates
// login, session decoding, and token refresh for the Job Tendencies API.
//
// Session model: the Identity Platform refresh token is stored inside an
// AES-256-GCM-encrypted httpOnly cookie (no server-side session store). On each
// authenticated request the middleware decodes the cookie, verifies the ID token,
// and transparently refreshes it when it is within 5 minutes of expiry.
package auth

import (
	"context"
	"fmt"
	"time"

	domainauth "github.com/g-trinh/job-tendencies/internal/domain/auth"
)

// LoginResult is the output of [Service.Login].
type LoginResult struct {
	// User is the authenticated identity returned by Identity Platform.
	User domainauth.User
	// CookieValue is the AES-256-GCM-encrypted, base64url-encoded session payload
	// to set as the httpOnly session cookie value. The browser receives no token.
	CookieValue string
}

// MeResult is the output of [Service.Me].
type MeResult struct {
	// User is the authenticated identity from the verified session.
	User domainauth.User
	// UpdatedCookieValue is non-empty when the session's ID token was transparently
	// refreshed near expiry. The caller must overwrite the session cookie with this value.
	UpdatedCookieValue string
}

// Service orchestrates authentication use cases: login, session verification, and
// transparent token refresh. Dependencies are injected; no global state.
type Service struct {
	idp       domainauth.IDPClient
	verifier  domainauth.TokenVerifier
	cookieKey []byte // 32-byte AES-256-GCM key
}

// New constructs an auth [Service]. cookieKey must be exactly 32 bytes
// (AES-256-GCM requirement); returns an error otherwise.
func New(idp domainauth.IDPClient, verifier domainauth.TokenVerifier, cookieKey []byte) (*Service, error) {
	if idp == nil {
		return nil, fmt.Errorf("auth: idp client is required")
	}
	if verifier == nil {
		return nil, fmt.Errorf("auth: token verifier is required")
	}
	if len(cookieKey) != 32 {
		return nil, fmt.Errorf("auth: cookieKey must be exactly 32 bytes for AES-256-GCM; got %d", len(cookieKey))
	}
	return &Service{idp: idp, verifier: verifier, cookieKey: cookieKey}, nil
}

// Login authenticates email and password against Identity Platform. On success it
// returns the encrypted session cookie value (to be written as an httpOnly cookie)
// and the authenticated user. The browser receives no token — only the cookie.
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	result, err := s.idp.SignIn(ctx, email, password)
	if err != nil {
		return nil, fmt.Errorf("auth: sign in: %w", err)
	}

	sd := &sessionData{
		UID:          result.User.UID,
		Email:        result.User.Email,
		RefreshToken: result.RefreshToken,
		IDToken:      result.IDToken,
		ExpiresAt:    result.ExpiresAt.Unix(),
	}
	cookieValue, err := encodeSession(s.cookieKey, sd)
	if err != nil {
		return nil, fmt.Errorf("auth: encoding session: %w", err)
	}

	return &LoginResult{User: result.User, CookieValue: cookieValue}, nil
}

// Me decodes the encrypted session cookie, optionally refreshes the ID token when it
// is within 5 minutes of expiry, and returns the verified user. When the session was
// refreshed, UpdatedCookieValue is non-empty and the caller must set a new cookie.
//
// Returns [domainauth.ErrTokenInvalid] when the cookie cannot be decoded or decrypted.
// Returns [domainauth.ErrTokenExpired] when the refresh token itself has expired.
func (s *Service) Me(ctx context.Context, cookieValue string) (*MeResult, error) {
	sd, err := decodeSession(s.cookieKey, cookieValue)
	if err != nil {
		// Wrap as token-invalid so the middleware maps it to 401.
		return nil, fmt.Errorf("%w: decoding session cookie", domainauth.ErrTokenInvalid)
	}

	idToken := sd.IDToken
	var updatedCookieValue string

	// Refresh the ID token when it is within 5 minutes of expiry, so most requests
	// never hit an expired token (transparent refresh from the user's perspective).
	if time.Until(sessionExpiresAt(sd)) < 5*time.Minute {
		refreshResult, err := s.idp.RefreshIDToken(ctx, sd.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("auth: refreshing session: %w", err)
		}
		sd.IDToken = refreshResult.IDToken
		sd.RefreshToken = refreshResult.RefreshToken
		sd.ExpiresAt = refreshResult.ExpiresAt.Unix()
		idToken = refreshResult.IDToken

		updatedCookieValue, err = encodeSession(s.cookieKey, sd)
		if err != nil {
			return nil, fmt.Errorf("auth: encoding refreshed session: %w", err)
		}
	}

	user, err := s.verifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("auth: verifying session: %w", err)
	}

	return &MeResult{User: *user, UpdatedCookieValue: updatedCookieValue}, nil
}
