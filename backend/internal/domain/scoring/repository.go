package scoring

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository is the scoring context's write port. It upserts job_score rows and
// provides point lookups for existing scores. The Postgres implementation lives in
// internal/infra/scoring.
type Repository interface {
	// Upsert inserts or updates the fit score for a (job, profile) pair. It is
	// idempotent: re-running the scorer for the same pair overwrites the prior result.
	Upsert(ctx context.Context, score JobScore) error
	// FindByJobAndProfile returns the stored score for a (job, profile) pair, or a
	// kernel.NotFoundError when none exists yet.
	FindByJobAndProfile(ctx context.Context, jobID kernel.JobID, profileID kernel.ProfileID) (JobScore, error)
}
