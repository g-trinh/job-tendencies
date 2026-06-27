package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
)

// ActiveProfileResolver returns the single active profile. Implemented by
// app/profiles.Service.
type ActiveProfileResolver interface {
	ActiveProfile(ctx context.Context) (profiles.Profile, error)
}

// ProfileService exposes profile CRUD and activation. Implemented by
// app/profiles.Service.
type ProfileService interface {
	ActiveProfileResolver
	ListProfiles(ctx context.Context) ([]profiles.Profile, error)
	ProfileByID(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error)
	CreateProfile(ctx context.Context, name, location string, keywords []string) (profiles.Profile, error)
	UpdateProfile(ctx context.Context, id kernel.ProfileID, name, location string, keywords []string) (profiles.Profile, error)
	DeleteProfile(ctx context.Context, id kernel.ProfileID) error
	ActivateProfile(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error)
}

// profileResponse is the JSON shape of a profile.
type profileResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	SearchKeywords []string `json:"search_keywords"`
	Location       string   `json:"location"`
	IsActive       bool     `json:"is_active"`
}

func toProfileResponse(p profiles.Profile) profileResponse {
	kw := p.SearchKeywords
	if kw == nil {
		kw = []string{}
	}
	return profileResponse{
		ID:             string(p.ID),
		Name:           p.Name,
		SearchKeywords: kw,
		Location:       p.Location,
		IsActive:       p.IsActive,
	}
}

// GetActiveProfile handles GET /api/active-profile, returning the active profile.
func GetActiveProfile(svc ActiveProfileResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := svc.ActiveProfile(r.Context())
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toProfileResponse(p))
	}
}

// PutActiveProfile handles PUT /api/active-profile. Body: {"profile_id":"<id>"}.
// Switches the active profile and returns the newly active profile.
func PutActiveProfile(svc ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ProfileID string `json:"profile_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ProfileID == "" {
			RespondError(w, r, &kernel.ValidationError{Field: "profile_id", Message: "required"})
			return
		}
		p, err := svc.ActivateProfile(r.Context(), kernel.ProfileID(body.ProfileID))
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toProfileResponse(p))
	}
}

// ListProfiles handles GET /api/profiles, returning all profiles.
func ListProfiles(svc ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.ListProfiles(r.Context())
		if err != nil {
			RespondError(w, r, err)
			return
		}
		out := make([]profileResponse, 0, len(list))
		for _, p := range list {
			out = append(out, toProfileResponse(p))
		}
		respond(w, http.StatusOK, out)
	}
}

// GetProfile handles GET /api/profiles/{id}.
func GetProfile(svc ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ProfileID(chi.URLParam(r, "id"))
		p, err := svc.ProfileByID(r.Context(), id)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toProfileResponse(p))
	}
}

// createProfileRequest is the request body for POST /api/profiles.
type createProfileRequest struct {
	Name           string   `json:"name"`
	SearchKeywords []string `json:"search_keywords"`
	Location       string   `json:"location"`
}

// PostProfile handles POST /api/profiles, creating a new inactive profile.
func PostProfile(svc ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body createProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		p, err := svc.CreateProfile(r.Context(), body.Name, body.Location, body.SearchKeywords)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusCreated, toProfileResponse(p))
	}
}

// PutProfile handles PUT /api/profiles/{id}, updating name, keywords, and location.
func PutProfile(svc ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ProfileID(chi.URLParam(r, "id"))
		var body createProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		p, err := svc.UpdateProfile(r.Context(), id, body.Name, body.Location, body.SearchKeywords)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toProfileResponse(p))
	}
}

// DeleteProfile handles DELETE /api/profiles/{id}.
func DeleteProfile(svc ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ProfileID(chi.URLParam(r, "id"))
		if err := svc.DeleteProfile(r.Context(), id); err != nil {
			RespondError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
