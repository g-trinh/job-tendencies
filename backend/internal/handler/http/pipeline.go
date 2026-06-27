package handler

import (
	"context"
	"net/http"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// PipelineRunner records and triggers on-demand pipeline runs. Implemented by
// app/pipeline.Service.
type PipelineRunner interface {
	CreateRun(ctx context.Context, profileID kernel.ProfileID) (kernel.ScrapeRunID, error)
}

// pipelineRunResponse is the JSON shape returned by POST /api/pipeline/runs.
type pipelineRunResponse struct {
	RunID string `json:"run_id"`
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
