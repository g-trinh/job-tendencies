// Package contacts is the contacts-CRM bounded context. It owns the Contact aggregate —
// a recruiter record auto-populated from the extraction pipeline and deduplicated by
// email or LinkedIn URL. Contacts are linked from jobs via job.contact_id.
package contacts

import (
	"fmt"
	"strings"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Contact is a recruiter record. It is auto-populated from extraction and may also be
// manually created or edited. Deduplication uses dedup_key: "email:<email>" when an
// email is present, otherwise "linkedin:<url>". At least one of email or linkedin_url
// must be supplied to construct a Contact via NewContact.
type Contact struct {
	// ID is the contact's stable identifier.
	ID kernel.ContactID
	// Name is the recruiter's full name.
	Name string
	// Company is the recruiter's employer.
	Company string
	// Email is the recruiter's email address; used as the primary dedup key.
	Email string
	// LinkedInURL is the recruiter's LinkedIn profile URL; fallback dedup key when
	// no email is available.
	LinkedInURL string
	// Phone is the recruiter's phone number (optional).
	Phone string
	// Notes is free-text recruiter notes.
	Notes string
	// Tags is the label set (e.g. ["in-house", "responsive"]).
	Tags []string
	// DedupKey is the canonical identifier used for upsert: "email:<email>" when
	// email is non-empty, otherwise "linkedin:<url>".
	DedupKey string
}

// NewContact constructs a Contact with a computed DedupKey. At least one of email or
// linkedInURL must be non-empty. Email takes priority as the dedup key.
//
// Example:
//
//	c, err := contacts.NewContact("Alice Martin", "Acme Corp", "alice@acme.io", "", "", "", nil)
func NewContact(name, company, email, linkedInURL, phone, notes string, tags []string) (Contact, error) {
	email = strings.TrimSpace(email)
	linkedInURL = strings.TrimSpace(linkedInURL)

	if email == "" && linkedInURL == "" {
		return Contact{}, &kernel.ValidationError{
			Field:   "email",
			Message: "at least one of email or linkedin_url is required",
		}
	}

	dedupKey := computeDedupKey(email, linkedInURL)

	if tags == nil {
		tags = []string{}
	}

	return Contact{
		Name:        name,
		Company:     company,
		Email:       email,
		LinkedInURL: linkedInURL,
		Phone:       phone,
		Notes:       notes,
		Tags:        tags,
		DedupKey:    dedupKey,
	}, nil
}

// computeDedupKey returns "email:<email>" when email is non-empty, otherwise
// "linkedin:<url>". This is the canonical uniqueness key for contact deduplication.
func computeDedupKey(email, linkedInURL string) string {
	if email != "" {
		return fmt.Sprintf("email:%s", strings.ToLower(email))
	}
	return fmt.Sprintf("linkedin:%s", linkedInURL)
}

// ComputeDedupKey is the exported form of the dedup key computation, used by infra
// callers when they need to look up or upsert by identity without constructing a full
// Contact.
func ComputeDedupKey(email, linkedInURL string) string {
	return computeDedupKey(strings.TrimSpace(email), strings.TrimSpace(linkedInURL))
}
