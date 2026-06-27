// Package scraping contains the scrape-worker application service: the json_api crawl
// loop with high-water-mark incrementality, raw→GCS capture with content-hash dedup,
// and per-new-listing listing.extract fan-out. Port interfaces are declared here
// (the consumer) and implemented in infra/scraping, infra/blobstore, infra/messaging.
//
// See docs/architecture/pipeline.md §2 for the crawl algorithm.
package scraping

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/blobstore"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
	"github.com/g-trinh/job-tendencies/internal/domain/scraping"
)

// ExtractRawListingIDAttr is the listing.extract message attribute carrying the
// raw listing id to extract.
const ExtractRawListingIDAttr = "raw_listing_id"

// BoardAdapter is a board paired with its approved scraping spec. It is the scraping
// context's view of the board-manager data, mapped at the composition root so the two
// contexts do not share domain objects.
type BoardAdapter struct {
	BoardID kernel.BoardID
	Spec    llm.AdapterSpec
}

// ScrapeTarget is the active profile's board-side search filter.
type ScrapeTarget struct {
	ProfileID kernel.ProfileID
	Keywords  []string
	Location  string
}

// Card is one search-result entry returned by the SearchFetcher, with its raw payload
// captured for storage.
type Card struct {
	// ListingURL is the listing's URL on the board.
	ListingURL string
	// ExternalID is the board's stable listing id, when present.
	ExternalID string
	// Title, Company and Location are identity facts read verbatim from the search card.
	Title    string
	Company  string
	Location string
	// PostedAt is the listing's publication time; nil when the board omits it.
	PostedAt *time.Time
	// Raw is the verbatim payload to store (the search card, or the detail page).
	Raw []byte
}

// AdapterSource provides the approved adapter for every enabled board.
type AdapterSource interface {
	ApprovedBoardAdapters(ctx context.Context) ([]BoardAdapter, error)
}

// TargetSource resolves the active profile's search target.
type TargetSource interface {
	ActiveTarget(ctx context.Context) (ScrapeTarget, error)
}

// SearchFetcher fetches one search page for a board and returns its result cards.
// An empty slice signals there are no further pages.
type SearchFetcher interface {
	FetchPage(ctx context.Context, spec llm.AdapterSpec, target ScrapeTarget, page int) ([]Card, error)
}

// The RawListing capture port and the high-water-mark port are the scraping
// aggregate's repositories and live in domain/scraping (ADR-005), consumed here.

// Service runs the scrape pipeline stage in response to scrape.tick deliveries.
type Service struct {
	adapters  AdapterSource
	targets   TargetSource
	fetcher   SearchFetcher
	rawStore  blobstore.Storer
	rawRepo   scraping.RawListingRepository
	hwm       scraping.HighWaterMarkRepository
	publisher messaging.Publisher
	logger    *slog.Logger
}

// New constructs a scraping Service with all pipeline dependencies wired.
func New(
	adapters AdapterSource,
	targets TargetSource,
	fetcher SearchFetcher,
	rawStore blobstore.Storer,
	rawRepo scraping.RawListingRepository,
	hwm scraping.HighWaterMarkRepository,
	publisher messaging.Publisher,
	logger *slog.Logger,
) *Service {
	return &Service{
		adapters:  adapters,
		targets:   targets,
		fetcher:   fetcher,
		rawStore:  rawStore,
		rawRepo:   rawRepo,
		hwm:       hwm,
		publisher: publisher,
		logger:    logger,
	}
}

// HandleScrapeTick is invoked for each verified scrape.tick push delivery. It resolves
// the active target, then crawls every enabled board's approved adapter.
func (s *Service) HandleScrapeTick(ctx context.Context, _ messaging.Message) error {
	target, err := s.targets.ActiveTarget(ctx)
	if err != nil {
		return fmt.Errorf("resolving active target: %w", err)
	}

	adapters, err := s.adapters.ApprovedBoardAdapters(ctx)
	if err != nil {
		return fmt.Errorf("loading approved adapters: %w", err)
	}

	for _, adapter := range adapters {
		if err := s.crawlBoard(ctx, adapter, target); err != nil {
			return fmt.Errorf("crawling board %q: %w", adapter.BoardID, err)
		}
	}
	return nil
}

