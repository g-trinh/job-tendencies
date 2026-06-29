// Package dashboard provides the Postgres implementation of the dashboard read
// queries (app/dashboard.DashboardQuery). All queries are profile-scoped: jobs
// are linked to a profile via the job_source → raw_listing join. This is a
// read-model context; it never writes.
package dashboard

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/g-trinh/job-tendencies/internal/app/dashboard"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

const (
	defaultFrequencyLimit = 20
	defaultTrendLimit     = 10
	defaultMatchLimit     = 20
)

// Repository satisfies app/dashboard.DashboardQuery over Postgres.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a dashboard Repository over the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SkillFrequency returns the top-N skills by occurrence count across all jobs
// scoped to the active profile. Each job is counted once per skill even if it
// was discovered via multiple boards.
//
// AC P3-DA-1: ranked skill counts scoped to the active profile.
func (r *Repository) SkillFrequency(ctx context.Context, profileID kernel.ProfileID, filter dashboard.SkillFrequencyFilter) ([]dashboard.SkillFrequencyEntry, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultFrequencyLimit
	}

	const query = `
		SELECT skill, COUNT(*) AS cnt
		FROM (
			SELECT DISTINCT j.id, skill
			FROM job j
			JOIN job_source js ON js.job_id = j.id
			JOIN raw_listing rl ON rl.id = js.raw_listing_id,
			LATERAL unnest(j.skills) AS skill
			WHERE rl.profile_id = $1
		) dedup
		GROUP BY skill
		ORDER BY cnt DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, string(profileID), limit)
	if err != nil {
		return nil, fmt.Errorf("querying skill frequency: %w", err)
	}
	defer rows.Close()

	var out []dashboard.SkillFrequencyEntry
	for rows.Next() {
		var e dashboard.SkillFrequencyEntry
		if err := rows.Scan(&e.Skill, &e.Count); err != nil {
			return nil, fmt.Errorf("scanning skill frequency row: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating skill frequency rows: %w", err)
	}
	if out == nil {
		out = []dashboard.SkillFrequencyEntry{}
	}
	return out, nil
}

// SkillTrend returns per-bucket skill counts for the top-N most frequent skills
// in the active profile's job set. The bucket granularity ('week' or 'month')
// is embedded directly in the query; it is validated before construction.
//
// AC P3-DA-2: per-period skill counts; default bucket is week.
func (r *Repository) SkillTrend(ctx context.Context, profileID kernel.ProfileID, filter dashboard.SkillTrendFilter) ([]dashboard.SkillTrendEntry, error) {
	bucket := filter.Bucket
	if bucket == "" {
		bucket = dashboard.TrendBucketWeek
	}
	// Validate bucket to prevent SQL injection; only two known values exist.
	if bucket != dashboard.TrendBucketWeek && bucket != dashboard.TrendBucketMonth {
		return nil, fmt.Errorf("invalid trend bucket %q: must be 'week' or 'month'", bucket)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultTrendLimit
	}

	// date_trunc requires a string literal, not a bind parameter.
	query := fmt.Sprintf(`
		WITH top_skills AS (
			SELECT skill
			FROM (
				SELECT DISTINCT j.id, skill
				FROM job j
				JOIN job_source js ON js.job_id = j.id
				JOIN raw_listing rl ON rl.id = js.raw_listing_id,
				LATERAL unnest(j.skills) AS skill
				WHERE rl.profile_id = $1
			) dedup
			GROUP BY skill
			ORDER BY COUNT(*) DESC
			LIMIT $2
		),
		dedup_jobs AS (
			SELECT DISTINCT j.id, j.first_seen, skill
			FROM job j
			JOIN job_source js ON js.job_id = j.id
			JOIN raw_listing rl ON rl.id = js.raw_listing_id,
			LATERAL unnest(j.skills) AS skill
			WHERE rl.profile_id = $1
			  AND skill IN (SELECT skill FROM top_skills)
		)
		SELECT date_trunc('%s', first_seen) AS period, skill, COUNT(*) AS cnt
		FROM dedup_jobs
		GROUP BY period, skill
		ORDER BY period DESC, cnt DESC`, string(bucket))

	rows, err := r.pool.Query(ctx, query, string(profileID), limit)
	if err != nil {
		return nil, fmt.Errorf("querying skill trend: %w", err)
	}
	defer rows.Close()

	var out []dashboard.SkillTrendEntry
	for rows.Next() {
		var e dashboard.SkillTrendEntry
		if err := rows.Scan(&e.Period, &e.Skill, &e.Count); err != nil {
			return nil, fmt.Errorf("scanning skill trend row: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating skill trend rows: %w", err)
	}
	if out == nil {
		out = []dashboard.SkillTrendEntry{}
	}
	return out, nil
}

// TopMatches returns jobs ordered by weighted_score descending. Only jobs that
// pass the dealbreaker gate (passes_dealbreakers = true) are included; jobs that
// have not been scored yet are excluded entirely.
//
// AC P3-DA-3: ordered by weighted_score, dealbreaker-failed excluded from top.
func (r *Repository) TopMatches(ctx context.Context, profileID kernel.ProfileID, filter dashboard.MatchFilter) ([]dashboard.MatchEntry, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultMatchLimit
	}

	const query = `
		SELECT j.id, j.title, j.company, j.location, j.url,
		       j.skills, j.remote_policy, j.contract_type, j.salary_min, j.salary_max,
		       js.weighted_score, js.passes_dealbreakers
		FROM job j
		JOIN job_score js ON js.job_id = j.id AND js.profile_id = $1
		WHERE js.passes_dealbreakers = TRUE
		  AND EXISTS (
		      SELECT 1
		      FROM job_source jsrc
		      JOIN raw_listing rl ON rl.id = jsrc.raw_listing_id
		      WHERE jsrc.job_id = j.id AND rl.profile_id = $1
		  )
		ORDER BY js.weighted_score DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, string(profileID), limit)
	if err != nil {
		return nil, fmt.Errorf("querying top matches: %w", err)
	}
	defer rows.Close()

	var out []dashboard.MatchEntry
	for rows.Next() {
		var (
			e      dashboard.MatchEntry
			skills []string
			id     string
		)
		if err := rows.Scan(
			&id, &e.Title, &e.Company, &e.Location, &e.URL,
			&skills, &e.RemotePolicy, &e.ContractType,
			&e.SalaryMin, &e.SalaryMax,
			&e.WeightedScore, &e.PassesDealbreakers,
		); err != nil {
			return nil, fmt.Errorf("scanning match row: %w", err)
		}
		e.JobID = kernel.JobID(id)
		e.Skills = skills
		if e.Skills == nil {
			e.Skills = []string{}
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating match rows: %w", err)
	}
	if out == nil {
		out = []dashboard.MatchEntry{}
	}
	return out, nil
}

