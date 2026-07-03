package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/g-trinh/job-tendencies/internal/config"
	domainllm "github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// Provider is the union of every LLM capability used across the application:
// adapter generation, listing extraction, and LinkedIn PDF identity extraction.
// Exactly one Provider implementation serves every LLM task (ADR-006) — there is
// no per-task routing or fallback between providers.
type Provider interface {
	GenerateAdapter(ctx context.Context, boardURL string, exampleResponse string) (*domainllm.AdapterSpec, error)
	Extract(ctx context.Context, raw string) (*domainllm.ExtractedListing, error)
	ExtractIdentity(ctx context.Context, pdf []byte) (*domainllm.ExtractedIdentity, error)
}

var _ Provider = (*Client)(nil)
var _ Provider = (*deepSeekClient)(nil)

// NewProvider constructs the single LLM Provider selected by cfg.LLMProvider
// (ADR-006). It fails when the provider is not one of "claude" or "deepseek",
// or when the selected provider's required configuration is missing.
func NewProvider(cfg *config.Config, logger *slog.Logger) (Provider, error) {
	switch cfg.LLMProvider {
	case "claude":
		return New(cfg.AnthropicAPIKey, cfg.LLMModelID, logger), nil
	case "deepseek":
		return newDeepSeek(cfg.DeepSeekAPIKey, cfg.DeepSeekModelID, cfg.DeepSeekBaseURL, logger)
	default:
		return nil, fmt.Errorf("llm: unknown provider %q: must be \"claude\" or \"deepseek\"", cfg.LLMProvider)
	}
}
