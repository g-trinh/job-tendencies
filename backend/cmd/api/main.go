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

	appauth "github.com/g-trinh/job-tendencies/internal/app/auth"
	appboards "github.com/g-trinh/job-tendencies/internal/app/boards"
	appcontacts "github.com/g-trinh/job-tendencies/internal/app/contacts"
	appdashboard "github.com/g-trinh/job-tendencies/internal/app/dashboard"
	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	apppipeline "github.com/g-trinh/job-tendencies/internal/app/pipeline"
	appprofiles "github.com/g-trinh/job-tendencies/internal/app/profiles"
	"github.com/g-trinh/job-tendencies/internal/config"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
	infraauth "github.com/g-trinh/job-tendencies/internal/infra/auth"
	infraboards "github.com/g-trinh/job-tendencies/internal/infra/boards"
	infracontacts "github.com/g-trinh/job-tendencies/internal/infra/contacts"
	infradashboard "github.com/g-trinh/job-tendencies/internal/infra/dashboard"
	"github.com/g-trinh/job-tendencies/internal/infra/db"
	infrajobs "github.com/g-trinh/job-tendencies/internal/infra/jobs"
	infrallm "github.com/g-trinh/job-tendencies/internal/infra/llm"
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

	// Auth service: Identity Platform client + encrypted session cookie.
	cookieKey, err := cfg.SessionCookieKeyBytes()
	if err != nil {
		slog.Error("loading session cookie key", "err", err)
		os.Exit(1)
	}
	idpClient, err := infraauth.NewClient(cfg.IDPAPIKey, cfg.GCPProjectID)
	if err != nil {
		slog.Error("creating Identity Platform client", "err", err)
		os.Exit(1)
	}
	authSvc, err := appauth.New(idpClient, idpClient, cookieKey)
	if err != nil {
		slog.Error("creating auth service", "err", err)
		os.Exit(1)
	}

	// LLM client shared across services that need adapter generation or extraction.
	llmClient := infrallm.New(cfg.AnthropicAPIKey, cfg.LLMModelID, logger)

	// Application services wired over the Postgres repositories.
	boardSvc := appboards.New(infraboards.NewRepository(pool), llmClient)
	profileSvc := appprofiles.NewWithExtractor(infraprofiles.NewRepository(pool), llmClient)
	jobRepo := infrajobs.NewRepository(pool)
	jobSvc := appjobs.NewWithWriter(jobRepo, jobRepo)
	contactSvc := appcontacts.New(infracontacts.NewRepository(pool))
	pipelineSvc := apppipeline.New(infrapipeline.NewRepository(pool), scrapePublisher)
	dashboardSvc := appdashboard.New(infradashboard.NewRepository(pool))

	r := handler.NewRouter(logger)
	r.Get("/healthz", handleHealthz)
	r.Get("/livez", handleHealthz)

	r.Route("/api", func(api chi.Router) {
		api.Use(handler.NewCORSMiddleware(cfg.AllowedOrigins))

		// Auth endpoints — public (no session guard). The login and logout routes
		// must remain reachable without a valid session cookie.
		api.Post("/auth/login", handler.PostLogin(authSvc, cfg.CookieSecure))
		api.Post("/auth/logout", handler.PostLogout(cfg.CookieSecure))

		// All remaining /api routes require a valid session cookie + CSRF check.
		api.Group(func(guarded chi.Router) {
			guarded.Use(handler.RequireAuth(authSvc, cfg.CookieSecure))
			guarded.Use(handler.RequireCSRF(cfg.AllowedOrigins))

			// Auth — me endpoint (reads session from context set by RequireAuth).
			guarded.Get("/auth/me", handler.GetMe())

			// Boards.
			guarded.Get("/boards", handler.ListBoards(boardSvc))
			guarded.Post("/boards", handler.PostBoard(boardSvc))
			guarded.Put("/boards/{id}", handler.PutBoard(boardSvc))
			guarded.Delete("/boards/{id}", handler.DeleteBoard(boardSvc))
			guarded.Get("/boards/{id}/adapter", handler.GetBoardAdapter(boardSvc))
			guarded.Post("/boards/{id}/adapter/generate", handler.PostGenerateAdapter(boardSvc))
			guarded.Post("/boards/{id}/adapter/approve", handler.PostApproveAdapter(boardSvc))

			// Schedule.
			guarded.Get("/schedule", handler.GetSchedule(boardSvc))
			guarded.Put("/schedule", handler.PutSchedule(boardSvc))

			// Profiles.
			guarded.Get("/profiles", handler.ListProfiles(profileSvc))
			guarded.Post("/profiles", handler.PostProfile(profileSvc))
			guarded.Get("/profiles/{id}", handler.GetProfile(profileSvc))
			guarded.Put("/profiles/{id}", handler.PutProfile(profileSvc))
			guarded.Delete("/profiles/{id}", handler.DeleteProfile(profileSvc))
			guarded.Get("/active-profile", handler.GetActiveProfile(profileSvc))
			guarded.Put("/active-profile", handler.PutActiveProfile(profileSvc))
			guarded.Patch("/profiles/{id}/identity", handler.PatchProfileIdentity(profileSvc))
			guarded.Post("/profiles/{id}/identity/import", handler.PostImportIdentity(profileSvc))
			guarded.Get("/profiles/{id}/conditions", handler.GetProfile(profileSvc))
			guarded.Put("/profiles/{id}/conditions", handler.PutProfileConditions(profileSvc))
			guarded.Get("/profiles/{id}/weights", handler.GetProfile(profileSvc))
			guarded.Put("/profiles/{id}/weights", handler.PutProfileWeights(profileSvc))

			// Contacts.
			guarded.Get("/contacts", handler.ListContacts(contactSvc))
			guarded.Get("/contacts/export.csv", handler.ExportContacts(contactSvc))
			guarded.Post("/contacts", handler.PostContact(contactSvc))
			guarded.Get("/contacts/{id}", handler.GetContact(contactSvc))
			guarded.Put("/contacts/{id}", handler.PutContact(contactSvc))
			guarded.Delete("/contacts/{id}", handler.DeleteContact(contactSvc))

			// Pipeline.
			guarded.Post("/pipeline/runs", handler.CreatePipelineRun(pipelineSvc, profileSvc))
			guarded.Get("/pipeline/runs", handler.ListPipelineRuns(pipelineSvc))
			guarded.Get("/pipeline/runs/{id}", handler.GetPipelineRun(pipelineSvc))

			// Profile-scoped routes require a valid X-Active-Profile header in addition to auth.
			guarded.Group(func(scoped chi.Router) {
				scoped.Use(handler.RequireActiveProfileMiddleware)
				scoped.Get("/jobs", handler.ListJobs(jobSvc))
				scoped.Get("/jobs/{id}", handler.GetJob(jobSvc))
				scoped.Get("/jobs/{id}/original", handler.GetJobOriginal(jobSvc))
				scoped.Patch("/jobs/{id}/application", handler.PatchJobApplication(jobSvc))

				// Dashboard.
				scoped.Get("/dashboard/skills/frequency", handler.GetDashboardSkillFrequency(dashboardSvc))
				scoped.Get("/dashboard/skills/trend", handler.GetDashboardSkillTrend(dashboardSvc))
				scoped.Get("/dashboard/matches", handler.GetDashboardMatches(dashboardSvc))
				scoped.Get("/dashboard/stats", handler.GetDashboardStats(dashboardSvc))
			})
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
