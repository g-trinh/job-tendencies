package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/api/idtoken"
)

// TokenPayload carries the verified claims from an OIDC token. It is stored in
// the request context by OIDCMiddleware and retrieved by push handlers.
type TokenPayload struct {
	// Email is the service account email extracted from the "email" claim.
	Email string
}

type oidcPayloadKey struct{}

// TokenPayloadFromContext retrieves the verified OIDC token payload injected by
// OIDCMiddleware. Returns nil when no token has been verified on this request.
func TokenPayloadFromContext(ctx context.Context) *TokenPayload {
	p, _ := ctx.Value(oidcPayloadKey{}).(*TokenPayload)
	return p
}

// TokenVerifier validates an OIDC ID token string against the given audience and
// returns a TokenPayload carrying the verified claims.
// The real implementation delegates to google.golang.org/api/idtoken.Validate.
// Tests inject a fake that avoids network calls.
type TokenVerifier interface {
	Verify(ctx context.Context, token, audience string) (*TokenPayload, error)
}

// GoogleTokenVerifier implements TokenVerifier using Google's idtoken package.
// It validates the token signature, expiry, issuer, and audience.
type GoogleTokenVerifier struct{}

// Verify validates an OIDC token using Google's public JWKS endpoint. It returns
// a TokenPayload carrying the email claim on success, or an error when validation
// fails for any reason (expired, wrong audience, bad signature, etc.).
func (GoogleTokenVerifier) Verify(ctx context.Context, token, audience string) (*TokenPayload, error) {
	payload, err := idtoken.Validate(ctx, token, audience)
	if err != nil {
		return nil, fmt.Errorf("validating oidc token: %w", err)
	}

	email, _ := payload.Claims["email"].(string)
	return &TokenPayload{Email: email}, nil
}

// OIDCMiddleware returns a chi middleware that:
//  1. Extracts the Bearer token from the Authorization header.
//  2. Validates it using verifier against audience.
//  3. Checks that the token's email claim matches allowedSA.
//  4. Stores the payload in the request context and calls next.
//
// It returns 401 when the Authorization header is missing or the token is invalid,
// and 403 when the token is valid but the service account is not authorised.
func OIDCMiddleware(verifier TokenVerifier, audience, allowedSA string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "unauthorized: missing bearer token", http.StatusUnauthorized)
				return
			}

			rawToken := strings.TrimPrefix(authHeader, "Bearer ")
			payload, err := verifier.Verify(r.Context(), rawToken, audience)
			if err != nil {
				http.Error(w, "unauthorized: token validation failed", http.StatusUnauthorized)
				return
			}

			if payload.Email != allowedSA {
				http.Error(w, "forbidden: service account not authorised", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), oidcPayloadKey{}, payload)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
