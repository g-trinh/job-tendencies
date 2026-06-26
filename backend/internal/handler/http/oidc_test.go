package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// AC: valid OIDC token is accepted; invalid/absent token is rejected 401/403.

const (
	testAudience = "https://scrape-worker-abc.run.app"
	testPushSA   = "pubsub-push-dev@job-tendencies-dev.iam.gserviceaccount.com"
	testOtherSA  = "other-sa@other-project.iam.gserviceaccount.com"
)

// fakeVerifier is a test double for TokenVerifier. It returns a fixed payload
// when the token equals "valid-token", or an error otherwise.
type fakeVerifier struct {
	email   string
	failErr error
}

func (f *fakeVerifier) Verify(_ context.Context, token, _ string) (*handler.TokenPayload, error) {
	if f.failErr != nil {
		return nil, f.failErr
	}
	return &handler.TokenPayload{Email: f.email}, nil
}

func TestOIDCMiddleware(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		authHeader string
		verifier   handler.TokenVerifier
		wantStatus int
	}{
		{
			name:       "missing Authorization header returns 401",
			authHeader: "",
			verifier:   &fakeVerifier{email: testPushSA},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Authorization without Bearer prefix returns 401",
			authHeader: "Basic dXNlcjpwYXNz",
			verifier:   &fakeVerifier{email: testPushSA},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token (verifier returns error) returns 401",
			authHeader: "Bearer bad-token",
			verifier:   &fakeVerifier{failErr: fmt.Errorf("token expired")},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token but wrong SA returns 403",
			authHeader: "Bearer valid-token",
			verifier:   &fakeVerifier{email: testOtherSA},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "valid token with correct SA is accepted",
			authHeader: "Bearer valid-token",
			verifier:   &fakeVerifier{email: testPushSA},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := chi.NewRouter()
			r.With(handler.OIDCMiddleware(tc.verifier, testAudience, testPushSA)).
				Post("/push/scrape-tick", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

			req := httptest.NewRequest(http.MethodPost, "/push/scrape-tick", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestOIDCMiddleware_PayloadStoredInContext(t *testing.T) {
	t.Parallel()

	var capturedEmail string

	r := chi.NewRouter()
	r.With(handler.OIDCMiddleware(&fakeVerifier{email: testPushSA}, testAudience, testPushSA)).
		Post("/push/scrape-tick", func(w http.ResponseWriter, r *http.Request) {
			if p := handler.TokenPayloadFromContext(r.Context()); p != nil {
				capturedEmail = p.Email
			}
			w.WriteHeader(http.StatusOK)
		})

	req := httptest.NewRequest(http.MethodPost, "/push/scrape-tick", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rec.Code)
	}
	if capturedEmail != testPushSA {
		t.Errorf("captured email = %q; want %q", capturedEmail, testPushSA)
	}
}
