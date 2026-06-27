package boards

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// BoardView is a board paired with its approved adapter (nil when the board has no
// approved adapter yet). It is the board-manager read projection.
type BoardView struct {
	Board   Board
	Adapter *Adapter
}

// Schedule is the single global cron expression applied to Cloud Scheduler. There
// is exactly one row in the schedule table; the application layer enforces this via
// an upsert.
type Schedule struct {
	// Cron is the cron expression string (e.g. "0 2 * * *").
	Cron string
}

// Repository is the board-manager aggregate's persistence port. Aggregate repository
// interfaces live in the domain per ADR-005; the Postgres implementation lives in
// internal/infra/boards.
type Repository interface {
	// ListBoards returns every board with its approved adapter (nil when none).
	ListBoards(ctx context.Context) ([]BoardView, error)
	// ApprovedAdapters returns the approved adapter for every enabled board.
	ApprovedAdapters(ctx context.Context) ([]Adapter, error)
	// BoardByID returns one board, or a kernel.NotFoundError.
	BoardByID(ctx context.Context, id kernel.BoardID) (Board, error)
	// CreateBoard persists a new board and returns its assigned id.
	CreateBoard(ctx context.Context, b Board) (kernel.BoardID, error)
	// UpdateBoard persists name, base_url, and enabled changes for the board.
	UpdateBoard(ctx context.Context, b Board) error
	// DeleteBoard removes a board by id. It returns a kernel.NotFoundError when
	// the board does not exist.
	DeleteBoard(ctx context.Context, id kernel.BoardID) error

	// GetAdapter returns the most recent adapter for the board (draft preferred over
	// approved by insertion order), or a kernel.NotFoundError when none exists.
	GetAdapter(ctx context.Context, boardID kernel.BoardID) (Adapter, error)
	// SaveDraftAdapter inserts a new draft adapter for the board. The version is
	// set to (current max version + 1) by the implementation. Returns the new adapter id.
	SaveDraftAdapter(ctx context.Context, a Adapter) (kernel.AdapterID, error)
	// ApproveAdapter sets the given adapter's status to 'approved' in a single
	// transaction. Returns a kernel.NotFoundError when the adapter does not exist.
	ApproveAdapter(ctx context.Context, adapterID kernel.AdapterID, boardID kernel.BoardID) error

	// GetSchedule returns the single global cron schedule, or a kernel.NotFoundError
	// when no schedule has been configured yet.
	GetSchedule(ctx context.Context) (Schedule, error)
	// UpsertSchedule creates or replaces the global cron schedule.
	UpsertSchedule(ctx context.Context, s Schedule) error
}
