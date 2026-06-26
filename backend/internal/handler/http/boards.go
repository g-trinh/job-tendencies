package handler

import (
	"context"
	"net/http"

	appboards "github.com/g-trinh/job-tendencies/internal/app/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// BoardLister lists boards with their approved adapter. Implemented by app/boards.Service.
type BoardLister interface {
	ListBoards(ctx context.Context) ([]appboards.BoardView, error)
}

// boardResponse is the JSON shape of a board returned by GET /api/boards.
type boardResponse struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	BaseURL string           `json:"base_url"`
	Enabled bool             `json:"enabled"`
	Adapter *adapterResponse `json:"adapter"`
}

// adapterResponse is the JSON shape of a board's approved adapter.
type adapterResponse struct {
	ID        string         `json:"id"`
	Status    string         `json:"status"`
	FetchMode string         `json:"fetch_mode"`
	Version   int            `json:"version"`
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

func toBoardResponse(v appboards.BoardView) boardResponse {
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
