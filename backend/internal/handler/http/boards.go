package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// BoardService exposes board-manager read and write use cases. Implemented by
// app/boards.Service.
type BoardService interface {
	ListBoards(ctx context.Context) ([]boards.BoardView, error)
	CreateBoard(ctx context.Context, name, baseURL string) (boards.Board, error)
	UpdateBoard(ctx context.Context, id kernel.BoardID, name, baseURL string, enabled bool) (boards.Board, error)
	DeleteBoard(ctx context.Context, id kernel.BoardID) error
	GetBoardAdapter(ctx context.Context, boardID kernel.BoardID) (boards.Adapter, error)
	GenerateAdapter(ctx context.Context, boardID kernel.BoardID, exampleResponse string) (boards.Adapter, error)
	ApproveBoardAdapter(ctx context.Context, boardID kernel.BoardID) (boards.Adapter, error)
}

// ScheduleService manages the single global cron schedule. Implemented by
// app/boards.Service.
type ScheduleService interface {
	GetSchedule(ctx context.Context) (boards.Schedule, error)
	UpsertSchedule(ctx context.Context, cron string) (boards.Schedule, error)
}

// scheduleResponse is the JSON shape of the global schedule.
type scheduleResponse struct {
	Cron string `json:"cron"`
}

// BoardLister lists boards with their approved adapter. Kept for backwards
// compatibility with callers that only need the read path.
type BoardLister interface {
	ListBoards(ctx context.Context) ([]boards.BoardView, error)
}

// boardResponse is the JSON shape of a board returned by the boards endpoints.
type boardResponse struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	BaseURL string           `json:"base_url"`
	Enabled bool             `json:"enabled"`
	Adapter *adapterResponse `json:"adapter"`
}

// adapterResponse is the JSON shape of a board's approved adapter.
type adapterResponse struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	FetchMode string          `json:"fetch_mode"`
	Version   int             `json:"version"`
	Spec      llm.AdapterSpec `json:"spec"`
}

// ListBoards handles GET /api/boards. It returns every board with its approved
// adapter (null when the board has no approved adapter).
func ListBoards(lister BoardLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		views, err := lister.ListBoards(r.Context())
		if err != nil {
			RespondError(w, r, err)
			return
		}

		out := make([]boardResponse, 0, len(views))
		for _, v := range views {
			out = append(out, toBoardResponse(v))
		}
		respond(w, http.StatusOK, out)
	}
}

// boardWriteRequest is the shared request body for POST and PUT /api/boards.
type boardWriteRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Enabled *bool  `json:"enabled"`
}

// PostBoard handles POST /api/boards, creating a new board (enabled by default).
func PostBoard(svc BoardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body boardWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		b, err := svc.CreateBoard(r.Context(), body.Name, body.BaseURL)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusCreated, boardResponse{
			ID:      string(b.ID),
			Name:    b.Name,
			BaseURL: b.BaseURL,
			Enabled: b.Enabled,
		})
	}
}

// PutBoard handles PUT /api/boards/{id}, updating name, base_url, and enabled.
func PutBoard(svc BoardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.BoardID(chi.URLParam(r, "id"))
		var body boardWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		enabled := true
		if body.Enabled != nil {
			enabled = *body.Enabled
		}
		b, err := svc.UpdateBoard(r.Context(), id, body.Name, body.BaseURL, enabled)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, boardResponse{
			ID:      string(b.ID),
			Name:    b.Name,
			BaseURL: b.BaseURL,
			Enabled: b.Enabled,
		})
	}
}

// DeleteBoard handles DELETE /api/boards/{id}.
func DeleteBoard(svc BoardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.BoardID(chi.URLParam(r, "id"))
		if err := svc.DeleteBoard(r.Context(), id); err != nil {
			RespondError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func toBoardResponse(v boards.BoardView) boardResponse {
	resp := boardResponse{
		ID:      string(v.Board.ID),
		Name:    v.Board.Name,
		BaseURL: v.Board.BaseURL,
		Enabled: v.Board.Enabled,
	}
	if v.Adapter != nil {
		resp.Adapter = toAdapterResponse(*v.Adapter)
	}
	return resp
}

func toAdapterResponse(a boards.Adapter) *adapterResponse {
	return &adapterResponse{
		ID:        string(a.ID),
		Status:    string(a.Status),
		FetchMode: string(a.FetchMode),
		Version:   a.Version,
		Spec:      a.Spec,
	}
}

// GetBoardAdapter handles GET /api/boards/{id}/adapter. Returns the most recent
// adapter for the board (draft or approved).
func GetBoardAdapter(svc BoardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.BoardID(chi.URLParam(r, "id"))
		a, err := svc.GetBoardAdapter(r.Context(), id)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toAdapterResponse(a))
	}
}

// generateAdapterRequest is the JSON body for POST /api/boards/{id}/adapter/generate.
type generateAdapterRequest struct {
	// ExampleResponse is the raw HTML or JSON page captured from the board's search
	// or listing URL. The LLM uses it to infer selectors, URL templates, and pagination.
	ExampleResponse string `json:"example_response"`
}

// PostGenerateAdapter handles POST /api/boards/{id}/adapter/generate. It calls the LLM
// to produce a declarative AdapterSpec draft from the supplied example page. The draft
// is persisted and returned; it must be reviewed and approved before the scraper uses it.
func PostGenerateAdapter(svc BoardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.BoardID(chi.URLParam(r, "id"))
		var body generateAdapterRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		a, err := svc.GenerateAdapter(r.Context(), id, body.ExampleResponse)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusCreated, toAdapterResponse(a))
	}
}

// PostApproveAdapter handles POST /api/boards/{id}/adapter/approve. Validates the
// latest draft adapter and promotes it to approved, superseding the prior approved
// adapter. Returns 400 with field-level errors when the spec is invalid.
func PostApproveAdapter(svc BoardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.BoardID(chi.URLParam(r, "id"))
		a, err := svc.ApproveBoardAdapter(r.Context(), id)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toAdapterResponse(a))
	}
}

// GetSchedule handles GET /api/schedule, returning the global cron schedule.
func GetSchedule(svc ScheduleService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sch, err := svc.GetSchedule(r.Context())
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, scheduleResponse{Cron: sch.Cron})
	}
}

// PutSchedule handles PUT /api/schedule. Body: {"cron":"<expression>"}.
// Persists the global cron expression.
func PutSchedule(svc ScheduleService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Cron string `json:"cron"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		sch, err := svc.UpsertSchedule(r.Context(), body.Cron)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, scheduleResponse{Cron: sch.Cron})
	}
}
