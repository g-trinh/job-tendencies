// Package config loads and validates runtime configuration from environment
// variables. All Job Tendencies binaries (api, scrape-worker, extract-worker)
// call [Load] at startup; the function fails fast when any required variable is
// absent so misconfigured deployments surface immediately rather than at the
// first I/O call.
//
// Required variables:
//   - DATABASE_URL — PostgreSQL connection string used by goose migrations.
//     Example: postgres://user:pass@host:5432/dbname?sslmode=require
//     For Cloud SQL, use the Auth Proxy socket: host=/cloudsql/<instance> ...
//
// Optional variables with defaults:
//   - PORT              — TCP port the HTTP server listens on (default: 8080). Cloud Run sets this.
//   - LOG_LEVEL         — slog verbosity: debug, info, warn, error (default: info).
//   - LLM_PROVIDER      — LLM provider selection: "claude" or "deepseek" (default: claude).
//     One provider serves every LLM task (adapter generation, listing extraction,
//     identity import) — ADR-006. Load fails fast on any other value.
//   - ANTHROPIC_API_KEY — Anthropic API key; required for LLM port when LLM_PROVIDER=claude.
//   - LLM_MODEL_ID      — Claude model id (default: claude-opus-4-8).
//   - DEEPSEEK_API_KEY  — DeepSeek API key; required when LLM_PROVIDER=deepseek.
//   - DEEPSEEK_MODEL_ID — DeepSeek model id; required when LLM_PROVIDER=deepseek (no default).
//   - DEEPSEEK_BASE_URL — DeepSeek API base URL (default: https://api.deepseek.com).
//   - LLM_BATCH_ENABLED — Route scheduled bulk extraction through the Anthropic Batch
//     API cost lever (ADR-004, pipeline.md §3). Default: false. P5-5: the extension
//     point is wired (extract-worker reads this flag and the propagated run trigger),
//     but real batch submission is deferred behind open question #1 (Batch API latency
//     vs the scheduled cron window, PM-blocked) — see docs/architecture/tech_debt.md.
//     Enabling this today only logs a warning and falls back to synchronous extraction.
//   - GCP_PROJECT_ID    — GCP project id; required for Pub/Sub and GCS.
//   - GCS_RAW_BUCKET    — GCS bucket name for raw HTML/JSON payloads.
//   - PUBSUB_SCRAPE_TOPIC_ID   — Pub/Sub topic id for scrape.tick events.
//   - PUBSUB_EXTRACT_TOPIC_ID  — Pub/Sub topic id for listing.extract events.
//   - CLOUD_SQL_INSTANCE — Cloud SQL instance connection name (project:region:instance).
//   - DB_IAM_USER        — IAM DB user (SA email for deployed; developer Google account for local ADC).
//   - DB_NAME            — Postgres database name (default: job_tendencies).
//   - WORKER_SERVICE_URL — Cloud Run service URL used as the OIDC token audience.
//   - PUBSUB_PUSH_SA     — Service account email authorised to deliver Pub/Sub push messages.
//   - ALLOWED_ORIGINS   — Comma-separated list of browser origins permitted to call the API
//     cross-origin. Example: https://job-tendencies-dev.web.app,http://localhost:5173
//     Optional. When unset, no cross-origin requests are permitted (no wildcard fallback).
//
// Auth-specific variables (api binary only):
//   - IDP_API_KEY        — Identity Platform web API key. Required by the api binary for
//     the auth endpoints.
//   - SESSION_COOKIE_KEY — Hex-encoded 32-byte AES-256 key used to encrypt session cookies.
//     Must be exactly 64 hex characters. Required by the api binary.
//     Generate: openssl rand -hex 32
//   - COOKIE_SECURE      — Set to "false" to disable the Secure flag on session cookies.
//     Defaults to "true". Use COOKIE_SECURE=false for local HTTP development only.
package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DefaultLLMModelID is the default Claude model used when LLM_MODEL_ID is not set.
const DefaultLLMModelID = "claude-opus-4-8"

// DefaultLLMProvider is the default LLM provider used when LLM_PROVIDER is not set.
const DefaultLLMProvider = "claude"

// DefaultDeepSeekBaseURL is the default DeepSeek API base URL used when
// DEEPSEEK_BASE_URL is not set.
const DefaultDeepSeekBaseURL = "https://api.deepseek.com"

