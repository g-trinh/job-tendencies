// Command scrape-worker is the Job Tendencies scrape pipeline worker. It runs on
// Cloud Run and receives scrape.tick Pub/Sub push messages (OIDC-authenticated).
// It fetches raw listings from job boards' JSON APIs, stores them in GCS, tracks the
// high-water-mark in Postgres, and publishes per-listing listing.extract events.
//
// Cloud Run settings: max-instances=1, concurrency=1 (single-user rate limiting
// is handled in-process via x/time/rate, per ADR-003).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appboards "github.com/g-trinh/job-tendencies/internal/app/boards"
	appprofiles "github.com/g-trinh/job-tendencies/internal/app/profiles"
	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/config"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
	"github.com/g-trinh/job-tendencies/internal/infra/blobstore"
	infraboards "github.com/g-trinh/job-tendencies/internal/infra/boards"
	"github.com/g-trinh/job-tendencies/internal/infra/db"
	"github.com/g-trinh/job-tendencies/internal/infra/messaging"
	infraprofiles "github.com/g-trinh/job-tendencies/internal/infra/profiles"
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
	defer stop()

	pool, closePool, err := db.NewPool(ctx, cfg.CloudSQLInstance, cfg.DBIAMUser, cfg.DBName)
	if err != nil {
		slog.Error("connecting to database", "err", err)
		os.Exit(1)
	}
	defer closePool()

	rawStore, err := blobstore.NewGCSBlobStore(ctx, cfg.GCSRawBucket)
	if err != nil {
		slog.Error("creating gcs blobstore", "err", err)
		os.Exit(1)
	}

	extractPublisher, err := messaging.NewPubSubPublisher(ctx, cfg.GCPProjectID, cfg.PubSubExtractTopicID)
	if err != nil {
		slog.Error("creating extract publisher", "err", err)
		os.Exit(1)
	}
	defer extractPublisher.Stop()

	boardSvc := appboards.New(infraboards.NewRepository(pool))
	profileSvc := appprofiles.New(infraprofiles.NewRepository(pool))

	scrapingSvc := appscraping.New(
		adapterSource{boards: boardSvc},
		targetSource{profiles: profileSvc},
		infrascraping.NewFetcher(),
		rawStore,
		infrascraping.NewRawListingRepository(pool),
		infrascraping.NewHighWaterMarkRepository(pool),
		extractPublisher,
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
	r.With(oidcMiddleware).Post("/push/scrape-tick", handler.PushScrapeTick(scrapingSvc, logger))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

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

// adapterSource maps the board-manager service into the scraping context's AdapterSource
// port, keeping the two contexts from sharing domain objects.
type adapterSource struct {
	boards *appboards.Service
}

func (a adapterSource) ApprovedBoardAdapters(ctx context.Context) ([]appscraping.BoardAdapter, error) {
	adapters, err := a.boards.ApprovedAdapters(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]appscraping.BoardAdapter, 0, len(adapters))
	for _, ad := range adapters {
		out = append(out, appscraping.BoardAdapter{BoardID: ad.BoardID, Spec: ad.Spec})
	}
	return out, nil
}

// targetSource maps the profiles service into the scraping context's TargetSource port.
type targetSource struct {
	profiles *appprofiles.Service
}

func (t targetSource) ActiveTarget(ctx context.Context) (appscraping.ScrapeTarget, error) {
	p, err := t.profiles.ActiveProfile(ctx)
	if err != nil {
		return appscraping.ScrapeTarget{}, err
	}
	return appscraping.ScrapeTarget{
		ProfileID: p.ID,
		Keywords:  p.SearchKeywords,
		Location:  p.Location,
	}, nil
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
