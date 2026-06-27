package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/contacts"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// fakeContactService is an in-memory fake of handler.ContactService.
type fakeContactService struct {
	records map[string]contacts.Contact // keyed by dedup_key for upsert logic
	byID    map[kernel.ContactID]contacts.Contact
	err     error
	nextID  int
}

func newFakeContactService() *fakeContactService {
	return &fakeContactService{
		records: make(map[string]contacts.Contact),
		byID:    make(map[kernel.ContactID]contacts.Contact),
	}
}

func (f *fakeContactService) nextContactID() kernel.ContactID {
	f.nextID++
	return kernel.ContactID("contact-" + strings.Repeat("0", 3-len(string(rune('0'+f.nextID)))) + string(rune('0'+f.nextID)))
}

func (f *fakeContactService) ListContacts(_ context.Context) ([]contacts.Contact, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]contacts.Contact, 0, len(f.byID))
	for _, c := range f.byID {
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeContactService) GetContact(_ context.Context, id kernel.ContactID) (contacts.Contact, error) {
	if f.err != nil {
		return contacts.Contact{}, f.err
	}
	c, ok := f.byID[id]
	if !ok {
		return contacts.Contact{}, &kernel.NotFoundError{Kind: "contact", ID: string(id)}
	}
	return c, nil
}

func (f *fakeContactService) UpsertContact(_ context.Context, name, company, email, linkedInURL, phone, notes string, tags []string) (contacts.Contact, bool, error) {
	if f.err != nil {
		return contacts.Contact{}, false, f.err
	}
	c, err := contacts.NewContact(name, company, email, linkedInURL, phone, notes, tags)
	if err != nil {
		return contacts.Contact{}, false, err
	}

	existing, found := f.records[c.DedupKey]
	if found {
		// merge: update non-empty fields
		existing.Name = name
		existing.Company = company
		existing.Notes = notes
		f.records[c.DedupKey] = existing
		f.byID[existing.ID] = existing
		return existing, false, nil
	}

	c.ID = kernel.ContactID("c-new")
	f.records[c.DedupKey] = c
	f.byID[c.ID] = c
	return c, true, nil
}

func (f *fakeContactService) UpdateContact(_ context.Context, id kernel.ContactID, name, company, email, linkedInURL, phone, notes string, tags []string) (contacts.Contact, error) {
	if f.err != nil {
		return contacts.Contact{}, f.err
	}
	c, ok := f.byID[id]
	if !ok {
		return contacts.Contact{}, &kernel.NotFoundError{Kind: "contact", ID: string(id)}
	}
	c.Name = name
	c.Company = company
	c.Email = email
	c.LinkedInURL = linkedInURL
	c.Phone = phone
	c.Notes = notes
	c.Tags = tags
	f.byID[id] = c
	return c, nil
}

func (f *fakeContactService) DeleteContact(_ context.Context, id kernel.ContactID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.byID[id]; !ok {
		return &kernel.NotFoundError{Kind: "contact", ID: string(id)}
	}
	delete(f.byID, id)
	return nil
}

func newContactRouter(svc *fakeContactService) *chi.Mux {
	r := handler.NewRouter(slog.Default())
	r.Get("/api/contacts", handler.ListContacts(svc))
	r.Post("/api/contacts", handler.PostContact(svc))
	r.Get("/api/contacts/{id}", handler.GetContact(svc))
	r.Put("/api/contacts/{id}", handler.PutContact(svc))
	r.Delete("/api/contacts/{id}", handler.DeleteContact(svc))
	return r
}

// AC: Upserting a contact with an existing email|linkedin merges, not duplicates.

func TestPostContact_Upsert(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		firstBody      string
		secondBody     string
		wantFirstCode  int
		wantSecondCode int
		wantCount      int
	}{
		{
			name:           "creates new contact and returns 201",
			firstBody:      `{"name":"Alice","email":"alice@example.com"}`,
			wantFirstCode:  http.StatusCreated,
			wantCount:      1,
		},
		{
			name:           "merges on duplicate email — returns 200 and no new record",
			firstBody:      `{"name":"Alice","email":"alice@example.com"}`,
			secondBody:     `{"name":"Alice Updated","email":"alice@example.com"}`,
			wantFirstCode:  http.StatusCreated,
			wantSecondCode: http.StatusOK,
			wantCount:      1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newFakeContactService()
			r := newContactRouter(svc)

			req1 := httptest.NewRequest(http.MethodPost, "/api/contacts", strings.NewReader(tc.firstBody))
			req1.Header.Set("Content-Type", "application/json")
			rec1 := httptest.NewRecorder()
			r.ServeHTTP(rec1, req1)

			if rec1.Code != tc.wantFirstCode {
				t.Errorf("first POST status = %d; want %d (body: %s)", rec1.Code, tc.wantFirstCode, rec1.Body.String())
			}

			if tc.secondBody != "" {
				req2 := httptest.NewRequest(http.MethodPost, "/api/contacts", strings.NewReader(tc.secondBody))
				req2.Header.Set("Content-Type", "application/json")
				rec2 := httptest.NewRecorder()
				r.ServeHTTP(rec2, req2)

				if rec2.Code != tc.wantSecondCode {
					t.Errorf("second POST status = %d; want %d (body: %s)", rec2.Code, tc.wantSecondCode, rec2.Body.String())
				}
			}

			if tc.wantCount > 0 && len(svc.byID) != tc.wantCount {
				t.Errorf("contact count = %d; want %d", len(svc.byID), tc.wantCount)
			}
		})
	}
}

func TestPostContact_ValidationErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "returns 400 when neither email nor linkedin provided",
			body:       `{"name":"Alice","company":"Acme"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns 400 for invalid JSON",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "accepts contact with only linkedin_url",
			body:       `{"name":"Bob","linkedin_url":"https://linkedin.com/in/bob"}`,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newFakeContactService()
			r := newContactRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/api/contacts", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestGetContact_ReturnsContactByID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		id         string
		wantStatus int
		wantName   string
	}{
		{
			name:       "returns contact by id",
			id:         "c-1",
			wantStatus: http.StatusOK,
			wantName:   "Carol",
		},
		{
			name:       "returns 404 for unknown contact",
			id:         "unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newFakeContactService()
			svc.byID["c-1"] = contacts.Contact{ID: "c-1", Name: "Carol", Email: "carol@example.com", DedupKey: "email:carol@example.com", Tags: []string{}}
			r := newContactRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/contacts/"+tc.id, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantName != "" {
				var resp struct {
					Name string `json:"name"`
				}
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				if resp.Name != tc.wantName {
					t.Errorf("name = %q; want %q", resp.Name, tc.wantName)
				}
			}
		})
	}
}

func TestDeleteContact_RemovesContact(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "deletes existing contact with 204",
			id:         "c-1",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "returns 404 for unknown contact",
			id:         "unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newFakeContactService()
			svc.byID["c-1"] = contacts.Contact{ID: "c-1", Name: "Carol", Tags: []string{}}
			r := newContactRouter(svc)

			req := httptest.NewRequest(http.MethodDelete, "/api/contacts/"+tc.id, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}
