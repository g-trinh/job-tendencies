// Package scraping contains the scrape-worker application service: the two-phase crawl
// loop (json_api and html modes) with high-water-mark incrementality, raw→GCS capture
// with content-hash dedup, per-board run tracking, and per-new-listing listing.extract
// fan-out. Port interfaces are declared here (the consumer) and implemented in
// infra/scraping, infra/blobstore, infra/messaging, and infra/pipeline.
//
// See docs/architecture/pipeline.md §2 for the crawl algorithm.
package scraping

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// TriggerAttr is the listing.extract message attribute carrying the run trigger
// propagated from scrape.tick ("scheduled" | "on_demand"). extract-worker reads it to
// gate Batch API routing (P5-5, config-gated, default off — see tech_debt.md open
// question #1).
const TriggerAttr = "trigger"

// TriggerScheduled marks a run started by Cloud Scheduler's global cron.
const TriggerScheduled = "scheduled"

// scrapeTickRunAttr is the scrape.tick message attribute carrying the run id.
// Matches app/pipeline.ScrapeTickRunAttr; defined here to avoid cross-context import.
const scrapeTickRunAttr = "run_id"

// scrapeTickTriggerAttr is the scrape.tick message attribute carrying the run trigger.
// Matches app/pipeline.ScrapeTickTriggerAttr; defined here to avoid cross-context import
// (same rationale as scrapeTickRunAttr above).
const scrapeTickTriggerAttr = "trigger"

// triggerOnDemand is the safe default when a scrape.tick message carries no trigger at
// all (keeps every downstream listing.extract on the synchronous extraction path).
const triggerOnDemand = "on_demand"

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

// RunTracker records live crawl progress for observability (data-model.md scrape_run,
// scrape_run_board). When the scrape.tick message carries a run_id the tracker is
// wired; on first-ever invocations with no run_id it creates a scheduled run.
// Implemented by infra/pipeline.Repository; nil → no-op via noOpRunTracker.
type RunTracker interface {
	// MarkRunning transitions an existing run to running, or creates a new scheduled run.
	// Returns the run id to use for the rest of the crawl.
	MarkRunning(ctx context.Context, profileID kernel.ProfileID, runID kernel.ScrapeRunID) (kernel.ScrapeRunID, error)
	// TrackBoard opens a per-board row for the run and returns its id.
	TrackBoard(ctx context.Context, runID kernel.ScrapeRunID, boardID kernel.BoardID) (kernel.ScrapeRunBoardID, error)
	// FinishBoard records the board's final counts and optional error (empty = success).
	FinishBoard(ctx context.Context, id kernel.ScrapeRunBoardID, pagesF, listingsC int, errMsg string) error
	// FinishRun marks the run done or error.
	FinishRun(ctx context.Context, id kernel.ScrapeRunID, status string) error
}

// The RawListing capture port and the high-water-mark port are the scraping
// aggregate's repositories and live in domain/scraping (ADR-005), consumed here.

// JobExpirer marks jobs no longer present in a board's incremental re-scan window as
// expired, and reactivates any job that reappears (P5-3, pipeline.md §5 "Expiry").
// Implemented by infra/jobs.Repository; nil -> no-op via noOpExpirer, matching the
// RunTracker pattern.
type JobExpirer interface {
	// MarkExpired sets expired_at=now for jobs sourced from boardID/profileID whose
	// source_url is not in activeSourceURLs, and clears expired_at for any job whose
	// source_url is. activeSourceURLs is the set of listing URLs actually re-scanned
	// this run (bounded by the incremental cutoff), so listings outside the re-scan
	// window are never mistakenly marked expired.
	MarkExpired(ctx context.Context, boardID kernel.BoardID, profileID kernel.ProfileID, activeSourceURLs []string, now time.Time) error
}

// Service runs the scrape pipeline stage in response to scrape.tick deliveries.
type Service struct {
	adapters  AdapterSource
	targets   TargetSource
	fetcher   SearchFetcher
	rawStore  blobstore.Storer
	rawRepo   scraping.RawListingRepository
	hwm       scraping.HighWaterMarkRepository
	publisher messaging.Publisher
	tracker   RunTracker
	expirer   JobExpirer
	logger    *slog.Logger
}

// New constructs a scraping Service with all pipeline dependencies wired.
// Pass nil for tracker and/or expirer to disable run tracking / expiry marking
// (no-ops are used instead).
func New(
	adapters AdapterSource,
	targets TargetSource,
	fetcher SearchFetcher,
	rawStore blobstore.Storer,
	rawRepo scraping.RawListingRepository,
	hwm scraping.HighWaterMarkRepository,
	publisher messaging.Publisher,
	tracker RunTracker,
	expirer JobExpirer,
	logger *slog.Logger,
) *Service {
	if tracker == nil {
		tracker = noOpRunTracker{}
	}
	if expirer == nil {
		expirer = noOpExpirer{}
	}
	return &Service{
		adapters:  adapters,
		targets:   targets,
		fetcher:   fetcher,
		rawStore:  rawStore,
		rawRepo:   rawRepo,
		hwm:       hwm,
		publisher: publisher,
		tracker:   tracker,
		expirer:   expirer,
		logger:    logger,
	}
}

