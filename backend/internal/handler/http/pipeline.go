package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/pipeline"
)

// PipelineRunner records and triggers on-demand pipeline runs. Implemented by
// app/pipeline.Service.
type PipelineRunner interface {
	CreateRun(ctx context.Context, profileID kernel.ProfileID) (kernel.ScrapeRunID, error)
}

// PipelineRunReader exposes run history for status polling. Implemented by
// app/pipeline.Service.
type PipelineRunReader interface {
	ListRuns(ctx context.Context) ([]pipeline.ScrapeRun, error)
	GetRun(ctx context.Context, id kernel.ScrapeRunID) (pipeline.ScrapeRun, error)
}

// pipelineRunResponse is the JSON shape returned by POST /api/pipeline/runs.
type pipelineRunResponse struct {
	RunID string `json:"run_id"`
}

// scrapeRunListResponse is the JSON shape returned by GET /api/pipeline/runs.
type scrapeRunListResponse struct {
	Runs []scrapeRunSummary `json:"runs"`
}

// scrapeRunSummary is one entry in the run list.
type scrapeRunSummary struct {
	RunID      string     `json:"run_id"`
	ProfileID  string     `json:"profile_id"`
	Trigger    string     `json:"trigger"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// scrapeRunDetailResponse is the JSON shape returned by GET /api/pipeline/runs/{id}.
type scrapeRunDetailResponse struct {
	scrapeRunSummary
	Boards []scrapeRunBoardResponse `json:"boards"`
}

// scrapeRunBoardResponse is one board's progress within a run.
type scrapeRunBoardResponse struct {
	BoardID          string     `json:"board_id"`
	Status           string     `json:"status"`
	PagesFetched     int        `json:"pages_fetched"`
	ListingsCaptured int        `json:"listings_captured"`
	Error            string     `json:"error,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
}

// CreatePipelineRun handles POST /api/pipeline/runs. It resolves the active profile,
// records a run and publishes scrape.tick, returning the run id for polling.
func CreatePipelineRun(runner PipelineRunner, profiles ActiveProfileResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, err := profiles.ActiveProfile(r.Context())
		if err != nil {
			RespondError(w, r, err)
			return
		}

		runID, err := runner.CreateRun(r.Context(), profile.ID)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusAccepted, pipelineRunResponse{RunID: string(runID)})
	}
}

// ListPipelineRuns handles GET /api/pipeline/runs. It returns recent runs ordered
// newest first so the UI can show crawl history.
func ListPipelineRuns(reader PipelineRunReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runs, err := reader.ListRuns(r.Context())
		if err != nil {
			RespondError(w, r, err)
			return
		}
		resp := scrapeRunListResponse{Runs: make([]scrapeRunSummary, 0, len(runs))}
		for _, run := range runs {
			resp.Runs = append(resp.Runs, toRunSummary(run))
		}
		respond(w, http.StatusOK, resp)
	}
}

// GetPipelineRun handles GET /api/pipeline/runs/{id}. It returns a run's status and
// per-board progress breakdown for status polling.
func GetPipelineRun(reader PipelineRunReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ScrapeRunID(chi.URLParam(r, "id"))
		run, err := reader.GetRun(r.Context(), id)
		if err != nil {
			RespondError(w, r, err)
			return
		}

		resp := scrapeRunDetailResponse{
			scrapeRunSummary: toRunSummary(run),
			Boards:           make([]scrapeRunBoardResponse, 0, len(run.Boards)),
		}
		for _, b := range run.Boards {
			resp.Boards = append(resp.Boards, scrapeRunBoardResponse{
				BoardID:          string(b.BoardID),
				Status:           b.Status,
				PagesFetched:     b.PagesFetched,
				ListingsCaptured: b.ListingsCaptured,
				Error:            b.Error,
				StartedAt:        b.StartedAt,
				FinishedAt:       b.FinishedAt,
			})
		}
		respond(w, http.StatusOK, resp)
	}
}

func toRunSummary(run pipeline.ScrapeRun) scrapeRunSummary {
	return scrapeRunSummary{
		RunID:      string(run.ID),
		ProfileID:  string(run.ProfileID),
		Trigger:    run.Trigger,
		Status:     run.Status,
		CreatedAt:  run.CreatedAt,
		FinishedAt: run.FinishedAt,
	}
}
