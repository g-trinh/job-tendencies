package contacts

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// UpsertResult carries the result of an Upsert call.
type UpsertResult struct {
	// ID is the stable identifier of the contact (existing or newly created).
	ID kernel.ContactID
	// Created is true when a new contact was inserted; false when an existing
	// contact was merged (dedup hit).
	Created bool
}

// Repository is the contacts aggregate's persistence port. Aggregate repository
// interfaces live in the domain per ADR-005; the Postgres implementation lives in
// internal/infra/contacts.
type Repository interface {
	// List returns all contacts ordered by name. When tag is non-empty, only
	// contacts whose tags array contains that tag are returned.
	List(ctx context.Context, tag string) ([]Contact, error)
	// GetByID returns one contact, or a kernel.NotFoundError.
	GetByID(ctx context.Context, id kernel.ContactID) (Contact, error)
	// Upsert inserts a new contact or merges with an existing one that shares the
	// same dedup_key. Merging updates name, company, phone, notes, and tags. Returns
	// the contact id and whether a new record was created.
	Upsert(ctx context.Context, c Contact) (UpsertResult, error)
	// Update persists changes to an existing contact's fields.
	Update(ctx context.Context, c Contact) error
	// Delete removes a contact by id. It returns a kernel.NotFoundError when the
	// contact does not exist.
	Delete(ctx context.Context, id kernel.ContactID) error
}
