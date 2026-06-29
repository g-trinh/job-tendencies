package jobs

import (
	"context"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository is the Job aggregate's write-side persistence port; it persists whole Job
// aggregates together with their source linkage. The job-browser read path does NOT use
// this port — reads go through the app/jobs query service (read/write split, ADR-005).
type Repository interface {
	// Create persists a new Job aggregate together with its source linkage and
	// returns the assigned id.
	Create(ctx context.Context, job Job) (kernel.JobID, error)
	// FindByFingerprint looks up an existing job by its cross-board dedup key.
	// Returns (id, true, nil) on a hit and ("", false, nil) when not found.
	FindByFingerprint(ctx context.Context, fingerprint string) (kernel.JobID, bool, error)
	// MergeSource appends a JobSource row to an existing job (idempotent on duplicate
	// raw_listing_id), advances last_seen, and sets contact_id when the job does not
	// already have one. Use this when a fingerprint match collapses two listings from
	// different boards into the same Job.
	MergeSource(ctx context.Context, jobID kernel.JobID, source JobSource, lastSeen time.Time, contactID *kernel.ContactID) error
}
