package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// t.Setenv modifies global process environment, so these tests must not
	// run in parallel with each other or with any test that reads the same vars.
	tests := []struct {
		name        string
		env         map[string]string
		wantErr     bool
		errContains string
		wantPort    string
		wantLevel   string
	}{
		{
			name:      "all required vars set — returns valid config",
			env:       map[string]string{"DATABASE_URL": "postgres://localhost/testdb"},
			wantErr:   false,
			wantPort:  "8080",
			wantLevel: "info",
		},
		{
			name:        "DATABASE_URL missing — fails fast",
			env:         map[string]string{},
			wantErr:     true,
			errContains: "DATABASE_URL",
		},
		{
			name: "optional PORT overrides default",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/testdb",
				"PORT":         "9090",
			},
			wantErr:  false,
			wantPort: "9090",
		},
		{
			name: "optional LOG_LEVEL overrides default",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/testdb",
				"LOG_LEVEL":    "debug",
			},
			wantErr:   false,
			wantLevel: "debug",
		},
		{
			name: "LLM_MODEL_ID defaults to claude-opus-4-8 when not set",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/testdb",
			},
			wantErr: false,
		},
		{
			name: "optional LLM_MODEL_ID overrides default",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/testdb",
				"LLM_MODEL_ID": "claude-sonnet-4-6",
			},
			wantErr: false,
		},
		{
			name: "LLM_PROVIDER defaults to claude when not set",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/testdb",
			},
			wantErr: false,
		},
		{
			name: "deepseek provider without api key and model id fails fast",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/testdb",
				"LLM_PROVIDER": "deepseek",
			},
			wantErr:     true,
			errContains: "DEEPSEEK_API_KEY",
		},
		{
			name: "deepseek provider with api key and model id loads",
			env: map[string]string{
				"DATABASE_URL":      "postgres://localhost/testdb",
				"LLM_PROVIDER":      "deepseek",
				"DEEPSEEK_API_KEY":  "sk-test",
				"DEEPSEEK_MODEL_ID": "deepseek-chat",
			},
			wantErr: false,
		},
		{
			name: "invalid LLM_PROVIDER fails fast",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/testdb",
				"LLM_PROVIDER": "gpt5",
			},
			wantErr:     true,
			errContains: "LLM_PROVIDER",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Isolate: clear all vars this loader reads, then apply test values.
			for _, k := range []string{
				"DATABASE_URL", "PORT", "LOG_LEVEL",
				"LLM_PROVIDER", "ANTHROPIC_API_KEY", "LLM_MODEL_ID",
				"DEEPSEEK_API_KEY", "DEEPSEEK_MODEL_ID", "DEEPSEEK_BASE_URL",
				"LLM_BATCH_ENABLED", "GCP_PROJECT_ID",
				"GCS_RAW_BUCKET", "PUBSUB_SCRAPE_TOPIC_ID", "PUBSUB_EXTRACT_TOPIC_ID",
				"CLOUD_SQL_INSTANCE", "DB_IAM_USER", "DB_NAME",
				"WORKER_SERVICE_URL", "PUBSUB_PUSH_SA",
			} {
				t.Setenv(k, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			got, err := Load()

			if tc.wantErr {
				if err == nil {
					t.Fatal("Load() returned nil error; want non-nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("Load() error = %q; want it to contain %q", err.Error(), tc.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}

			if tc.wantPort != "" && got.Port != tc.wantPort {
				t.Errorf("Port = %q; want %q", got.Port, tc.wantPort)
			}
			if tc.wantLevel != "" && got.LogLevel != tc.wantLevel {
				t.Errorf("LogLevel = %q; want %q", got.LogLevel, tc.wantLevel)
			}
		})
	}
}

// TestLoad_LLMBatchEnabled verifies P5-5: the Batch API cost lever defaults to false and
// is only enabled by an explicit, valid LLM_BATCH_ENABLED=true.
func TestLoad_LLMBatchEnabled(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{name: "unset defaults to false", env: "", want: false},
		{name: "true enables it", env: "true", want: true},
		{name: "invalid value falls back to false", env: "not-a-bool", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://localhost/testdb")
			t.Setenv("LLM_BATCH_ENABLED", tc.env)

			got, err := Load()
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if got.LLMBatchEnabled != tc.want {
				t.Errorf("LLMBatchEnabled = %v; want %v", got.LLMBatchEnabled, tc.want)
			}
		})
	}
}
