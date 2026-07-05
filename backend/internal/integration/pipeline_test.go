// Package integration exercises the full pipeline flow — scrape → extract → dedup →
// score → job visible — across the scraping, extraction, scoring and job-browser
// bounded contexts wired together the same way cmd/scrape-worker and cmd/extract-worker
// wire them in production (P5-6, pipeline.md full flow).
//
// It runs entirely in-process against in-memory fakes for the outer ports (blobstore,
// raw listing storage, LLM extraction, scoring/profile storage): no real Postgres, GCS,
// or Pub/Sub broker is available in this environment (Makefile's "Open Question #1" also
// blocks pushing images / deploying to Cloud Run dev, so a literal deployed dev pipeline
// run is an infra/ops follow-up, not something this suite can execute). What this test
// does prove, faithfully, is that the real application services — appscraping.Service,
// appextraction.Service, appscoring.Service, appjobs.Service, using the same
// infra/scoring adapters cmd/extract-worker wires — cooperate correctly end to end: a
// captured listing becomes a scored, browsable job, and redelivery does not duplicate it
// (P5-1, gating this task).
package integration

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	appextraction "github.com/g-trinh/job-tendencies/internal/app/extraction"
	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	appscoring "github.com/g-trinh/job-tendencies/internal/app/scoring"
	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/domain/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
	"github.com/g-trinh/job-tendencies/internal/domain/scoring"
	domainscraping "github.com/g-trinh/job-tendencies/internal/domain/scraping"
	infrascoring "github.com/g-trinh/job-tendencies/internal/infra/scoring"
)

const (
	testProfileID = kernel.ProfileID("profile-1")
	testBoardID   = kernel.BoardID("board-1")
)

// ---------------------------------------------------------------------------
// In-memory fakes for the outer ports. Every fake here mirrors the shape of the
// corresponding infra/* Postgres/GCS implementation closely enough that swapping one
// in for the other is a pure wiring change, not a behaviour change.
// ---------------------------------------------------------------------------

// memoryBlobStore is a shared in-memory stand-in for GCS: what scrape-worker Stores,
// extract-worker Loads back by the same ref.
type memoryBlobStore struct {
	objects map[string][]byte
}

func newMemoryBlobStore() *memoryBlobStore { return &memoryBlobStore{objects: map[string][]byte{}} }

func (b *memoryBlobStore) Store(_ context.Context, path string, data []byte) error {
	b.objects[path] = data
	return nil
}

func (b *memoryBlobStore) Load(_ context.Context, path string) ([]byte, error) {
	data, ok := b.objects[path]
	if !ok {
		return nil, fmt.Errorf("object %q not found", path)
	}
	return data, nil
}

// memoryRawListingStore is a shared in-memory stand-in for the raw_listing table: it
// implements both domain/scraping.RawListingRepository (scrape-worker's write side) and
// domain/scraping.RawListingSource (extract-worker's read side), the same table
// two different worker binaries read/write in production.
type memoryRawListingStore struct {
	byID        map[kernel.RawListingID]domainscraping.RawListing
	byHash      map[string]bool
	nextID      int
	markedCount map[kernel.RawListingID]int
}

func newMemoryRawListingStore() *memoryRawListingStore {
	return &memoryRawListingStore{
		byID:        map[kernel.RawListingID]domainscraping.RawListing{},
		byHash:      map[string]bool{},
		markedCount: map[kernel.RawListingID]int{},
	}
}

func (s *memoryRawListingStore) ExistsByContentHash(_ context.Context, _ kernel.BoardID, _ kernel.ProfileID, hash string) (bool, error) {
	return s.byHash[hash], nil
}

func (s *memoryRawListingStore) Save(_ context.Context, l domainscraping.RawListing) (kernel.RawListingID, error) {
	s.nextID++
	id := kernel.RawListingID(fmt.Sprintf("raw-%d", s.nextID))
	l.ID = id
	s.byID[id] = l
	s.byHash[l.ContentHash] = true
	return id, nil
}

