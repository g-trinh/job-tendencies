// Package pipeline contains the on-demand pipeline-trigger application service. It
// records a scrape run and publishes scrape.tick — the same topic the scheduled cron
// uses, so scheduled and on-demand runs share one path (ADR-003, pipeline.md §6).
package pipeline

import (
	"context"
	"fmt"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
	"github.com/g-trinh/job-tendencies/internal/domain/pipeline"
)

const (
	// TriggerOnDemand marks a run started from the API.
	TriggerOnDemand = "on_demand"

	// ScrapeTickProfileAttr is the scrape.tick attribute carrying the run's profile id.
	ScrapeTickProfileAttr = "profile_id"
	// ScrapeTickRunAttr is the scrape.tick attribute carrying the run id.
	ScrapeTickRunAttr = "run_id"
	// ScrapeTickTriggerAttr is the scrape.tick attribute carrying the run trigger
	// ("scheduled" | "on_demand"). scrape-worker propagates it onto every listing.extract
	// message it publishes so extract-worker can gate Batch API routing (P5-5).
	ScrapeTickTriggerAttr = "trigger"
)

// Service triggers on-demand pipeline runs and exposes run history for polling.
type Service struct {
	runs      pipeline.RunRepository
	publisher messaging.Publisher
}

// New constructs a pipeline Service over a run repository and the scrape.tick publisher.
func New(runs pipeline.RunRepository, publisher messaging.Publisher) *Service {
	return &Service{runs: runs, publisher: publisher}
}

// CreateRun records an on-demand run for the profile and publishes scrape.tick. It
// returns the new run id for polling.
func (s *Service) CreateRun(ctx context.Context, profileID kernel.ProfileID) (kernel.ScrapeRunID, error) {
	runID, err := s.runs.CreateRun(ctx, profileID, TriggerOnDemand)
	if err != nil {
		return "", fmt.Errorf("creating scrape run: %w", err)
	}

	msg := messaging.Message{
		Attributes: map[string]string{
			ScrapeTickProfileAttr: string(profileID),
			ScrapeTickRunAttr:     string(runID),
			ScrapeTickTriggerAttr: TriggerOnDemand,
		},
	}
	if err := s.publisher.Publish(ctx, msg); err != nil {
		return "", fmt.Errorf("publishing scrape.tick for run %q: %w", runID, err)
	}
	return runID, nil
}

// ListRuns returns recent scrape runs ordered newest-first for the pipeline dashboard.
func (s *Service) ListRuns(ctx context.Context) ([]pipeline.ScrapeRun, error) {
	runs, err := s.runs.ListRuns(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing scrape runs: %w", err)
	}
	return runs, nil
}

// GetRun returns a scrape run with its per-board breakdown for status polling.
func (s *Service) GetRun(ctx context.Context, id kernel.ScrapeRunID) (pipeline.ScrapeRun, error) {
	run, err := s.runs.GetRun(ctx, id)
	if err != nil {
		return pipeline.ScrapeRun{}, fmt.Errorf("getting scrape run: %w", err)
	}
	return run, nil
}
