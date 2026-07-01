// Package handler provides the chi HTTP router, middleware stack, and error-mapping
// helpers for the Job Tendencies API.
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	domainauth "github.com/g-trinh/job-tendencies/internal/domain/auth"
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

// RequireAuth returns middleware that validates the session cookie on every request.
// When the session's ID token is within 5 minutes of expiry, it is transparently
// refreshed and the caller receives an updated cookie in the response. The authenticated
// user is stored in the request context and retrieved via [AuthUser].
//
// Routes that must remain public (e.g. POST /api/auth/login) should be registered
// outside the router group that mounts this middleware.
//
// Returns 401 when the cookie is absent, invalid, or cannot be decrypted.
func RequireAuth(svc AuthService, cookieSecure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				// http.ErrNoCookie is the only possible error from r.Cookie.
				RespondError(w, r, kernel.ErrUnauthorized)
				return
			}

			result, err := svc.Me(r.Context(), cookie.Value)
			if err != nil {
				switch {
				case errors.Is(err, domainauth.ErrTokenInvalid),
					errors.Is(err, domainauth.ErrTokenExpired),
					errors.Is(err, domainauth.ErrInvalidCredentials):
					RespondError(w, r, kernel.ErrUnauthorized)
				default:
					RespondError(w, r, kernel.ErrUnauthorized)
				}
				return
			}

			// Propagate the refreshed cookie when the session was transparently renewed.
			if result.UpdatedCookieValue != "" {
				http.SetCookie(w, newSessionCookie(result.UpdatedCookieValue, cookieSecure))
			}

			ctx := contextWithAuthUser(r.Context(), result.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireCSRF returns middleware that protects state-changing requests against
// cross-site request forgery using a SameSite=Strict cookie combined with an
// Origin/Referer header check.
//
// For state-changing methods (POST, PUT, PATCH, DELETE): if the request carries
// an Origin header that is not in allowedOrigins (when that list is non-empty),
// the request is rejected with 403. Requests without an Origin header (e.g. curl,
// Postman, or same-origin browser requests that omit Origin) are allowed through.
//
// SameSite=Strict on the session cookie already prevents cross-site cookie delivery;
// the origin check adds defence-in-depth for deployments that configure AllowedOrigins.
func RequireCSRF(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(allowedOrigins) > 0 && isStateChangingMethod(r.Method) {
				origin := r.Header.Get("Origin")
				if origin == "" {
					// Fall back to extracting the origin from Referer.
					origin = refererOrigin(r.Header.Get("Referer"))
				}
				if origin != "" && !originAllowed(origin, allowedOrigins) {
					http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isStateChangingMethod reports whether m is an HTTP method that mutates server state.
func isStateChangingMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

// originAllowed reports whether origin is contained in the allowed list.
func originAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}

// refererOrigin extracts the scheme+host from a Referer URL, returning an empty
// string when the Referer is absent or not a valid URL.
func refererOrigin(referer string) string {
	if referer == "" {
		return ""
	}
	// Extract scheme://host from the Referer without importing net/url to keep it simple.
	// Referer format: https://host/path
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(referer, prefix) {
			rest := referer[len(prefix):]
			if idx := strings.IndexByte(rest, '/'); idx != -1 {
				return prefix + rest[:idx]
			}
			return prefix + rest
		}
	}
	return ""
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
