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
