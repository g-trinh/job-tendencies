package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	appauth "github.com/g-trinh/job-tendencies/internal/app/auth"
	domainauth "github.com/g-trinh/job-tendencies/internal/domain/auth"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// sessionCookieName is the name of the httpOnly session cookie set by the auth endpoints.
// The cookie carries an AES-256-GCM-encrypted payload containing the IdP refresh token.
//
// MUST be "__session": Firebase Hosting strips every cookie except one named
// __session before forwarding a request to the Cloud Run backend (and on the way
// back), so any other name silently disappears behind the /api rewrite.
const sessionCookieName = "__session"

// authUserKey is the context key used by RequireAuth to store the authenticated user.
const authUserKey contextKey = "auth_user"

// AuthService handles authentication use cases for the Job Tendencies API.
// Implemented by app/auth.Service.
type AuthService interface {
	// Login authenticates email+password against Identity Platform. Returns an encrypted
	// session cookie value (to set as httpOnly cookie) and the authenticated user.
	Login(ctx context.Context, email, password string) (*appauth.LoginResult, error)

	// Me decodes the session cookie, optionally refreshes the ID token near expiry, and
	// returns the verified user. UpdatedCookieValue is non-empty when the session was refreshed.
	Me(ctx context.Context, cookieValue string) (*appauth.MeResult, error)
}

// loginRequest is the JSON body expected by POST /api/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// userResponse is the JSON body returned by login and me endpoints.
// It intentionally omits any token — the browser only receives a cookie.
type userResponse struct {
	UID   string `json:"uid"`
	Email string `json:"email"`
}

// PostLogin handles POST /api/auth/login. It authenticates email+password via Identity
// Platform and sets an httpOnly, Secure, SameSite=Strict session cookie. The JSON
// response body contains only the user — no token is ever sent to the browser.
//
// Returns 400 on missing/invalid body, 401 on wrong credentials.
func PostLogin(svc AuthService, cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		if req.Email == "" || req.Password == "" {
			RespondError(w, r, &kernel.ValidationError{Field: "email/password", Message: "both fields are required"})
			return
		}

		result, err := svc.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			// Map auth domain errors to HTTP status codes.
			switch {
			case isAuthError(err):
				RespondError(w, r, kernel.ErrUnauthorized)
			default:
				RespondError(w, r, err)
			}
			return
		}

		http.SetCookie(w, newSessionCookie(result.CookieValue, cookieSecure))
		respond(w, http.StatusOK, userResponse{UID: result.User.UID, Email: result.User.Email})
	}
}

// PostLogout handles POST /api/auth/logout. It clears the session cookie by setting
// an expired replacement. Server-side token revocation is not available via the
// Identity Platform REST API without the Admin SDK; the session is invalidated from
// the browser's perspective by removing the cookie.
func PostLogout(cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, clearSessionCookie(cookieSecure))
		respond(w, http.StatusOK, map[string]string{"status": "logged out"})
	}
}

// GetMe handles GET /api/auth/me. It reads the authenticated user stored in the
// request context by RequireAuth middleware and returns it as JSON. If the middleware
// refreshed the session (UpdatedCookieValue non-empty), the new cookie is also set.
//
// Returns 401 when the RequireAuth middleware was not applied or the session is invalid.
func GetMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := AuthUser(r)
		if !ok {
			RespondError(w, r, kernel.ErrUnauthorized)
			return
		}
		respond(w, http.StatusOK, userResponse{UID: user.UID, Email: user.Email})
	}
}

// AuthUser retrieves the authenticated user from the request context. It is set by
// RequireAuth middleware on all guarded routes.
// Returns the user and true when present; zero value and false otherwise.
func AuthUser(r *http.Request) (domainauth.User, bool) {
	u, ok := r.Context().Value(authUserKey).(domainauth.User)
	return u, ok && u.UID != ""
}

// contextWithAuthUser stores the authenticated user in the context. Called by RequireAuth.
func contextWithAuthUser(ctx context.Context, user domainauth.User) context.Context {
	return context.WithValue(ctx, authUserKey, user)
}

// newSessionCookie constructs the httpOnly session cookie with the correct security flags.
func newSessionCookie(value string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}

// clearSessionCookie constructs an expired cookie to instruct the browser to delete
// the session cookie. MaxAge=-1 is the standard way to clear a cookie.
func clearSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
}

// isAuthError returns true for domain errors that should map to HTTP 401.
func isAuthError(err error) bool {
	return errors.Is(err, domainauth.ErrInvalidCredentials) ||
		errors.Is(err, domainauth.ErrTokenExpired) ||
		errors.Is(err, domainauth.ErrTokenInvalid)
}
