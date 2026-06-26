// Command extract-worker is the Job Tendencies extraction worker. It runs on Cloud Run
// and receives listing.extract Pub/Sub push messages (OIDC-authenticated). It loads raw
// listings from GCS, sends them to Claude for structured extraction, and creates one job
// per listing. Phase 2 skips dedup/merge, contacts and scoring.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appextraction "github.com/g-trinh/job-tendencies/internal/app/extraction"
	"github.com/g-trinh/job-tendencies/internal/config"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
	"github.com/g-trinh/job-tendencies/internal/infra/blobstore"
	"github.com/g-trinh/job-tendencies/internal/infra/db"
	infrajobs "github.com/g-trinh/job-tendencies/internal/infra/jobs"
	infrallm "github.com/g-trinh/job-tendencies/internal/infra/llm"
	infrascraping "github.com/g-trinh/job-tendencies/internal/infra/scraping"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("loading config", "err", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)

	pool, closePool, err := db.NewPool(ctx, cfg.CloudSQLInstance, cfg.DBIAMUser, cfg.DBName)
	if err != nil {
		slog.Error("connecting to database", "err", err)
		os.Exit(1)
	}

	rawStore, err := blobstore.NewGCSBlobStore(ctx, cfg.GCSRawBucket)
	if err != nil {
		slog.Error("creating gcs blobstore", "err", err)
		os.Exit(1)
	}

	// Register cleanup only after all fatal startup steps succeed so the os.Exit
	// branches above run with no pending defers.
	defer stop()
	defer closePool()

	extractor := infrallm.New(cfg.AnthropicAPIKey, cfg.LLMModelID, logger)

	extractionSvc := appextraction.New(
		infrascraping.NewRawListingRepository(pool),
		rawStore,
		extractor,
		infrajobs.NewRepository(pool),
		logger,
	)

	r := handler.NewRouter(logger)
	r.Get("/healthz", handleHealthz)
	r.Get("/livez", handleHealthz)

	oidcMiddleware := handler.OIDCMiddleware(
		handler.GoogleTokenVerifier{},
		cfg.WorkerServiceURL,
		cfg.PubSubPushSA,
	)
	r.With(oidcMiddleware).Post("/push/listing-extract", handler.PushListingExtract(extractionSvc, logger))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("extract-worker starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("extract-worker server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("extract-worker shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("extract-worker shutdown error", "err", err)
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
