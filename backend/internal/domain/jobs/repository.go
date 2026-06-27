package jobs

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository is the Job aggregate's write-side persistence port. Per ADR-005 the
// aggregate repository interface lives in the domain (not the consumer package); it
// persists whole Job aggregates together with their source linkage. The job-browser
// read path does NOT use this port — reads go through the app/jobs query service,
// which projects read models directly from storage.
type Repository interface {
	// Create persists a new Job aggregate together with its source linkage and
	// returns the assigned id.
	Create(ctx context.Context, job Job) (kernel.JobID, error)
}
