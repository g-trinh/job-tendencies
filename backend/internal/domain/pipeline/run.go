package pipeline

import (
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// ScrapeRun is one pipeline execution (scheduled cron or on-demand API trigger)
// recorded for observability. See data-model.md (scrape_run, scrape_run_board).
type ScrapeRun struct {
	// ID is the run's stable identifier.
	ID kernel.ScrapeRunID
	// ProfileID is the search profile this run was scoped to.
	ProfileID kernel.ProfileID
	// Trigger is "on_demand" or "scheduled".
	Trigger string
	// Status is the run lifecycle state: queued → running → done | error.
	Status string
	// CreatedAt is when the run was recorded.
	CreatedAt time.Time
	// FinishedAt is when the run reached its terminal state (nil if still running).
	FinishedAt *time.Time
	// Boards holds the per-board progress entries; populated only by GetRun.
	Boards []ScrapeRunBoard
}

// ScrapeRunBoard is the per-board progress entry within a scrape run. It records
// page and listing counts so the UI can show live crawl progress.
type ScrapeRunBoard struct {
	// ID is the entry's stable identifier.
	ID kernel.ScrapeRunBoardID
	// RunID back-references the parent scrape run.
	RunID kernel.ScrapeRunID
	// BoardID is the board that was crawled.
	BoardID kernel.BoardID
	// Status is running → done | error.
	Status string
	// PagesFetched is the number of search pages fetched during the crawl.
	PagesFetched int
	// ListingsCaptured is the number of genuinely new raw listings stored.
	ListingsCaptured int
	// Error is non-empty when the board crawl failed.
	Error string
	// StartedAt is when this board's crawl began.
	StartedAt *time.Time
	// FinishedAt is when this board's crawl ended (nil if still running).
	FinishedAt *time.Time
}
