package profiles

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository is the profiles aggregate's persistence port. Aggregate repository
// interfaces live in the domain per ADR-005; the Postgres implementation lives in
// internal/infra/profiles.
type Repository interface {
	// ActiveProfile returns the single active profile, or a kernel.NotFoundError
	// when no profile is marked active.
	ActiveProfile(ctx context.Context) (Profile, error)
	// ProfileByID returns one profile, or a kernel.NotFoundError.
	ProfileByID(ctx context.Context, id kernel.ProfileID) (Profile, error)
	// List returns all profiles ordered by name.
	List(ctx context.Context) ([]Profile, error)
	// Create persists a new profile and returns its assigned id.
	Create(ctx context.Context, p Profile) (kernel.ProfileID, error)
	// Update persists name, search_keywords, and location changes for the profile.
	Update(ctx context.Context, p Profile) error
	// Delete removes a profile by id. It returns a kernel.NotFoundError when the
	// profile does not exist.
	Delete(ctx context.Context, id kernel.ProfileID) error
	// Activate switches the active profile to id in a single transaction, leaving
	// exactly one active profile afterwards. It returns a kernel.NotFoundError when
	// id does not exist.
	Activate(ctx context.Context, id kernel.ProfileID) error
	// UpdateIdentity persists skills and seniority for a profile. Used by the manual
	// edit path; does not touch raw_experience.
	UpdateIdentity(ctx context.Context, id kernel.ProfileID, skills []string, seniority kernel.Seniority) error
	// UpdateIdentityFromImport persists skills, seniority, and raw experience from a
	// LinkedIn PDF import. This is the single-import path; for manual edits use UpdateIdentity.
	UpdateIdentityFromImport(ctx context.Context, id kernel.ProfileID, skills []string, seniority kernel.Seniority, rawExperience string) error
	// UpdateConditions persists the conditions (dealbreakers + preferences) for a profile.
	UpdateConditions(ctx context.Context, id kernel.ProfileID, c ProfileConditions) error
	// UpdateWeights persists the fit-score weights for a profile. Caller must validate
	// that weights sum to 100 before calling (domain rule, not enforced in storage).
	UpdateWeights(ctx context.Context, id kernel.ProfileID, w FitWeights) error
}
