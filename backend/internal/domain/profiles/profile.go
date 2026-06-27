// Package profiles is the profiles bounded context. It owns the search persona used
// to scope every job/dashboard/browser view and to drive the scraper's board-side
// filtering (keywords + location). Each profile is a named search configuration;
// exactly one profile is active at a time. Phase 3 adds identity, conditions, and
// fit-score weights (see identity.go).
package profiles

import (
	"strings"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Profile is a search persona. Exactly one profile is active at a time; the active
// profile scopes scoped resources and supplies the scraper's keywords and location.
type Profile struct {
	// ID is the profile's stable identifier.
	ID kernel.ProfileID
	// Name is the human-readable persona name.
	Name string
	// SearchKeywords are the keywords pushed into each board's search query.
	SearchKeywords []string
	// Location is the geographic search target pushed into each board's search query.
	Location string
	// IsActive reports whether this is the single active profile.
	IsActive bool

	// Identity: extracted from the LinkedIn PDF import or manually maintained.
	Skills    []string
	Seniority kernel.Seniority

	// Conditions: dealbreakers and preferences used by the scoring pipeline.
	Conditions ProfileConditions

	// Weights: user-configured fit-score weights (soft components, sum = 100).
	Weights FitWeights
}

// NewProfile constructs a Profile with validated name, location, and keywords. It
// validates that the name is non-empty; keywords default to an empty slice when nil.
// Identity, conditions, and weights are zeroed and set via dedicated service methods.
//
// Example:
//
//	p, err := profiles.NewProfile("Go Backend Paris", "Paris", []string{"golang", "backend"})
func NewProfile(name, location string, keywords []string) (Profile, error) {
	if strings.TrimSpace(name) == "" {
		return Profile{}, &kernel.ValidationError{Field: "name", Message: "required"}
	}
	if keywords == nil {
		keywords = []string{}
	}
	return Profile{
		Name:           strings.TrimSpace(name),
		Location:       location,
		SearchKeywords: keywords,
		Weights:        DefaultFitWeights(),
		Conditions: ProfileConditions{
			DealBreakerRequiredSkills: []string{},
			PreferredSkills:           []string{},
		},
	}, nil
}
