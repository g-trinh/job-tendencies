package handler

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
