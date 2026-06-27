package scraping

import (
	"context"
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

type fakeAdapterSource struct{}

func (fakeAdapterSource) ApprovedBoardAdapters(context.Context) ([]BoardAdapter, error) {
	return []BoardAdapter{{
		BoardID: testBoardID,
		Spec: llm.AdapterSpec{
			Search:      llm.SearchConfig{Pagination: llm.PaginationConfig{Start: 1}},
			Listing:     llm.ListingConfig{Fetch: llm.ListingFetchUseSearchPayload},
			Incremental: llm.IncrementalConfig{OverlapBuffer: "36h", SafetyMaxPages: 5},
		},
	}}, nil
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

func newTestService(store *fakeStore, raw *fakeRawRepo, hwm *fakeHWM, pub *fakePublisher) *Service {
	return New(fakeAdapterSource{}, fakeTargetSource{}, fakeFetcher{posted: time.Now().UTC()},
		store, raw, hwm, pub, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestService_HandleScrapeTick(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	raw := &fakeRawRepo{seen: map[string]bool{}}
	hwm := &fakeHWM{}
	pub := &fakePublisher{}
	svc := newTestService(store, raw, hwm, pub)
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