func (s *memoryRawListingStore) Get(_ context.Context, id kernel.RawListingID) (domainscraping.RawListing, error) {
	l, ok := s.byID[id]
	if !ok {
		return domainscraping.RawListing{}, &kernel.NotFoundError{Kind: "raw_listing", ID: string(id)}
	}
	return l, nil
}

func (s *memoryRawListingStore) MarkExtracted(_ context.Context, id kernel.RawListingID) error {
	s.markedCount[id]++
	return nil
}

// memoryHWM is a no-baseline high-water-mark: every crawl in this test is a
// first-ever crawl, which is enough to exercise capture -> extract -> score.
type memoryHWM struct{ value *time.Time }

func (h *memoryHWM) Get(context.Context, kernel.BoardID, kernel.ProfileID) (*time.Time, error) {
	return h.value, nil
}

func (h *memoryHWM) Set(_ context.Context, _ kernel.BoardID, _ kernel.ProfileID, t time.Time) error {
	h.value = &t
	return nil
}

// fakeAdapterSource returns one board with a minimal, valid adapter spec.
type fakeAdapterSource struct{}

func (fakeAdapterSource) ApprovedBoardAdapters(context.Context) ([]appscraping.BoardAdapter, error) {
	return []appscraping.BoardAdapter{{
		BoardID: testBoardID,
		Spec: llm.AdapterSpec{
			Incremental: llm.IncrementalConfig{OverlapBuffer: "36h", SafetyMaxPages: 3},
			Listing:     llm.ListingConfig{Fetch: llm.ListingFetchUseSearchPayload},
			Search:      llm.SearchConfig{Pagination: llm.PaginationConfig{Start: 1}},
		},
	}}, nil
}

// fakeTargetSource resolves the single test profile as the active scrape target.
type fakeTargetSource struct{}

func (fakeTargetSource) ActiveTarget(context.Context) (appscraping.ScrapeTarget, error) {
	return appscraping.ScrapeTarget{ProfileID: testProfileID, Keywords: []string{"go"}, Location: "Paris"}, nil
}

// fakeSearchFetcher returns one listing on page 1, nothing after — a minimal but
// realistic single-board, single-listing crawl.
type fakeSearchFetcher struct{ served bool }

func (f *fakeSearchFetcher) FetchPage(_ context.Context, _ llm.AdapterSpec, _ appscraping.ScrapeTarget, page int) ([]appscraping.Card, error) {
	if page != 1 || f.served {
		return nil, nil
	}
	f.served = true
	now := time.Now().UTC()
	return []appscraping.Card{{
		ListingURL: "https://wttj.co/jobs/go-engineer",
		ExternalID: "wttj-1",
		Title:      "Go Engineer",
		Company:    "Acme Corp",
		Location:   "Paris",
		PostedAt:   &now,
		Raw:        []byte(`{"title":"Go Engineer","company":"Acme Corp"}`),
	}}, nil
}

// fakeExtractor returns a fixed, fully-populated structured extraction for any raw
// payload, standing in for the real Claude call.
type fakeExtractor struct{}

func (fakeExtractor) Extract(context.Context, string) (*llm.ExtractedListing, error) {
	salary := int64(60000)
	listing := &llm.ExtractedListing{
		Skills:        llm.ExtractedField[[]string]{Value: []string{"go", "postgresql"}, Confidence: 90},
		RemotePolicy:  llm.ExtractedField[kernel.RemotePolicy]{Value: kernel.RemotePolicyHybrid, Confidence: 85},
		OfficeDays:    llm.ExtractedField[int]{Value: 2, Confidence: 70},
		ContractType:  llm.ExtractedField[kernel.ContractType]{Value: kernel.ContractTypeCDI, Confidence: 95},
		WorkingDays:   llm.ExtractedField[kernel.WorkingDays]{Value: kernel.WorkingDaysFullTime, Confidence: 80},
		SalaryMin:     llm.ExtractedField[*int64]{Value: &salary, Confidence: 75},
		SalaryMax:     llm.ExtractedField[*int64]{Value: &salary, Confidence: 75},
		Seniority:     llm.ExtractedField[kernel.Seniority]{Value: kernel.SenioritySenior, Confidence: 60},
		Recruiter:     llm.ExtractedField[*llm.Recruiter]{Value: nil, Confidence: 0},
		Understanding: 88,
	}
	return listing, nil
}

