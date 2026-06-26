package llm_test

import (
	"encoding/json"
	"testing"

	infrallm "github.com/g-trinh/job-tendencies/internal/infra/llm"
)

// AC: smoke Extract against a fixture returns per-field confidence + understanding.
// AC: model id read from config (defaults to claude-opus-4-8).
//
// NOTE: Tests against the live Claude API require a valid ANTHROPIC_API_KEY and
// are not run in CI without that credential. The unit tests below verify the
// internal parsing logic without network calls.

func TestParseExtractedListing_WithFixture(t *testing.T) {
	t.Parallel()

	// fixture is a synthetic Claude tool_use response matching the extraction schema.
	fixture := map[string]interface{}{
		"skills":        map[string]interface{}{"value": []string{"Go", "PostgreSQL", "Docker"}, "confidence": 90},
		"remote_policy": map[string]interface{}{"value": "hybrid", "confidence": 85},
		"office_days":   map[string]interface{}{"value": 2, "confidence": 80},
		"contract_type": map[string]interface{}{"value": "cdi", "confidence": 95},
		"working_days":  map[string]interface{}{"value": "full_time", "confidence": 88},
		"salary_min":    map[string]interface{}{"value": 55000, "confidence": 70},
		"salary_max":    map[string]interface{}{"value": 70000, "confidence": 70},
		"seniority":     map[string]interface{}{"value": "senior", "confidence": 82},
		"recruiter": map[string]interface{}{
			"value": map[string]interface{}{
				"name":  "Alice Dupont",
				"email": "alice@company.fr",
			},
			"confidence": 60,
		},
		"understanding": 88,
	}

	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshalling fixture: %v", err)
	}

	listing, err := infrallm.ParseExtractedListingForTest(raw)
	if err != nil {
		t.Fatalf("ParseExtractedListing() unexpected error: %v", err)
	}

	if listing.Understanding.Int() != 88 {
		t.Errorf("Understanding = %d; want 88", listing.Understanding.Int())
	}
	if listing.Skills.Confidence.Int() != 90 {
		t.Errorf("Skills.Confidence = %d; want 90", listing.Skills.Confidence.Int())
	}
	if len(listing.Skills.Value) != 3 {
		t.Errorf("Skills.Value len = %d; want 3", len(listing.Skills.Value))
	}
	if listing.SalaryMin.Value == nil {
		t.Fatal("SalaryMin.Value is nil; want non-nil")
	}
	if *listing.SalaryMin.Value != 55000 {
		t.Errorf("SalaryMin.Value = %d; want 55000", *listing.SalaryMin.Value)
	}
	if listing.Recruiter.Value == nil {
		t.Fatal("Recruiter.Value is nil; want non-nil")
	}
	if listing.Recruiter.Value.Name != "Alice Dupont" {
		t.Errorf("Recruiter.Name = %q; want %q", listing.Recruiter.Value.Name, "Alice Dupont")
	}
}

func TestParseExtractedListing_NullSalaryAndRecruiter(t *testing.T) {
	t.Parallel()

	// AC: salary absent → field null, confidence 0.
	fixture := map[string]interface{}{
		"skills":        map[string]interface{}{"value": []string{}, "confidence": 50},
		"remote_policy": map[string]interface{}{"value": "on_site", "confidence": 70},
		"office_days":   map[string]interface{}{"value": 5, "confidence": 70},
		"contract_type": map[string]interface{}{"value": "cdd", "confidence": 65},
		"working_days":  map[string]interface{}{"value": "full_time", "confidence": 60},
		"salary_min":    map[string]interface{}{"value": nil, "confidence": 0},
		"salary_max":    map[string]interface{}{"value": nil, "confidence": 0},
		"seniority":     map[string]interface{}{"value": "mid", "confidence": 55},
		"recruiter":     map[string]interface{}{"value": nil, "confidence": 0},
		"understanding": 45,
	}

	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshalling fixture: %v", err)
	}

	listing, err := infrallm.ParseExtractedListingForTest(raw)
	if err != nil {
		t.Fatalf("ParseExtractedListing() unexpected error: %v", err)
	}

	if listing.SalaryMin.Value != nil {
		t.Errorf("SalaryMin.Value = %v; want nil", listing.SalaryMin.Value)
	}
	if listing.SalaryMin.Confidence.Int() != 0 {
		t.Errorf("SalaryMin.Confidence = %d; want 0", listing.SalaryMin.Confidence.Int())
	}
	if listing.Recruiter.Value != nil {
		t.Errorf("Recruiter.Value = %v; want nil", listing.Recruiter.Value)
	}
	if listing.Understanding.Int() != 45 {
		t.Errorf("Understanding = %d; want 45", listing.Understanding.Int())
	}
}

func TestDefaultModelID(t *testing.T) {
	t.Parallel()

	// AC: model id defaults to claude-opus-4-8 when empty.
	if infrallm.DefaultModelID != "claude-opus-4-8" {
		t.Errorf("DefaultModelID = %q; want %q", infrallm.DefaultModelID, "claude-opus-4-8")
	}
}
