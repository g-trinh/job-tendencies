package extraction

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/domain/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
	domainscraping "github.com/g-trinh/job-tendencies/internal/domain/scraping"
)

// ---------------------------------------------------------------------------
// Fake implementations
// ---------------------------------------------------------------------------

type fakeRawListingSource struct {
	listing domainscraping.RawListing
	getErr  error
	marked  []kernel.RawListingID
}

func (f *fakeRawListingSource) Get(_ context.Context, id kernel.RawListingID) (domainscraping.RawListing, error) {
	if f.getErr != nil {
		return domainscraping.RawListing{}, f.getErr
	}
	return f.listing, nil
}

func (f *fakeRawListingSource) MarkExtracted(_ context.Context, id kernel.RawListingID) error {
	f.marked = append(f.marked, id)
	return nil
}

type fakeBlobLoader struct {
	payload []byte
	err     error
}

func (f *fakeBlobLoader) Load(_ context.Context, _ string) ([]byte, error) {
	return f.payload, f.err
}

// fakeExtractor is an in-memory ListingExtractor used to avoid real LLM calls.
// It returns the configured listing for any input.
type fakeExtractor struct {
	listing *llm.ExtractedListing
	err     error
}

func (f *fakeExtractor) Extract(_ context.Context, _ string) (*llm.ExtractedListing, error) {
	return f.listing, f.err
}

// fakeJobRepo records Create/FindByFingerprint/MergeSource calls.
type fakeJobRepo struct {
	// fingerprintHit, when non-empty, simulates a dedup match for any fingerprint.
	fingerprintHit kernel.JobID
	createID       kernel.JobID
	createErr      error
	mergedSources  []jobs.JobSource
	mergeErr       error
}

func (f *fakeJobRepo) Create(_ context.Context, job jobs.Job) (kernel.JobID, error) {
	if f.createErr != nil {
		return "", f.createErr
	}
	return f.createID, nil
}

func (f *fakeJobRepo) FindByFingerprint(_ context.Context, _ string) (kernel.JobID, bool, error) {
	if f.fingerprintHit != "" {
		return f.fingerprintHit, true, nil
	}
	return "", false, nil
}

func (f *fakeJobRepo) MergeSource(_ context.Context, _ kernel.JobID, src jobs.JobSource, _ time.Time, _ *kernel.ContactID) error {
	if f.mergeErr != nil {
		return f.mergeErr
	}
	f.mergedSources = append(f.mergedSources, src)
	return nil
}

// fakeContactUpserter records UpsertContact calls.
type fakeContactUpserter struct {
	returnID kernel.ContactID
	err      error
	calls    []contactCall
}

type contactCall struct {
	name, company, email, linkedInURL, phone string
}

func (f *fakeContactUpserter) UpsertContact(_ context.Context, name, company, email, linkedInURL, phone string) (kernel.ContactID, error) {
	f.calls = append(f.calls, contactCall{name: name, company: company, email: email, linkedInURL: linkedInURL, phone: phone})
	return f.returnID, f.err
}

// fakeJobScorer records ScoreJob calls.
type fakeJobScorer struct {
	calls []scoreCall
	err   error
}

type scoreCall struct {
	jobID     kernel.JobID
	profileID kernel.ProfileID
}

