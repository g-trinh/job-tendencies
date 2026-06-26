//go:build integration

package llm_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	infra "github.com/g-trinh/job-tendencies/internal/infra/llm"
)

const sampleListing = `
Software Engineer - Backend (CDI)
Company: Acme Corp, Paris 9e

We are looking for a senior backend engineer to join our platform team.

Requirements:
- 5+ years of Go experience
- Proficiency in PostgreSQL, Redis
- Experience with GCP or AWS
- REST and gRPC API design

Contract: CDI (permanent contract)
Working: Full time, hybrid (3 days on-site, 2 remote)
Salary: 65 000 - 80 000 EUR / year
Start: ASAP

Recruiter: Marie Dupont, marie.dupont@acmecorp.fr
`

func TestLiveExtract(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := infra.New(apiKey, "", logger)

	result, err := client.Extract(context.Background(), sampleListing)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("ExtractedListing:\n%s", out)

	if len(result.Skills.Value) == 0 {
		t.Error("expected at least one skill extracted")
	}
	if result.Understanding.Int() < 50 {
		t.Errorf("understanding too low: %d", result.Understanding.Int())
	}
	if string(result.ContractType.Value) != "cdi" {
		t.Errorf("expected cdi contract type, got %q", result.ContractType.Value)
	}
	if string(result.RemotePolicy.Value) != "hybrid" {
		t.Errorf("expected hybrid remote policy, got %q", result.RemotePolicy.Value)
	}
	if result.SalaryMin.Value == nil || *result.SalaryMin.Value < 60000 {
		t.Errorf("salary_min expected >= 60000, got %v", result.SalaryMin.Value)
	}
	if result.Recruiter.Value == nil {
		t.Error("expected recruiter to be extracted")
	} else {
		t.Logf("recruiter: name=%q email=%q", result.Recruiter.Value.Name, result.Recruiter.Value.Email)
	}
}

func TestLiveGenerateAdapter(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := infra.New(apiKey, "", logger)

	exampleResponse := `{
  "jobs": [
    {"id": "123", "title": "Backend Engineer", "url": "/jobs/123", "posted_at": "2025-06-01"},
    {"id": "124", "title": "Frontend Engineer", "url": "/jobs/124", "posted_at": "2025-06-02"}
  ],
  "total": 2,
  "page": 1
}`

	spec, err := client.GenerateAdapter(
		context.Background(),
		"https://jobs.example.com/api/jobs",
		exampleResponse,
	)
	if err != nil {
		t.Fatalf("GenerateAdapter failed: %v", err)
	}

	out, _ := json.MarshalIndent(spec, "", "  ")
	t.Logf("AdapterSpec:\n%s", out)

	if spec.Board == "" {
		t.Error("expected non-empty board field")
	}
	if spec.FetchMode == "" {
		t.Error("expected non-empty fetch_mode")
	}
	if spec.Search.URLTemplate == "" {
		t.Error("expected non-empty search url_template")
	}
}