// HandleScrapeTick is invoked for each verified scrape.tick push delivery. It resolves
// the active target, then crawls every enabled board's approved adapter, tracking
// per-board progress when the message carries a run_id attribute.
func (s *Service) HandleScrapeTick(ctx context.Context, msg messaging.Message) error {
	trigger := resolveTrigger(msg)

	target, err := s.targets.ActiveTarget(ctx)
	if err != nil {
		return fmt.Errorf("resolving active target: %w", err)
	}

	adapters, err := s.adapters.ApprovedBoardAdapters(ctx)
	if err != nil {
		return fmt.Errorf("loading approved adapters: %w", err)
	}

	// Use run_id from the message when present (on-demand); otherwise the tracker
	// creates a new scheduled run.
	existingRunID := kernel.ScrapeRunID(msg.Attributes[scrapeTickRunAttr])
	runID, err := s.tracker.MarkRunning(ctx, target.ProfileID, existingRunID)
	if err != nil {
		s.logger.WarnContext(ctx, "run tracking: mark running failed, continuing without tracking",
			"run_id", string(existingRunID), "err", err)
		runID = existingRunID
	}

	var runErr error
	for _, adapter := range adapters {
		boardRunID, trackErr := s.tracker.TrackBoard(ctx, runID, adapter.BoardID)
		if trackErr != nil {
			s.logger.WarnContext(ctx, "run tracking: track board failed",
				"board_id", string(adapter.BoardID), "err", trackErr)
		}

		pages, listings, crawlErr := s.crawlBoard(ctx, adapter, target, trigger)

		errMsg := ""
		if crawlErr != nil {
			runErr = fmt.Errorf("crawling board %q: %w", adapter.BoardID, crawlErr)
			errMsg = crawlErr.Error()
		}
		if finErr := s.tracker.FinishBoard(ctx, boardRunID, pages, listings, errMsg); finErr != nil {
			s.logger.WarnContext(ctx, "run tracking: finish board failed",
				"board_id", string(adapter.BoardID), "err", finErr)
		}
		if runErr != nil {
			break
		}
	}

	runStatus := "done"
	if runErr != nil {
		runStatus = "error"
	}
	if finErr := s.tracker.FinishRun(ctx, runID, runStatus); finErr != nil {
		s.logger.WarnContext(ctx, "run tracking: finish run failed",
			"run_id", string(runID), "err", finErr)
	}
	return runErr
}

// crawlBoard runs the high-water-mark incremental crawl for one board (pipeline.md §2).
// It returns the number of pages scanned and the number of genuinely new listings captured.
// trigger is propagated onto every listing.extract message this crawl publishes (P5-5).
func (s *Service) crawlBoard(ctx context.Context, adapter BoardAdapter, target ScrapeTarget, trigger string) (pagesScanned, listingsCaptured int, err error) {
	hwm, err := s.hwm.Get(ctx, adapter.BoardID, target.ProfileID)
	if err != nil {
		return 0, 0, fmt.Errorf("reading high-water-mark: %w", err)
	}

	cutoff, err := computeCutoff(hwm, adapter.Spec.Incremental.OverlapBuffer)
	if err != nil {
		return 0, 0, err
	}

	var newest *time.Time
	page := adapter.Spec.Search.Pagination.Start
	// seenURLs collects every listing actually re-scanned this run (i.e. within the
	// incremental cutoff), the baseline P5-3 expiry marking compares against.
	seenURLs := make([]string, 0, 16)

	for pagesScanned = 0; pagesScanned < adapter.Spec.Incremental.SafetyMaxPages; pagesScanned++ {
		if err := ctx.Err(); err != nil {
			return pagesScanned, listingsCaptured, fmt.Errorf("scrape cancelled: %w", err)
		}

		cards, err := s.fetcher.FetchPage(ctx, adapter.Spec, target, page)
		if err != nil {
			return pagesScanned, listingsCaptured, fmt.Errorf("fetching search page %d: %w", page, err)
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
			seenURLs = append(seenURLs, card.ListingURL)
			captured, err := s.captureCard(ctx, adapter.BoardID, target.ProfileID, card, trigger)
			if err != nil {
				return pagesScanned, listingsCaptured, err
			}
			if captured {
				listingsCaptured++
			}
		}
		if reachedCutoff {
			break
		}
		page++
	}

	if newest != nil {
		if err := s.hwm.Set(ctx, adapter.BoardID, target.ProfileID, *newest); err != nil {
			return pagesScanned, listingsCaptured, fmt.Errorf("advancing high-water-mark: %w", err)
		}
	}

	// P5-3: only mark expiry on incremental runs. A nil cutoff means this is the
	// board's first-ever crawl (no prior baseline), so nothing has gone missing yet.
	if cutoff != nil {
		if err := s.expirer.MarkExpired(ctx, adapter.BoardID, target.ProfileID, seenURLs, time.Now().UTC()); err != nil {
			return pagesScanned, listingsCaptured, fmt.Errorf("marking expired jobs for board %q: %w", adapter.BoardID, err)
		}
	}
	return pagesScanned, listingsCaptured, nil
}

