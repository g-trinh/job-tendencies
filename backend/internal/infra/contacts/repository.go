// Package contacts provides the Postgres implementation of the contacts-CRM repository.
// dedup_key is the uniqueness key used for upsert; the DB unique index on dedup_key
// guarantees deduplication at the storage level.
package contacts

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/g-trinh/job-tendencies/internal/domain/contacts"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Repository persists and reads contacts in Postgres. It satisfies
// domain/contacts.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

var _ contacts.Repository = (*Repository)(nil)

// NewRepository constructs a Postgres contact repository over the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List returns all contacts ordered by name.
func (r *Repository) List(ctx context.Context) ([]contacts.Contact, error) {
	const query = `
		SELECT id, name, company, email, linkedin_url, phone, notes, tags, dedup_key
		FROM contact ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying contacts: %w", err)
	}
	defer rows.Close()

	var out []contacts.Contact
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating contact rows: %w", err)
	}
	return out, nil
}

// GetByID returns one contact or a kernel.NotFoundError.
func (r *Repository) GetByID(ctx context.Context, id kernel.ContactID) (contacts.Contact, error) {
	const query = `
		SELECT id, name, company, email, linkedin_url, phone, notes, tags, dedup_key
		FROM contact WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, string(id))
	c, err := scanContact(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return contacts.Contact{}, &kernel.NotFoundError{Kind: "contact", ID: string(id)}
	}
	if err != nil {
		return contacts.Contact{}, err
	}
	return c, nil
}

// Upsert inserts a new contact or merges with an existing one sharing the same
// dedup_key. On conflict the name, company, phone, and notes columns are updated;
// tags are merged (union). Returns the contact id and whether a new record was created.
func (r *Repository) Upsert(ctx context.Context, c contacts.Contact) (contacts.UpsertResult, error) {
	const query = `
		INSERT INTO contact (name, company, email, linkedin_url, phone, notes, tags, dedup_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (dedup_key) DO UPDATE
		    SET name        = EXCLUDED.name,
		        company     = EXCLUDED.company,
		        phone       = EXCLUDED.phone,
		        notes       = CASE WHEN EXCLUDED.notes != '' THEN EXCLUDED.notes ELSE contact.notes END,
		        tags        = (
		            SELECT array_agg(DISTINCT t ORDER BY t)
		            FROM unnest(contact.tags || EXCLUDED.tags) t
		        ),
		        updated_at  = now()
		RETURNING id, (xmax = 0) AS inserted`

	var id string
	var inserted bool
	err := r.pool.QueryRow(ctx, query,
		c.Name, c.Company, c.Email, c.LinkedInURL, c.Phone, c.Notes, c.Tags, c.DedupKey,
	).Scan(&id, &inserted)
	if err != nil {
		return contacts.UpsertResult{}, fmt.Errorf("upserting contact: %w", err)
	}
	return contacts.UpsertResult{ID: kernel.ContactID(id), Created: inserted}, nil
}

// Update persists all editable fields for the contact.
func (r *Repository) Update(ctx context.Context, c contacts.Contact) error {
	const query = `
		UPDATE contact
		SET name = $1, company = $2, email = $3, linkedin_url = $4,
		    phone = $5, notes = $6, tags = $7, updated_at = now()
		WHERE id = $8`

	tag, err := r.pool.Exec(ctx, query,
		c.Name, c.Company, c.Email, c.LinkedInURL, c.Phone, c.Notes, c.Tags, string(c.ID))
	if err != nil {
		return fmt.Errorf("updating contact %q: %w", c.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "contact", ID: string(c.ID)}
	}
	return nil
}

// Delete removes a contact by id.
func (r *Repository) Delete(ctx context.Context, id kernel.ContactID) error {
	const query = `DELETE FROM contact WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("deleting contact %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "contact", ID: string(id)}
	}
	return nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanContact(row rowScanner) (contacts.Contact, error) {
	var c contacts.Contact
	if err := row.Scan(&c.ID, &c.Name, &c.Company, &c.Email, &c.LinkedInURL,
		&c.Phone, &c.Notes, &c.Tags, &c.DedupKey); err != nil {
		return contacts.Contact{}, fmt.Errorf("scanning contact row: %w", err)
	}
	return c, nil
}
