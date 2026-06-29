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

	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/domain/blobstore"
	"github.com/g-trinh/job-tendencies/internal/domain/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
	domainscraping "github.com/g-trinh/job-tendencies/internal/domain/scraping"
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

// toRef maps the scraping context's RawListing aggregate into the extraction context's
// own view, keeping extraction decoupled from the scraping aggregate's full shape.
func toRef(l domainscraping.RawListing) RawListingRef {
	return RawListingRef{
		ID:        l.ID,
		BoardID:   l.BoardID,
		ProfileID: l.ProfileID,
		Title:     l.Title,
		Company:   l.Company,
		Location:  l.Location,
		SourceURL: l.SourceURL,
		RawRef:    l.RawRef,
	}
}

// ContactUpserter is the extraction context's consumer interface for the contacts
// context (ADR-001). It carries only the fields needed to upsert a recruiter contact
// extracted from a listing.
type ContactUpserter interface {
	// UpsertContact creates or merges a recruiter contact by email or LinkedIn URL
	// and returns the stable contact id.
	UpsertContact(ctx context.Context, name, company, email, linkedInURL, phone string) (kernel.ContactID, error)
}

// JobScorer is the extraction context's consumer interface for the scoring context
// (ADR-001). The extraction worker calls it once per job after the job row is written.
type JobScorer interface {
	// ScoreJob computes and persists the fit score for the (job, profile) pair.
	ScoreJob(ctx context.Context, jobID kernel.JobID, profileID kernel.ProfileID) error
}

// Service handles listing.extract pipeline events. Its raw-listing read/lifecycle port
// and Job write port are the scraping and jobs aggregate repositories declared in the
// domain (ADR-005); the extraction stage maps the scraping aggregate to its own
// RawListingRef before building a Job.
//
// contacts and scorer are optional: when nil, P3-EX-3 (contact upsert) and P3-EX-4
// (scoring) are skipped. They are wired in the full extract-worker binary.
type Service struct {
	rawListings domainscraping.RawListingSource
	rawStore    blobstore.Loader
	extractor   llm.ListingExtractor
	jobs        jobs.Repository
	contacts    ContactUpserter
	scorer      JobScorer
	logger      *slog.Logger
}

