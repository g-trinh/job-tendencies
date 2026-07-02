// Package auth defines the authentication domain types, port interfaces, and errors
// for the Job Tendencies API. All identity calls are proxied through the backend;
// the browser never holds a token — only an httpOnly session cookie.
//
// Bounded context: auth sits outside the core domain contexts (boards, jobs, etc.)
// and is consumed only by the app/auth service and the HTTP middleware layer.
package auth

import (
	"context"
	"errors"
	"time"
)

// User is the authenticated identity resolved from a verified Identity Platform ID token.
type User struct {
	// UID is the unique user identifier assigned by Identity Platform.
	UID string
	// Email is the user's verified email address.
	Email string
}

// SignInResult holds the tokens returned by Identity Platform after a successful sign-in.
type SignInResult struct {
	// User is the authenticated identity.
	User User
	// IDToken is the Firebase ID token (JWT, valid for 1 hour).
	IDToken string
	// RefreshToken is the long-lived token used to obtain new ID tokens.
	RefreshToken string
	// ExpiresAt is when the ID token expires.
	ExpiresAt time.Time
}

// RefreshResult holds the tokens returned after refreshing an ID token.
type RefreshResult struct {
	// User is the authenticated identity. Email may be empty (not returned by the
	// refresh endpoint); callers should use the value stored in the session.
	User User
	// IDToken is the new Firebase ID token (JWT, valid for 1 hour).
	IDToken string
	// RefreshToken is the (possibly rotated) refresh token.
	RefreshToken string
	// ExpiresAt is when the new ID token expires.
	ExpiresAt time.Time
}

var (
	// ErrInvalidCredentials is returned when the email or password supplied to SignIn
	// is incorrect, or when a refresh token is revoked or invalid.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrTokenExpired is returned when an ID token has passed its expiry time.
	ErrTokenExpired = errors.New("token expired")

	// ErrTokenInvalid is returned when an ID token fails signature or claim validation
	// (wrong audience, wrong issuer, forged signature, or malformed JWT).
	ErrTokenInvalid = errors.New("token invalid")
)

// IDPClient wraps the Identity Platform REST API for sign-in and token refresh.
// The concrete implementation lives in internal/infra/auth and satisfies this
// interface implicitly. Consumed by internal/app/auth.
type IDPClient interface {
	// SignIn authenticates the supplied email and password and returns tokens on success.
	// Returns ErrInvalidCredentials when the credentials are wrong.
	SignIn(ctx context.Context, email, password string) (*SignInResult, error)

	// RefreshIDToken exchanges a refresh token for a fresh ID token.
	// Returns ErrInvalidCredentials when the refresh token is revoked or invalid.
	RefreshIDToken(ctx context.Context, refreshToken string) (*RefreshResult, error)
}

// TokenVerifier verifies a Firebase ID token and returns the authenticated user.
// The concrete implementation lives in internal/infra/auth and satisfies this
// interface implicitly. Consumed by internal/app/auth.
type TokenVerifier interface {
	// VerifyIDToken validates the token's RS256 signature against Google's public JWKS
	// and verifies the audience, issuer, and expiry claims.
	// Returns ErrTokenExpired when the token is expired, ErrTokenInvalid for all other failures.
	VerifyIDToken(ctx context.Context, idToken string) (*User, error)
}
