package llm_test

import (
	"encoding/json"
	"testing"

	infrallm "github.com/g-trinh/job-tendencies/internal/infra/llm"
)

// AC: parseExtractedIdentity maps skills, experience, and seniority from Claude tool output.

func TestParseExtractedIdentity_WithFixture(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		fixture       map[string]interface{}
		wantSkillsLen int
		wantSeniority string
		wantExpEmpty  bool
	}{
		{
			name: "maps skills experience and seniority from valid tool output",
			fixture: map[string]interface{}{
				"skills":     []string{"Go", "PostgreSQL", "Kubernetes"},
				"experience": "Software Engineer at Acme Corp (2019–2024)",
				"seniority":  "senior",
			},
			wantSkillsLen: 3,
			wantSeniority: "senior",
			wantExpEmpty:  false,
		},
		{
			name: "returns empty skills slice when skills array is empty",
			fixture: map[string]interface{}{
				"skills":     []string{},
				"experience": "Junior Developer at Startup (2023–2024)",
				"seniority":  "entry",
			},
			wantSkillsLen: 0,
			wantSeniority: "entry",
			wantExpEmpty:  false,
		},
		{
			name: "returns empty experience when experience field is empty string",
			fixture: map[string]interface{}{
				"skills":     []string{"Python"},
				"experience": "",
				"seniority":  "mid",
			},
			wantSkillsLen: 1,
			wantSeniority: "mid",
			wantExpEmpty:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(tc.fixture)
			if err != nil {
				t.Fatalf("marshalling fixture: %v", err)
			}

			got, err := infrallm.ParseExtractedIdentityForTest(raw)
			if err != nil {
				t.Fatalf("ParseExtractedIdentity() unexpected error: %v", err)
			}

			if len(got.Skills) != tc.wantSkillsLen {
				t.Errorf("Skills len = %d; want %d", len(got.Skills), tc.wantSkillsLen)
			}
			if string(got.Seniority) != tc.wantSeniority {
				t.Errorf("Seniority = %q; want %q", got.Seniority, tc.wantSeniority)
			}
			if tc.wantExpEmpty && got.RawExperience != "" {
				t.Errorf("RawExperience = %q; want empty", got.RawExperience)
			}
			if !tc.wantExpEmpty && got.RawExperience == "" {
				t.Error("RawExperience is empty; want non-empty")
			}
		})
	}
}

func TestParseExtractedIdentity_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := infrallm.ParseExtractedIdentityForTest(json.RawMessage(`{bad`))
	if err == nil {
		t.Error("expected error for invalid JSON; got nil")
	}
}
