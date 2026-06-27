// Package jobs contains the job-browser query service: the read side of the
// job-browser context (CQRS-lite, ADR-005). It returns read DTOs (JobView) projected
// directly from storage and never goes through the Job aggregate's write repository.
// Reads are scoped to the active profile. Phase 2 has no filters or sorting.
package jobs

import (
	"context"
	"fmt"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// JobSourceView records which raw listing (and board) a job was extracted from.
type JobSourceView struct {
	BoardID      kernel.BoardID
	RawListingID kernel.RawListingID
	SourceURL    string
}

// JobView is the job-browser read model. It carries exactly the fields the browser
// renders; per ADR-005 it is projected from storage by the query port and is distinct
// from the domain Job aggregate used on the write side.
type JobView struct {
	ID                 kernel.JobID
	Title              string
	Company            string
	Location           string
	URL                string
	Skills             []string
	RemotePolicy       kernel.RemotePolicy
	OfficeDays         int
	ContractType       kernel.ContractType
	WorkingDays        kernel.WorkingDays
	SalaryMin          *int64
	SalaryMax          *int64
	Seniority          kernel.Seniority
	FieldConfidence    map[string]int
	UnderstandingScore kernel.Understanding
	Sources            []JobSourceView
}

// JobQuery reads job views scoped to a profile, querying storage directly. It is the
// read side and deliberately does not reuse the domain Job repository (ADR-005).
type JobQuery interface {
	// ListByProfile returns every job view that has a source listing captured for the profile.
	ListByProfile(ctx context.Context, profileID kernel.ProfileID) ([]JobView, error)
	// GetByProfile returns one job view scoped to the profile, or a kernel.NotFoundError.
	GetByProfile(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (JobView, error)
}

// Service exposes job-browser read use cases to the API.
type Service struct {
	query JobQuery
}

// New constructs a jobs query Service.
func New(query JobQuery) *Service {
	return &Service{query: query}
}

// ListJobs returns all job views scoped to the active profile.
func (s *Service) ListJobs(ctx context.Context, profileID kernel.ProfileID) ([]JobView, error) {
	out, err := s.query.ListByProfile(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("listing jobs for profile %q: %w", profileID, err)
	}
	return out, nil
}

// GetJob returns one job view scoped to the active profile.
func (s *Service) GetJob(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (JobView, error) {
	view, err := s.query.GetByProfile(ctx, profileID, id)
	if err != nil {
		return JobView{}, fmt.Errorf("getting job %q for profile %q: %w", id, profileID, err)
	}
	return view, nil
}
