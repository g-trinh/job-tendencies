// Command scrape-worker is the Job Tendencies scrape pipeline worker. It runs on
// Cloud Run and receives scrape.tick Pub/Sub push messages (OIDC-authenticated).
// It fetches raw listings from job boards, stores them in GCS, tracks the
// high-water-mark in Postgres, and publishes per-listing listing.extract events.
//
// Cloud Run settings: max-instances=1, concurrency=1 (single-user rate limiting
// is handled in-process via x/time/rate, per ADR-003).
//
// Phase 0: boots a minimal HTTP server exposing /healthz only. Pub/Sub push
// handler and scraping logic will be added in later phases.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/g-trinh/job-tendencies/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("loading config", "err", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// /healthz is reserved by Cloud Run's ingress (requests never reach the
	// container), so also expose /livez for reachable health checks.
	r.Get("/healthz", handleHealthz)
	r.Get("/livez", handleHealthz)

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