// captureCard stores one card's raw payload (idempotent by content hash) and publishes
// a listing.extract message for genuinely new listings, carrying trigger onward so
// extract-worker can gate Batch API routing (P5-5). It returns true when the card was
// captured (not already seen by content_hash).
func (s *Service) captureCard(ctx context.Context, boardID kernel.BoardID, profileID kernel.ProfileID, card Card, trigger string) (bool, error) {
	contentHash := hashContent(card.Raw)

	// The exists-check and Save below are not a single transaction, so two concurrent
	// crawls could both pass the check and try to Save. The scrape-worker runs at Cloud
	// Run concurrency=1, max-instances=1 (ADR-003), so that race cannot occur today; the
	// (board_id, profile_id, content_hash) unique index (migration 00007) is the backstop
	// that makes a racing Save fail rather than insert a duplicate.
	seen, err := s.rawRepo.ExistsByContentHash(ctx, boardID, profileID, contentHash)
	if err != nil {
		return false, fmt.Errorf("checking content hash: %w", err)
	}
	if seen {
		return false, nil // overlap re-scan: skip, not duplicate
	}

	rawRef := fmt.Sprintf("raw/%s/%s.json", boardID, contentHash)
	if err := s.rawStore.Store(ctx, rawRef, card.Raw); err != nil {
		return false, fmt.Errorf("storing raw payload: %w", err)
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
		return false, fmt.Errorf("saving raw listing: %w", err)
	}

	msg := messaging.Message{
		Data: []byte(id),
		Attributes: map[string]string{
			ExtractRawListingIDAttr: string(id),
			TriggerAttr:             trigger,
		},
	}
	if err := s.publisher.Publish(ctx, msg); err != nil {
		return false, fmt.Errorf("publishing listing.extract for %q: %w", id, err)
	}

	s.logger.InfoContext(ctx, "raw listing captured",
		"raw_listing_id", string(id), "board_id", string(boardID), "content_hash", contentHash)
	return true, nil
}

// resolveTrigger reads the run trigger off a scrape.tick message: the attribute when
// present (on-demand runs, set by app/pipeline.Service.CreateRun), else the Cloud
// Scheduler payload's {"trigger":"scheduled"} JSON body (infrastructure.md §5 scheduler
// module), else triggerOnDemand as the safe default so an unrecognised message never
// accidentally routes toward the (currently unimplemented) batch path.
func resolveTrigger(msg messaging.Message) string {
	if t := msg.Attributes[scrapeTickTriggerAttr]; t != "" {
		return t
	}
	var payload struct {
		Trigger string `json:"trigger"`
	}
	if len(msg.Data) > 0 {
		if err := json.Unmarshal(msg.Data, &payload); err == nil && payload.Trigger != "" {
			return payload.Trigger
		}
	}
	return triggerOnDemand
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

// noOpRunTracker is used when no run tracking is configured (nil passed to New).
type noOpRunTracker struct{}

func (noOpRunTracker) MarkRunning(_ context.Context, _ kernel.ProfileID, id kernel.ScrapeRunID) (kernel.ScrapeRunID, error) {
	return id, nil
}
func (noOpRunTracker) TrackBoard(_ context.Context, _ kernel.ScrapeRunID, _ kernel.BoardID) (kernel.ScrapeRunBoardID, error) {
	return "", nil
}
func (noOpRunTracker) FinishBoard(_ context.Context, _ kernel.ScrapeRunBoardID, _, _ int, _ string) error {
	return nil
}
func (noOpRunTracker) FinishRun(_ context.Context, _ kernel.ScrapeRunID, _ string) error {
	return nil
}

// noOpExpirer is used when no job expirer is configured (nil passed to New).
type noOpExpirer struct{}

func (noOpExpirer) MarkExpired(context.Context, kernel.BoardID, kernel.ProfileID, []string, time.Time) error {
	return nil
}
