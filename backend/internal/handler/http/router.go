package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter constructs the root chi router with base middleware: request ID,
// panic recovery, and structured slog request logging. It does not mount any
// domain routes — callers add routes after construction.
//
// Example:
//
//	r := handler.NewRouter(logger)
//	r.Mount("/api", handler.ScopedRouter(r))
func NewRouter(logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(logger))
	return r
}

// ScopedRoutes returns a sub-router where all mounted routes require a valid
// X-Active-Profile header. Use this for all endpoints that are scoped to the
// active search profile (jobs, dashboard, contacts, application kanban).
func ScopedRoutes(r chi.Router) chi.Router {
	return r.With(requireActiveProfile)
}

// NewCORSMiddleware returns a CORS middleware configured for the Job Tendencies
// API. It restricts cross-origin access to the explicit list of origins supplied
// at startup (from ALLOWED_ORIGINS), allows the custom X-Active-Profile header
// required by scoped routes, and handles OPTIONS preflight requests.
//
// Mounting this only on the /api sub-router keeps health-check endpoints (/healthz,
// /livez) out of CORS scope, and ensures the scrape-worker and extract-worker
// binaries — which share NewRouter but never add this middleware — are unaffected.
//
// When allowedOrigins is empty, the middleware is a no-op passthrough: no CORS
// headers are added and no wildcard fallback is used. This is the fail-safe
// behaviour for deployments that do not need cross-origin access.
//
// Example:
//
//	r.Route("/api", func(api chi.Router) {
//	    api.Use(handler.NewCORSMiddleware(cfg.AllowedOrigins))
//	    // ... domain routes
//	})
func NewCORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	if len(allowedOrigins) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders: []string{"Accept", "Content-Type", "X-Active-Profile"},
		MaxAge:         300,
	})
}
