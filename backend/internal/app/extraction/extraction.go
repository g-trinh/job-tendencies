// Package extraction contains the extract-worker application service: load a raw
// listing from GCS, run Claude structured extraction through the llm port, and create
// one Job. Phase 2 skips dedup/merge, contacts and scoring. Raw is never translated.
//
// See docs/architecture/pipeline.md §3 and ADR-004.
package extraction

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/blobstore"
	"github.com/g-trinh/job-tendencies/internal/domain/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
)

// RawListingRef is the extraction context's view of a captured raw listing: enough to
// load the payload from GCS and link the resulting job back to its source.
type RawListingRef struct {
	ID        kernel.RawListingID
	BoardID   kernel.BoardID
	ProfileID kernel.ProfileID
	Title     string
	Company   string
	Location  string
	SourceURL string
	RawRef    string
}

// RawListingSource loads raw-listing metadata and marks listings extracted.
type RawListingSource interface {
	Get(ctx context.Context, id kernel.RawListingID) (RawListingRef, error)
	MarkExtracted(ctx context.Context, id kernel.RawListingID) error
}

// JobWriter creates a Job (with its source linkage) and returns its id.
type JobWriter interface {
	Create(ctx context.Context, job jobs.Job) (kernel.JobID, error)
}

// Service handles listing.extract pipeline events.
type Service struct {
	rawListings RawListingSource
	rawStore    blobstore.Loader
	extractor   llm.ListingExtractor
	jobs        JobWriter
	logger      *slog.Logger
}

// New constructs an extraction Service with all dependencies wired.
func New(
	rawListings RawListingSource,
	rawStore blobstore.Loader,
	extractor llm.ListingExtractor,
	jobWriter JobWriter,
	logger *slog.Logger,
) *Service {
	return &Service{
		rawListings: rawListings,
		rawStore:    rawStore,
		extractor:   extractor,
		jobs:        jobWriter,
		logger:      logger,
	}
}

// HandleListingExtract is invoked for each verified listing.extract push delivery. It
// loads the raw payload, extracts structured fields via Claude, and creates one Job.
func (s *Service) HandleListingExtract(ctx context.Context, msg messaging.Message) error {
	rawListingID := kernel.RawListingID(msg.Attributes[appscraping.ExtractRawListingIDAttr])
	if rawListingID == "" {
		rawListingID = kernel.RawListingID(msg.Data)
	}
	if rawListingID == "" {
		return fmt.Errorf("listing.extract message carries no raw listing id")
	}

	ref, err := s.rawListings.Get(ctx, rawListingID)
	if err != nil {
		return fmt.Errorf("loading raw listing %q: %w", rawListingID, err)
	}

	raw, err := s.rawStore.Load(ctx, ref.RawRef)
	if err != nil {
		return fmt.Errorf("loading raw payload %q: %w", ref.RawRef, err)
	}

	extracted, err := s.extractor.Extract(ctx, string(raw))
	if err != nil {
		return fmt.Errorf("extracting listing %q: %w", rawListingID, err)
	}

	job := buildJob(extracted, ref, time.Now().UTC())
	jobID, err := s.jobs.Create(ctx, job)
	if err != nil {
		return fmt.Errorf("creating job from listing %q: %w", rawListingID, err)
	}

	if err := s.rawListings.MarkExtracted(ctx, rawListingID); err != nil {
		return fmt.Errorf("marking listing %q extracted: %w", rawListingID, err)
	}

	s.logger.InfoContext(ctx, "job created from listing",
		"job_id", string(jobID), "raw_listing_id", string(rawListingID),
		"understanding", extracted.Understanding.Int())
	return nil
}

// buildJob maps an ExtractedListing plus its source reference into a Job aggregate,
// flattening per-field confidence into the FieldConfidence map. It is pure so the
// mapping can be unit-tested without the LLM or datastore.
func buildJob(e *llm.ExtractedListing, ref RawListingRef, now time.Time) jobs.Job {
	return jobs.Job{
		Title:        ref.Title,
		Company:      ref.Company,
		Location:     ref.Location,
		URL:          ref.SourceURL,
		Skills:       e.Skills.Value,
		RemotePolicy: e.RemotePolicy.Value,
		OfficeDays:   e.OfficeDays.Value,
		ContractType: e.ContractType.Value,
		WorkingDays:  e.WorkingDays.Value,
		SalaryMin:    e.SalaryMin.Value,
		SalaryMax:    e.SalaryMax.Value,
		Seniority:    e.Seniority.Value,
		FieldConfidence: map[string]int{
			"skills":        e.Skills.Confidence.Int(),
			"remote_policy": e.RemotePolicy.Confidence.Int(),
			"office_days":   e.OfficeDays.Confidence.Int(),
			"contract_type": e.ContractType.Confidence.Int(),
			"working_days":  e.WorkingDays.Confidence.Int(),
			"salary_min":    e.SalaryMin.Confidence.Int(),
			"salary_max":    e.SalaryMax.Confidence.Int(),
			"seniority":     e.Seniority.Confidence.Int(),
		},
		UnderstandingScore: e.Understanding,
		FirstSeen:          now,
		LastSeen:           now,
		Sources: []jobs.JobSource{{
			BoardID:      ref.BoardID,
			RawListingID: ref.ID,
			SourceURL:    ref.SourceURL,
		}},
	}
}
