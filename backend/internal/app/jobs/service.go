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

	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
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
//
// Page/PageSize implement offset pagination (ADR-007): Page is 1-based and PageSize
// is the number of rows per page. Callers (the HTTP handler) are responsible for
// clamping Page to >= 1 and PageSize to 1..100 before this filter reaches the query
// port — the query port trusts these values are already valid.
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
	Page          int    // 1-based page number; expected >= 1
	PageSize      int    // rows per page; expected in 1..100
}

// JobListResult is the paginated outcome of a job list query (ADR-007): the page of
// items plus the total row count across the entire filtered result set (not just the
// page), so the caller can compute total_pages.
type JobListResult struct {
	Items []JobView
	Total int
}

// ApplicationResult is returned after upserting a job application kanban status.
type ApplicationResult struct {
	Status    kernel.ApplicationStatus
	UpdatedAt time.Time
}

// JobQuery reads job views scoped to a profile, querying storage directly. It is the
// read side and deliberately does not reuse the domain Job repository (ADR-005).
type JobQuery interface {
	// ListByProfile returns a page of job views scoped to the profile, filtered and
	// sorted, plus the total row count across the whole filtered result (ADR-007).
	ListByProfile(ctx context.Context, profileID kernel.ProfileID, filter JobListFilter) (JobListResult, error)
	// GetByProfile returns one job view scoped to the profile, or a kernel.NotFoundError.
	GetByProfile(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (JobView, error)
}

// JobApplicationWriter persists per-(profile, job) kanban status changes.
type JobApplicationWriter interface {
	// UpsertApplication sets or updates the application status for a (profile, job) pair.
	UpsertApplication(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID, status kernel.ApplicationStatus) (ApplicationResult, error)
}

// JobRawListingSource resolves the raw listing ids retained for a job's sources, scoped
// to the profile, so P5-4 re-extraction can re-publish listing.extract for each one.
// Implemented by infra/jobs.Repository.
type JobRawListingSource interface {
	// RawListingIDsByJob returns the raw listing ids the job was extracted from, scoped
	// to the profile. Returns an empty slice (not an error) when the job exists but has
	// no sources visible to this profile; the caller treats that as not-found.
	RawListingIDsByJob(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID) ([]kernel.RawListingID, error)
}

// Service exposes job-browser read and kanban write use cases to the API.
type Service struct {
	query       JobQuery
	appW        JobApplicationWriter
	rawListings JobRawListingSource
	publisher   messaging.Publisher
}

// New constructs a jobs query Service with read-only access.
func New(query JobQuery) *Service {
	return &Service{query: query}
}

// NewWithWriter constructs a jobs Service with both read and kanban-write access.
func NewWithWriter(query JobQuery, appW JobApplicationWriter) *Service {
	return &Service{query: query, appW: appW}
}

// WithReextraction attaches the dependencies needed for ReextractJob (P5-4): a source of
// a job's retained raw listing ids and the listing.extract publisher.
func (s *Service) WithReextraction(rawListings JobRawListingSource, publisher messaging.Publisher) *Service {
	s.rawListings = rawListings
	s.publisher = publisher
	return s
}

// ListJobs returns a page of job views scoped to the active profile with optional
// filtering, plus the total row count (ADR-007).
func (s *Service) ListJobs(ctx context.Context, profileID kernel.ProfileID, filter JobListFilter) (JobListResult, error) {
	out, err := s.query.ListByProfile(ctx, profileID, filter)
	if err != nil {
		return JobListResult{}, fmt.Errorf("listing jobs for profile %q: %w", profileID, err)
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

// ReextractJob re-publishes listing.extract for every raw listing retained for the job
// (P5-4, pipeline.md §5 "Re-extraction"): raw payloads are never deleted from GCS, so
// this lets an improved extractor reprocess a job without a new scrape. Returns a
// kernel.NotFoundError when the job has no sources visible to this profile.
func (s *Service) ReextractJob(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID) error {
	if s.rawListings == nil || s.publisher == nil {
		return fmt.Errorf("job re-extraction is not configured")
	}

	ids, err := s.rawListings.RawListingIDsByJob(ctx, profileID, jobID)
	if err != nil {
		return fmt.Errorf("resolving raw listings for job %q: %w", jobID, err)
	}
	if len(ids) == 0 {
		return &kernel.NotFoundError{Kind: "job", ID: string(jobID)}
	}

	for _, id := range ids {
		msg := messaging.Message{
			Data:       []byte(id),
			Attributes: map[string]string{appscraping.ExtractRawListingIDAttr: string(id)},
		}
		if err := s.publisher.Publish(ctx, msg); err != nil {
			return fmt.Errorf("publishing listing.extract for raw listing %q: %w", id, err)
		}
	}
	return nil
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
