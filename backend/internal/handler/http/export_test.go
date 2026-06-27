package handler

import (
	"context"
	"net/http"

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
