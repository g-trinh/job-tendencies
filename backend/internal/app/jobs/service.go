// Package jobs contains the job-browser query service: the read side of the
// job-browser context (CQRS-lite, ADR-005). It returns read DTOs (JobView) projected
// directly from storage and never goes through the Job aggregate's write repository.
// Reads are scoped to the active profile. Phase 3 adds filters, sort, board names,
// application status, and description.
package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// JobSourceView records which raw listing (and board) a job was extracted from.
type JobSourceView struct {
	BoardID   kernel.BoardID
	SourceURL string
	BoardName string
}

// JobView is the job-browser read model. It carries exactly the fields the browser
// renders; per ADR-005 it is projected from storage by the query port and is distinct
// from the domain Job aggregate used on the write side.
//
// FitScore is always nil until the scoring pipeline (Track B) lands.
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
	Description        string
	ContactID          *string
	FirstSeen          time.Time
	LastSeen           time.Time
	ExpiredAt          *time.Time
	ApplicationStatus  *kernel.ApplicationStatus
	FitScore           *float64
	Sources            []JobSourceView
}

// JobListFilter holds optional filter and sort parameters for the list query.
// Zero values mean "no filter". Sort defaults to "date" DESC when unset.
type JobListFilter struct {
	Skills        []string
	RemotePolicy  string
	ContractType  string
	SalaryMin     *int64
	SalaryMax     *int64
	Location      string
	BoardID       string
	Since         *time.Time
	ConfidenceMin *int
	Sort          string // "date" | "fit" | "salary"; default "date"
	SortDir       string // "asc" | "desc"; default "desc"
}

// ApplicationResult is returned after upserting a job application kanban status.
type ApplicationResult struct {
	Status    kernel.ApplicationStatus
	UpdatedAt time.Time
}

// JobQuery reads job views scoped to a profile, querying storage directly. It is the
// read side and deliberately does not reuse the domain Job repository (ADR-005).
type JobQuery interface {
	// ListByProfile returns job views scoped to the profile, filtered and sorted.
	ListByProfile(ctx context.Context, profileID kernel.ProfileID, filter JobListFilter) ([]JobView, error)
	// GetByProfile returns one job view scoped to the profile, or a kernel.NotFoundError.
	GetByProfile(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (JobView, error)
}

// JobApplicationWriter persists per-(profile, job) kanban status changes.
type JobApplicationWriter interface {
	// UpsertApplication sets or updates the application status for a (profile, job) pair.
	UpsertApplication(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID, status kernel.ApplicationStatus) (ApplicationResult, error)
}

// Service exposes job-browser read and kanban write use cases to the API.
type Service struct {
	query JobQuery
	appW  JobApplicationWriter
}

// New constructs a jobs query Service with read-only access.
func New(query JobQuery) *Service {
	return &Service{query: query}
}

// NewWithWriter constructs a jobs Service with both read and kanban-write access.
func NewWithWriter(query JobQuery, appW JobApplicationWriter) *Service {
	return &Service{query: query, appW: appW}
}

// ListJobs returns job views scoped to the active profile with optional filtering.
func (s *Service) ListJobs(ctx context.Context, profileID kernel.ProfileID, filter JobListFilter) ([]JobView, error) {
	out, err := s.query.ListByProfile(ctx, profileID, filter)
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

// SetApplicationStatus upserts the kanban status for a (profile, job) pair.
func (s *Service) SetApplicationStatus(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID, status kernel.ApplicationStatus) (ApplicationResult, error) {
	if s.appW == nil {
		return ApplicationResult{}, fmt.Errorf("application writer not configured")
	}
	result, err := s.appW.UpsertApplication(ctx, profileID, jobID, status)
	if err != nil {
		return ApplicationResult{}, fmt.Errorf("setting application status for job %q: %w", jobID, err)
	}
	return result, nil
}
