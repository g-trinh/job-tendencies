// Package pipeline is the pipeline bounded context. It owns scrape-run tracking: each
// crawl invocation (scheduled cron or on-demand API trigger) is recorded as a run so
// it can be polled. Phase 2 persists only the run id and trigger; richer run state
// lands in a later phase.
package pipeline

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// RunRepository records scrape runs. Per ADR-005 the aggregate repository interface
// lives in the domain.
type RunRepository interface {
	// CreateRun inserts a run for the profile with the given trigger and returns its id.
	CreateRun(ctx context.Context, profileID kernel.ProfileID, trigger string) (kernel.ScrapeRunID, error)
}
