// Command scrape-worker is the Job Tendencies scrape pipeline worker. It runs on
// Cloud Run and receives scrape.tick Pub/Sub push messages (OIDC-authenticated).
// It fetches raw listings from job boards, stores them in GCS, tracks the
// high-water-mark in Postgres, and publishes per-listing listing.extract events.
//
// Cloud Run settings: max-instances=1, concurrency=1 (single-user rate limiting
// is handled in-process via x/time/rate, per ADR-003).
//
// Required env vars for push handling:
//   - WORKER_SERVICE_URL — this service's Cloud Run URL (used as OIDC audience).
//   - PUBSUB_PUSH_SA     — push-auth SA email (pubsub-push-dev@…).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/config"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("loading config", "err", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	// Build the router with base middleware.
	r := handler.NewRouter(logger)

	// Health probes — /healthz is reserved by Cloud Run ingress; /livez is reachable.
	r.Get("/healthz", handleHealthz)
	r.Get("/livez", handleHealthz)

	// Pub/Sub push route — protected by OIDC verification.
	// Phase 1 stub: scraping.Service logs the event and returns nil.
	scrapingSvc := scraping.New(logger)
	oidcMiddleware := handler.OIDCMiddleware(
		handler.GoogleTokenVerifier{},
		cfg.WorkerServiceURL,
		cfg.PubSubPushSA,
	)
	r.With(oidcMiddleware).Post("/push/scrape-tick", handler.PushScrapeTick(scrapingSvc, logger))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		slog.Info("scrape-worker starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("scrape-worker server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("scrape-worker shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("scrape-worker shutdown error", "err", err)
	}
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// newLogger constructs a JSON structured logger at the requested level.
// Unknown level strings fall back to info.
func newLogger(level string) *slog.Logger {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
