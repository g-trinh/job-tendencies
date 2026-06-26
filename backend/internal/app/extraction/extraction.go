// Package extraction contains the application service for the extract-worker pipeline stage.
// Phase 1: stub implementation that logs the received event and returns immediately.
// The real extraction logic (GCS load, Claude extraction, dedup, scoring) will replace
// this stub in Phase 3.
package extraction

import (
	"context"
	"log/slog"

	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
)

// Service handles listing.extract pipeline events dispatched by the push handler.
type Service struct {
	logger *slog.Logger
}

// New constructs an extraction Service.
func New(logger *slog.Logger) *Service {
	return &Service{logger: logger}
}

// HandleListingExtract is invoked for each verified listing.extract push delivery.
// Phase 1 stub: logs the event and returns nil (ack).
// Phase 3 will load the raw payload from GCS, call the LLM extraction port,
// deduplicate against existing jobs, upsert contacts, and score the result.
func (s *Service) HandleListingExtract(ctx context.Context, msg messaging.Message) error {
	s.logger.InfoContext(ctx, "listing.extract received (stub)",
		"data_len", len(msg.Data),
		"attributes", msg.Attributes,
	)
	return nil
}
