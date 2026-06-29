// Command api is the Job Tendencies REST API server. It serves the React SPA's
// REST calls and publishes on-demand pipeline events to Pub/Sub. This binary wires
// the dependencies the API needs: Postgres pool, application services, chi router,
// and a Pub/Sub publisher for on-demand pipeline runs.
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

	appboards "github.com/g-trinh/job-tendencies/internal/app/boards"
	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	apppipeline "github.com/g-trinh/job-tendencies/internal/app/pipeline"
	appprofiles "github.com/g-trinh/job-tendencies/internal/app/profiles"
	"github.com/g-trinh/job-tendencies/internal/config"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
	infraboards "github.com/g-trinh/job-tendencies/internal/infra/boards"
	"github.com/g-trinh/job-tendencies/internal/infra/db"
	infrajobs "github.com/g-trinh/job-tendencies/internal/infra/jobs"
	"github.com/g-trinh/job-tendencies/internal/infra/messaging"
	infrapipeline "github.com/g-trinh/job-tendencies/internal/infra/pipeline"
	infraprofiles "github.com/g-trinh/job-tendencies/internal/infra/profiles"
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

	scrapePublisher, err := messaging.NewPubSubPublisher(ctx, cfg.GCPProjectID, cfg.PubSubScrapeTopicID)
	if err != nil {
		slog.Error("creating scrape publisher", "err", err)
		os.Exit(1)
	}

	// Register cleanup only after all fatal startup steps succeed so the os.Exit
	// branches above run with no pending defers.
	defer stop()
	defer closePool()
	defer scrapePublisher.Stop()

	// Application services wired over the Postgres repositories.
	boardSvc := appboards.New(infraboards.NewRepository(pool))
	profileSvc := appprofiles.New(infraprofiles.NewRepository(pool))
	jobSvc := appjobs.New(infrajobs.NewRepository(pool))
	pipelineSvc := apppipeline.New(infrapipeline.NewRepository(pool), scrapePublisher)

	r := handler.NewRouter(logger)
	r.Get("/healthz", handleHealthz)
	r.Get("/livez", handleHealthz)

	r.Route("/api", func(api chi.Router) {
		api.Use(handler.NewCORSMiddleware(cfg.AllowedOrigins))
		api.Get("/boards", handler.ListBoards(boardSvc))
		api.Get("/active-profile", handler.GetActiveProfile(profileSvc))
		api.Post("/pipeline/runs", handler.CreatePipelineRun(pipelineSvc, profileSvc))

		// Profile-scoped routes require a valid X-Active-Profile header.
		api.Group(func(scoped chi.Router) {
			handler.ScopedRoutes(scoped)
			scoped.Get("/jobs", handler.ListJobs(jobSvc))
			scoped.Get("/jobs/{id}", handler.GetJob(jobSvc))
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("api server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("api server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("api server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("api server shutdown error", "err", err)
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