// DefaultDBName is the default Postgres database name.
const DefaultDBName = "job_tendencies"

// Config holds all runtime configuration for a Job Tendencies binary.
// The zero value is invalid; obtain a valid instance via [Load].
type Config struct {
	// Port is the TCP port the HTTP server listens on. Cloud Run sets PORT=8080.
	Port string

	// DatabaseURL is the PostgreSQL connection string used by goose migrations.
	// Required. For the running application, prefer CloudSQLInstance + DBIAMUser + DBName.
	DatabaseURL string

	// LogLevel controls slog verbosity: debug, info, warn, error.
	// Defaults to "info". Unrecognised values are silently treated as "info".
	LogLevel string

	// LLMProvider selects which LLM provider serves every LLM task (adapter
	// generation, listing extraction, identity import) — ADR-006. One of
	// "claude" or "deepseek". Defaults to DefaultLLMProvider when not set.
	LLMProvider string

	// AnthropicAPIKey is the Anthropic API key used by infra/llm.
	// Optional at load time; infra/llm validates it on construction.
	AnthropicAPIKey string

	// LLMModelID is the Claude model id passed to the Anthropic API.
	// Defaults to DefaultLLMModelID when not set.
	LLMModelID string

	// DeepSeekAPIKey is the DeepSeek API key used by infra/llm when
	// LLMProvider is "deepseek". Required in that case.
	DeepSeekAPIKey string

	// DeepSeekModelID is the DeepSeek model id passed to the DeepSeek API.
	// Required when LLMProvider is "deepseek"; there is no default.
	DeepSeekModelID string

	// DeepSeekBaseURL is the DeepSeek API base URL. Defaults to
	// DefaultDeepSeekBaseURL when not set.
	DeepSeekBaseURL string

	// LLMBatchEnabled gates routing scheduled bulk extraction through the Anthropic
	// Batch API (P5-5). Defaults to false. See the LLM_BATCH_ENABLED doc above.
	LLMBatchEnabled bool

	// GCPProjectID is the GCP project id used by Pub/Sub and GCS clients.
	GCPProjectID string

	// GCSRawBucket is the GCS bucket name for raw HTML/JSON payloads.
	GCSRawBucket string

	// PubSubScrapeTopicID is the Pub/Sub topic id for scrape.tick messages.
	PubSubScrapeTopicID string

	// PubSubExtractTopicID is the Pub/Sub topic id for listing.extract messages.
	PubSubExtractTopicID string

	// CloudSQLInstance is the Cloud SQL instance connection name
	// (format: project:region:instance, e.g. job-tendencies-dev:europe-west9:jt-dev-pg).
	// Used by infra/db to establish a connection via the Cloud SQL Go connector.
	CloudSQLInstance string

	// DBIAMUser is the IAM database user name.
	// For service accounts: sa-email@project.iam.gserviceaccount.com
	// For human developers using ADC: their Google account email.
	DBIAMUser string

	// DBName is the Postgres database name. Defaults to DefaultDBName.
	DBName string

	// WorkerServiceURL is the Cloud Run service URL used as the OIDC token audience
	// when verifying Pub/Sub push requests. Example: https://scrape-worker-xxx.run.app
	WorkerServiceURL string

	// PubSubPushSA is the service account email authorised to deliver Pub/Sub push
	// messages (pubsub-push-dev@job-tendencies-dev.iam.gserviceaccount.com).
	PubSubPushSA string

	// AllowedOrigins is the list of browser origins permitted to make cross-origin
	// requests to the API. Populated from the ALLOWED_ORIGINS environment variable
	// (comma-separated). Empty means no cross-origin requests are allowed.
	AllowedOrigins []string

	// IDPAPIKey is the Identity Platform web API key used to call the Firebase
	// Auth REST endpoints. Required for the api binary; loaded from IDP_API_KEY.
	// Optional at config load time; validated by app/auth.New.
	IDPAPIKey string

	// SessionCookieKey is the hex-encoded 32-byte AES-256-GCM key used to encrypt
	// and decrypt httpOnly session cookies. Required for the api binary.
	// Must be exactly 64 hex characters (32 bytes). Loaded from SESSION_COOKIE_KEY.
	// Parse it into bytes with [Config.SessionCookieKeyBytes].
	SessionCookieKey string

	// CookieSecure controls whether session cookies carry the Secure flag (HTTPS only).
	// Defaults to true. Set COOKIE_SECURE=false for local HTTP development.
	CookieSecure bool
}

