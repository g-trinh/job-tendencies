// Package jobs provides the Postgres implementation of the Job aggregate's write
// repository (domain/jobs.Repository) and the job-browser read query
// (app/jobs.JobQuery + app/jobs.JobApplicationWriter). A Job and its JobSource rows
// are written in one transaction; reads are scoped to a profile via the job_source →
// raw_listing link and projected into read DTOs (ADR-005). Phase 3 adds filters, sort,
// board names, application status, description, and the job_application kanban table.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository persists and reads jobs in Postgres. It satisfies domain/jobs.Repository
// (write side), app/jobs.JobQuery (read side), and app/jobs.JobApplicationWriter
// (kanban write side).
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
			 field_confidence, understanding_score, first_seen, last_seen,
			 fingerprint, contact_id, expired_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id`

	var contactID *string
	if job.ContactID != nil {
		s := string(*job.ContactID)
		contactID = &s
	}

	var jobID string
	err = tx.QueryRow(ctx, insertJob,
		job.Title, job.Company, job.Location, job.URL,
		job.Skills, string(job.RemotePolicy), job.OfficeDays, string(job.ContractType),
		string(job.WorkingDays), job.SalaryMin, job.SalaryMax, string(job.Seniority),
		confidence, job.UnderstandingScore.Int(), job.FirstSeen, job.LastSeen,
		job.Fingerprint, contactID, job.ExpiredAt,
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

// ListByProfile returns job views scoped to the profile with optional filtering and
// sorting. Sources (with board names) are loaded in a single follow-up query.
func (r *Repository) ListByProfile(ctx context.Context, profileID kernel.ProfileID, filter appjobs.JobListFilter) ([]appjobs.JobView, error) {
	conditions := []string{"rl.profile_id = $1"}
	args := []any{string(profileID)}

	addArg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if len(filter.Skills) > 0 {
		conditions = append(conditions, "j.skills && "+addArg(filter.Skills))
	}
	if filter.RemotePolicy != "" {
		conditions = append(conditions, "j.remote_policy = "+addArg(filter.RemotePolicy))
	}
	if filter.ContractType != "" {
		conditions = append(conditions, "j.contract_type = "+addArg(filter.ContractType))
	}
	if filter.SalaryMin != nil {
		conditions = append(conditions, "(j.salary_max IS NULL OR j.salary_max >= "+addArg(*filter.SalaryMin)+")")
	}
	if filter.SalaryMax != nil {
		conditions = append(conditions, "(j.salary_min IS NULL OR j.salary_min <= "+addArg(*filter.SalaryMax)+")")
	}
	if filter.Location != "" {
		conditions = append(conditions, "j.location ILIKE '%'||"+addArg(filter.Location)+"||'%'")
	}
	if filter.BoardID != "" {
		conditions = append(conditions, "js.board_id = "+addArg(filter.BoardID))
	}
	if filter.Since != nil {
		conditions = append(conditions, "j.first_seen >= "+addArg(*filter.Since))
	}
	if filter.ConfidenceMin != nil {
		conditions = append(conditions, "j.understanding_score >= "+addArg(*filter.ConfidenceMin))
	}

	orderCol := "j.first_seen"
	if filter.Sort == "salary" {
		orderCol = "j.salary_min NULLS LAST"
	}
	orderDir := "DESC"
	if strings.EqualFold(filter.SortDir, "asc") {
		orderDir = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT j.id, j.title, j.company, j.location, j.url, j.skills,
		       j.remote_policy, j.office_days, j.contract_type,
		       j.working_days, j.salary_min, j.salary_max, j.seniority,
		       j.field_confidence, j.understanding_score, j.description,
		       j.contact_id, j.first_seen, j.last_seen, j.expired_at,
		       ja.status
		FROM job j
		JOIN job_source js ON js.job_id = j.id
		JOIN raw_listing rl ON rl.id = js.raw_listing_id
		LEFT JOIN job_application ja ON ja.job_id = j.id AND ja.profile_id = $1
		WHERE %s
		ORDER BY %s %s`,
		strings.Join(conditions, " AND "), orderCol, orderDir)

	rows, err := r.pool.Query(ctx, query, args...)
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

	if len(out) > 0 {
		ids := make([]string, len(out))
		for i, j := range out {
			ids[i] = string(j.ID)
		}
		sourcesMap, err := r.sourcesByJobBulk(ctx, ids)
		if err != nil {
			return nil, err
		}
		for i := range out {
			out[i].Sources = sourcesMap[out[i].ID]
		}
	}
	return out, nil
}

