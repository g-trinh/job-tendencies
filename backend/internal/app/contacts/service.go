// Package contacts contains the contacts-CRM application service. It exposes CRUD
// and upsert use cases for the Contact aggregate. Contacts are deduplicated by email
// or LinkedIn URL via the dedup_key mechanism in the domain. The aggregate repository
// interface lives in the domain (domain/contacts.Repository, ADR-005) and is
// implemented in infra/contacts.
package contacts

import (
	"context"
	"fmt"

	"github.com/g-trinh/job-tendencies/internal/domain/contacts"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Service exposes contacts-CRM use cases to the API and the extraction pipeline.
type Service struct {
	repo contacts.Repository
}

// New constructs a contacts Service.
func New(repo contacts.Repository) *Service {
	return &Service{repo: repo}
}

// ListContacts returns all contacts, optionally filtered by tag. An empty tag
// string returns all contacts.
func (s *Service) ListContacts(ctx context.Context, tag string) ([]contacts.Contact, error) {
	list, err := s.repo.List(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("listing contacts: %w", err)
	}
	return list, nil
}

// GetContact returns one contact by id.
func (s *Service) GetContact(ctx context.Context, id kernel.ContactID) (contacts.Contact, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return contacts.Contact{}, fmt.Errorf("getting contact %q: %w", id, err)
	}
	return c, nil
}

// UpsertContact validates and upserts a contact, deduplicating by email or LinkedIn
// URL. When a contact with the same dedup_key exists, it is merged. Returns the
// contact and whether it was newly created.
func (s *Service) UpsertContact(ctx context.Context, name, company, email, linkedInURL, phone, notes string, tags []string) (contacts.Contact, bool, error) {
	c, err := contacts.NewContact(name, company, email, linkedInURL, phone, notes, tags)
	if err != nil {
		return contacts.Contact{}, false, fmt.Errorf("validating contact: %w", err)
	}
	result, err := s.repo.Upsert(ctx, c)
	if err != nil {
		return contacts.Contact{}, false, fmt.Errorf("upserting contact: %w", err)
	}
	c.ID = result.ID
	return c, result.Created, nil
}

// UpdateContact persists all editable fields for an existing contact.
func (s *Service) UpdateContact(ctx context.Context, id kernel.ContactID, name, company, email, linkedInURL, phone, notes string, tags []string) (contacts.Contact, error) {
	c, err := contacts.NewContact(name, company, email, linkedInURL, phone, notes, tags)
	if err != nil {
		return contacts.Contact{}, fmt.Errorf("validating contact: %w", err)
	}
	c.ID = id
	if err := s.repo.Update(ctx, c); err != nil {
		return contacts.Contact{}, fmt.Errorf("updating contact %q: %w", id, err)
	}
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return contacts.Contact{}, fmt.Errorf("reading updated contact %q: %w", id, err)
	}
	return updated, nil
}

// DeleteContact removes a contact by id.
func (s *Service) DeleteContact(ctx context.Context, id kernel.ContactID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting contact %q: %w", id, err)
	}
	return nil
}
