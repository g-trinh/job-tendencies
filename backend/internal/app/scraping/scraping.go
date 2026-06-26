// Package scraping contains the application service for the scrape-worker pipeline stage.
// Phase 1: stub implementation that logs the received event and returns immediately.
// The real scraping logic (board adapter evaluation, GCS storage, HWM, fan-out publish)
// will replace this stub in Phase 2.
package scraping

import (
	"context"
	"log/slog"

	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
)

// Service handles scrape.tick pipeline events dispatched by the push handler.
// Implement this to drive the scraping pipeline.
type Service struct {
	logger *slog.Logger
}

// New constructs a scraping Service.
func New(logger *slog.Logger) *Service {
	return &Service{logger: logger}
}

// HandleScrapeTick is invoked for each verified scrape.tick push delivery.
// Phase 1 stub: logs the event and returns nil (ack).
// Phase 2 will fetch board adapters, run the scraper, store raw listings in GCS,
// advance the high-water-mark, and publish listing.extract messages.
func (s *Service) HandleScrapeTick(ctx context.Context, msg messaging.Message) error {
	s.logger.InfoContext(ctx, "scrape.tick received (stub)",
		"data_len", len(msg.Data),
		"attributes", msg.Attributes,
	)
	return nil
}
