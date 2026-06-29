package scraping

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
	"github.com/g-trinh/job-tendencies/internal/domain/scraping"
)

const testBoardID = kernel.BoardID("board-1")

type fakeAdapterSource struct{ spec llm.AdapterSpec }

func (f fakeAdapterSource) ApprovedBoardAdapters(context.Context) ([]BoardAdapter, error) {
	spec := f.spec
	if spec.Incremental.OverlapBuffer == "" {
		spec.Incremental.OverlapBuffer = "36h"
	}
	if spec.Incremental.SafetyMaxPages == 0 {
		spec.Incremental.SafetyMaxPages = 5
	}
	if spec.Listing.Fetch == "" {
		spec.Listing.Fetch = llm.ListingFetchUseSearchPayload
	}
	if spec.Search.Pagination.Start == 0 {
		spec.Search.Pagination.Start = 1
	}
	return []BoardAdapter{{BoardID: testBoardID, Spec: spec}}, nil
}

type fakeTargetSource struct{}

func (fakeTargetSource) ActiveTarget(context.Context) (ScrapeTarget, error) {
	return ScrapeTarget{ProfileID: "profile-1", Keywords: []string{"go"}, Location: "Paris"}, nil
}

// fakeFetcher returns the same two cards on page 1 and nothing after, so each run
// re-scans the identical listings (simulating the incremental overlap window).
type fakeFetcher struct{ posted time.Time }

func (f fakeFetcher) FetchPage(_ context.Context, _ llm.AdapterSpec, _ ScrapeTarget, page int) ([]Card, error) {
	if page != 1 {
		return nil, nil
	}
	return []Card{
		{ListingURL: "u1", Title: "A", PostedAt: &f.posted, Raw: []byte(`{"id":1}`)},
		{ListingURL: "u2", Title: "B", PostedAt: &f.posted, Raw: []byte(`{"id":2}`)},
	}, nil
}

// cutoffFetcher returns cards with progressively older timestamps so the HWM cutoff
// test can verify the crawl stops mid-page. Each page holds one card whose PostedAt
// decrements by 48 h relative to the previous page (newest-first order).
type cutoffFetcher struct {
	newest time.Time
}

func (f cutoffFetcher) FetchPage(_ context.Context, _ llm.AdapterSpec, _ ScrapeTarget, page int) ([]Card, error) {
	t := f.newest.Add(-time.Duration(page-1) * 48 * time.Hour)
	raw := []byte(fmt.Sprintf(`{"page":%d}`, page))
	return []Card{
		{ListingURL: fmt.Sprintf("u%d", page), PostedAt: &t, Raw: raw},
	}, nil
}

type fakeStore struct{ writes int }

func (s *fakeStore) Store(context.Context, string, []byte) error { s.writes++; return nil }
func (s *fakeStore) Load(context.Context, string) ([]byte, error) {
	return nil, io.EOF
}

// fakeRawRepo records saved content hashes so re-saves of the same payload are detected.
type fakeRawRepo struct {
	seen  map[string]bool
	saved int
}

func (r *fakeRawRepo) ExistsByContentHash(_ context.Context, _ kernel.BoardID, _ kernel.ProfileID, hash string) (bool, error) {
	return r.seen[hash], nil
}

func (r *fakeRawRepo) Save(_ context.Context, l scraping.RawListing) (kernel.RawListingID, error) {
	r.seen[l.ContentHash] = true
	r.saved++
	return kernel.RawListingID(l.ContentHash), nil
}

type fakeHWM struct{ value *time.Time }

func (h *fakeHWM) Get(context.Context, kernel.BoardID, kernel.ProfileID) (*time.Time, error) {
	return h.value, nil
}

func (h *fakeHWM) Set(_ context.Context, _ kernel.BoardID, _ kernel.ProfileID, t time.Time) error {
	h.value = &t
	return nil
}

type fakePublisher struct{ published int }

func (p *fakePublisher) Publish(context.Context, messaging.Message) error { p.published++; return nil }

func newTestService(fetcher SearchFetcher, raw *fakeRawRepo, hwm *fakeHWM, pub *fakePublisher) *Service {
	return newTestServiceWithSpec(llm.AdapterSpec{}, fetcher, raw, hwm, pub)
}

