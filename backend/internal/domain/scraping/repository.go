package scraping

import (
	"context"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// RawListingRepository is the RawListing aggregate's capture-side persistence port: it
// stores newly captured listings and answers content-hash dedup during a crawl. Per
// ADR-005 the aggregate repository interface lives in the domain.
type RawListingRepository interface {
	// ExistsByContentHash reports whether a raw listing with this hash already exists
	// for the board.
	ExistsByContentHash(ctx context.Context, boardID kernel.BoardID, contentHash string) (bool, error)
	// Save stores a raw listing and returns its assigned id.
	Save(ctx context.Context, listing RawListing) (kernel.RawListingID, error)
}

// RawListingSource is the RawListing aggregate's read/lifecycle port used downstream of
// capture: the extraction stage loads a captured listing and marks it extracted once a
// job has been created from it.
type RawListingSource interface {
	// Get returns the captured raw listing, or a kernel.NotFoundError.
	Get(ctx context.Context, id kernel.RawListingID) (RawListing, error)
	// MarkExtracted records that a job has been created from this raw listing.
	MarkExtracted(ctx context.Context, id kernel.RawListingID) error
}

// HighWaterMarkRepository reads and advances the per-(board, profile) incremental
// cursor that drives high-water-mark crawling (pipeline.md §2).
type HighWaterMarkRepository interface {
	// Get returns the most recent posted_at seen on the previous run, or nil on first crawl.
	Get(ctx context.Context, boardID kernel.BoardID, profileID kernel.ProfileID) (*time.Time, error)
	// Set advances the cursor to cursorPostedAt.
	Set(ctx context.Context, boardID kernel.BoardID, profileID kernel.ProfileID, cursorPostedAt time.Time) error
}
