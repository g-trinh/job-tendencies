package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appauth "github.com/g-trinh/job-tendencies/internal/app/auth"
	domainauth "github.com/g-trinh/job-tendencies/internal/domain/auth"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// --- RequireAuth middleware tests ---

// AC: Unauthenticated request to a guarded /api route → 401; authenticated → 200.

func TestRequireAuth(t *testing.T) {
	t.Parallel()

	validUser := domainauth.User{UID: "uid-auth", Email: "auth@example.com"}

	cases := []struct {
		name       string
		svc        *fakeAuthService
		hasCookie  bool
		wantStatus int
	}{
		{
			name: "request with valid session cookie returns 200",
			svc: &fakeAuthService{meResult: &appauth.MeResult{
				User: validUser,
			}},
			hasCookie:  true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "request without session cookie returns 401",
			svc:        &fakeAuthService{},
			hasCookie:  false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "request with invalid session cookie returns 401",
			svc: &fakeAuthService{
				meErr: domainauth.ErrTokenInvalid,
			},
			hasCookie:  true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "request with expired session cookie returns 401",
			svc: &fakeAuthService{
				meErr: domainauth.ErrTokenExpired,
			},
			hasCookie:  true,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := chi.NewRouter()
			r.Group(func(guarded chi.Router) {
				guarded.Use(handler.RequireAuth(tc.svc, false))
				guarded.Get("/api/boards", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			})

			req := httptest.NewRequest(http.MethodGet, "/api/boards", nil)
			if tc.hasCookie {
				req.AddCookie(&http.Cookie{Name: "__session", Value: "some-cookie-value"})
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

// AC: Transparent refresh sets updated cookie on response.

func TestRequireAuth_TransparentRefreshSetsCookie(t *testing.T) {
	t.Parallel()

	svc := &fakeAuthService{meResult: &appauth.MeResult{
		User:               domainauth.User{UID: "uid-refresh", Email: "r@example.com"},
		UpdatedCookieValue: "new-encrypted-cookie",
	}}

	r := chi.NewRouter()
	r.Group(func(guarded chi.Router) {
		guarded.Use(handler.RequireAuth(svc, false))
		guarded.Get("/api/boards", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/boards", nil)
	req.AddCookie(&http.Cookie{Name: "__session", Value: "old-cookie-value"})
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "__session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "updated session cookie must be set on the response")
	assert.Equal(t, "new-encrypted-cookie", sessionCookie.Value)
}

// AC: /healthz remains reachable unauthenticated.

func TestHealthz_IsPublic(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// /api is guarded by RequireAuth; /healthz is not.
	r.Group(func(guarded chi.Router) {
		guarded.Use(handler.RequireAuth(&fakeAuthService{meErr: domainauth.ErrTokenInvalid}, false))
		guarded.Get("/api/boards", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "/healthz must be reachable without auth")

	req2 := httptest.NewRequest(http.MethodGet, "/api/boards", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code, "guarded /api route must reject unauthenticated request")
}

// AC: /api/auth/login remains reachable unauthenticated.

func TestLoginRoute_IsPublic(t *testing.T) {
	t.Parallel()

	// Simulate the router structure used in production: auth routes outside the guarded group.
	svc := &fakeAuthService{loginResult: &appauth.LoginResult{
		User:        domainauth.User{UID: "uid-1", Email: "u@example.com"},
		CookieValue: "cookie",
	}}

	r := chi.NewRouter()
	r.Post("/api/auth/login", handler.PostLogin(svc, false))
	r.Group(func(guarded chi.Router) {
		guarded.Use(handler.RequireAuth(&fakeAuthService{meErr: domainauth.ErrTokenInvalid}, false))
		guarded.Get("/api/boards", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"email":"u@example.com","password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "/api/auth/login must be reachable without session cookie")
}

// --- RequireCSRF middleware tests ---

// AC: A state-changing request without a valid CSRF token is rejected.

func TestRequireCSRF(t *testing.T) {
	t.Parallel()

	allowedOrigins := []string{"https://job-tendencies.web.app", "http://localhost:5173"}

	cases := []struct {
		name       string
		method     string
		origin     string
		wantStatus int
	}{
		{
			name:       "POST with allowed origin passes CSRF check",
			method:     http.MethodPost,
			origin:     "https://job-tendencies.web.app",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with disallowed origin returns 403",
			method:     http.MethodPost,
			origin:     "https://evil.com",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "POST without Origin header is allowed (curl/Postman)",
			method:     http.MethodPost,
			origin:     "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET with disallowed origin is allowed (read-only)",
			method:     http.MethodGet,
			origin:     "https://evil.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "DELETE with disallowed origin returns 403",
			method:     http.MethodDelete,
			origin:     "https://evil.com",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "PUT with allowed origin passes",
			method:     http.MethodPut,
			origin:     "http://localhost:5173",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := chi.NewRouter()
			r.Use(handler.RequireCSRF(allowedOrigins))
			r.Method(tc.method, "/api/boards", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tc.method, "/api/boards", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestRequireCSRF_EmptyAllowedOrigins_AllowsAll(t *testing.T) {
	t.Parallel()

	// When no AllowedOrigins are configured, CSRF check is skipped.
	r := chi.NewRouter()
	r.Use(handler.RequireCSRF(nil))
	r.Post("/api/boards", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boards", nil)
	req.Header.Set("Origin", "https://anyone.example.com")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code,
		"when AllowedOrigins is empty, all origins pass (SameSite=Strict on cookie provides CSRF protection)")
}
