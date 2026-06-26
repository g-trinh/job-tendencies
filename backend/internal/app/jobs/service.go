// Package jobs contains the job-browser application service: list and detail reads
// scoped to the active profile. The repository interface is declared here (consumer)
// and implemented in infra/jobs. Phase 2 has no filters or sorting.
package jobs

import (
	"context"
	"fmt"

	"github.com/g-trinh/job-tendencies/internal/domain/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository reads jobs scoped to a profile from the datastore.
type Repository interface {
	// ListByProfile returns every job that has a source listing captured for the profile.
	ListByProfile(ctx context.Context, profileID kernel.ProfileID) ([]jobs.Job, error)
	// GetByProfile returns one job scoped to the profile, or a kernel.NotFoundError.
	GetByProfile(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (jobs.Job, error)
}

// Service exposes job read use cases to the API.
type Service struct {
	repo Repository
}

// New constructs a jobs Service.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListJobs returns all jobs scoped to the active profile.
func (s *Service) ListJobs(ctx context.Context, profileID kernel.ProfileID) ([]jobs.Job, error) {
	out, err := s.repo.ListByProfile(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("listing jobs for profile %q: %w", profileID, err)
	}
	return out, nil
}

// GetJob returns one job scoped to the active profile.
func (s *Service) GetJob(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (jobs.Job, error) {
	job, err := s.repo.GetByProfile(ctx, profileID, id)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("getting job %q for profile %q: %w", id, profileID, err)
	}
	return job, nil
}
