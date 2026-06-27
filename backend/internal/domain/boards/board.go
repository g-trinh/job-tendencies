// Package boards is the board-manager bounded context. It owns the Board source
// and its scraping Adapter (declarative AdapterSpec, draft → approved lifecycle).
// The scraping context consumes approved adapters to crawl each board.
package boards

import (
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// AdapterStatus is the review lifecycle state of an Adapter. Only an approved
// adapter is evaluated by the scraper (ADR-004).
type AdapterStatus string

const (
	// AdapterStatusDraft is a generated-but-unapproved adapter; never crawled.
	AdapterStatusDraft AdapterStatus = "draft"
	// AdapterStatusApproved is a human-reviewed adapter cleared to go live.
	AdapterStatusApproved AdapterStatus = "approved"
)

// Board is a job board source that Job Tendencies can scrape.
type Board struct {
	// ID is the board's stable identifier.
	ID kernel.BoardID
	// Name is the human-readable board name (e.g. "Welcome to the Jungle").
	Name string
	// BaseURL is the board's public base URL.
	BaseURL string
	// Enabled reports whether the board is included in scrape runs.
	Enabled bool
}

// Adapter is a board's declarative scraping configuration. Its Spec is data, never
// executed code; the scraper evaluates it only once Status is approved.
type Adapter struct {
	// ID is the adapter's stable identifier.
	ID kernel.AdapterID
	// BoardID is the board this adapter scrapes.
	BoardID kernel.BoardID
	// Status gates whether the scraper may evaluate this adapter.
	Status AdapterStatus
	// FetchMode mirrors Spec.FetchMode (json_api or html) for indexing convenience.
	FetchMode llm.FetchMode
	// Spec is the declarative crawl configuration.
	Spec llm.AdapterSpec
	// Version increments each time the adapter is regenerated.
	Version int
}

// IsApproved reports whether the adapter is cleared to be evaluated by the scraper.
func (a Adapter) IsApproved() bool { return a.Status == AdapterStatusApproved }
