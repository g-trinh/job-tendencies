package handler_test

import (
	"context"
	"encoding/json"
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

// --- Test double ---

type fakeAuthService struct {
	loginResult *appauth.LoginResult
	loginErr    error
	meResult    *appauth.MeResult
	meErr       error
}

func (f *fakeAuthService) Login(_ context.Context, _, _ string) (*appauth.LoginResult, error) {
	return f.loginResult, f.loginErr
}

func (f *fakeAuthService) Me(_ context.Context, _ string) (*appauth.MeResult, error) {
	return f.meResult, f.meErr
}

// --- Login handler tests ---

// AC: Successful login sets an httpOnly cookie and returns the user (no token in JSON).

func TestPostLogin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		body        string
		svc         *fakeAuthService
		wantStatus  int
		wantCookie  bool
		wantUID     string
		wantEmail   string
		wantNoToken bool // true = JSON body must not contain any token field
	}{
		{
			name: "valid credentials set cookie and return user without token",
			body: `{"email":"user@example.com","password":"secret"}`,
			svc: &fakeAuthService{loginResult: &appauth.LoginResult{
				User:        domainauth.User{UID: "uid-1", Email: "user@example.com"},
				CookieValue: "encrypted-session-value",
			}},
			wantStatus:  http.StatusOK,
			wantCookie:  true,
			wantUID:     "uid-1",
			wantEmail:   "user@example.com",
			wantNoToken: true,
		},
		{
			name:       "invalid credentials return 401",
			body:       `{"email":"bad@example.com","password":"wrong"}`,
			svc:        &fakeAuthService{loginErr: domainauth.ErrInvalidCredentials},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing email returns 400",
			body:       `{"password":"secret"}`,
			svc:        &fakeAuthService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing password returns 400",
			body:       `{"email":"user@example.com"}`,
			svc:        &fakeAuthService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON body returns 400",
			body:       `not-json`,
			svc:        &fakeAuthService{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := chi.NewRouter()
			r.Post("/api/auth/login", handler.PostLogin(tc.svc, false))

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCookie {
				cookies := rec.Result().Cookies()
				var sessionCookie *http.Cookie
				for _, c := range cookies {
					if c.Name == "session" {
						sessionCookie = c
						break
					}
				}
				require.NotNil(t, sessionCookie, "session cookie must be set on successful login")
				assert.True(t, sessionCookie.HttpOnly, "session cookie must be HttpOnly")
				assert.Equal(t, http.SameSiteStrictMode, sessionCookie.SameSite, "session cookie must be SameSite=Strict")
				assert.Equal(t, "encrypted-session-value", sessionCookie.Value)
			}

			if tc.wantUID != "" {
				var body map[string]interface{}
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
				assert.Equal(t, tc.wantUID, body["uid"])
				assert.Equal(t, tc.wantEmail, body["email"])
			}

			if tc.wantNoToken {
				var body map[string]interface{}
				_ = json.NewDecoder(rec.Body).Decode(&body)
				_, hasToken := body["token"]
				_, hasIDToken := body["id_token"]
				_, hasIDTokenCamel := body["idToken"]
				assert.False(t, hasToken, "response must not contain a 'token' field")
				assert.False(t, hasIDToken, "response must not contain an 'id_token' field")
				assert.False(t, hasIDTokenCamel, "response must not contain an 'idToken' field")
			}
		})
	}
}

// --- Logout handler tests ---

// AC: Logout invalidates the session; subsequent /api/auth/me → 401.

func TestPostLogout(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Post("/api/auth/logout", handler.PostLogout(false))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// The session cookie must be cleared (MaxAge=-1 or expires in the past).
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "session cookie must be set on logout (to clear it)")
	assert.Equal(t, -1, sessionCookie.MaxAge, "cleared cookie must have MaxAge=-1")
	assert.Empty(t, sessionCookie.Value, "cleared cookie value must be empty")
}

// --- Me handler tests ---

// AC: GET /api/auth/me returns the user when the cookie is valid, 401 otherwise.

func TestGetMe(t *testing.T) {
	t.Parallel()

	validUser := domainauth.User{UID: "uid-me", Email: "me@example.com"}

	cases := []struct {
		name       string
		injectUser *domainauth.User // nil = no user in context (no auth middleware)
		wantStatus int
		wantUID    string
	}{
		{
			name:       "authenticated user returns 200 with user info",
			injectUser: &validUser,
			wantStatus: http.StatusOK,
			wantUID:    "uid-me",
		},
		{
			name:       "unauthenticated request returns 401",
			injectUser: nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := chi.NewRouter()
			// Simulate RequireAuth by injecting the user into context when present.
			if tc.injectUser != nil {
				user := *tc.injectUser
				r.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						ctx := handler.WithAuthUserForTest(req.Context(), user)
						next.ServeHTTP(w, req.WithContext(ctx))
					})
				})
			}
			r.Get("/api/auth/me", handler.GetMe())

			req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantUID != "" {
				var body map[string]interface{}
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
				assert.Equal(t, tc.wantUID, body["uid"])
			}
		})
	}
}
