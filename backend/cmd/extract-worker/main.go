// Command extract-worker is the Job Tendencies extraction worker. It runs on Cloud Run
// and receives listing.extract Pub/Sub push messages (OIDC-authenticated). It loads raw
// listings from GCS, sends them to Claude for structured extraction, deduplicates across
// boards via fingerprint, upserts recruiter contacts, creates or merges Jobs, and
// triggers fit scoring for the owning profile. Raw listings are never translated.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appcontacts "github.com/g-trinh/job-tendencies/internal/app/contacts"
	appextraction "github.com/g-trinh/job-tendencies/internal/app/extraction"
	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	appprofiles "github.com/g-trinh/job-tendencies/internal/app/profiles"
	appscoring "github.com/g-trinh/job-tendencies/internal/app/scoring"
	"github.com/g-trinh/job-tendencies/internal/config"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
	"github.com/g-trinh/job-tendencies/internal/infra/blobstore"
	infracontacts "github.com/g-trinh/job-tendencies/internal/infra/contacts"
	"github.com/g-trinh/job-tendencies/internal/infra/db"
	infrajobs "github.com/g-trinh/job-tendencies/internal/infra/jobs"
	infrallm "github.com/g-trinh/job-tendencies/internal/infra/llm"
	infraprofiles "github.com/g-trinh/job-tendencies/internal/infra/profiles"
	infrascoring "github.com/g-trinh/job-tendencies/internal/infra/scoring"
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

	// LLM provider (ADR-006: one provider serves every LLM task).
	extractor, err := infrallm.NewProvider(cfg, logger)
	if err != nil {
		slog.Error("creating llm provider", "err", err)
		os.Exit(1)
	}

	// Register cleanup only after all fatal startup steps succeed so the os.Exit
	// branches above run with no pending defers.
	defer stop()
	defer closePool()

	jobRepo := infrajobs.NewRepository(pool)

	// P3-EX-3: contacts service wrapped in the extraction context's ContactUpserter.
	contactsSvc := appcontacts.New(infracontacts.NewRepository(pool))
	contactsAdapter := &contactsAdapter{svc: contactsSvc}

	// P3-EX-4: scoring service wired with the jobs and profiles adapters.
	jobsSvc := appjobs.NewWithWriter(jobRepo, jobRepo)
	profilesSvc := appprofiles.New(infraprofiles.NewRepository(pool))
	scoringSvc := appscoring.New(
		infrascoring.NewJobsAdapter(jobsSvc),
		infrascoring.NewProfilesAdapter(profilesSvc),
		infrascoring.NewRepository(pool),
	)
	scorerAdapter := &scorerAdapter{svc: scoringSvc}

	extractionSvc := appextraction.New(
		infrascraping.NewRawListingRepository(pool),
		rawStore,
		extractor,
		jobRepo,
		logger,
	).WithContacts(contactsAdapter).WithScorer(scorerAdapter).WithBatchEnabled(cfg.LLMBatchEnabled)

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

// contactsAdapter bridges app/contacts.Service to app/extraction.ContactUpserter.
// It narrows the contacts service signature to the fields needed for extraction
// and returns only the stable contact id.
type contactsAdapter struct {
	svc *appcontacts.Service
}

func (a *contactsAdapter) UpsertContact(ctx context.Context, name, company, email, linkedInURL, phone string) (kernel.ContactID, error) {
	c, _, err := a.svc.UpsertContact(ctx, name, company, email, linkedInURL, phone, "", nil)
	if err != nil {
		return "", fmt.Errorf("upserting recruiter contact: %w", err)
	}
	return c.ID, nil
}

// scorerAdapter bridges app/scoring.Service to app/extraction.JobScorer.
// Scoring failures are surfaced as errors; the caller decides whether to treat
// them as fatal or log-and-continue.
type scorerAdapter struct {
	svc *appscoring.Service
}

func (a *scorerAdapter) ScoreJob(ctx context.Context, jobID kernel.JobID, profileID kernel.ProfileID) error {
	_, err := a.svc.ScoreJob(ctx, jobID, profileID)
	if err != nil {
		return fmt.Errorf("scoring job %q for profile %q: %w", jobID, profileID, err)
	}
	return nil
}

// Verify adapters satisfy the extraction consumer interfaces at compile time.
var (
	_ appextraction.ContactUpserter = (*contactsAdapter)(nil)
	_ appextraction.JobScorer       = (*scorerAdapter)(nil)
)

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