// memoryJobStore is a shared in-memory stand-in for the job + job_source tables. It
// implements domain/jobs.Repository (extraction's write side) and exposes GetJob/
// ListJobs matching app/jobs.JobQuery, so a real appjobs.Service can be built over it
// — the same read side the job-browser API and the scoring JobsAdapter both use.
type memoryJobStore struct {
	byID        map[kernel.JobID]jobs.Job
	byFP        map[string]kernel.JobID
	nextID      int
	createCount int
	mergeCount  int
}

func newMemoryJobStore() *memoryJobStore {
	return &memoryJobStore{byID: map[kernel.JobID]jobs.Job{}, byFP: map[string]kernel.JobID{}}
}

func (s *memoryJobStore) Create(_ context.Context, job jobs.Job) (kernel.JobID, error) {
	s.nextID++
	id := kernel.JobID(fmt.Sprintf("job-%d", s.nextID))
	job.ID = id
	s.byID[id] = job
	if job.Fingerprint != nil {
		s.byFP[*job.Fingerprint] = id
	}
	s.createCount++
	return id, nil
}

func (s *memoryJobStore) FindByFingerprint(_ context.Context, fingerprint string) (kernel.JobID, bool, error) {
	id, ok := s.byFP[fingerprint]
	return id, ok, nil
}

func (s *memoryJobStore) MergeSource(_ context.Context, jobID kernel.JobID, source jobs.JobSource, lastSeen time.Time, contactID *kernel.ContactID) error {
	job := s.byID[jobID]
	job.Sources = append(job.Sources, source)
	job.LastSeen = lastSeen
	if job.ContactID == nil {
		job.ContactID = contactID
	}
	s.byID[jobID] = job
	s.mergeCount++
	return nil
}

// GetJob implements the narrow jobsServiceFacade infra/scoring.JobsAdapter consumes,
// and app/jobs.JobQuery.GetByProfile via the same shape (P5-6 exercises both callers).
func (s *memoryJobStore) GetJob(_ context.Context, _ kernel.ProfileID, id kernel.JobID) (appjobs.JobView, error) {
	return s.GetByProfile(context.Background(), testProfileID, id)
}

func (s *memoryJobStore) GetByProfile(_ context.Context, _ kernel.ProfileID, id kernel.JobID) (appjobs.JobView, error) {
	job, ok := s.byID[id]
	if !ok {
		return appjobs.JobView{}, &kernel.NotFoundError{Kind: "job", ID: string(id)}
	}
	return toJobView(job), nil
}

func (s *memoryJobStore) ListByProfile(_ context.Context, _ kernel.ProfileID, _ appjobs.JobListFilter) (appjobs.JobListResult, error) {
	out := make([]appjobs.JobView, 0, len(s.byID))
	for _, job := range s.byID {
		out = append(out, toJobView(job))
	}
	return appjobs.JobListResult{Items: out, Total: len(out)}, nil
}

func toJobView(job jobs.Job) appjobs.JobView {
	sources := make([]appjobs.JobSourceView, 0, len(job.Sources))
	for _, src := range job.Sources {
		sources = append(sources, appjobs.JobSourceView{BoardID: src.BoardID, SourceURL: src.SourceURL, BoardName: "WTTJ"})
	}
	return appjobs.JobView{
		ID: job.ID, Title: job.Title, Company: job.Company, Location: job.Location, URL: job.URL,
		Skills: job.Skills, RemotePolicy: job.RemotePolicy, OfficeDays: job.OfficeDays,
		ContractType: job.ContractType, WorkingDays: job.WorkingDays,
		SalaryMin: job.SalaryMin, SalaryMax: job.SalaryMax, Seniority: job.Seniority,
		FieldConfidence: job.FieldConfidence, UnderstandingScore: job.UnderstandingScore,
		ContactID: contactIDPtr(job.ContactID), FirstSeen: job.FirstSeen, LastSeen: job.LastSeen,
		ExpiredAt: job.ExpiredAt, Sources: sources,
	}
}

