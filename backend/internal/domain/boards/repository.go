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

// Repository is the board-manager aggregate's persistence port. The board-manager is
// read-only in Phase 2; the scrape-worker consumes ApprovedAdapters to know what to crawl.
type Repository interface {
	// ListBoards returns every board with its approved adapter (nil when none).
	ListBoards(ctx context.Context) ([]BoardView, error)
	// ApprovedAdapters returns the approved adapter for every enabled board.
	ApprovedAdapters(ctx context.Context) ([]Adapter, error)
	// BoardByID returns one board, or a kernel.NotFoundError.
	BoardByID(ctx context.Context, id kernel.BoardID) (Board, error)
}
