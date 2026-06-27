// Package profiles contains the profiles application service. It exposes read and
// write use cases for the profiles aggregate. The aggregate repository interface lives
// in the domain (domain/profiles.Repository, ADR-005) and is implemented in
// infra/profiles.
package profiles

import (
	"context"
	"fmt"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
)

// Service exposes profile use cases to the API and the scrape-worker.
type Service struct {
	repo profiles.Repository
}

// New constructs a profiles Service.
func New(repo profiles.Repository) *Service {
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

// ListProfiles returns all profiles.
func (s *Service) ListProfiles(ctx context.Context) ([]profiles.Profile, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing profiles: %w", err)
	}
	return list, nil
}

// CreateProfile validates and persists a new profile. The new profile is always
// created inactive; call ActivateProfile to switch the active profile.
func (s *Service) CreateProfile(ctx context.Context, name, location string, keywords []string) (profiles.Profile, error) {
	p, err := profiles.NewProfile(name, location, keywords)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("validating profile: %w", err)
	}
	id, err := s.repo.Create(ctx, p)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("creating profile: %w", err)
	}
	p.ID = id
	return p, nil
}

// UpdateProfile persists name, search_keywords, and location changes. Activation
// state is unaffected; use ActivateProfile to switch the active profile.
func (s *Service) UpdateProfile(ctx context.Context, id kernel.ProfileID, name, location string, keywords []string) (profiles.Profile, error) {
	p, err := profiles.NewProfile(name, location, keywords)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("validating profile: %w", err)
	}
	p.ID = id
	if err := s.repo.Update(ctx, p); err != nil {
		return profiles.Profile{}, fmt.Errorf("updating profile %q: %w", id, err)
	}
	updated, err := s.repo.ProfileByID(ctx, id)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("reading updated profile %q: %w", id, err)
	}
	return updated, nil
}

// DeleteProfile removes a profile by id.
func (s *Service) DeleteProfile(ctx context.Context, id kernel.ProfileID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting profile %q: %w", id, err)
	}
	return nil
}

// ActivateProfile switches the active profile to id. Exactly one profile is active
// afterwards; all others are deactivated. Returns the newly active profile.
func (s *Service) ActivateProfile(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error) {
	if err := s.repo.Activate(ctx, id); err != nil {
		return profiles.Profile{}, fmt.Errorf("activating profile %q: %w", id, err)
	}
	p, err := s.repo.ProfileByID(ctx, id)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("reading activated profile %q: %w", id, err)
	}
	return p, nil
}

// PatchIdentity updates the identity fields (skills and seniority) for a profile.
// This is the manual-edit path; the LinkedIn import path (P3-PR-2) is separate.
func (s *Service) PatchIdentity(ctx context.Context, id kernel.ProfileID, skills []string, seniority kernel.Seniority) (profiles.Profile, error) {
	if skills == nil {
		skills = []string{}
	}
	if err := s.repo.UpdateIdentity(ctx, id, skills, seniority); err != nil {
		return profiles.Profile{}, fmt.Errorf("patching identity for profile %q: %w", id, err)
	}
	p, err := s.repo.ProfileByID(ctx, id)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("reading updated profile %q: %w", id, err)
	}
	return p, nil
}

// UpdateConditions persists the dealbreakers and preferences for a profile.
func (s *Service) UpdateConditions(ctx context.Context, id kernel.ProfileID, c profiles.ProfileConditions) (profiles.Profile, error) {
	if err := s.repo.UpdateConditions(ctx, id, c); err != nil {
		return profiles.Profile{}, fmt.Errorf("updating conditions for profile %q: %w", id, err)
	}
	p, err := s.repo.ProfileByID(ctx, id)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("reading updated profile %q: %w", id, err)
	}
	return p, nil
}

// UpdateWeights validates and persists the fit-score weights for a profile.
// Returns a validation error when the weights do not sum to 100.
func (s *Service) UpdateWeights(ctx context.Context, id kernel.ProfileID, w profiles.FitWeights) (profiles.Profile, error) {
	if err := w.Validate(); err != nil {
		return profiles.Profile{}, &kernel.ValidationError{Field: "weights", Message: err.Error()}
	}
	if err := s.repo.UpdateWeights(ctx, id, w); err != nil {
		return profiles.Profile{}, fmt.Errorf("updating weights for profile %q: %w", id, err)
	}
	p, err := s.repo.ProfileByID(ctx, id)
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("reading updated profile %q: %w", id, err)
	}
	return p, nil
}
