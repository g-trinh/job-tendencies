package extraction

import (
	"testing"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
)

func TestBuildJob(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	salary := int64(55000)
	recruiter := &llm.Recruiter{Name: "Alice", Email: "alice@acme.io"}

	extracted := &llm.ExtractedListing{
		Skills:        llm.ExtractedField[[]string]{Value: []string{"go", "sql"}, Confidence: 90},
		RemotePolicy:  llm.ExtractedField[kernel.RemotePolicy]{Value: kernel.RemotePolicyHybrid, Confidence: 80},
		SalaryMin:     llm.ExtractedField[*int64]{Value: &salary, Confidence: 70},
		Seniority:     llm.ExtractedField[kernel.Seniority]{Value: kernel.SenioritySenior, Confidence: 60},
		Recruiter:     llm.ExtractedField[*llm.Recruiter]{Value: recruiter, Confidence: 75},
		Understanding: 88,
	}
	ref := RawListingRef{
		ID:        "raw-1",
		BoardID:   "board-1",
		Title:     "Go Engineer",
		Company:   "Acme",
		Location:  "Paris",
		SourceURL: "https://wttj/jobs/1",
	}

	job := buildJob(extracted, ref, now)

	if job.Title != "Go Engineer" || job.Company != "Acme" || job.Location != "Paris" {
		t.Fatalf("identity fields not captured verbatim: %+v", job)
	}
	if job.URL != ref.SourceURL {
		t.Fatalf("job.URL = %q, want %q (canonical source url)", job.URL, ref.SourceURL)
	}
	if job.UnderstandingScore.Int() != 88 {
		t.Fatalf("understanding = %d, want 88", job.UnderstandingScore.Int())
	}
	if job.FieldConfidence["skills"] != 90 || job.FieldConfidence["remote_policy"] != 80 {
		t.Fatalf("field confidence not flattened: %+v", job.FieldConfidence)
	}
	if job.FieldConfidence["recruiter"] != 75 {
		t.Fatalf("recruiter confidence not in field_confidence map: got %d, want 75", job.FieldConfidence["recruiter"])
	}
	if len(job.Sources) != 1 || job.Sources[0].RawListingID != "raw-1" || job.Sources[0].BoardID != "board-1" {
		t.Fatalf("source linkage wrong: %+v", job.Sources)
	}
	if !job.FirstSeen.Equal(now) || !job.LastSeen.Equal(now) {
		t.Fatalf("timestamps not set to now")
	}
}
