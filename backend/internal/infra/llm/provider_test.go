package llm

import (
	"testing"

	"github.com/g-trinh/job-tendencies/internal/config"
)

// AC: NewProvider selects claude by default (ADR-006, config LLM_PROVIDER default).
// AC: NewProvider selects deepseek when LLM_PROVIDER=deepseek and DeepSeek config is present.
// AC: NewProvider errors on an unknown provider value.

func TestNewProvider(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		cfg          *config.Config
		wantErr      bool
		wantDeepSeek bool
	}{
		{
			name:         "default provider constructs the claude client",
			cfg:          &config.Config{LLMProvider: "claude", LLMModelID: "claude-opus-4-8"},
			wantErr:      false,
			wantDeepSeek: false,
		},
		{
			name: "deepseek provider with valid config constructs the deepseek client",
			cfg: &config.Config{
				LLMProvider:     "deepseek",
				DeepSeekAPIKey:  "test-key",
				DeepSeekModelID: "deepseek-chat",
			},
			wantErr:      false,
			wantDeepSeek: true,
		},
		{
			name:    "unknown provider errors",
			cfg:     &config.Config{LLMProvider: "gpt5"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			provider, err := NewProvider(tc.cfg, testLogger())

			if tc.wantErr {
				if err == nil {
					t.Fatal("NewProvider() expected error; got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewProvider() unexpected error: %v", err)
			}

			_, isDeepSeek := provider.(*deepSeekClient)
			if isDeepSeek != tc.wantDeepSeek {
				t.Errorf("provider is *deepSeekClient = %v; want %v", isDeepSeek, tc.wantDeepSeek)
			}
		})
	}
}
