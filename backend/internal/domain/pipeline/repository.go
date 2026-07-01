// Package pipeline is the pipeline bounded context. It owns scrape-run tracking: each
// crawl invocation (scheduled cron or on-demand API trigger) is recorded as a run so
// it can be polled. See data-model.md (scrape_run, scrape_run_board).
package pipeline

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// RunRepository is the read/write persistence port for scrape run lifecycle and history.
// Implementations live in infra/pipeline.
type RunRepository interface {
	// CreateRun inserts a run for the profile with the given trigger and returns its id.
	CreateRun(ctx context.Context, profileID kernel.ProfileID, trigger string) (kernel.ScrapeRunID, error)
	// StartRun transitions a run from queued to running.
	StartRun(ctx context.Context, id kernel.ScrapeRunID) error
	// FinishRun records the run's terminal status (done or error) and completion time.
	FinishRun(ctx context.Context, id kernel.ScrapeRunID, status string) error
	// CreateRunBoard opens a per-board entry for the run (status: running) and returns its id.
	CreateRunBoard(ctx context.Context, runID kernel.ScrapeRunID, boardID kernel.BoardID) (kernel.ScrapeRunBoardID, error)
	// FinishRunBoard records the board's final counts and optional error message.
	// An empty errMsg indicates success.
	FinishRunBoard(ctx context.Context, id kernel.ScrapeRunBoardID, pagesF, listingsC int, errMsg string) error
	// ListRuns returns recent scrape runs ordered by creation time (newest first).
	ListRuns(ctx context.Context) ([]ScrapeRun, error)
	// GetRun returns a run with its per-board breakdown, or kernel.NotFoundError.
	GetRun(ctx context.Context, id kernel.ScrapeRunID) (ScrapeRun, error)
}
