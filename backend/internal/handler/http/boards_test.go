package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// fakeBoardService is an in-memory fake of handler.BoardService.
type fakeBoardService struct {
	views []boards.BoardView
	err   error
}

func (f *fakeBoardService) ListBoards(_ context.Context) ([]boards.BoardView, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.views, nil
}

func (f *fakeBoardService) CreateBoard(_ context.Context, name, baseURL string) (boards.Board, error) {
	if f.err != nil {
		return boards.Board{}, f.err
	}
	b, err := boards.NewBoard(name, baseURL)
	if err != nil {
		return boards.Board{}, fmt.Errorf("validating board: %w", err)
	}
	b.ID = "new-board-id"
	f.views = append(f.views, boards.BoardView{Board: b})
	return b, nil
}

func (f *fakeBoardService) UpdateBoard(_ context.Context, id kernel.BoardID, name, baseURL string, enabled bool) (boards.Board, error) {
	if f.err != nil {
		return boards.Board{}, f.err
	}
	for i, v := range f.views {
		if v.Board.ID == id {
			f.views[i].Board.Name = name
			f.views[i].Board.BaseURL = baseURL
			f.views[i].Board.Enabled = enabled
			return f.views[i].Board, nil
		}
	}
	return boards.Board{}, &kernel.NotFoundError{Kind: "board", ID: string(id)}
}

func (f *fakeBoardService) DeleteBoard(_ context.Context, id kernel.BoardID) error {
	if f.err != nil {
		return f.err
	}
	for i, v := range f.views {
		if v.Board.ID == id {
			f.views = slices.Delete(f.views, i, i+1)
			return nil
		}
	}
	return &kernel.NotFoundError{Kind: "board", ID: string(id)}
}

func (f *fakeBoardService) GetBoardAdapter(_ context.Context, boardID kernel.BoardID) (boards.Adapter, error) {
	if f.err != nil {
		return boards.Adapter{}, f.err
	}
	for _, v := range f.views {
		if v.Board.ID == boardID && v.Adapter != nil {
			return *v.Adapter, nil
		}
	}
	return boards.Adapter{}, &kernel.NotFoundError{Kind: "adapter", ID: string(boardID)}
}

func (f *fakeBoardService) ApproveBoardAdapter(_ context.Context, boardID kernel.BoardID) (boards.Adapter, error) {
	if f.err != nil {
		return boards.Adapter{}, f.err
	}
	for i, v := range f.views {
		if v.Board.ID == boardID && v.Adapter != nil {
			f.views[i].Adapter.Status = boards.AdapterStatusApproved
			return *f.views[i].Adapter, nil
		}
	}
	return boards.Adapter{}, &kernel.NotFoundError{Kind: "adapter", ID: string(boardID)}
}

func newBoardRouter(svc *fakeBoardService) *chi.Mux {
	r := handler.NewRouter(slog.Default())
	r.Get("/api/boards", handler.ListBoards(svc))
	r.Post("/api/boards", handler.PostBoard(svc))
	r.Put("/api/boards/{id}", handler.PutBoard(svc))
	r.Delete("/api/boards/{id}", handler.DeleteBoard(svc))
	return r
}

// AC: CRUD works; four boards seeded.

func TestPostBoard_CreatesBoard(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		body        string
		wantStatus  int
		wantName    string
		wantEnabled bool
	}{
		{
			name:        "creates board with valid body",
			body:        `{"name":"Indeed","base_url":"https://www.indeed.com"}`,
			wantStatus:  http.StatusCreated,
			wantName:    "Indeed",
			wantEnabled: true,
		},
		{
			name:       "returns 400 when name is empty",
			body:       `{"name":"","base_url":"https://www.indeed.com"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns 400 when base_url is empty",
			body:       `{"name":"Indeed","base_url":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns 400 for invalid JSON",
			body:       `bad`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeBoardService{}
			r := newBoardRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/api/boards", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantName != "" {
				var resp struct {
					Name    string `json:"name"`
					Enabled bool   `json:"enabled"`
				}
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				if resp.Name != tc.wantName {
					t.Errorf("name = %q; want %q", resp.Name, tc.wantName)
				}
				if resp.Enabled != tc.wantEnabled {
					t.Errorf("enabled = %v; want %v", resp.Enabled, tc.wantEnabled)
				}
			}
		})
	}
}

// AC: Disabling all boards is allowed (UI warns later — FE concern).

func TestPutBoard_EnabledToggle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		id          string
		body        string
		wantStatus  int
		wantEnabled bool
	}{
		{
			name:        "disables board when enabled=false",
			id:          "b-1",
			body:        `{"name":"WTTJ","base_url":"https://www.wttj.co","enabled":false}`,
			wantStatus:  http.StatusOK,
			wantEnabled: false,
		},
		{
			name:        "re-enables board when enabled=true",
			id:          "b-1",
			body:        `{"name":"WTTJ","base_url":"https://www.wttj.co","enabled":true}`,
			wantStatus:  http.StatusOK,
			wantEnabled: true,
		},
		{
			name:       "returns 404 for unknown board",
			id:         "unknown",
			body:       `{"name":"X","base_url":"https://x.com","enabled":false}`,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeBoardService{
				views: []boards.BoardView{
					{Board: boards.Board{ID: "b-1", Name: "WTTJ", BaseURL: "https://www.wttj.co", Enabled: true}},
				},
			}
			r := newBoardRouter(svc)

			req := httptest.NewRequest(http.MethodPut, "/api/boards/"+tc.id, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus == http.StatusOK {
				var resp struct {
					Enabled bool `json:"enabled"`
				}
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				if resp.Enabled != tc.wantEnabled {
					t.Errorf("enabled = %v; want %v", resp.Enabled, tc.wantEnabled)
				}
			}
		})
	}
}

func TestDeleteBoard_RemovesBoard(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "deletes existing board with 204",
			id:         "b-1",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "returns 404 for unknown board",
			id:         "unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeBoardService{
				views: []boards.BoardView{
					{Board: boards.Board{ID: "b-1", Name: "WTTJ", Enabled: true}},
				},
			}
			r := newBoardRouter(svc)

			req := httptest.NewRequest(http.MethodDelete, "/api/boards/"+tc.id, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}