// Load reads configuration from environment variables and returns a populated
// [Config]. It returns a non-nil error listing every missing required variable
// so operators can fix all problems in a single deployment cycle.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                 envOrDefault("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		LogLevel:             envOrDefault("LOG_LEVEL", "info"),
		LLMProvider:          envOrDefault("LLM_PROVIDER", DefaultLLMProvider),
		AnthropicAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		LLMModelID:           envOrDefault("LLM_MODEL_ID", DefaultLLMModelID),
		DeepSeekAPIKey:       os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekModelID:      os.Getenv("DEEPSEEK_MODEL_ID"),
		DeepSeekBaseURL:      envOrDefault("DEEPSEEK_BASE_URL", DefaultDeepSeekBaseURL),
		LLMBatchEnabled:      parseBoolDefault("LLM_BATCH_ENABLED", false),
		GCPProjectID:         os.Getenv("GCP_PROJECT_ID"),
		GCSRawBucket:         os.Getenv("GCS_RAW_BUCKET"),
		PubSubScrapeTopicID:  os.Getenv("PUBSUB_SCRAPE_TOPIC_ID"),
		PubSubExtractTopicID: os.Getenv("PUBSUB_EXTRACT_TOPIC_ID"),
		CloudSQLInstance:     os.Getenv("CLOUD_SQL_INSTANCE"),
		DBIAMUser:            os.Getenv("DB_IAM_USER"),
		DBName:               envOrDefault("DB_NAME", DefaultDBName),
		WorkerServiceURL:     os.Getenv("WORKER_SERVICE_URL"),
		PubSubPushSA:         os.Getenv("PUBSUB_PUSH_SA"),
		AllowedOrigins:       parseOrigins(os.Getenv("ALLOWED_ORIGINS")),
		IDPAPIKey:            strings.TrimSpace(os.Getenv("IDP_API_KEY")),
		SessionCookieKey:     os.Getenv("SESSION_COOKIE_KEY"),
		CookieSecure:         parseBoolDefault("COOKIE_SECURE", true),
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	switch cfg.LLMProvider {
	case "claude":
		// No additional required vars beyond the existing optional ANTHROPIC_API_KEY,
		// validated at infra/llm construction time (unchanged behaviour).
	case "deepseek":
		if cfg.DeepSeekAPIKey == "" {
			missing = append(missing, "DEEPSEEK_API_KEY")
		}
		if cfg.DeepSeekModelID == "" {
			missing = append(missing, "DEEPSEEK_MODEL_ID")
		}
	default:
		return nil, fmt.Errorf("config: invalid LLM_PROVIDER %q: must be \"claude\" or \"deepseek\"", cfg.LLMProvider)
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("config: missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// envOrDefault returns the value of the named environment variable, or
// fallback when the variable is unset or empty.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// SessionCookieKeyBytes decodes the hex-encoded SessionCookieKey into raw bytes.
// Returns an error when the field is empty, not valid hex, or does not decode to
// exactly 32 bytes (required for AES-256-GCM).
func (c *Config) SessionCookieKeyBytes() ([]byte, error) {
	if c.SessionCookieKey == "" {
		return nil, fmt.Errorf("SESSION_COOKIE_KEY is required for the api binary")
	}
	// Trim whitespace: secret managers commonly store a trailing newline.
	key, err := hex.DecodeString(strings.TrimSpace(c.SessionCookieKey))
	if err != nil {
		return nil, fmt.Errorf("SESSION_COOKIE_KEY: invalid hex encoding: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("SESSION_COOKIE_KEY: must decode to exactly 32 bytes (64 hex chars); got %d bytes", len(key))
	}
	return key, nil
}

// parseBoolDefault parses a boolean environment variable. Returns defaultVal when
// the variable is unset or the value is not a valid bool.
func parseBoolDefault(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}

// parseOrigins splits a comma-separated list of origins into a slice, trimming
// whitespace from each entry and dropping empty tokens. Returns nil when v is empty.
func parseOrigins(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