func contactIDPtr(id *kernel.ContactID) *string {
	if id == nil {
		return nil
	}
	s := string(*id)
	return &s
}

// memoryScoringRepo is a shared in-memory stand-in for the job_score table.
type memoryScoringRepo struct {
	scores map[string]scoring.JobScore
}

func newMemoryScoringRepo() *memoryScoringRepo {
	return &memoryScoringRepo{scores: map[string]scoring.JobScore{}}
}

func scoreKey(jobID kernel.JobID, profileID kernel.ProfileID) string {
	return string(jobID) + "|" + string(profileID)
}

func (r *memoryScoringRepo) Upsert(_ context.Context, score scoring.JobScore) error {
	r.scores[scoreKey(score.JobID, score.ProfileID)] = score
	return nil
}

func (r *memoryScoringRepo) FindByJobAndProfile(_ context.Context, jobID kernel.JobID, profileID kernel.ProfileID) (scoring.JobScore, error) {
	sc, ok := r.scores[scoreKey(jobID, profileID)]
	if !ok {
		return scoring.JobScore{}, &kernel.NotFoundError{Kind: "job_score", ID: scoreKey(jobID, profileID)}
	}
	return sc, nil
}

// fakeProfileFacade satisfies infra/scoring's profilesServiceFacade directly, standing
// in for appprofiles.Service.ProfileByID, with a trivial profile (no dealbreakers) so
// any extracted job passes the gate and gets a well-defined weighted score.
type fakeProfileFacade struct{ profile profiles.Profile }

func (f fakeProfileFacade) ProfileByID(context.Context, kernel.ProfileID) (profiles.Profile, error) {
	return f.profile, nil
}

func testProfile(t *testing.T) profiles.Profile {
	t.Helper()
	p, err := profiles.NewProfile("Go Backend Paris", "Paris", []string{"go"})
	if err != nil {
		t.Fatalf("building test profile: %v", err)
	}
	p.ID = testProfileID
	return p
}

// syncPublisher bridges scrape-worker's listing.extract publish directly into
// extract-worker's push handler, synchronously, standing in for the Pub/Sub round trip
// (push subscription -> OIDC-verified POST) neither of which is available in this
// environment. Each call records the delivery count so tests can assert on redelivery.
type syncPublisher struct {
	extraction  *appextraction.Service
	deliveries  int
	deliverFunc func(context.Context, messaging.Message) error
}

func (p *syncPublisher) Publish(ctx context.Context, msg messaging.Message) error {
	p.deliveries++
	deliver := p.deliverFunc
	if deliver == nil {
		deliver = p.extraction.HandleListingExtract
	}
	return deliver(ctx, msg)
}

// ---------------------------------------------------------------------------
// TestPipeline_ScrapeExtractDedupScoreJobVisible (P5-6)
// ---------------------------------------------------------------------------

