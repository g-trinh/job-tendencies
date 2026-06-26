// Package profiles is the profiles bounded context. It owns the search persona used
// to scope every job/dashboard/browser view and to drive the scraper's board-side
// filtering (keywords + location). Phase 2 ships a single hardcoded active profile.
package profiles

import "github.com/g-trinh/job-tendencies/internal/domain/kernel"

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
}
