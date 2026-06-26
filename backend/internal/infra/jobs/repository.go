// Package jobs provides the Postgres implementation of the job-browser repository and
// the extraction context's JobWriter. A Job and its JobSource rows are written in one
// transaction; reads are scoped to a profile via the job_source → raw_listing link.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appextraction "github.com/g-trinh/job-tendencies/internal/app/extraction"
	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository persists and reads jobs in Postgres. It satisfies app/jobs.Repository
// and app/extraction.JobWriter.
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

// ListByProfile returns every job with a source listing captured for the profile.
func (r *Repository) ListByProfile(ctx context.Context, profileID kernel.ProfileID) ([]jobs.Job, error) {
	const query = `
		SELECT DISTINCT j.id, j.title, j.company, j.location, j.url, j.skills,
		       j.remote_policy, j.office_days, j.contract_type,
		       j.working_days, j.salary_min, j.salary_max, j.seniority,
		       j.field_confidence, j.understanding_score, j.first_seen, j.last_seen
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

	var out []jobs.Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating job rows: %w", err)
	}
	return out, nil
}

// GetByProfile returns one job scoped to the profile, with its sources loaded.
func (r *Repository) GetByProfile(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (jobs.Job, error) {
	const query = `
		SELECT j.id, j.title, j.company, j.location, j.url, j.skills,
		       j.remote_policy, j.office_days, j.contract_type,
		       j.working_days, j.salary_min, j.salary_max, j.seniority,
		       j.field_confidence, j.understanding_score, j.first_seen, j.last_seen
		FROM job j
		WHERE j.id = $1
		  AND EXISTS (
		    SELECT 1 FROM job_source js
		    JOIN raw_listing rl ON rl.id = js.raw_listing_id
		    WHERE js.job_id = j.id AND rl.profile_id = $2)`

	row := r.pool.QueryRow(ctx, query, string(id), string(profileID))
	job, err := scanJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return jobs.Job{}, &kernel.NotFoundError{Kind: "job", ID: string(id)}
	}
	if err != nil {
		return jobs.Job{}, err
	}

	sources, err := r.sourcesByJob(ctx, id)
	if err != nil {
		return jobs.Job{}, err
	}
	job.Sources = sources
	return job, nil
}

func (r *Repository) sourcesByJob(ctx context.Context, id kernel.JobID) ([]jobs.JobSource, error) {
	const query = `SELECT board_id, raw_listing_id, source_url FROM job_source WHERE job_id = $1`
	rows, err := r.pool.Query(ctx, query, string(id))
	if err != nil {
		return nil, fmt.Errorf("querying job sources: %w", err)
	}
	defer rows.Close()

	var sources []jobs.JobSource
	for rows.Next() {
		var boardID, rawListingID, sourceURL string
		if err := rows.Scan(&boardID, &rawListingID, &sourceURL); err != nil {
			return nil, fmt.Errorf("scanning job source: %w", err)
		}
		sources = append(sources, jobs.JobSource{
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

// scanJob scans a job row into a domain Job (without sources).
func scanJob(row rowScanner) (jobs.Job, error) {
	var (
		id            string
		remotePolicy  string
		contractType  string
		workingDays   string
		seniority     string
		confidence    []byte
		understanding int
		job           jobs.Job
	)
	if err := row.Scan(&id, &job.Title, &job.Company, &job.Location, &job.URL,
		&job.Skills, &remotePolicy, &job.OfficeDays, &contractType,
		&workingDays, &job.SalaryMin, &job.SalaryMax, &seniority,
		&confidence, &understanding, &job.FirstSeen, &job.LastSeen); err != nil {
		return jobs.Job{}, fmt.Errorf("scanning job row: %w", err)
	}

	job.ID = kernel.JobID(id)
	job.RemotePolicy = kernel.RemotePolicy(remotePolicy)
	job.ContractType = kernel.ContractType(contractType)
	job.WorkingDays = kernel.WorkingDays(workingDays)
	job.Seniority = kernel.Seniority(seniority)
	if u, err := kernel.NewUnderstanding(uint8(min(understanding, 100))); err == nil {
		job.UnderstandingScore = u
	}
	if len(confidence) > 0 {
		if err := json.Unmarshal(confidence, &job.FieldConfidence); err != nil {
			return jobs.Job{}, fmt.Errorf("unmarshalling field confidence: %w", err)
		}
	}
	return job, nil
}

// Ensure the repository satisfies the app-layer ports at compile time.
var (
	_ appjobs.Repository      = (*Repository)(nil)
	_ appextraction.JobWriter = (*Repository)(nil)
)
