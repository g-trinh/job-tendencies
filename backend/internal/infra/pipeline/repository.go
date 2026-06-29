// Package pipeline provides the Postgres implementation of the pipeline run repository.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/pipeline"
)

// Repository records scrape runs in Postgres. It satisfies domain/pipeline.RunRepository
// and also app/scraping.RunTracker (the per-board progress port used during a live crawl).
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Postgres pipeline run repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateRun inserts a scrape run and returns its generated id.
func (r *Repository) CreateRun(ctx context.Context, profileID kernel.ProfileID, trigger string) (kernel.ScrapeRunID, error) {
	const query = `
		INSERT INTO scrape_run (profile_id, trigger, status)
		VALUES ($1, $2, 'queued')
		RETURNING id`
	var id string
	if err := r.pool.QueryRow(ctx, query, string(profileID), trigger).Scan(&id); err != nil {
		return "", fmt.Errorf("inserting scrape run: %w", err)
	}
	return kernel.ScrapeRunID(id), nil
}

// StartRun transitions a run to the running state.
func (r *Repository) StartRun(ctx context.Context, id kernel.ScrapeRunID) error {
	const query = `UPDATE scrape_run SET status='running' WHERE id=$1`
	if _, err := r.pool.Exec(ctx, query, string(id)); err != nil {
		return fmt.Errorf("starting scrape run %q: %w", id, err)
	}
	return nil
}

// FinishRun records the run's terminal status (done or error) and completion time.
func (r *Repository) FinishRun(ctx context.Context, id kernel.ScrapeRunID, status string) error {
	if id == "" {
		return nil // no-op: run was not tracked
	}
	const query = `UPDATE scrape_run SET status=$1, finished_at=now() WHERE id=$2`
	if _, err := r.pool.Exec(ctx, query, status, string(id)); err != nil {
		return fmt.Errorf("finishing scrape run %q: %w", id, err)
	}
	return nil
}

// CreateRunBoard opens a per-board entry for the run, setting its status to running.
func (r *Repository) CreateRunBoard(ctx context.Context, runID kernel.ScrapeRunID, boardID kernel.BoardID) (kernel.ScrapeRunBoardID, error) {
	if runID == "" {
		return "", nil // no-op: run was not tracked
	}
	const query = `
		INSERT INTO scrape_run_board (run_id, board_id, status, started_at)
		VALUES ($1, $2, 'running', now())
		RETURNING id`
	var id string
	if err := r.pool.QueryRow(ctx, query, string(runID), string(boardID)).Scan(&id); err != nil {
		return "", fmt.Errorf("creating scrape run board %q/%q: %w", runID, boardID, err)
	}
	return kernel.ScrapeRunBoardID(id), nil
}

// FinishRunBoard records the board's final counts and optional error message.
// An empty errMsg indicates success (status → done); a non-empty one sets status → error.
func (r *Repository) FinishRunBoard(ctx context.Context, id kernel.ScrapeRunBoardID, pagesF, listingsC int, errMsg string) error {
	if id == "" {
		return nil // no-op: board was not tracked
	}
	status := "done"
	if errMsg != "" {
		status = "error"
	}
	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}
	const query = `
		UPDATE scrape_run_board
		SET status=$1, pages_fetched=$2, listings_captured=$3, error=$4, finished_at=now()
		WHERE id=$5`
	if _, err := r.pool.Exec(ctx, query, status, pagesF, listingsC, errPtr, string(id)); err != nil {
		return fmt.Errorf("finishing scrape run board %q: %w", id, err)
	}
	return nil
}

// MarkRunning satisfies app/scraping.RunTracker: if existingRunID is empty it creates a
// new scheduled run; otherwise it transitions the existing run to running. Returns the
// run id to use for the rest of the crawl.
func (r *Repository) MarkRunning(ctx context.Context, profileID kernel.ProfileID, existingRunID kernel.ScrapeRunID) (kernel.ScrapeRunID, error) {
	if existingRunID == "" {
		id, err := r.CreateRun(ctx, profileID, "scheduled")
		if err != nil {
			return "", fmt.Errorf("creating scheduled run: %w", err)
		}
		existingRunID = id
	}
	if err := r.StartRun(ctx, existingRunID); err != nil {
		return "", fmt.Errorf("marking run running: %w", err)
	}
	return existingRunID, nil
}

