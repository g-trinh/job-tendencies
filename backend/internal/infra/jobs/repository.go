// Package jobs provides the Postgres implementation of the Job aggregate's write
// repository (domain/jobs.Repository) and the job-browser read query
// (app/jobs.JobQuery). A Job and its JobSource rows are written in one transaction;
// reads are scoped to a profile via the job_source → raw_listing link and projected
// into read DTOs (ADR-005).
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository persists and reads jobs in Postgres. It satisfies domain/jobs.Repository
// (write side) and app/jobs.JobQuery (read side).
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Postgres job repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a job and its source linkage in a single transaction.
func (r *Repository) Create(ctx context.Context, job jobs.Job) (kernel.JobID, error) {
	confidence, err := json.Marshal(job.FieldConfidence)
	if err != nil {
		return "", fmt.Errorf("marshalling field confidence: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertJob = `
		INSERT INTO job
			(title, company, location, url, skills, remote_policy, office_days,
			 contract_type, working_days, salary_min, salary_max, seniority,
			 field_confidence, understanding_score, first_seen, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id`

	var jobID string
	err = tx.QueryRow(ctx, insertJob,
		job.Title, job.Company, job.Location, job.URL,
		job.Skills, string(job.RemotePolicy), job.OfficeDays, string(job.ContractType),
		string(job.WorkingDays), job.SalaryMin, job.SalaryMax, string(job.Seniority),
		confidence, job.UnderstandingScore.Int(), job.FirstSeen, job.LastSeen,
	).Scan(&jobID)
	if err != nil {
		return "", fmt.Errorf("inserting job: %w", err)
	}

	const insertSource = `
		INSERT INTO job_source (job_id, raw_listing_id, board_id, source_url)
		VALUES ($1, $2, $3, $4)`
	for _, src := range job.Sources {
		if _, err := tx.Exec(ctx, insertSource,
			jobID, string(src.RawListingID), string(src.BoardID), src.SourceURL); err != nil {
			return "", fmt.Errorf("inserting job source: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("committing job transaction: %w", err)
	}
	return kernel.JobID(jobID), nil
}

// ListByProfile returns every job view with a source listing captured for the profile.
func (r *Repository) ListByProfile(ctx context.Context, profileID kernel.ProfileID) ([]appjobs.JobView, error) {
	const query = `
		SELECT DISTINCT j.id, j.title, j.company, j.location, j.url, j.skills,
		       j.remote_policy, j.office_days, j.contract_type,
		       j.working_days, j.salary_min, j.salary_max, j.seniority,
		       j.field_confidence, j.understanding_score, j.first_seen
		FROM job j
		JOIN job_source js ON js.job_id = j.id
		JOIN raw_listing rl ON rl.id = js.raw_listing_id
		WHERE rl.profile_id = $1
		ORDER BY j.first_seen DESC`

	rows, err := r.pool.Query(ctx, query, string(profileID))
	if err != nil {
		return nil, fmt.Errorf("querying jobs: %w", err)
	}
	defer rows.Close()

	var out []appjobs.JobView
	for rows.Next() {
		view, err := scanJobView(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating job rows: %w", err)
	}
	return out, nil
}

// GetByProfile returns one job view scoped to the profile, with its sources loaded.
func (r *Repository) GetByProfile(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (appjobs.JobView, error) {
	const query = `
		SELECT j.id, j.title, j.company, j.location, j.url, j.skills,
		       j.remote_policy, j.office_days, j.contract_type,
		       j.working_days, j.salary_min, j.salary_max, j.seniority,
		       j.field_confidence, j.understanding_score, j.first_seen
		FROM job j
		WHERE j.id = $1
		  AND EXISTS (
		    SELECT 1 FROM job_source js
		    JOIN raw_listing rl ON rl.id = js.raw_listing_id
		    WHERE js.job_id = j.id AND rl.profile_id = $2)`

	row := r.pool.QueryRow(ctx, query, string(id), string(profileID))
	view, err := scanJobView(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return appjobs.JobView{}, &kernel.NotFoundError{Kind: "job", ID: string(id)}
	}
	if err != nil {
		return appjobs.JobView{}, err
	}

	sources, err := r.sourcesByJob(ctx, id)
	if err != nil {
		return appjobs.JobView{}, err
	}
	view.Sources = sources
	return view, nil
}

func (r *Repository) sourcesByJob(ctx context.Context, id kernel.JobID) ([]appjobs.JobSourceView, error) {
	const query = `SELECT board_id, raw_listing_id, source_url FROM job_source WHERE job_id = $1`
	rows, err := r.pool.Query(ctx, query, string(id))
	if err != nil {
		return nil, fmt.Errorf("querying job sources: %w", err)
	}
	defer rows.Close()

	var sources []appjobs.JobSourceView
	for rows.Next() {
		var boardID, rawListingID, sourceURL string
		if err := rows.Scan(&boardID, &rawListingID, &sourceURL); err != nil {
			return nil, fmt.Errorf("scanning job source: %w", err)
		}
		sources = append(sources, appjobs.JobSourceView{
			BoardID:      kernel.BoardID(boardID),
			RawListingID: kernel.RawListingID(rawListingID),
			SourceURL:    sourceURL,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating job source rows: %w", err)
	}
	return sources, nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanJobView scans a job row into a job-browser read DTO (without sources). first_seen
// drives ordering but is not part of the read model, so it is scanned and discarded.
func scanJobView(row rowScanner) (appjobs.JobView, error) {
	var (
		id            string
		remotePolicy  string
		contractType  string
		workingDays   string
		seniority     string
		confidence    []byte
		understanding int
		firstSeen     any
		view          appjobs.JobView
	)
	if err := row.Scan(&id, &view.Title, &view.Company, &view.Location, &view.URL,
		&view.Skills, &remotePolicy, &view.OfficeDays, &contractType,
		&workingDays, &view.SalaryMin, &view.SalaryMax, &seniority,
		&confidence, &understanding, &firstSeen); err != nil {
		return appjobs.JobView{}, fmt.Errorf("scanning job row: %w", err)
	}

	view.ID = kernel.JobID(id)
	view.RemotePolicy = kernel.RemotePolicy(remotePolicy)
	view.ContractType = kernel.ContractType(contractType)
	view.WorkingDays = kernel.WorkingDays(workingDays)
	view.Seniority = kernel.Seniority(seniority)
	if u, err := kernel.NewUnderstanding(uint8(min(understanding, 100))); err == nil {
		view.UnderstandingScore = u
	}
	if len(confidence) > 0 {
		if err := json.Unmarshal(confidence, &view.FieldConfidence); err != nil {
			return appjobs.JobView{}, fmt.Errorf("unmarshalling field confidence: %w", err)
		}
	}
	return view, nil
}

// Ensure the repository satisfies the write and read ports at compile time.
var (
	_ jobs.Repository  = (*Repository)(nil)
	_ appjobs.JobQuery = (*Repository)(nil)
)
