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
	domainllm "github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// Service exposes board-manager use cases to the API and the scrape-worker.
type Service struct {
	repo      boards.Repository
	generator domainllm.AdapterGenerator
}

// New constructs a board-manager Service. generator is the LLM adapter generator used by
// GenerateAdapter; it may be nil when that endpoint is not wired (e.g. in tests that only
// exercise CRUD paths), in which case GenerateAdapter returns an error.
func New(repo boards.Repository, generator domainllm.AdapterGenerator) *Service {
	return &Service{repo: repo, generator: generator}
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

// GenerateAdapter calls the LLM to generate a declarative AdapterSpec draft for the
// given board, using exampleResponse as the page sample the model analyses. The draft
// is persisted with status=draft and must be reviewed and approved via
// ApproveBoardAdapter before the scraper can evaluate it.
//
// exampleResponse is the raw HTML or JSON captured from the board's search or listing
// URL — the content the LLM uses to infer selectors, URL templates, and pagination.
func (s *Service) GenerateAdapter(ctx context.Context, boardID kernel.BoardID, exampleResponse string) (boards.Adapter, error) {
	if exampleResponse == "" {
		return boards.Adapter{}, &kernel.ValidationError{Field: "example_response", Message: "required"}
	}
	if s.generator == nil {
		return boards.Adapter{}, fmt.Errorf("adapter generator not configured")
	}

	b, err := s.repo.BoardByID(ctx, boardID)
	if err != nil {
		return boards.Adapter{}, fmt.Errorf("getting board %q: %w", boardID, err)
	}

	spec, err := s.generator.GenerateAdapter(ctx, b.BaseURL, exampleResponse)
	if err != nil {
		return boards.Adapter{}, fmt.Errorf("generating adapter for board %q: %w", boardID, err)
	}

	draft := boards.Adapter{
		BoardID:   boardID,
		Status:    boards.AdapterStatusDraft,
		FetchMode: spec.FetchMode,
		Spec:      *spec,
	}
	id, err := s.repo.SaveDraftAdapter(ctx, draft)
	if err != nil {
		return boards.Adapter{}, fmt.Errorf("saving draft adapter for board %q: %w", boardID, err)
	}
	draft.ID = id
	return draft, nil
}
