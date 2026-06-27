// Package boards contains the board-manager application service: listing boards
// together with their approved adapter. The aggregate repository interface lives in
// the domain (domain/boards.Repository, ADR-005) and is implemented in infra/boards.
package boards

import (
	"context"
	"fmt"

	"github.com/g-trinh/job-tendencies/internal/domain/boards"
)

// Service exposes board-manager read use cases to the API and the scrape-worker.
type Service struct {
	repo boards.Repository
}

// New constructs a board-manager Service.
func New(repo boards.Repository) *Service {
	return &Service{repo: repo}
}

// ListBoards returns all boards, each with its approved adapter when present.
func (s *Service) ListBoards(ctx context.Context) ([]boards.BoardView, error) {
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
