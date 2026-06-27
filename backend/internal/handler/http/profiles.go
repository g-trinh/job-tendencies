package handler

import (
	"context"
	"net/http"

	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
)

// ActiveProfileResolver returns the single active profile. Implemented by
// app/profiles.Service.
type ActiveProfileResolver interface {
	ActiveProfile(ctx context.Context) (profiles.Profile, error)
}

// activeProfileResponse is the JSON shape returned by GET /api/active-profile.
type activeProfileResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	SearchKeywords []string `json:"search_keywords"`
	Location       string   `json:"location"`
}

// GetActiveProfile handles GET /api/active-profile, returning the active profile id
// and its search target (keywords + location).
func GetActiveProfile(resolver ActiveProfileResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := resolver.ActiveProfile(r.Context())
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, activeProfileResponse{
			ID:             string(p.ID),
			Name:           p.Name,
			SearchKeywords: p.SearchKeywords,
			Location:       p.Location,
		})
	}
}