func newTestServiceWithSpec(spec llm.AdapterSpec, fetcher SearchFetcher, raw *fakeRawRepo, hwm *fakeHWM, pub *fakePublisher) *Service {
	store := &fakeStore{}
	return New(
		fakeAdapterSource{spec: spec},
		fakeTargetSource{},
		fetcher,
		store,
		raw,
		hwm,
		pub,
		nil, // no-op tracker
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

// --- P3-SCR-1/2: basic crawl + content_hash dedup ---

func TestService_HandleScrapeTick(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	raw := &fakeRawRepo{seen: map[string]bool{}}
	hwm := &fakeHWM{}
	pub := &fakePublisher{}
	svc := newTestService(fakeFetcher{posted: now}, raw, hwm, pub)
	ctx := context.Background()

	if err := svc.HandleScrapeTick(ctx, messaging.Message{}); err != nil {
		t.Fatalf("first run error = %v", err)
	}
	if pub.published != 2 {
		t.Fatalf("first run published = %d, want 2 (one per new listing)", pub.published)
	}
	if raw.saved != 2 {
		t.Fatalf("first run saved = %d, want 2", raw.saved)
	}
	if hwm.value == nil {
		t.Fatalf("first run did not advance the high-water-mark")
	}

	// Second run re-scans the identical cards within the overlap window: content_hash
	// dedup must skip them — no new saves and no duplicate publishes.
	if err := svc.HandleScrapeTick(ctx, messaging.Message{}); err != nil {
		t.Fatalf("second run error = %v", err)
	}
	if pub.published != 2 {
		t.Fatalf("second run published = %d, want still 2 (overlap skipped by content_hash)", pub.published)
	}
	if raw.saved != 2 {
		t.Fatalf("second run saved = %d, want still 2", raw.saved)
	}
}

// --- P3-SCR-3: HWM cutoff + safety cap ---

// TestCrawlBoard_SafetyCapEnforced verifies that the first crawl (no HWM) stops at
// safety_max_pages even when the fetcher would return pages indefinitely.
func TestCrawlBoard_SafetyCapEnforced(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	// cutoffFetcher returns one card per page with posted_at going backward; with no HWM
	// the cutoff is nil so the crawler must stop at safety_max_pages.
	raw := &fakeRawRepo{seen: map[string]bool{}}
	hwm := &fakeHWM{} // nil HWM → first crawl
	pub := &fakePublisher{}

	spec := llm.AdapterSpec{
		Incremental: llm.IncrementalConfig{OverlapBuffer: "36h", SafetyMaxPages: 3},
		Listing:     llm.ListingConfig{Fetch: llm.ListingFetchUseSearchPayload},
		Search:      llm.SearchConfig{Pagination: llm.PaginationConfig{Start: 1}},
	}
	svc := newTestServiceWithSpec(spec, cutoffFetcher{newest: now}, raw, hwm, pub)

	if err := svc.HandleScrapeTick(context.Background(), messaging.Message{}); err != nil {
		t.Fatalf("HandleScrapeTick error = %v", err)
	}

	// 3 pages × 1 card each = 3 listings (safety cap = 3 pages)
	if raw.saved != 3 {
		t.Fatalf("saved = %d, want 3 (one per page up to safety cap)", raw.saved)
	}
	if hwm.value == nil {
		t.Fatalf("HWM not set after first crawl")
	}
}

// TestCrawlBoard_CutoffStopsPagination verifies that a subsequent crawl stops when
// cards' posted_at falls below (hwm − overlap_buffer), not at the safety cap.
func TestCrawlBoard_CutoffStopsPagination(t *testing.T) {
	t.Parallel()

	// HWM = now → cutoff = now - 36h.
	// cutoffFetcher page 1: now (> cutoff) → captured.
	// cutoffFetcher page 2: now - 48h (< cutoff) → stop.
	// Safety cap = 20, so it must stop at page 2, not at the cap.
	now := time.Now().UTC()
	hwmTime := now

	raw := &fakeRawRepo{seen: map[string]bool{}}
	hwm := &fakeHWM{value: &hwmTime}
	pub := &fakePublisher{}

	spec := llm.AdapterSpec{
		Incremental: llm.IncrementalConfig{OverlapBuffer: "36h", SafetyMaxPages: 20},
		Listing:     llm.ListingConfig{Fetch: llm.ListingFetchUseSearchPayload},
		Search:      llm.SearchConfig{Pagination: llm.PaginationConfig{Start: 1}},
	}
	svc := newTestServiceWithSpec(spec, cutoffFetcher{newest: now}, raw, hwm, pub)

	if err := svc.HandleScrapeTick(context.Background(), messaging.Message{}); err != nil {
		t.Fatalf("HandleScrapeTick error = %v", err)
	}

	// page 1 captured, page 2 triggers cutoff → stop before reaching safety cap.
	if raw.saved != 1 {
		t.Fatalf("saved = %d, want 1 (stopped at cutoff after page 1, not safety cap)", raw.saved)
	}
}

// TestComputeCutoff_NilHWMYieldsNil verifies the first-ever-crawl path: a nil HWM
// produces a nil cutoff so the crawler runs to the safety cap.
func TestComputeCutoff_NilHWMYieldsNil(t *testing.T) {
	t.Parallel()

	cutoff, err := computeCutoff(nil, "36h")
	if err != nil {
		t.Fatalf("computeCutoff(nil) error = %v", err)
	}
	if cutoff != nil {
		t.Fatalf("computeCutoff(nil) = %v, want nil", cutoff)
	}
}

// TestComputeCutoff_AppliesOverlapBuffer verifies that the cutoff is hwm minus the
// overlap buffer, so late-indexed posts inside the window are re-scanned.
func TestComputeCutoff_AppliesOverlapBuffer(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	cutoff, err := computeCutoff(&base, "36h")
	if err != nil {
		t.Fatalf("computeCutoff error = %v", err)
	}
	if cutoff == nil {
		t.Fatalf("computeCutoff = nil, want time")
	}
	want := base.Add(-36 * time.Hour)
	if !cutoff.Equal(want) {
		t.Fatalf("cutoff = %v, want %v", *cutoff, want)
	}
}
