// Package handler provides the chi HTTP router, middleware stack, and error-mapping
// helpers for the Job Tendencies API.
package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

type contextKey string

const activeProfileKey contextKey = "active_profile_id"

// activeProfileHeader is the request header that carries the active profile id.
const activeProfileHeader = "X-Active-Profile"

// ActiveProfileID extracts the active profile ID from the request context.
// It is set by requireActiveProfile middleware on scoped routes.
// Returns the ID and true when present; empty string and false otherwise.
func ActiveProfileID(r *http.Request) (kernel.ProfileID, bool) {
	id, ok := r.Context().Value(activeProfileKey).(kernel.ProfileID)
	return id, ok && id != ""
}

// requireActiveProfile is middleware that reads the X-Active-Profile header and
// stores the profile ID in the request context. It returns 400 when the header
// is missing or empty. This middleware is applied to all scoped routes.
func requireActiveProfile(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(activeProfileHeader)
		if id == "" {
			RespondError(w, r, &kernel.ValidationError{
				Field:   activeProfileHeader,
				Message: "header is required for this endpoint",
			})
			return
		}
		ctx := r.Context()
		ctx = contextWithProfileID(ctx, kernel.ProfileID(id))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requestLogger returns a middleware that writes a structured slog log line for
// every request. It logs method, path, status code, and elapsed duration.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				logger.InfoContext(r.Context(), "http request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"duration_ms", time.Since(start).Milliseconds(),
					"request_id", middleware.GetReqID(r.Context()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
