package handler_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// AC: a scoped route rejects a missing X-Active-Profile with a 400.

func TestScopedRoutes_RejectsRequestWithoutActiveProfileHeader(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.With(handler.RequireActiveProfileForTest).Get("/scoped", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cases := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{
			name:       "missing X-Active-Profile returns 400",
			header:     "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "present X-Active-Profile is accepted",
			header:     "profile-123",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
			if tc.header != "" {
				req.Header.Set("X-Active-Profile", tc.header)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestActiveProfileID_ExtractedFromContext(t *testing.T) {
	t.Parallel()

	var captured kernel.ProfileID

	r := chi.NewRouter()
	r.With(handler.RequireActiveProfileForTest).Get("/scoped", func(w http.ResponseWriter, r *http.Request) {
		id, ok := handler.ActiveProfileID(r)
		if !ok {
			http.Error(w, "no profile", http.StatusInternalServerError)
			return
		}
		captured = id
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	req.Header.Set("X-Active-Profile", "profile-abc")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rec.Code)
	}
	if captured != "profile-abc" {
		t.Errorf("captured profile id = %q; want %q", captured, "profile-abc")
	}
}

// AC: errors map to status codes.

func TestRespondError_DomainErrorMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "ErrNotFound maps to 404",
			err:        kernel.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantMsg:    "resource not found",
		},
		{
			name:       "NotFoundError maps to 404 with kind context",
			err:        &kernel.NotFoundError{Kind: "job", ID: "job-123"},
			wantStatus: http.StatusNotFound,
			wantMsg:    `job "job-123" not found`,
		},
		{
			name:       "ErrInvalidInput maps to 400",
			err:        kernel.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "invalid input",
		},
		{
			name:       "ValidationError maps to 400 with field context",
			err:        &kernel.ValidationError{Field: "salary_min", Message: "must be positive"},
			wantStatus: http.StatusBadRequest,
			wantMsg:    "validation error on salary_min: must be positive",
		},
		{
			name:       "ErrConflict maps to 409",
			err:        kernel.ErrConflict,
			wantStatus: http.StatusConflict,
			wantMsg:    "conflict",
		},
		{
			name:       "ErrUnauthorized maps to 401",
			err:        kernel.ErrUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "unauthorized",
		},
		{
			name:       "unknown error maps to 500 without leaking detail",
			err:        errors.New("db connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "internal server error",
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := handler.NewRouter(logger)
			r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
				handler.RespondError(w, r, tc.err)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}

			var body struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decoding response body: %v", err)
			}
			if body.Error != tc.wantMsg {
				t.Errorf("body.error = %q; want %q", body.Error, tc.wantMsg)
			}
		})
	}
}