// TrackBoard satisfies app/scraping.RunTracker: creates a per-board entry and returns
// its id. A no-op (empty runID) returns an empty ScrapeRunBoardID without error.
func (r *Repository) TrackBoard(ctx context.Context, runID kernel.ScrapeRunID, boardID kernel.BoardID) (kernel.ScrapeRunBoardID, error) {
	return r.CreateRunBoard(ctx, runID, boardID)
}

// FinishBoard satisfies app/scraping.RunTracker by delegating to FinishRunBoard.
func (r *Repository) FinishBoard(ctx context.Context, id kernel.ScrapeRunBoardID, pagesF, listingsC int, errMsg string) error {
	return r.FinishRunBoard(ctx, id, pagesF, listingsC, errMsg)
}

// ListRuns returns recent scrape runs ordered by creation time, newest first.
func (r *Repository) ListRuns(ctx context.Context) ([]pipeline.ScrapeRun, error) {
	const query = `
		SELECT id, profile_id, trigger, status, created_at, finished_at
		FROM scrape_run
		ORDER BY created_at DESC
		LIMIT 50`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing scrape runs: %w", err)
	}
	defer rows.Close()

	var runs []pipeline.ScrapeRun
	for rows.Next() {
		var run pipeline.ScrapeRun
		var id, profileID, trigger, status string
		if err := rows.Scan(&id, &profileID, &trigger, &status,
			&run.CreatedAt, &run.FinishedAt); err != nil {
			return nil, fmt.Errorf("scanning scrape run: %w", err)
		}
		run.ID = kernel.ScrapeRunID(id)
		run.ProfileID = kernel.ProfileID(profileID)
		run.Trigger = trigger
		run.Status = status
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating scrape runs: %w", err)
	}
	return runs, nil
}

// GetRun returns a run with its per-board breakdown, or kernel.NotFoundError.
func (r *Repository) GetRun(ctx context.Context, id kernel.ScrapeRunID) (pipeline.ScrapeRun, error) {
	const runQuery = `
		SELECT id, profile_id, trigger, status, created_at, finished_at
		FROM scrape_run WHERE id=$1`
	var run pipeline.ScrapeRun
	var rawID, profileID, trigger, status string
	err := r.pool.QueryRow(ctx, runQuery, string(id)).
		Scan(&rawID, &profileID, &trigger, &status, &run.CreatedAt, &run.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return pipeline.ScrapeRun{}, &kernel.NotFoundError{Kind: "scrape_run", ID: string(id)}
	}
	if err != nil {
		return pipeline.ScrapeRun{}, fmt.Errorf("querying scrape run %q: %w", id, err)
	}
	run.ID = kernel.ScrapeRunID(rawID)
	run.ProfileID = kernel.ProfileID(profileID)
	run.Trigger = trigger
	run.Status = status

	const boardsQuery = `
		SELECT id, run_id, board_id, status, pages_fetched, listings_captured,
		       COALESCE(error, ''), started_at, finished_at
		FROM scrape_run_board WHERE run_id=$1
		ORDER BY started_at`
	rows, err := r.pool.Query(ctx, boardsQuery, string(id))
	if err != nil {
		return pipeline.ScrapeRun{}, fmt.Errorf("querying run boards for %q: %w", id, err)
	}
	defer rows.Close()

	for rows.Next() {
		var b pipeline.ScrapeRunBoard
		var bid, bRunID, bBoardID, bStatus, bErr string
		var startedAt time.Time
		if err := rows.Scan(&bid, &bRunID, &bBoardID, &bStatus,
			&b.PagesFetched, &b.ListingsCaptured, &bErr, &startedAt, &b.FinishedAt); err != nil {
			return pipeline.ScrapeRun{}, fmt.Errorf("scanning run board: %w", err)
		}
		b.ID = kernel.ScrapeRunBoardID(bid)
		b.RunID = kernel.ScrapeRunID(bRunID)
		b.BoardID = kernel.BoardID(bBoardID)
		b.Status = bStatus
		b.Error = bErr
		b.StartedAt = &startedAt
		run.Boards = append(run.Boards, b)
	}
	if err := rows.Err(); err != nil {
		return pipeline.ScrapeRun{}, fmt.Errorf("iterating run boards for %q: %w", id, err)
	}
	return run, nil
}

var _ pipeline.RunRepository = (*Repository)(nil)
