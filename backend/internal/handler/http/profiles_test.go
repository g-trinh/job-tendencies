package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// fakeProfileService is an in-memory fake of handler.ProfileService.
type fakeProfileService struct {
	list     []profiles.Profile
	active   *profiles.Profile
	err      error
}

func (f *fakeProfileService) ActiveProfile(_ context.Context) (profiles.Profile, error) {
	if f.err != nil {
		return profiles.Profile{}, f.err
	}
	if f.active == nil {
		return profiles.Profile{}, &kernel.NotFoundError{Kind: "profile", ID: "active"}
	}
	return *f.active, nil
}

func (f *fakeProfileService) ListProfiles(_ context.Context) ([]profiles.Profile, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

func (f *fakeProfileService) ProfileByID(_ context.Context, id kernel.ProfileID) (profiles.Profile, error) {
	if f.err != nil {
		return profiles.Profile{}, f.err
	}
	for _, p := range f.list {
		if p.ID == id {
			return p, nil
		}
	}
	return profiles.Profile{}, &kernel.NotFoundError{Kind: "profile", ID: string(id)}
}

func (f *fakeProfileService) CreateProfile(_ context.Context, name, location string, keywords []string) (profiles.Profile, error) {
	if f.err != nil {
		return profiles.Profile{}, f.err
	}
	if name == "" {
		return profiles.Profile{}, &kernel.ValidationError{Field: "name", Message: "required"}
	}
	p := profiles.Profile{
		ID:             kernel.ProfileID("new-id"),
		Name:           name,
		Location:       location,
		SearchKeywords: keywords,
	}
	f.list = append(f.list, p)
	return p, nil
}

func (f *fakeProfileService) UpdateProfile(_ context.Context, id kernel.ProfileID, name, location string, keywords []string) (profiles.Profile, error) {
	if f.err != nil {
		return profiles.Profile{}, f.err
	}
	for i, p := range f.list {
		if p.ID == id {
			f.list[i].Name = name
			f.list[i].Location = location
			f.list[i].SearchKeywords = keywords
			return f.list[i], nil
		}
	}
	return profiles.Profile{}, &kernel.NotFoundError{Kind: "profile", ID: string(id)}
}

func (f *fakeProfileService) DeleteProfile(_ context.Context, id kernel.ProfileID) error {
	if f.err != nil {
		return f.err
	}
	for i, p := range f.list {
		if p.ID == id {
			f.list = append(f.list[:i], f.list[i+1:]...)
			return nil
		}
	}
	return &kernel.NotFoundError{Kind: "profile", ID: string(id)}
}

func (f *fakeProfileService) ActivateProfile(_ context.Context, id kernel.ProfileID) (profiles.Profile, error) {
	if f.err != nil {
		return profiles.Profile{}, f.err
	}
	for i := range f.list {
		f.list[i].IsActive = false
	}
	for i, p := range f.list {
		if p.ID == id {
			f.list[i].IsActive = true
			active := f.list[i]
			f.active = &active
			return active, nil
		}
	}
	return profiles.Profile{}, &kernel.NotFoundError{Kind: "profile", ID: string(id)}
}

func newProfileRouter(svc *fakeProfileService) *chi.Mux {
	logger := slog.Default()
	r := handler.NewRouter(logger)
	r.Get("/api/profiles", handler.ListProfiles(svc))
	r.Post("/api/profiles", handler.PostProfile(svc))
	r.Get("/api/profiles/{id}", handler.GetProfile(svc))
	r.Put("/api/profiles/{id}", handler.PutProfile(svc))
	r.Delete("/api/profiles/{id}", handler.DeleteProfile(svc))
	r.Get("/api/active-profile", handler.GetActiveProfile(svc))
	r.Put("/api/active-profile", handler.PutActiveProfile(svc))
	return r
}

// AC: Creating/activating a profile leaves exactly one active.

func TestPostProfile_CreatesInactiveProfile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantName   string
	}{
		{
			name:       "creates profile with valid body",
			body:       `{"name":"Go Backend","location":"Paris","search_keywords":["golang"]}`,
			wantStatus: http.StatusCreated,
			wantName:   "Go Backend",
		},
		{
			name:       "returns 400 when name is empty",
			body:       `{"name":"","location":"Paris"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns 400 for invalid JSON",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeProfileService{}
			r := newProfileRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/api/profiles", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantName != "" {
				var resp struct {
					Name     string `json:"name"`
					IsActive bool   `json:"is_active"`
				}
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				if resp.Name != tc.wantName {
					t.Errorf("name = %q; want %q", resp.Name, tc.wantName)
				}
				if resp.IsActive {
					t.Error("new profile should be inactive; got is_active=true")
				}
			}
		})
	}
}

// AC: PUT /api/active-profile switches the active profile.

func TestPutActiveProfile_SwitchesActiveProfile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		body       string
		svc        *fakeProfileService
		wantStatus int
		wantActive string
	}{
		{
			name: "switches active profile and returns it",
			body: `{"profile_id":"p-2"}`,
			svc: &fakeProfileService{
				list: []profiles.Profile{
					{ID: "p-1", Name: "Profile A", IsActive: true},
					{ID: "p-2", Name: "Profile B", IsActive: false},
				},
			},
			wantStatus: http.StatusOK,
			wantActive: "Profile B",
		},
		{
			name:       "returns 404 when profile does not exist",
			body:       `{"profile_id":"unknown"}`,
			svc:        &fakeProfileService{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "returns 400 when profile_id is missing",
			body:       `{}`,
			svc:        &fakeProfileService{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := newProfileRouter(tc.svc)

			req := httptest.NewRequest(http.MethodPut, "/api/active-profile", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantActive != "" {
				var resp struct {
					Name     string `json:"name"`
					IsActive bool   `json:"is_active"`
				}
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				if resp.Name != tc.wantActive {
					t.Errorf("active profile name = %q; want %q", resp.Name, tc.wantActive)
				}
				if !resp.IsActive {
					t.Error("activated profile should have is_active=true")
				}
			}
		})
	}
}

func TestDeleteProfile_RemovesProfile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		id         string
		svc        *fakeProfileService
		wantStatus int
	}{
		{
			name: "deletes existing profile with 204",
			id:   "p-1",
			svc: &fakeProfileService{
				list: []profiles.Profile{{ID: "p-1", Name: "Profile A"}},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "returns 404 for unknown profile",
			id:         "unknown",
			svc:        &fakeProfileService{},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "returns 500 when service fails",
			id:   "p-1",
			svc: &fakeProfileService{
				list: []profiles.Profile{{ID: "p-1"}},
				err:  errors.New("db down"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := newProfileRouter(tc.svc)
			req := httptest.NewRequest(http.MethodDelete, "/api/profiles/"+tc.id, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}