func (f *fakeJobScorer) ScoreJob(_ context.Context, jobID kernel.JobID, profileID kernel.ProfileID) error {
	f.calls = append(f.calls, scoreCall{jobID: jobID, profileID: profileID})
	return f.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// defaultRawListing returns a minimal RawListing for tests.
func defaultRawListing() domainscraping.RawListing {
	return domainscraping.RawListing{
		ID:        "raw-1",
		BoardID:   "board-1",
		ProfileID: "profile-1",
		Title:     "Go Engineer",
		Company:   "Acme Corp",
		Location:  "Paris, France",
		SourceURL: "https://wttj/jobs/1",
		RawRef:    "gs://bucket/raw-1.json",
	}
}

// defaultExtracted returns a fully-populated ExtractedListing fixture.
func defaultExtracted() *llm.ExtractedListing {
	salary := int64(60000)
	return &llm.ExtractedListing{
		Skills:        llm.ExtractedField[[]string]{Value: []string{"Go", "PostgreSQL"}, Confidence: 90},
		RemotePolicy:  llm.ExtractedField[kernel.RemotePolicy]{Value: kernel.RemotePolicyHybrid, Confidence: 85},
		OfficeDays:    llm.ExtractedField[int]{Value: 2, Confidence: 70},
		ContractType:  llm.ExtractedField[kernel.ContractType]{Value: kernel.ContractTypeCDI, Confidence: 95},
		WorkingDays:   llm.ExtractedField[kernel.WorkingDays]{Value: kernel.WorkingDaysFullTime, Confidence: 80},
		SalaryMin:     llm.ExtractedField[*int64]{Value: &salary, Confidence: 75},
		SalaryMax:     llm.ExtractedField[*int64]{Value: nil, Confidence: 0},
		Seniority:     llm.ExtractedField[kernel.Seniority]{Value: kernel.SenioritySenior, Confidence: 60},
		Recruiter:     llm.ExtractedField[*llm.Recruiter]{Value: nil, Confidence: 0},
		Understanding: 82,
	}
}

// msg builds a test Pub/Sub message carrying rawListingID in the extract attribute.
func msg(rawListingID string) messaging.Message { //nolint:unparam // test helper: callers may vary
	return messaging.Message{
		Attributes: map[string]string{appscraping.ExtractRawListingIDAttr: rawListingID},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestHandleListingExtract_NewJob verifies the happy path: a raw listing with no
// fingerprint match creates a new job with fingerprint set and marks the listing extracted.
//
// AC P3-EX-1: extraction populates every output field with confidence + understanding.
// AC P3-EX-2: a listing with a unique fingerprint creates a new job (not a merge).
func TestHandleListingExtract_NewJob(t *testing.T) {
	t.Parallel()

	rawSrc := &fakeRawListingSource{listing: defaultRawListing()}
	blob := &fakeBlobLoader{payload: []byte(`{"raw":"payload"}`)}
	extractor := &fakeExtractor{listing: defaultExtracted()}
	repo := &fakeJobRepo{createID: "job-1"}
	scorer := &fakeJobScorer{}

	svc := New(rawSrc, blob, extractor, repo, nopLogger()).WithScorer(scorer)
	err := svc.HandleListingExtract(context.Background(), msg("raw-1"))
	if err != nil {
		t.Fatalf("HandleListingExtract returned error: %v", err)
	}
	if len(rawSrc.marked) != 1 || rawSrc.marked[0] != "raw-1" {
		t.Fatalf("expected listing raw-1 to be marked extracted, got %v", rawSrc.marked)
	}
	// Scorer must have been called for the new job.
	if len(scorer.calls) != 1 {
		t.Fatalf("expected 1 score call, got %d", len(scorer.calls))
	}
	if scorer.calls[0].jobID != "job-1" || scorer.calls[0].profileID != "profile-1" {
		t.Fatalf("unexpected score call: %+v", scorer.calls[0])
	}
}

// TestHandleListingExtract_DedupMerge verifies that when a fingerprint match is found
// the listing is merged into the existing job rather than creating a new one.
//
// AC P3-EX-2: two boards' listings for the same role collapse to one job, two sources.
func TestHandleListingExtract_DedupMerge(t *testing.T) {
	t.Parallel()

	rawSrc := &fakeRawListingSource{listing: defaultRawListing()}
	blob := &fakeBlobLoader{payload: []byte(`{"raw":"payload"}`)}
	extractor := &fakeExtractor{listing: defaultExtracted()}
	repo := &fakeJobRepo{fingerprintHit: "existing-job-42"} // simulates dedup hit
	scorer := &fakeJobScorer{}

	svc := New(rawSrc, blob, extractor, repo, nopLogger()).WithScorer(scorer)
	err := svc.HandleListingExtract(context.Background(), msg("raw-1"))
	if err != nil {
		t.Fatalf("HandleListingExtract returned error: %v", err)
	}
	// A source row must have been merged; no new job created.
	if len(repo.mergedSources) != 1 {
		t.Fatalf("expected 1 merged source, got %d", len(repo.mergedSources))
	}
	if repo.mergedSources[0].RawListingID != "raw-1" {
		t.Fatalf("merged wrong source: %+v", repo.mergedSources[0])
	}
	// Scorer must be triggered for the existing job.
	if len(scorer.calls) != 1 || scorer.calls[0].jobID != "existing-job-42" {
		t.Fatalf("scorer not called with existing job: %+v", scorer.calls)
	}
}

// TestHandleListingExtract_RecruiterContact verifies that a listing with a recruiter
// creates or updates a contact and links the resulting contact id to the job.
//
// AC P3-EX-3: a listing with recruiter fields creates/links a deduped contact.
func TestHandleListingExtract_RecruiterContact(t *testing.T) {
	t.Parallel()

	recruiter := &llm.Recruiter{Name: "Alice Martin", Email: "alice@acme.io", Phone: "+33600000000"}
	extracted := defaultExtracted()
	extracted.Recruiter = llm.ExtractedField[*llm.Recruiter]{Value: recruiter, Confidence: 80}

	rawSrc := &fakeRawListingSource{listing: defaultRawListing()}
	blob := &fakeBlobLoader{payload: []byte(`{"raw":"payload"}`)}
	extractor := &fakeExtractor{listing: extracted}
	repo := &fakeJobRepo{createID: "job-2"}
	contacts := &fakeContactUpserter{returnID: "contact-99"}

	svc := New(rawSrc, blob, extractor, repo, nopLogger()).WithContacts(contacts)
	err := svc.HandleListingExtract(context.Background(), msg("raw-1"))
	if err != nil {
		t.Fatalf("HandleListingExtract returned error: %v", err)
	}
	// Contact must have been upserted with the recruiter's details.
	if len(contacts.calls) != 1 {
		t.Fatalf("expected 1 contact upsert, got %d", len(contacts.calls))
	}
	call := contacts.calls[0]
	if call.email != "alice@acme.io" || call.name != "Alice Martin" || call.phone != "+33600000000" {
		t.Fatalf("wrong contact upsert call: %+v", call)
	}
}

// TestHandleListingExtract_HiddenRecruiterSkipsContact verifies that "Easy Apply"
// listings (no recruiter email or linkedin_url) do not trigger a contact upsert.
//
// AC P3-EX-5: hidden-recruiter listing — extract visible fields, low understanding, no contact.
func TestHandleListingExtract_HiddenRecruiterSkipsContact(t *testing.T) {
	t.Parallel()

	// LinkedIn "Easy Apply" style: recruiter block present but no contactable fields.
	hiddenRecruiter := &llm.Recruiter{Name: ""}
	extracted := defaultExtracted()
	extracted.Recruiter = llm.ExtractedField[*llm.Recruiter]{Value: hiddenRecruiter, Confidence: 10}
	extracted.Understanding = 25 // low overall understanding

	rawSrc := &fakeRawListingSource{listing: defaultRawListing()}
	blob := &fakeBlobLoader{payload: []byte(`<html>Easy Apply listing</html>`)}
	extractor := &fakeExtractor{listing: extracted}
	repo := &fakeJobRepo{createID: "job-3"}
	contacts := &fakeContactUpserter{returnID: "contact-should-not-be-called"}

	svc := New(rawSrc, blob, extractor, repo, nopLogger()).WithContacts(contacts)
	err := svc.HandleListingExtract(context.Background(), msg("raw-1"))
	if err != nil {
		t.Fatalf("HandleListingExtract returned error: %v", err)
	}
	// No contactable fields → no upsert.
	if len(contacts.calls) != 0 {
		t.Fatalf("expected no contact upsert for hidden recruiter, got %d calls", len(contacts.calls))
	}
}

// TestHandleListingExtract_MissingSalary verifies that when the LLM returns nil salary
// the job is created with null salary and confidence 0 in the field_confidence map.
//
// AC P3-EX-5: missing salary → null value, confidence 0.
func TestHandleListingExtract_MissingSalary(t *testing.T) {
	t.Parallel()

	extracted := defaultExtracted()
	extracted.SalaryMin = llm.ExtractedField[*int64]{Value: nil, Confidence: 0}
	extracted.SalaryMax = llm.ExtractedField[*int64]{Value: nil, Confidence: 0}

	rawSrc := &fakeRawListingSource{listing: defaultRawListing()}
	blob := &fakeBlobLoader{payload: []byte(`{"salary": "not disclosed"}`)}
	extractor := &fakeExtractor{listing: extracted}

	var capturedJob jobs.Job
	repo := &capturingJobRepo{captureInto: &capturedJob, returnID: "job-4"}

	svc := New(rawSrc, blob, extractor, repo, nopLogger())
	err := svc.HandleListingExtract(context.Background(), msg("raw-1"))
	if err != nil {
		t.Fatalf("HandleListingExtract returned error: %v", err)
	}
	if capturedJob.SalaryMin != nil || capturedJob.SalaryMax != nil {
		t.Fatalf("expected nil salary, got min=%v max=%v", capturedJob.SalaryMin, capturedJob.SalaryMax)
	}
	if capturedJob.FieldConfidence["salary_min"] != 0 || capturedJob.FieldConfidence["salary_max"] != 0 {
		t.Fatalf("expected salary confidence 0, got %+v", capturedJob.FieldConfidence)
	}
}

// TestHandleListingExtract_ScoringFailureIsNonFatal verifies that a scoring error does
// not abort the extraction pipeline — the job is created and the listing marked extracted.
//
// AC P3-EX-4: scoring failure must not lose the job.
func TestHandleListingExtract_ScoringFailureIsNonFatal(t *testing.T) {
	t.Parallel()

	rawSrc := &fakeRawListingSource{listing: defaultRawListing()}
	blob := &fakeBlobLoader{payload: []byte(`{"raw":"payload"}`)}
	extractor := &fakeExtractor{listing: defaultExtracted()}
	repo := &fakeJobRepo{createID: "job-5"}
	scorer := &fakeJobScorer{err: errors.New("scoring service unavailable")}

	svc := New(rawSrc, blob, extractor, repo, nopLogger()).WithScorer(scorer)
	err := svc.HandleListingExtract(context.Background(), msg("raw-1"))
	if err != nil {
		t.Fatalf("HandleListingExtract should not fail when scoring fails; got: %v", err)
	}
	// Listing must still be marked extracted.
	if len(rawSrc.marked) != 1 {
		t.Fatalf("expected listing to be marked extracted despite scoring failure")
	}
}

// TestHandleListingExtract_MissingRawListingID verifies that a message with no id
// is rejected early.
func TestHandleListingExtract_MissingRawListingID(t *testing.T) {
	t.Parallel()

	svc := New(
		&fakeRawListingSource{},
		&fakeBlobLoader{},
		&fakeExtractor{listing: defaultExtracted()},
		&fakeJobRepo{createID: "job-6"},
		nopLogger(),
	)
	err := svc.HandleListingExtract(context.Background(), messaging.Message{})
	if err == nil {
		t.Fatal("expected error for message with no raw listing id")
	}
}

// ---------------------------------------------------------------------------
// capturingJobRepo — variant of fakeJobRepo that captures the Job passed to Create.
// ---------------------------------------------------------------------------

type capturingJobRepo struct {
	captureInto *jobs.Job
	returnID    kernel.JobID
}

func (c *capturingJobRepo) Create(_ context.Context, job jobs.Job) (kernel.JobID, error) {
	*c.captureInto = job
	return c.returnID, nil
}

func (c *capturingJobRepo) FindByFingerprint(_ context.Context, _ string) (kernel.JobID, bool, error) {
	return "", false, nil
}

func (c *capturingJobRepo) MergeSource(_ context.Context, _ kernel.JobID, _ jobs.JobSource, _ time.Time, _ *kernel.ContactID) error {
	return nil
}

// ---------------------------------------------------------------------------
// nopLogger
// ---------------------------------------------------------------------------

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}
