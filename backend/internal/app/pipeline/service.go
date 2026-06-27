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
)

// Service triggers on-demand pipeline runs.
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
		},
	}
	if err := s.publisher.Publish(ctx, msg); err != nil {
		return "", fmt.Errorf("publishing scrape.tick for run %q: %w", runID, err)
	}
	return runID, nil
}
