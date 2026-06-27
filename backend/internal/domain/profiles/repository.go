package profiles

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository is the profiles aggregate's persistence port. Per ADR-005 the aggregate
// repository interface lives in the domain. Phase 2 exposes only active-profile
// resolution and lookup by id.
type Repository interface {
	// ActiveProfile returns the single active profile, or a kernel.NotFoundError
	// when no profile is marked active.
	ActiveProfile(ctx context.Context) (Profile, error)
	// ProfileByID returns one profile, or a kernel.NotFoundError.
	ProfileByID(ctx context.Context, id kernel.ProfileID) (Profile, error)
}
