// Package config loads and validates runtime configuration from environment
// variables. All Job Tendencies binaries (api, scrape-worker, extract-worker)
// call [Load] at startup; the function fails fast when any required variable is
// absent so misconfigured deployments surface immediately rather than at the
// first I/O call.
//
// Required variables:
//   - DATABASE_URL — PostgreSQL connection string (libpq format or DSN).
//     Example: postgres://user:pass@host:5432/dbname?sslmode=require
//
// Optional variables with defaults:
//   - PORT              — TCP port the HTTP server listens on (default: 8080). Cloud Run sets this automatically.
//   - LOG_LEVEL         — slog verbosity: debug, info, warn, error (default: info).
//   - ANTHROPIC_API_KEY — Anthropic API key; required for binaries that call the LLM port.
//   - LLM_MODEL_ID      — Claude model to use for extraction/adapter gen (default: claude-opus-4-8).
//   - GCP_PROJECT_ID    — GCP project id; required for Pub/Sub and GCS.
//   - GCS_RAW_BUCKET    — GCS bucket name for raw HTML/JSON payloads.
//   - PUBSUB_SCRAPE_TOPIC_ID   — Pub/Sub topic id for scrape.tick events.
//   - PUBSUB_EXTRACT_TOPIC_ID  — Pub/Sub topic id for listing.extract events.
package config

import (
	"fmt"
	"os"
	"strings"
)

// DefaultLLMModelID is the default Claude model used when LLM_MODEL_ID is not set.
const DefaultLLMModelID = "claude-opus-4-8"

// Config holds all runtime configuration for a Job Tendencies binary.
// The zero value is invalid; obtain a valid instance via [Load].
type Config struct {
	// Port is the TCP port the HTTP server listens on. Cloud Run sets PORT=8080.
	Port string

	// DatabaseURL is the PostgreSQL connection string. Required.
	// Format: postgres://user:pass@host:5432/dbname?sslmode=require
	DatabaseURL string

	// LogLevel controls slog verbosity: debug, info, warn, error.
	// Defaults to "info". Unrecognised values are silently treated as "info".
	LogLevel string

	// AnthropicAPIKey is the Anthropic API key used by infra/llm.
	// Optional at load time; infra/llm validates it on construction.
	AnthropicAPIKey string

	// LLMModelID is the Claude model id passed to the Anthropic API.
	// Defaults to DefaultLLMModelID when not set.
	LLMModelID string

	// GCPProjectID is the GCP project id used by Pub/Sub and GCS clients.
	// Optional at load time; infra adapters validate it on construction.
	GCPProjectID string

	// GCSRawBucket is the GCS bucket name for raw HTML/JSON payloads.
	// Required by infra/blobstore.
	GCSRawBucket string

	// PubSubScrapeTopicID is the Pub/Sub topic id for scrape.tick messages.
	PubSubScrapeTopicID string

	// PubSubExtractTopicID is the Pub/Sub topic id for listing.extract messages.
	PubSubExtractTopicID string
}

// Load reads configuration from environment variables and returns a populated
// [Config]. It returns a non-nil error listing every missing required variable
// so operators can fix all problems in a single deployment cycle.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                 envOrDefault("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		LogLevel:             envOrDefault("LOG_LEVEL", "info"),
		AnthropicAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		LLMModelID:           envOrDefault("LLM_MODEL_ID", DefaultLLMModelID),
		GCPProjectID:         os.Getenv("GCP_PROJECT_ID"),
		GCSRawBucket:         os.Getenv("GCS_RAW_BUCKET"),
		PubSubScrapeTopicID:  os.Getenv("PUBSUB_SCRAPE_TOPIC_ID"),
		PubSubExtractTopicID: os.Getenv("PUBSUB_EXTRACT_TOPIC_ID"),
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
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