// crawlBoard runs the high-water-mark incremental crawl for one board (pipeline.md §2).
func (s *Service) crawlBoard(ctx context.Context, adapter BoardAdapter, target ScrapeTarget) error {
	hwm, err := s.hwm.Get(ctx, adapter.BoardID, target.ProfileID)
	if err != nil {
		return fmt.Errorf("reading high-water-mark: %w", err)
	}

	cutoff, err := computeCutoff(hwm, adapter.Spec.Incremental.OverlapBuffer)
	if err != nil {
		return err
	}

	var newest *time.Time
	page := adapter.Spec.Search.Pagination.Start

	for pagesScanned := 0; pagesScanned < adapter.Spec.Incremental.SafetyMaxPages; pagesScanned++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("scrape cancelled: %w", err)
		}

		cards, err := s.fetcher.FetchPage(ctx, adapter.Spec, target, page)
		if err != nil {
			return fmt.Errorf("fetching search page %d: %w", page, err)
		}
		if len(cards) == 0 {
			break
		}

		reachedCutoff := false
		for _, card := range cards {
			newest = laterOf(newest, card.PostedAt)
			if cutoff != nil && card.PostedAt != nil && card.PostedAt.Before(*cutoff) {
				reachedCutoff = true
				break
			}
			if err := s.captureCard(ctx, adapter.BoardID, target.ProfileID, card); err != nil {
				return err
			}
		}
		if reachedCutoff {
			break
		}
		page++
	}

	if newest != nil {
		if err := s.hwm.Set(ctx, adapter.BoardID, target.ProfileID, *newest); err != nil {
			return fmt.Errorf("advancing high-water-mark: %w", err)
		}
	}
	return nil
}

// captureCard stores one card's raw payload (idempotent by content hash) and publishes
// a listing.extract message for genuinely new listings.
func (s *Service) captureCard(ctx context.Context, boardID kernel.BoardID, profileID kernel.ProfileID, card Card) error {
	contentHash := hashContent(card.Raw)

	seen, err := s.rawRepo.ExistsByContentHash(ctx, boardID, contentHash)
	if err != nil {
		return fmt.Errorf("checking content hash: %w", err)
	}
	if seen {
		return nil // overlap re-scan: skip, not duplicate
	}

	rawRef := fmt.Sprintf("raw/%s/%s.json", boardID, contentHash)
	if err := s.rawStore.Store(ctx, rawRef, card.Raw); err != nil {
		return fmt.Errorf("storing raw payload: %w", err)
	}

	postedAt := time.Time{}
	if card.PostedAt != nil {
		postedAt = *card.PostedAt
	}

	id, err := s.rawRepo.Save(ctx, scraping.RawListing{
		BoardID:          boardID,
		ProfileID:        profileID,
		Title:            card.Title,
		Company:          card.Company,
		Location:         card.Location,
		SourceURL:        card.ListingURL,
		RawRef:           rawRef,
		PostedAt:         postedAt,
		ContentHash:      contentHash,
		ExtractionStatus: scraping.ExtractionStatusPending,
	})
	if err != nil {
		return fmt.Errorf("saving raw listing: %w", err)
	}

	msg := messaging.Message{
		Data:       []byte(id),
		Attributes: map[string]string{ExtractRawListingIDAttr: string(id)},
	}
	if err := s.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("publishing listing.extract for %q: %w", id, err)
	}

	s.logger.InfoContext(ctx, "raw listing captured",
		"raw_listing_id", string(id), "board_id", string(boardID), "content_hash", contentHash)
	return nil
}

// computeCutoff returns the incremental stop boundary: hwm minus the overlap buffer.
// A nil hwm (first-ever crawl) yields a nil cutoff (crawl to the safety cap).
func computeCutoff(hwm *time.Time, overlapBuffer string) (*time.Time, error) {
	if hwm == nil {
		return nil, nil
	}
	buf, err := time.ParseDuration(overlapBuffer)
	if err != nil {
		return nil, fmt.Errorf("parsing overlap buffer %q: %w", overlapBuffer, err)
	}
	cutoff := hwm.Add(-buf)
	return &cutoff, nil
}

// laterOf returns the later of current and candidate, treating nil as "no value".
func laterOf(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.After(*current) {
		return candidate
	}
	return current
}

// hashContent returns the hex-encoded SHA-256 of the raw payload.
func hashContent(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
