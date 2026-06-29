// Package scoring provides the Postgres implementation of the scoring Repository
// (domain/scoring.Repository) and the adapters that satisfy the consumer interfaces
// defined in app/scoring. The adapters translate the jobs and profiles application
// services into the narrow JobReader and ProfileReader contracts required by the
// scoring pipeline (ADR-001: cross-context calls through app-service interfaces).
package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/scoring"
)

// Repository persists and reads job_score rows in Postgres. It satisfies
// domain/scoring.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

var _ scoring.Repository = (*Repository)(nil)

// NewRepository constructs a Postgres scoring repository over the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Upsert inserts or updates the fit score for a (job, profile) pair. The operation
// is idempotent: re-running the scorer for the same pair overwrites the prior result.
func (r *Repository) Upsert(ctx context.Context, score scoring.JobScore) error {
	breakdown, err := json.Marshal(score.Breakdown)
	if err != nil {
		return fmt.Errorf("marshalling component breakdown: %w", err)
	}

	const query = `
		INSERT INTO job_score
		    (job_id, profile_id, passes_dealbreakers, weighted_score, component_breakdown, scored_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (job_id, profile_id) DO UPDATE
		    SET passes_dealbreakers = EXCLUDED.passes_dealbreakers,
		        weighted_score      = EXCLUDED.weighted_score,
		        component_breakdown = EXCLUDED.component_breakdown,
		        scored_at           = EXCLUDED.scored_at`

	if _, err := r.pool.Exec(ctx, query,
		string(score.JobID), string(score.ProfileID),
		score.PassesDealbreakers, score.WeightedScore, breakdown, score.ScoredAt,
	); err != nil {
		return fmt.Errorf("upserting job score: %w", err)
	}
	return nil
}

// FindByJobAndProfile returns the stored score for a (job, profile) pair, or a
// kernel.NotFoundError when none exists yet.
func (r *Repository) FindByJobAndProfile(ctx context.Context, jobID kernel.JobID, profileID kernel.ProfileID) (scoring.JobScore, error) {
	const query = `
		SELECT job_id, profile_id, passes_dealbreakers, weighted_score,
		       component_breakdown, scored_at
		FROM job_score
		WHERE job_id = $1 AND profile_id = $2`

	row := r.pool.QueryRow(ctx, query, string(jobID), string(profileID))
	sc, err := scanJobScore(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return scoring.JobScore{}, &kernel.NotFoundError{Kind: "job_score", ID: string(jobID)}
	}
	if err != nil {
		return scoring.JobScore{}, err
	}
	return sc, nil
}

func scanJobScore(row interface{ Scan(dest ...any) error }) (scoring.JobScore, error) {
	var (
		jobID     string
		profileID string
		bdRaw     []byte
		sc        scoring.JobScore
	)
	if err := row.Scan(
		&jobID, &profileID,
		&sc.PassesDealbreakers, &sc.WeightedScore,
		&bdRaw, &sc.ScoredAt,
	); err != nil {
		return scoring.JobScore{}, fmt.Errorf("scanning job score: %w", err)
	}
	sc.JobID = kernel.JobID(jobID)
	sc.ProfileID = kernel.ProfileID(profileID)
	if len(bdRaw) > 0 {
		if err := json.Unmarshal(bdRaw, &sc.Breakdown); err != nil {
			return scoring.JobScore{}, fmt.Errorf("unmarshalling component breakdown: %w", err)
		}
	}
	return sc, nil
}
