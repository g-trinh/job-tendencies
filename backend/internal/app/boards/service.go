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

// GetBoardAdapter returns the most recent adapter for the board (draft or approved).
func (s *Service) GetBoardAdapter(ctx context.Context, boardID kernel.BoardID) (boards.Adapter, error) {
	a, err := s.repo.GetAdapter(ctx, boardID)
	if err != nil {
		return boards.Adapter{}, fmt.Errorf("getting adapter for board %q: %w", boardID, err)
	}
	return a, nil
}

// ValidateAdapterSpec validates the AdapterSpec shape and returns field-level errors.
// It does not persist anything.
func (s *Service) ValidateAdapterSpec(spec boards.Adapter) error {
	if err := spec.Spec.Validate(); err != nil {
		return fmt.Errorf("validating adapter spec: %w", err)
	}
	return nil
}

// ApproveBoardAdapter validates the latest draft adapter for the board and, if valid,
// promotes it to 'approved', superseding any previously approved adapter.
func (s *Service) ApproveBoardAdapter(ctx context.Context, boardID kernel.BoardID) (boards.Adapter, error) {
	a, err := s.repo.GetAdapter(ctx, boardID)
	if err != nil {
		return boards.Adapter{}, fmt.Errorf("getting adapter for board %q: %w", boardID, err)
	}
	if a.Status == boards.AdapterStatusApproved {
		return boards.Adapter{}, &kernel.ValidationError{Field: "adapter", Message: "adapter is already approved"}
	}
	if err := a.Spec.Validate(); err != nil {
		return boards.Adapter{}, fmt.Errorf("validating adapter spec: %w", err)
	}
	if err := s.repo.ApproveAdapter(ctx, a.ID, boardID); err != nil {
		return boards.Adapter{}, fmt.Errorf("approving adapter for board %q: %w", boardID, err)
	}
	approved, err := s.repo.GetAdapter(ctx, boardID)
	if err != nil {
		return boards.Adapter{}, fmt.Errorf("reading approved adapter for board %q: %w", boardID, err)
	}
	return approved, nil
}

// GetSchedule returns the single global cron schedule.
func (s *Service) GetSchedule(ctx context.Context) (boards.Schedule, error) {
	sch, err := s.repo.GetSchedule(ctx)
	if err != nil {
		return boards.Schedule{}, fmt.Errorf("getting schedule: %w", err)
	}
	return sch, nil
}

// UpsertSchedule creates or replaces the global cron schedule. It validates that
// the cron expression is non-empty.
func (s *Service) UpsertSchedule(ctx context.Context, cron string) (boards.Schedule, error) {
	if cron == "" {
		return boards.Schedule{}, &kernel.ValidationError{Field: "cron", Message: "required"}
	}
	sch := boards.Schedule{Cron: cron}
	if err := s.repo.UpsertSchedule(ctx, sch); err != nil {
		return boards.Schedule{}, fmt.Errorf("upserting schedule: %w", err)
	}
	return sch, nil
}
