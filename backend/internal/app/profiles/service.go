// Package profiles contains the profiles application service. Phase 2 exposes only
// active-profile resolution; the repository interface is declared here (consumer)
// and implemented in infra/profiles.
package profiles

import (
	"context"
	"fmt"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
)

// Repository reads profiles from the datastore.
type Repository interface {
	// ActiveProfile returns the single active profile, or a kernel.NotFoundError
	// when no profile is marked active.
	ActiveProfile(ctx context.Context) (profiles.Profile, error)
	// ProfileByID returns one profile, or a kernel.NotFoundError.
	ProfileByID(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error)
}

// Service exposes profile read use cases to the API and the scrape-worker.
type Service struct {
	repo Repository
}

// New constructs a profiles Service.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// ActiveProfile returns the single active profile.
func (s *Service) ActiveProfile(ctx context.Context) (profiles.Profile, error) {
	p, err := s.repo.ActiveProfile(ctx)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("resolving active profile: %w", err)
	}
	return p, nil
}

// ProfileByID returns one profile by id.
func (s *Service) ProfileByID(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error) {
	p, err := s.repo.ProfileByID(ctx, id)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("getting profile %q: %w", id, err)
	}
	return p, nil
}
