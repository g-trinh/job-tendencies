// Package jobs is the job-browser bounded context. It owns the Job aggregate — a
// structured, (eventually) deduplicated listing — together with the JobSource rows
// that record which raw listings it was extracted from ("found on: WTTJ, Indeed").
// Phase 2 creates one Job per raw listing with no dedup/merge/scoring.
package jobs

import (
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// JobSource links a Job to the raw listing (and board) it was extracted from.
type JobSource struct {
	// BoardID is the board the source listing came from.
	BoardID kernel.BoardID
	// RawListingID is the captured raw listing this job was extracted from.
	RawListingID kernel.RawListingID
	// SourceURL is the listing's URL on the board.
	SourceURL string
}

// Job is a structured job listing (aggregate root). Its structured fields are produced
// by LLM extraction; FieldConfidence holds the per-field extraction confidence (0–100)
// keyed by field name, and UnderstandingScore the overall parse quality (0–100).
type Job struct {
	// ID is the job's stable identifier (assigned on Create).
	ID kernel.JobID
	// Title, Company, Location and URL are identity facts captured verbatim from the
	// board search card (not LLM-extracted, never translated). They are the canonical
	// display fields and the deterministic inputs to the dedup fingerprint.
	Title    string
	Company  string
	Location string
	URL      string
	// Skills are the technologies/skills required by the listing.
	Skills []string
	// RemotePolicy is the advertised remote-work policy.
	RemotePolicy kernel.RemotePolicy
	// OfficeDays is the number of required on-site days per week.
	OfficeDays int
	// ContractType is the employment contract category.
	ContractType kernel.ContractType
	// WorkingDays is the weekly schedule.
	WorkingDays kernel.WorkingDays
	// SalaryMin and SalaryMax are whole euros; nil when the salary was not published.
	SalaryMin *int64
	SalaryMax *int64
	// Seniority is the experience level expected.
	Seniority kernel.Seniority
	// FieldConfidence is the per-field extraction confidence (0–100) keyed by field name.
	FieldConfidence map[string]int
	// UnderstandingScore is the overall parse-quality score (0–100).
	UnderstandingScore kernel.Understanding
	// FirstSeen and LastSeen bound when this job was observed.
	FirstSeen time.Time
	LastSeen  time.Time
	// Sources are the raw listings this job was extracted from (one in Phase 2).
	Sources []JobSource
}
