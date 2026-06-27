// Package profiles provides the Postgres implementation of the profiles repository.
// search_keywords is stored as a Postgres text[] column. The exactly-one-active
// invariant is enforced by the profile_single_active partial unique index in the
// schema; Activate switches the active profile in a single transaction.
package profiles

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
)

// Repository reads and writes profiles in Postgres. It satisfies domain/profiles.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

var _ profiles.Repository = (*Repository)(nil)

// NewRepository constructs a Postgres profile repository over the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ActiveProfile returns the single profile with is_active = true.
func (r *Repository) ActiveProfile(ctx context.Context) (profiles.Profile, error) {
	const query = `
		SELECT id, name, search_keywords, location, is_active
		FROM profile WHERE is_active = true LIMIT 1`
	return r.queryOne(ctx, query)
}

// ProfileByID returns one profile by id.
func (r *Repository) ProfileByID(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error) {
	const query = `
		SELECT id, name, search_keywords, location, is_active
		FROM profile WHERE id = $1`
	p, err := r.queryOne(ctx, query, string(id))
	if errors.Is(err, kernel.ErrNotFound) {
		return profiles.Profile{}, &kernel.NotFoundError{Kind: "profile", ID: string(id)}
	}
	return p, err
}

// List returns all profiles ordered by name.
func (r *Repository) List(ctx context.Context) ([]profiles.Profile, error) {
	const query = `
		SELECT id, name, search_keywords, location, is_active
		FROM profile ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying profiles: %w", err)
	}
	defer rows.Close()

	var out []profiles.Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating profile rows: %w", err)
	}
	return out, nil
}

// Create inserts a new profile and returns the assigned id. The profile's IsActive
// field is not used here; use Activate to switch the active profile.
func (r *Repository) Create(ctx context.Context, p profiles.Profile) (kernel.ProfileID, error) {
	const query = `
		INSERT INTO profile (name, search_keywords, location, is_active)
		VALUES ($1, $2, $3, false)
		RETURNING id`

	var id string
	err := r.pool.QueryRow(ctx, query, p.Name, p.SearchKeywords, p.Location).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("inserting profile: %w", err)
	}
	return kernel.ProfileID(id), nil
}

// Update persists name, search_keywords, and location changes for the profile.
func (r *Repository) Update(ctx context.Context, p profiles.Profile) error {
	const query = `
		UPDATE profile SET name = $1, search_keywords = $2, location = $3
		WHERE id = $4`

	tag, err := r.pool.Exec(ctx, query, p.Name, p.SearchKeywords, p.Location, string(p.ID))
	if err != nil {
		return fmt.Errorf("updating profile %q: %w", p.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "profile", ID: string(p.ID)}
	}
	return nil
}

// Delete removes a profile by id.
func (r *Repository) Delete(ctx context.Context, id kernel.ProfileID) error {
	const query = `DELETE FROM profile WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("deleting profile %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "profile", ID: string(id)}
	}
	return nil
}

// Activate switches the active profile in a single transaction: all profiles are
// deactivated first, then the target is activated. The DB partial unique index
// (profile_single_active) enforces that at most one row has is_active = true.
func (r *Repository) Activate(ctx context.Context, id kernel.ProfileID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `UPDATE profile SET is_active = false WHERE is_active = true`); err != nil {
		return fmt.Errorf("deactivating profiles: %w", err)
	}

	tag, err := tx.Exec(ctx, `UPDATE profile SET is_active = true WHERE id = $1`, string(id))
	if err != nil {
		return fmt.Errorf("activating profile %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "profile", ID: string(id)}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing activation: %w", err)
	}
	return nil
}

func (r *Repository) queryOne(ctx context.Context, query string, args ...any) (profiles.Profile, error) {
	var p profiles.Profile
	err := r.pool.QueryRow(ctx, query, args...).
		Scan(&p.ID, &p.Name, &p.SearchKeywords, &p.Location, &p.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return profiles.Profile{}, &kernel.NotFoundError{Kind: "profile", ID: "active"}
	}
	if err != nil {
		return profiles.Profile{}, fmt.Errorf("querying profile: %w", err)
	}
	return p, nil
}

// scanProfile scans a profile row from pgx.Rows.
func scanProfile(rows pgx.Rows) (profiles.Profile, error) {
	var p profiles.Profile
	if err := rows.Scan(&p.ID, &p.Name, &p.SearchKeywords, &p.Location, &p.IsActive); err != nil {
		return profiles.Profile{}, fmt.Errorf("scanning profile row: %w", err)
	}
	return p, nil
}
