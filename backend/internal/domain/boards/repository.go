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
}
