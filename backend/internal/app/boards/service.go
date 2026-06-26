// Package boards contains the board-manager application service: listing boards
// together with their approved adapter. Repository interfaces are declared here
// (the consumer) and implemented in infra/boards.
package boards

import (
	"context"
	"fmt"

	"github.com/g-trinh/job-tendencies/internal/domain/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// BoardView is a board paired with its approved adapter, if one exists. Adapter is
// nil when the board has no approved adapter yet.
type BoardView struct {
	Board   boards.Board
	Adapter *boards.Adapter
}

// Repository reads boards and their adapters from the datastore.
type Repository interface {
	// ListBoards returns every board with its approved adapter (nil when none).
	ListBoards(ctx context.Context) ([]BoardView, error)
	// ApprovedAdapters returns the approved adapter for every enabled board.
	// Used by the scrape-worker to know what to crawl.
	ApprovedAdapters(ctx context.Context) ([]boards.Adapter, error)
	// BoardByID returns one board, or a kernel.NotFoundError.
	BoardByID(ctx context.Context, id kernel.BoardID) (boards.Board, error)
}

// Service exposes board-manager read use cases to the API and the scrape-worker.
type Service struct {
	repo Repository
}

// New constructs a board-manager Service.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListBoards returns all boards, each with its approved adapter when present.
func (s *Service) ListBoards(ctx context.Context) ([]BoardView, error) {
	views, err := s.repo.ListBoards(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing boards: %w", err)
	}
	return views, nil
}

// ApprovedAdapters returns the approved adapter for every enabled board.
func (s *Service) ApprovedAdapters(ctx context.Context) ([]boards.Adapter, error) {
	adapters, err := s.repo.ApprovedAdapters(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing approved adapters: %w", err)
	}
	return adapters, nil
}
