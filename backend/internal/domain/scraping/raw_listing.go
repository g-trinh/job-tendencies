// Package scraping is the scraping bounded context. It owns the RawListing captured
// verbatim from a board and the per-(board, profile) high-water-mark that drives
// incremental crawling. Raw payloads are stored in GCS and never translated.
package scraping

import (
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// ExtractionStatus tracks whether a captured raw listing has been extracted into a job.
type ExtractionStatus string

const (
	// ExtractionStatusPending means the raw listing has been captured but not yet extracted.
	ExtractionStatusPending ExtractionStatus = "pending"
	// ExtractionStatusExtracted means a job row was created from this raw listing.
	ExtractionStatusExtracted ExtractionStatus = "extracted"
)

// RawListing is one job listing captured verbatim from a board during a crawl.
// The raw payload lives in GCS at RawRef; ContentHash deduplicates re-captures of the
// same payload across the incremental overlap window.
type RawListing struct {
	// ID is the raw listing's stable identifier (assigned on Save).
	ID kernel.RawListingID
	// BoardID is the board this listing was captured from.
	BoardID kernel.BoardID
	// ProfileID is the profile whose search produced this listing.
	ProfileID kernel.ProfileID
	// Title, Company and Location are identity facts captured verbatim from the search
	// card (never LLM-inferred, never translated); they feed the job's dedup fingerprint.
	Title    string
	Company  string
	Location string
	// SourceURL is the listing's URL on the board.
	SourceURL string
	// RawRef is the GCS object path of the verbatim payload.
	RawRef string
	// PostedAt is the listing's publication time, used as the incremental cursor.
	PostedAt time.Time
	// ContentHash is the SHA-256 of the raw payload; idempotency/dedup key within a board.
	ContentHash string
	// ExtractionStatus tracks extraction progress.
	ExtractionStatus ExtractionStatus
}