// TestPipeline_ScrapeExtractDedupScoreJobVisible wires the real scraping, extraction,
// scoring and job-browser application services together (the same composition
// cmd/scrape-worker + cmd/extract-worker use) and drives one scrape.tick through the
// full pipeline: capture -> extract -> dedup/merge -> score -> browsable job.
//
// AC (P5-6): the integration suite passes and exercises scrape -> extract -> dedup ->
// score -> job visible.
func TestPipeline_ScrapeExtractDedupScoreJobVisible(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	blobs := newMemoryBlobStore()
	rawStore := newMemoryRawListingStore()
	jobStore := newMemoryJobStore()
	scoringRepo := newMemoryScoringRepo()

	scoringSvc := appscoring.New(
		infrascoring.NewJobsAdapter(jobStore),
		infrascoring.NewProfilesAdapter(fakeProfileFacade{profile: testProfile(t)}),
		scoringRepo,
	)

	extractionSvc := appextraction.New(rawStore, blobs, fakeExtractor{}, jobStore, logger).
		WithScorer(scorerAdapter{svc: scoringSvc})

	pub := &syncPublisher{extraction: extractionSvc}

	scrapingSvc := appscraping.New(
		fakeAdapterSource{}, fakeTargetSource{}, &fakeSearchFetcher{},
		blobs, rawStore, &memoryHWM{}, pub, nil, nil, logger,
	)

	if err := scrapingSvc.HandleScrapeTick(context.Background(), messaging.Message{}); err != nil {
		t.Fatalf("HandleScrapeTick error = %v", err)
	}

	// scrape -> extract: exactly one listing.extract delivery for the one captured card.
	if pub.deliveries != 1 {
		t.Fatalf("listing.extract deliveries = %d, want 1", pub.deliveries)
	}

	// extract -> job: exactly one job created (dedup has nothing to merge yet on a
	// single-board, single-listing run).
	if jobStore.createCount != 1 {
		t.Fatalf("jobs created = %d, want 1", jobStore.createCount)
	}
	var jobID kernel.JobID
	for id := range jobStore.byID {
		jobID = id
	}

	// score: the scorer ran synchronously inside extraction and persisted a result.
	score, err := scoringSvc.GetScore(context.Background(), jobID, testProfileID)
	if err != nil {
		t.Fatalf("GetScore error = %v", err)
	}
	if !score.PassesDealbreakers {
		t.Fatalf("score.PassesDealbreakers = false, want true (profile has no dealbreakers)")
	}
	if score.WeightedScore <= 0 {
		t.Fatalf("score.WeightedScore = %v, want > 0", score.WeightedScore)
	}

	// job visible: the job-browser read service (real appjobs.Service, over the same
	// store) returns the job for the profile with its extracted fields.
	jobBrowserSvc := appjobs.New(jobStore)
	view, err := jobBrowserSvc.GetJob(context.Background(), testProfileID, jobID)
	if err != nil {
		t.Fatalf("GetJob error = %v", err)
	}
	if view.Title != "Go Engineer" || view.Company != "Acme Corp" {
		t.Fatalf("job view identity fields = %+v, want title=Go Engineer company=Acme Corp", view)
	}
	if len(view.Skills) == 0 {
		t.Fatalf("job view has no skills; extraction result was not applied")
	}
	list, err := jobBrowserSvc.ListJobs(context.Background(), testProfileID, appjobs.JobListFilter{})
	if err != nil {
		t.Fatalf("ListJobs error = %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("ListJobs returned %d jobs, want 1 (the job must be browsable)", len(list.Items))
	}

	// dedup / idempotency (P5-1, gating this task): redelivering the same
	// listing.extract message must merge into the existing job, not duplicate it.
	if err := extractionSvc.HandleListingExtract(context.Background(), lastRawListingMessage(rawStore)); err != nil {
		t.Fatalf("redelivery HandleListingExtract error = %v", err)
	}
	if jobStore.createCount != 1 {
		t.Fatalf("jobs created after redelivery = %d, want still 1", jobStore.createCount)
	}
	if jobStore.mergeCount != 1 {
		t.Fatalf("merge count after redelivery = %d, want 1", jobStore.mergeCount)
	}
}

// scorerAdapter bridges app/scoring.Service to app/extraction.JobScorer, mirroring
// cmd/extract-worker/main.go's scorerAdapter exactly.
type scorerAdapter struct{ svc *appscoring.Service }

func (a scorerAdapter) ScoreJob(ctx context.Context, jobID kernel.JobID, profileID kernel.ProfileID) error {
	_, err := a.svc.ScoreJob(ctx, jobID, profileID)
	if err != nil {
		return fmt.Errorf("scoring job %q for profile %q: %w", jobID, profileID, err)
	}
	return nil
}

// lastRawListingMessage rebuilds the listing.extract message for the single raw listing
// captured in this test, for the redelivery assertion.
func lastRawListingMessage(rawStore *memoryRawListingStore) messaging.Message {
	var id kernel.RawListingID
	for rid := range rawStore.byID {
		id = rid
	}
	return messaging.Message{
		Data:       []byte(id),
		Attributes: map[string]string{appscraping.ExtractRawListingIDAttr: string(id)},
	}
}
