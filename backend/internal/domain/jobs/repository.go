package jobs

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository is the Job aggregate's write-side persistence port; it persists whole Job
// aggregates together with their source linkage. The job-browser read path does NOT use
// this port — reads go through the app/jobs query service (read/write split, ADR-005).
type Repository interface {
	// Create persists a new Job aggregate together with its source linkage and
	// returns the assigned id.
	Create(ctx context.Context, job Job) (kernel.JobID, error)
}