// New constructs an extraction Service with the core extraction dependencies. Use
// WithContacts and WithScorer to wire optional pipeline stages.
func New(
	rawListings domainscraping.RawListingSource,
	rawStore blobstore.Loader,
	extractor llm.ListingExtractor,
	jobWriter jobs.Repository,
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

// WithContacts attaches a ContactUpserter so the extraction pipeline upserts recruiter
// contacts and links them to newly created jobs (P3-EX-3).
func (s *Service) WithContacts(c ContactUpserter) *Service {
	s.contacts = c
	return s
}

// WithScorer attaches a JobScorer so the extraction pipeline triggers scoring
// immediately after each job is created or merged (P3-EX-4).
func (s *Service) WithScorer(sc JobScorer) *Service {
	s.scorer = sc
	return s
}

// HandleListingExtract is invoked for each verified listing.extract push delivery. It
// loads the raw payload, extracts structured fields via Claude, deduplicates across
// boards via the fingerprint, optionally upserts the recruiter contact, creates or
// merges the Job, then triggers scoring for the active profile.
func (s *Service) HandleListingExtract(ctx context.Context, msg messaging.Message) error {
	// The scrape-worker publishes the raw listing id in both the typed attribute and the
	// message body (see app/scraping.captureCard). The attribute is the canonical source;
	// the body is a defensive fallback so an id is never silently lost if a transport (or
	// a test) populates only one of them. Reading both cannot mask data loss: when neither
	// carries an id we fail loudly below rather than extract a blank listing.
	rawListingID := kernel.RawListingID(msg.Attributes[appscraping.ExtractRawListingIDAttr])
	if rawListingID == "" {
		rawListingID = kernel.RawListingID(msg.Data)
	}
	if rawListingID == "" {
		return fmt.Errorf("listing.extract message carries no raw listing id")
	}

	rawListing, err := s.rawListings.Get(ctx, rawListingID)
	if err != nil {
		return fmt.Errorf("loading raw listing %q: %w", rawListingID, err)
	}
	ref := toRef(rawListing)

	raw, err := s.rawStore.Load(ctx, ref.RawRef)
	if err != nil {
		return fmt.Errorf("loading raw payload %q: %w", ref.RawRef, err)
	}

	extracted, err := s.extractor.Extract(ctx, string(raw))
	if err != nil {
		return fmt.Errorf("extracting listing %q: %w", rawListingID, err)
	}

	// P3-EX-3: upsert recruiter contact before creating the job so we have the
	// contact id available to set on the job row.
	contactID, err := s.upsertRecruiter(ctx, extracted)
	if err != nil {
		return fmt.Errorf("upserting recruiter contact for listing %q: %w", rawListingID, err)
	}

	now := time.Now().UTC()
	fingerprint := computeFingerprint(ref.Title, ref.Company, ref.Location)
	source := jobs.JobSource{
		BoardID:      ref.BoardID,
		RawListingID: ref.ID,
		SourceURL:    ref.SourceURL,
	}

	// P3-EX-2: look up by fingerprint — merge into the existing job when found,
	// create a new one otherwise.
	jobID, found, err := s.jobs.FindByFingerprint(ctx, fingerprint)
	if err != nil {
		return fmt.Errorf("checking fingerprint for listing %q: %w", rawListingID, err)
	}

	if found {
		if err := s.jobs.MergeSource(ctx, jobID, source, now, contactID); err != nil {
			return fmt.Errorf("merging source for listing %q into job %q: %w", rawListingID, jobID, err)
		}
		s.logger.InfoContext(ctx, "listing merged into existing job",
			"job_id", string(jobID), "raw_listing_id", string(rawListingID))
	} else {
		job := buildJob(extracted, ref, now)
		job.Fingerprint = &fingerprint
		job.ContactID = contactID
		jobID, err = s.jobs.Create(ctx, job)
		if err != nil {
			return fmt.Errorf("creating job from listing %q: %w", rawListingID, err)
		}
		s.logger.InfoContext(ctx, "job created from listing",
			"job_id", string(jobID), "raw_listing_id", string(rawListingID),
			"understanding", extracted.Understanding.Int())
	}

	// Not atomic with the job write above: on redelivery this can produce a duplicate
	// job. See tech_debt.md "Extraction is not idempotent".
	if err := s.rawListings.MarkExtracted(ctx, rawListingID); err != nil {
		return fmt.Errorf("marking listing %q extracted: %w", rawListingID, err)
	}

	// P3-EX-4: trigger scoring for the profile that owns this raw listing.
	if s.scorer != nil {
		if err := s.scorer.ScoreJob(ctx, jobID, ref.ProfileID); err != nil {
			// Scoring failure is non-fatal: log and continue so the job is not lost.
			s.logger.WarnContext(ctx, "scoring failed after extraction",
				"job_id", string(jobID), "profile_id", string(ref.ProfileID), "err", err)
		}
	}

	return nil
}

// upsertRecruiter upserts a contact when the extracted listing carries a recruiter.
// Returns nil when no recruiter was present (hidden or Easy Apply listings).
func (s *Service) upsertRecruiter(ctx context.Context, e *llm.ExtractedListing) (*kernel.ContactID, error) {
	if s.contacts == nil || e.Recruiter.Value == nil {
		return nil, nil
	}
	rec := e.Recruiter.Value
	if rec.Email == "" && rec.LinkedInURL == "" {
		// Recruiter block present but no dedup-key fields; skip to avoid a validation error.
		return nil, nil
	}
	id, err := s.contacts.UpsertContact(ctx, rec.Name, "", rec.Email, rec.LinkedInURL, rec.Phone)
	if err != nil {
		return nil, fmt.Errorf("upserting recruiter %q: %w", rec.Email, err)
	}
	return &id, nil
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
			"recruiter":     e.Recruiter.Confidence.Int(),
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