// Stats returns the five aggregated dashboard metrics for the active profile.
// Jobs are deduplicated before aggregation to avoid double-counting across boards.
// PctRemote counts jobs with remote_policy = 'full_remote'.
//
// AC P3-DA-4: all five stats scoped to the active profile.
func (r *Repository) Stats(ctx context.Context, profileID kernel.ProfileID) (dashboard.Stats, error) {
	const query = `
		WITH distinct_jobs AS (
			SELECT DISTINCT j.id, j.first_seen, j.remote_policy,
			                j.contract_type, j.salary_min
			FROM job j
			JOIN job_source jsrc ON jsrc.job_id = j.id
			JOIN raw_listing rl ON rl.id = jsrc.raw_listing_id
			WHERE rl.profile_id = $1
		)
		SELECT
			COUNT(*) AS total,
			COUNT(CASE WHEN first_seen::date >= CURRENT_DATE THEN 1 END) AS new_today,
			COUNT(CASE WHEN first_seen >= date_trunc('week', now() AT TIME ZONE 'UTC') THEN 1 END) AS new_this_week,
			COUNT(CASE WHEN remote_policy = 'full_remote' THEN 1 END)::float
				/ NULLIF(COUNT(*), 0) AS pct_remote,
			AVG(salary_min::float) AS avg_salary,
			mode() WITHIN GROUP (ORDER BY contract_type) AS top_contract_type
		FROM distinct_jobs`

	var (
		s               dashboard.Stats
		topContractType *string
	)
	err := r.pool.QueryRow(ctx, query, string(profileID)).Scan(
		&s.Total,
		&s.NewToday,
		&s.NewThisWeek,
		&s.PctRemote,
		&s.AvgSalary,
		&topContractType,
	)
	if err != nil {
		return dashboard.Stats{}, fmt.Errorf("querying dashboard stats: %w", err)
	}
	if topContractType != nil {
		s.TopContractType = *topContractType
	}
	return s, nil
}
