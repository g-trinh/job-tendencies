package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/pipeline"
)

// --- P3-SCR-5: GET /api/pipeline/runs ---

type fakeRunReader struct {
	runs []pipeline.ScrapeRun
	run  pipeline.ScrapeRun
	err  error
}

func (f *fakeRunReader) ListRuns(_ context.Context) ([]pipeline.ScrapeRun, error) {
	return f.runs, f.err
}

func (f *fakeRunReader) GetRun(_ context.Context, _ kernel.ScrapeRunID) (pipeline.ScrapeRun, error) {
	return f.run, f.err
}

func TestListPipelineRuns(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	reader := &fakeRunReader{
		runs: []pipeline.ScrapeRun{
			{
				ID:        "run-1",
				ProfileID: "p-1",
				Trigger:   "on_demand",
				Status:    "done",
				CreatedAt: now,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/runs", nil)
	rec := httptest.NewRecorder()
	ListPipelineRuns(reader).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp scrapeRunListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error = %v", err)
	}
	if len(resp.Runs) != 1 {
		t.Fatalf("len(runs) = %d, want 1", len(resp.Runs))
	}
	got := resp.Runs[0]
	if got.RunID != "run-1" {
		t.Errorf("run_id = %q, want %q", got.RunID, "run-1")
	}
	if got.Status != "done" {
		t.Errorf("status = %q, want %q", got.Status, "done")
	}
}

func TestGetPipelineRun(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	started := now.Add(time.Second)
	reader := &fakeRunReader{
		run: pipeline.ScrapeRun{
			ID:        "run-42",
			ProfileID: "p-1",
			Trigger:   "scheduled",
			Status:    "done",
			CreatedAt: now,
			Boards: []pipeline.ScrapeRunBoard{
				{
					ID:               "rb-1",
					BoardID:          "wttj",
					Status:           "done",
					PagesFetched:     5,
					ListingsCaptured: 30,
					StartedAt:        &started,
				},
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/pipeline/runs/{id}", GetPipelineRun(reader))

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/runs/run-42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp scrapeRunDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error = %v", err)
	}
	if resp.RunID != "run-42" {
		t.Errorf("run_id = %q, want %q", resp.RunID, "run-42")
	}
	if len(resp.Boards) != 1 {
		t.Fatalf("len(boards) = %d, want 1", len(resp.Boards))
	}
	b := resp.Boards[0]
	if b.BoardID != "wttj" {
		t.Errorf("board_id = %q, want %q", b.BoardID, "wttj")
	}
	if b.PagesFetched != 5 {
		t.Errorf("pages_fetched = %d, want 5", b.PagesFetched)
	}
	if b.ListingsCaptured != 30 {
		t.Errorf("listings_captured = %d, want 30", b.ListingsCaptured)
	}
}

func TestGetPipelineRun_NotFound(t *testing.T) {
	t.Parallel()

	reader := &fakeRunReader{err: &kernel.NotFoundError{Kind: "scrape_run", ID: "missing"}}

	r := chi.NewRouter()
	r.Get("/api/pipeline/runs/{id}", GetPipelineRun(reader))

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/runs/missing", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not found") {
		t.Errorf("body = %q, want 'not found'", rec.Body.String())
	}
}