// GetByProfile returns one job view scoped to the profile, with sources and full detail.
func (r *Repository) GetByProfile(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (appjobs.JobView, error) {
	const query = `
		SELECT j.id, j.title, j.company, j.location, j.url, j.skills,
		       j.remote_policy, j.office_days, j.contract_type,
		       j.working_days, j.salary_min, j.salary_max, j.seniority,
		       j.field_confidence, j.understanding_score, j.description,
		       j.contact_id, j.first_seen, j.last_seen, j.expired_at,
		       ja.status
		FROM job j
		LEFT JOIN job_application ja ON ja.job_id = j.id AND ja.profile_id = $2
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

	sources, err := r.sourcesByJobBulk(ctx, []string{string(id)})
	if err != nil {
		return appjobs.JobView{}, err
	}
	view.Sources = sources[id]
	return view, nil
}

// UpsertApplication inserts or updates the kanban status for a (profile, job) pair.
func (r *Repository) UpsertApplication(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID, status kernel.ApplicationStatus) (appjobs.ApplicationResult, error) {
	const query = `
		INSERT INTO job_application (profile_id, job_id, status, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (profile_id, job_id) DO UPDATE
			SET status = EXCLUDED.status, updated_at = now()
		RETURNING status, updated_at`

	var (
		s         string
		updatedAt time.Time
	)
	err := r.pool.QueryRow(ctx, query, string(profileID), string(jobID), string(status)).
		Scan(&s, &updatedAt)
	if err != nil {
		return appjobs.ApplicationResult{}, fmt.Errorf("upserting job application: %w", err)
	}
	return appjobs.ApplicationResult{
		Status:    kernel.ApplicationStatus(s),
		UpdatedAt: updatedAt,
	}, nil
}

// sourcesByJobBulk loads sources (with board names) for all listed job IDs in one
// query, returning a map keyed by JobID.
func (r *Repository) sourcesByJobBulk(ctx context.Context, ids []string) (map[kernel.JobID][]appjobs.JobSourceView, error) {
	const query = `
		SELECT js.job_id, js.board_id, js.source_url, b.name
		FROM job_source js
		JOIN board b ON b.id = js.board_id
		WHERE js.job_id = ANY($1)`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("querying job sources: %w", err)
	}
	defer rows.Close()

	out := make(map[kernel.JobID][]appjobs.JobSourceView)
	for rows.Next() {
		var jobID, boardID, sourceURL, boardName string
		if err := rows.Scan(&jobID, &boardID, &sourceURL, &boardName); err != nil {
			return nil, fmt.Errorf("scanning job source: %w", err)
		}
		jid := kernel.JobID(jobID)
		out[jid] = append(out[jid], appjobs.JobSourceView{
			BoardID:   kernel.BoardID(boardID),
			SourceURL: sourceURL,
			BoardName: boardName,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating job source rows: %w", err)
	}
	return out, nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanJobView scans a full job row into a job-browser read DTO. Column order must match
// the SELECT lists in ListByProfile and GetByProfile.
func scanJobView(row rowScanner) (appjobs.JobView, error) {
	var (
		id            string
		remotePolicy  string
		contractType  string
		workingDays   string
		seniority     string
		confidence    []byte
		understanding int
		contactID     *string
		appStatus     *string
		view          appjobs.JobView
	)
	if err := row.Scan(
		&id, &view.Title, &view.Company, &view.Location, &view.URL,
		&view.Skills, &remotePolicy, &view.OfficeDays, &contractType,
		&workingDays, &view.SalaryMin, &view.SalaryMax, &seniority,
		&confidence, &understanding, &view.Description,
		&contactID, &view.FirstSeen, &view.LastSeen, &view.ExpiredAt,
		&appStatus,
	); err != nil {
		return appjobs.JobView{}, fmt.Errorf("scanning job row: %w", err)
	}

	view.ID = kernel.JobID(id)
	view.RemotePolicy = kernel.RemotePolicy(remotePolicy)
	view.ContractType = kernel.ContractType(contractType)
	view.WorkingDays = kernel.WorkingDays(workingDays)
	view.Seniority = kernel.Seniority(seniority)
	view.ContactID = contactID
	if u, err := kernel.NewUnderstanding(uint8(min(understanding, 100))); err == nil {
		view.UnderstandingScore = u
	}
	if len(confidence) > 0 {
		if err := json.Unmarshal(confidence, &view.FieldConfidence); err != nil {
			return appjobs.JobView{}, fmt.Errorf("unmarshalling field confidence: %w", err)
		}
	}
	if appStatus != nil {
		s := kernel.ApplicationStatus(*appStatus)
		view.ApplicationStatus = &s
	}
	return view, nil
}

// FindByFingerprint looks up an existing job row by its dedup fingerprint.
// Returns (id, true, nil) on a hit; ("", false, nil) when no row matches.
func (r *Repository) FindByFingerprint(ctx context.Context, fingerprint string) (kernel.JobID, bool, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT id FROM job WHERE fingerprint = $1`, fingerprint).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("finding job by fingerprint: %w", err)
	}
	return kernel.JobID(id), true, nil
}

