package scraping

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/scraping"
)

// RawListingRepository persists captured raw listings in Postgres.
// It satisfies domain/scraping.RawListingRepository and domain/scraping.RawListingSource.
type RawListingRepository struct {
	pool *pgxpool.Pool
}

// NewRawListingRepository constructs a Postgres raw-listing repository.
func NewRawListingRepository(pool *pgxpool.Pool) *RawListingRepository {
	return &RawListingRepository{pool: pool}
}

// ExistsByContentHash reports whether a raw listing with this hash exists for the board.
func (r *RawListingRepository) ExistsByContentHash(ctx context.Context, boardID kernel.BoardID, contentHash string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM raw_listing WHERE board_id = $1 AND content_hash = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, string(boardID), contentHash).Scan(&exists); err != nil {
		return false, fmt.Errorf("checking raw listing content hash: %w", err)
	}
	return exists, nil
}

// Save inserts a raw listing and returns its generated id.
func (r *RawListingRepository) Save(ctx context.Context, listing scraping.RawListing) (kernel.RawListingID, error) {
	const query = `
		INSERT INTO raw_listing
			(board_id, profile_id, title, company, location, source_url, raw_ref,
			 posted_at, content_hash, extraction_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var postedAt *time.Time
	if !listing.PostedAt.IsZero() {
		postedAt = &listing.PostedAt
	}

	var id string
	err := r.pool.QueryRow(ctx, query,
		string(listing.BoardID), string(listing.ProfileID), listing.Title, listing.Company,
		listing.Location, listing.SourceURL, listing.RawRef, postedAt, listing.ContentHash,
		string(listing.ExtractionStatus),
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("inserting raw listing: %w", err)
	}
	return kernel.RawListingID(id), nil
}

// Get returns the captured raw listing aggregate, or a kernel.NotFoundError.
func (r *RawListingRepository) Get(ctx context.Context, id kernel.RawListingID) (scraping.RawListing, error) {
	const query = `
		SELECT id, board_id, profile_id, title, company, location, source_url, raw_ref
		FROM raw_listing WHERE id = $1`
	var listing scraping.RawListing
	var rawID, boardID, profileID string
	err := r.pool.QueryRow(ctx, query, string(id)).
		Scan(&rawID, &boardID, &profileID, &listing.Title, &listing.Company, &listing.Location,
			&listing.SourceURL, &listing.RawRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return scraping.RawListing{}, &kernel.NotFoundError{Kind: "raw_listing", ID: string(id)}
	}
	if err != nil {
		return scraping.RawListing{}, fmt.Errorf("querying raw listing %q: %w", id, err)
	}
	listing.ID = kernel.RawListingID(rawID)
	listing.BoardID = kernel.BoardID(boardID)
	listing.ProfileID = kernel.ProfileID(profileID)
	return listing, nil
}

// MarkExtracted flips a raw listing's extraction_status to extracted.
func (r *RawListingRepository) MarkExtracted(ctx context.Context, id kernel.RawListingID) error {
	const query = `UPDATE raw_listing SET extraction_status = 'extracted' WHERE id = $1`
	if _, err := r.pool.Exec(ctx, query, string(id)); err != nil {
		return fmt.Errorf("marking raw listing %q extracted: %w", id, err)
	}
	return nil
}

// HighWaterMarkRepository reads and advances the per-(board, profile) incremental cursor.
// It satisfies domain/scraping.HighWaterMarkRepository.
type HighWaterMarkRepository struct {
	pool *pgxpool.Pool
}

// NewHighWaterMarkRepository constructs a Postgres high-water-mark repository.
func NewHighWaterMarkRepository(pool *pgxpool.Pool) *HighWaterMarkRepository {
	return &HighWaterMarkRepository{pool: pool}
}

// Get returns the cursor for (board, profile), or nil on the first-ever crawl.
func (r *HighWaterMarkRepository) Get(ctx context.Context, boardID kernel.BoardID, profileID kernel.ProfileID) (*time.Time, error) {
	const query = `
		SELECT cursor_posted_at FROM scrape_high_water_mark
		WHERE board_id = $1 AND profile_id = $2`
	var cursor time.Time
	err := r.pool.QueryRow(ctx, query, string(boardID), string(profileID)).Scan(&cursor)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading high-water-mark: %w", err)
	}
	return &cursor, nil
}

// Set upserts the cursor for (board, profile).
func (r *HighWaterMarkRepository) Set(ctx context.Context, boardID kernel.BoardID, profileID kernel.ProfileID, cursorPostedAt time.Time) error {
	const query = `
		INSERT INTO scrape_high_water_mark (board_id, profile_id, cursor_posted_at, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (board_id, profile_id)
		DO UPDATE SET cursor_posted_at = EXCLUDED.cursor_posted_at, updated_at = now()`
	if _, err := r.pool.Exec(ctx, query, string(boardID), string(profileID), cursorPostedAt); err != nil {
		return fmt.Errorf("upserting high-water-mark: %w", err)
	}
	return nil
}

// Ensure the repositories satisfy the domain-layer ports at compile time.
var (
	_ scraping.RawListingRepository    = (*RawListingRepository)(nil)
	_ scraping.RawListingSource        = (*RawListingRepository)(nil)
	_ scraping.HighWaterMarkRepository = (*HighWaterMarkRepository)(nil)
)
