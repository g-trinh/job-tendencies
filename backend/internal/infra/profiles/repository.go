// Package profiles provides the Postgres implementation of the profiles repository.
// search_keywords is stored as a Postgres text[] column. The exactly-one-active
// invariant is enforced by the profile_single_active partial unique index in the
// schema; Activate switches the active profile in a single transaction. Phase 3
// adds identity (skills, seniority), conditions, and fit-score weights columns.
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

// selectCols is the column list shared by all SELECT queries so the scan order is
// always consistent.
const selectCols = `
	id, name, search_keywords, location, is_active,
	skills, seniority,
	dealbreaker_contract_type, dealbreaker_remote_policy,
	dealbreaker_salary_min, dealbreaker_required_skills,
	preferred_skills, preferred_max_office_days,
	preferred_location, preferred_working_days,
	weight_preferred_skills, weight_salary,
	weight_location, weight_office_days, weight_working_days`

// ActiveProfile returns the single profile with is_active = true.
func (r *Repository) ActiveProfile(ctx context.Context) (profiles.Profile, error) {
	query := `SELECT` + selectCols + `FROM profile WHERE is_active = true LIMIT 1`
	return r.queryOne(ctx, query)
}

// ProfileByID returns one profile by id.
func (r *Repository) ProfileByID(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error) {
	query := `SELECT` + selectCols + `FROM profile WHERE id = $1`
	p, err := r.queryOne(ctx, query, string(id))
	if errors.Is(err, kernel.ErrNotFound) {
		return profiles.Profile{}, &kernel.NotFoundError{Kind: "profile", ID: string(id)}
	}
	return p, err
}

// List returns all profiles ordered by name.
func (r *Repository) List(ctx context.Context) ([]profiles.Profile, error) {
	query := `SELECT` + selectCols + `FROM profile ORDER BY name`

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
		INSERT INTO profile (name, search_keywords, location, is_active,
		    weight_preferred_skills, weight_salary, weight_location,
		    weight_office_days, weight_working_days)
		VALUES ($1, $2, $3, false, $4, $5, $6, $7, $8)
		RETURNING id`

	w := p.Weights
	var id string
	err := r.pool.QueryRow(ctx, query,
		p.Name, p.SearchKeywords, p.Location,
		w.PreferredSkills, w.Salary, w.Location, w.OfficeDays, w.WorkingDays,
	).Scan(&id)
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

// UpdateIdentity persists skills and seniority on the profile.
func (r *Repository) UpdateIdentity(ctx context.Context, id kernel.ProfileID, skills []string, seniority kernel.Seniority) error {
	const query = `UPDATE profile SET skills = $1, seniority = $2 WHERE id = $3`
	tag, err := r.pool.Exec(ctx, query, skills, string(seniority), string(id))
	if err != nil {
		return fmt.Errorf("updating identity for profile %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "profile", ID: string(id)}
	}
	return nil
}

// UpdateConditions persists the dealbreakers and preferences for a profile.
func (r *Repository) UpdateConditions(ctx context.Context, id kernel.ProfileID, c profiles.ProfileConditions) error {
	const query = `
		UPDATE profile SET
		    dealbreaker_contract_type   = $1,
		    dealbreaker_remote_policy   = $2,
		    dealbreaker_salary_min      = $3,
		    dealbreaker_required_skills = $4,
		    preferred_skills            = $5,
		    preferred_max_office_days   = $6,
		    preferred_location          = $7,
		    preferred_working_days      = $8
		WHERE id = $9`

	var dct, drp *string
	if c.DealBreakerContractType != nil {
		s := string(*c.DealBreakerContractType)
		dct = &s
	}
	if c.DealBreakerRemotePolicy != nil {
		s := string(*c.DealBreakerRemotePolicy)
		drp = &s
	}
	reqSkills := c.DealBreakerRequiredSkills
	if reqSkills == nil {
		reqSkills = []string{}
	}
	prefSkills := c.PreferredSkills
	if prefSkills == nil {
		prefSkills = []string{}
	}
	tag, err := r.pool.Exec(ctx, query,
		dct, drp, c.DealBreakerSalaryMin, reqSkills,
		prefSkills, c.PreferredMaxOfficeDays,
		c.PreferredLocation, string(c.PreferredWorkingDays),
		string(id),
	)
	if err != nil {
		return fmt.Errorf("updating conditions for profile %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "profile", ID: string(id)}
	}
	return nil
}

// UpdateWeights persists the fit-score weights for a profile.
func (r *Repository) UpdateWeights(ctx context.Context, id kernel.ProfileID, w profiles.FitWeights) error {
	const query = `
		UPDATE profile SET
		    weight_preferred_skills = $1,
		    weight_salary           = $2,
		    weight_location         = $3,
		    weight_office_days      = $4,
		    weight_working_days     = $5
		WHERE id = $6`

	tag, err := r.pool.Exec(ctx, query,
		w.PreferredSkills, w.Salary, w.Location, w.OfficeDays, w.WorkingDays,
		string(id),
	)
	if err != nil {
		return fmt.Errorf("updating weights for profile %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "profile", ID: string(id)}
	}
	return nil
}

func (r *Repository) queryOne(ctx context.Context, query string, args ...any) (profiles.Profile, error) {
	p, err := scanProfile(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return profiles.Profile{}, &kernel.NotFoundError{Kind: "profile", ID: "active"}
	}
	return p, err
}

// scanProfile scans a profile row from pgx.Rows or pgx.Row.
func scanProfile(row interface {
	Scan(dest ...any) error
}) (profiles.Profile, error) {
	var (
		p                    profiles.Profile
		seniority            string
		dealBreakerCT        *string
		dealBreakerRP        *string
		preferredWorkingDays string
	)
	if err := row.Scan(
		&p.ID, &p.Name, &p.SearchKeywords, &p.Location, &p.IsActive,
		&p.Skills, &seniority,
		&dealBreakerCT, &dealBreakerRP,
		&p.Conditions.DealBreakerSalaryMin, &p.Conditions.DealBreakerRequiredSkills,
		&p.Conditions.PreferredSkills, &p.Conditions.PreferredMaxOfficeDays,
		&p.Conditions.PreferredLocation, &preferredWorkingDays,
		&p.Weights.PreferredSkills, &p.Weights.Salary,
		&p.Weights.Location, &p.Weights.OfficeDays, &p.Weights.WorkingDays,
	); err != nil {
		return profiles.Profile{}, fmt.Errorf("scanning profile row: %w", err)
	}

	p.Seniority = kernel.Seniority(seniority)
	p.Conditions.PreferredWorkingDays = kernel.WorkingDays(preferredWorkingDays)
	if dealBreakerCT != nil {
		ct := kernel.ContractType(*dealBreakerCT)
		p.Conditions.DealBreakerContractType = &ct
	}
	if dealBreakerRP != nil {
		rp := kernel.RemotePolicy(*dealBreakerRP)
		p.Conditions.DealBreakerRemotePolicy = &rp
	}
	if p.Conditions.DealBreakerRequiredSkills == nil {
		p.Conditions.DealBreakerRequiredSkills = []string{}
	}
	if p.Conditions.PreferredSkills == nil {
		p.Conditions.PreferredSkills = []string{}
	}
	if p.Skills == nil {
		p.Skills = []string{}
	}
	return p, nil
}