// MergeSource appends a JobSource row for an existing job, advances last_seen, and
// sets contact_id when the job's contact_id is currently NULL. The source insert is
// idempotent: a duplicate (job_id, raw_listing_id) row is silently skipped.
func (r *Repository) MergeSource(ctx context.Context, jobID kernel.JobID, source jobs.JobSource, lastSeen time.Time, contactID *kernel.ContactID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning merge transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertSource = `
		INSERT INTO job_source (job_id, raw_listing_id, board_id, source_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (job_id, raw_listing_id) DO NOTHING`
	if _, err := tx.Exec(ctx, insertSource,
		string(jobID), string(source.RawListingID), string(source.BoardID), source.SourceURL,
	); err != nil {
		return fmt.Errorf("inserting merged source: %w", err)
	}

	var contactParam *string
	if contactID != nil {
		s := string(*contactID)
		contactParam = &s
	}
	const updateJob = `
		UPDATE job
		SET last_seen  = $2,
		    contact_id = COALESCE(contact_id, $3)
		WHERE id = $1`
	if _, err := tx.Exec(ctx, updateJob, string(jobID), lastSeen, contactParam); err != nil {
		return fmt.Errorf("updating merged job metadata: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing merge transaction: %w", err)
	}
	return nil
}

// MarkExpired implements app/scraping.JobExpirer (P5-3): it sets expired_at=now for jobs
// sourced from boardID/profileID whose job_source.source_url is not in activeSourceURLs,
// and clears expired_at for any job whose source_url is (a listing that disappeared and
// later reappeared is active again, e.g. reposted). Scoping by profileID goes through
// raw_listing (the profile that captured the source), matching the read-side scoping
// used by ListByProfile/GetByProfile.
//
// An empty activeSourceURLs (board yielded zero re-scanned listings this run) expires
// every job currently sourced from that board/profile rather than reactivating none,
// which is the correct — if unlikely — behaviour when a board's search briefly returns
// nothing.
func (r *Repository) MarkExpired(ctx context.Context, boardID kernel.BoardID, profileID kernel.ProfileID, activeSourceURLs []string, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning expiry transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const expireStale = `
		UPDATE job j
		SET expired_at = $4
		FROM job_source js
		JOIN raw_listing rl ON rl.id = js.raw_listing_id
		WHERE js.job_id = j.id
		  AND js.board_id = $1
		  AND rl.profile_id = $2
		  AND NOT (js.source_url = ANY($3))
		  AND j.expired_at IS NULL`
	if _, err := tx.Exec(ctx, expireStale, string(boardID), string(profileID), activeSourceURLs, now); err != nil {
		return fmt.Errorf("marking stale jobs expired: %w", err)
	}

	const reactivateSeen = `
		UPDATE job j
		SET expired_at = NULL
		FROM job_source js
		JOIN raw_listing rl ON rl.id = js.raw_listing_id
		WHERE js.job_id = j.id
		  AND js.board_id = $1
		  AND rl.profile_id = $2
		  AND js.source_url = ANY($3)
		  AND j.expired_at IS NOT NULL`
	if _, err := tx.Exec(ctx, reactivateSeen, string(boardID), string(profileID), activeSourceURLs); err != nil {
		return fmt.Errorf("reactivating reappeared jobs: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing expiry transaction: %w", err)
	}
	return nil
}

var (
	_ jobs.Repository              = (*Repository)(nil)
	_ appjobs.JobQuery             = (*Repository)(nil)
	_ appjobs.JobApplicationWriter = (*Repository)(nil)
)
