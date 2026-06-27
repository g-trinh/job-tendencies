// Package profiles provides the Postgres implementation of the profiles repository.
// search_keywords is stored as a Postgres text[] column.
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

// Repository reads profiles from Postgres. It satisfies domain/profiles.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

// Ensure the repository satisfies the domain-layer port at compile time.
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
