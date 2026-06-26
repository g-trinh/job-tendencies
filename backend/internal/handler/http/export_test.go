package handler

import "net/http"

// RequireActiveProfileForTest exposes the internal requireActiveProfile middleware
// for use in package-external tests without requiring a live server.
func RequireActiveProfileForTest(next http.Handler) http.Handler {
	return requireActiveProfile(next)
}
