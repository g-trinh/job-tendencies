package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/contacts"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// ContactService exposes contacts-CRM use cases. Implemented by app/contacts.Service.
type ContactService interface {
	ListContacts(ctx context.Context) ([]contacts.Contact, error)
	GetContact(ctx context.Context, id kernel.ContactID) (contacts.Contact, error)
	UpsertContact(ctx context.Context, name, company, email, linkedInURL, phone, notes string, tags []string) (contacts.Contact, bool, error)
	UpdateContact(ctx context.Context, id kernel.ContactID, name, company, email, linkedInURL, phone, notes string, tags []string) (contacts.Contact, error)
	DeleteContact(ctx context.Context, id kernel.ContactID) error
}

// contactResponse is the JSON shape of a contact returned by the contacts endpoints.
type contactResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Company     string   `json:"company"`
	Email       string   `json:"email"`
	LinkedInURL string   `json:"linkedin_url"`
	Phone       string   `json:"phone"`
	Notes       string   `json:"notes"`
	Tags        []string `json:"tags"`
	DedupKey    string   `json:"dedup_key"`
}

// contactWriteRequest is the shared request body for POST and PUT /api/contacts.
type contactWriteRequest struct {
	Name        string   `json:"name"`
	Company     string   `json:"company"`
	Email       string   `json:"email"`
	LinkedInURL string   `json:"linkedin_url"`
	Phone       string   `json:"phone"`
	Notes       string   `json:"notes"`
	Tags        []string `json:"tags"`
}

func toContactResponse(c contacts.Contact) contactResponse {
	tags := c.Tags
	if tags == nil {
		tags = []string{}
	}
	return contactResponse{
		ID:          string(c.ID),
		Name:        c.Name,
		Company:     c.Company,
		Email:       c.Email,
		LinkedInURL: c.LinkedInURL,
		Phone:       c.Phone,
		Notes:       c.Notes,
		Tags:        tags,
		DedupKey:    c.DedupKey,
	}
}

// ListContacts handles GET /api/contacts, returning all contacts.
func ListContacts(svc ContactService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.ListContacts(r.Context())
		if err != nil {
			RespondError(w, r, err)
			return
		}
		out := make([]contactResponse, 0, len(list))
		for _, c := range list {
			out = append(out, toContactResponse(c))
		}
		respond(w, http.StatusOK, out)
	}
}

// GetContact handles GET /api/contacts/{id}.
func GetContact(svc ContactService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ContactID(chi.URLParam(r, "id"))
		c, err := svc.GetContact(r.Context(), id)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toContactResponse(c))
	}
}

// PostContact handles POST /api/contacts. It upserts by email or LinkedIn URL:
// when a contact with the same dedup_key already exists, it is merged instead of
// duplicated. Returns 201 when a new contact is created, 200 when merged.
func PostContact(svc ContactService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body contactWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		c, created, err := svc.UpsertContact(r.Context(),
			body.Name, body.Company, body.Email, body.LinkedInURL,
			body.Phone, body.Notes, body.Tags)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		respond(w, status, toContactResponse(c))
	}
}

// PutContact handles PUT /api/contacts/{id}, updating all editable fields.
func PutContact(svc ContactService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ContactID(chi.URLParam(r, "id"))
		var body contactWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		c, err := svc.UpdateContact(r.Context(), id,
			body.Name, body.Company, body.Email, body.LinkedInURL,
			body.Phone, body.Notes, body.Tags)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toContactResponse(c))
	}
}

// DeleteContact handles DELETE /api/contacts/{id}.
func DeleteContact(svc ContactService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ContactID(chi.URLParam(r, "id"))
		if err := svc.DeleteContact(r.Context(), id); err != nil {
			RespondError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
