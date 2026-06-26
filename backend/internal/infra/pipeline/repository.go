// Package pipeline provides the Postgres implementation of the pipeline run repository.
package pipeline

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	apppipeline "github.com/g-trinh/job-tendencies/internal/app/pipeline"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository records scrape runs in Postgres. It satisfies app/pipeline.RunRepository.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Postgres pipeline run repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateRun inserts a scrape run and returns its generated id.
func (r *Repository) CreateRun(ctx context.Context, profileID kernel.ProfileID, trigger string) (kernel.ScrapeRunID, error) {
	const query = `
		INSERT INTO scrape_run (profile_id, trigger, status)
		VALUES ($1, $2, 'queued')
		RETURNING id`
	var id string
	if err := r.pool.QueryRow(ctx, query, string(profileID), trigger).Scan(&id); err != nil {
		return "", fmt.Errorf("inserting scrape run: %w", err)
	}
	return kernel.ScrapeRunID(id), nil
}

// Ensure the repository satisfies the app-layer port at compile time.
var _ apppipeline.RunRepository = (*Repository)(nil)
