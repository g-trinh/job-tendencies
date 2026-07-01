package handler

import (
	"context"
	"net/http"

	domainauth "github.com/g-trinh/job-tendencies/internal/domain/auth"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// RequireActiveProfileForTest exposes the internal requireActiveProfile middleware
// for use in package-external tests without requiring a live server.
func RequireActiveProfileForTest(next http.Handler) http.Handler {
	return requireActiveProfile(next)
}

// WithActiveProfileID injects a profile id into the context for use in unit tests
// that bypass the requireActiveProfile middleware.
func WithActiveProfileID(ctx context.Context, id kernel.ProfileID) context.Context {
	return contextWithProfileID(ctx, id)
}

// WithAuthUserForTest injects an authenticated user into the context for use in
// unit tests that bypass the RequireAuth middleware.
func WithAuthUserForTest(ctx context.Context, user domainauth.User) context.Context {
	return contextWithAuthUser(ctx, user)
}

// RequireAuthForTest exposes RequireAuth for use in package-external middleware tests.
func RequireAuthForTest(svc AuthService, cookieSecure bool) func(http.Handler) http.Handler {
	return RequireAuth(svc, cookieSecure)
}

// RequireCSRFForTest exposes RequireCSRF for use in package-external middleware tests.
func RequireCSRFForTest(allowedOrigins []string) func(http.Handler) http.Handler {
	return RequireCSRF(allowedOrigins)
}
