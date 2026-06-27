// Package boards contains the board-manager application service: listing boards
// together with their approved adapter, and CRUD write operations. The aggregate
// repository interface lives in the domain (domain/boards.Repository, ADR-005) and
// is implemented in infra/boards.
package boards

import (
	"context"
	"fmt"

	"github.com/g-trinh/job-tendencies/internal/domain/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Service exposes board-manager use cases to the API and the scrape-worker.
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

// CreateBoard validates and persists a new board. The board is enabled by default.
func (s *Service) CreateBoard(ctx context.Context, name, baseURL string) (boards.Board, error) {
	b, err := boards.NewBoard(name, baseURL)
	if err != nil {
		return boards.Board{}, fmt.Errorf("validating board: %w", err)
	}
	id, err := s.repo.CreateBoard(ctx, b)
	if err != nil {
		return boards.Board{}, fmt.Errorf("creating board: %w", err)
	}
	b.ID = id
	return b, nil
}

// UpdateBoard persists name, base_url, and enabled changes for the board.
func (s *Service) UpdateBoard(ctx context.Context, id kernel.BoardID, name, baseURL string, enabled bool) (boards.Board, error) {
	if name == "" {
		return boards.Board{}, &kernel.ValidationError{Field: "name", Message: "required"}
	}
	if baseURL == "" {
		return boards.Board{}, &kernel.ValidationError{Field: "base_url", Message: "required"}
	}
	b := boards.Board{ID: id, Name: name, BaseURL: baseURL, Enabled: enabled}
	if err := s.repo.UpdateBoard(ctx, b); err != nil {
		return boards.Board{}, fmt.Errorf("updating board %q: %w", id, err)
	}
	return b, nil
}

// DeleteBoard removes a board by id.
func (s *Service) DeleteBoard(ctx context.Context, id kernel.BoardID) error {
	if err := s.repo.DeleteBoard(ctx, id); err != nil {
		return fmt.Errorf("deleting board %q: %w", id, err)
	}
	return nil
}
