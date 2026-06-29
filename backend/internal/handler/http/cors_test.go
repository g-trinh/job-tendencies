package handler_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// AC: a request from an allowed origin receives Access-Control-Allow-Origin echoed back.
// AC: a request from a disallowed origin receives no CORS headers.
// AC: an OPTIONS preflight listing X-Active-Profile in Access-Control-Request-Headers succeeds.

func TestNewCORSMiddleware_OriginHandling(t *testing.T) {
	t.Parallel()

	const allowedOrigin = "https://job-tendencies-dev.web.app"
	const localOrigin = "http://localhost:5173"
	const disallowedOrigin = "https://evil.example.com"

	cases := []struct {
		name            string
		allowedOrigins  []string
		requestOrigin   string
		wantAllowOrigin string // empty means header must be absent
	}{
		{
			name:            "allowed origin receives Access-Control-Allow-Origin",
			allowedOrigins:  []string{allowedOrigin},
			requestOrigin:   allowedOrigin,
			wantAllowOrigin: allowedOrigin,
		},
		{
			name:            "second origin in list is also allowed",
			allowedOrigins:  []string{allowedOrigin, localOrigin},
			requestOrigin:   localOrigin,
			wantAllowOrigin: localOrigin,
		},
		{
			name:            "disallowed origin receives no Access-Control-Allow-Origin",
			allowedOrigins:  []string{allowedOrigin},
			requestOrigin:   disallowedOrigin,
			wantAllowOrigin: "",
		},
		{
			name:            "no origins configured means no cross-origin access",
			allowedOrigins:  nil,
			requestOrigin:   allowedOrigin,
			wantAllowOrigin: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Mount CORS via api.Use on a sub-router, mirroring how cmd/api wires it,
			// so the middleware runs before chi's method dispatch on every request.
			r := handler.NewRouter(slog.Default())
			r.Route("/api", func(api chi.Router) {
				api.Use(handler.NewCORSMiddleware(tc.allowedOrigins))
				api.Get("/jobs", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			})

			req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
			req.Header.Set("Origin", tc.requestOrigin)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			got := rec.Header().Get("Access-Control-Allow-Origin")
			if got != tc.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q; want %q", got, tc.wantAllowOrigin)
			}
		})
	}
}

func TestNewCORSMiddleware_PreflightWithXActiveProfile(t *testing.T) {
	t.Parallel()

	const allowedOrigin = "https://job-tendencies-dev.web.app"

	cases := []struct {
		name               string
		requestHeaders     string // Access-Control-Request-Headers value
		wantStatus         int
		wantAllowedHeaders bool // Access-Control-Allow-Headers must be non-empty
	}{
		{
			// AC: OPTIONS preflight with X-Active-Profile in request headers succeeds so
			// scoped routes are reachable cross-origin.
			name:               "preflight with X-Active-Profile is accepted",
			requestHeaders:     "X-Active-Profile",
			wantStatus:         http.StatusOK,
			wantAllowedHeaders: true,
		},
		{
			name:               "preflight with Content-Type is accepted",
			requestHeaders:     "Content-Type",
			wantStatus:         http.StatusOK,
			wantAllowedHeaders: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := handler.NewRouter(slog.Default())
			r.Route("/api", func(api chi.Router) {
				api.Use(handler.NewCORSMiddleware([]string{allowedOrigin}))
				api.Get("/jobs", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			})

			req := httptest.NewRequest(http.MethodOptions, "/api/jobs", nil)
			req.Header.Set("Origin", allowedOrigin)
			req.Header.Set("Access-Control-Request-Method", http.MethodGet)
			req.Header.Set("Access-Control-Request-Headers", tc.requestHeaders)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}

			if tc.wantAllowedHeaders {
				allowedHeaders := rec.Header().Get("Access-Control-Allow-Headers")
				if allowedHeaders == "" {
					t.Errorf("Access-Control-Allow-Headers is empty; want non-empty (requested: %q)", tc.requestHeaders)
				}
			}
		})
	}
}
