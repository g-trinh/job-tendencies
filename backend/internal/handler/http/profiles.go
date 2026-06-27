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

// ProfileService exposes profile CRUD, activation, identity, conditions, and weights.
// Implemented by app/profiles.Service.
type ProfileService interface {
	ActiveProfileResolver
	ListProfiles(ctx context.Context) ([]profiles.Profile, error)
	ProfileByID(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error)
	CreateProfile(ctx context.Context, name, location string, keywords []string) (profiles.Profile, error)
	UpdateProfile(ctx context.Context, id kernel.ProfileID, name, location string, keywords []string) (profiles.Profile, error)
	DeleteProfile(ctx context.Context, id kernel.ProfileID) error
	ActivateProfile(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error)
	PatchIdentity(ctx context.Context, id kernel.ProfileID, skills []string, seniority kernel.Seniority) (profiles.Profile, error)
	UpdateConditions(ctx context.Context, id kernel.ProfileID, c profiles.ProfileConditions) (profiles.Profile, error)
	UpdateWeights(ctx context.Context, id kernel.ProfileID, w profiles.FitWeights) (profiles.Profile, error)
}

// profileConditionsResponse is the JSON shape of profile conditions.
type profileConditionsResponse struct {
	DealBreakerContractType   *string  `json:"dealbreaker_contract_type"`
	DealBreakerRemotePolicy   *string  `json:"dealbreaker_remote_policy"`
	DealBreakerSalaryMin      *int64   `json:"dealbreaker_salary_min"`
	DealBreakerRequiredSkills []string `json:"dealbreaker_required_skills"`
	PreferredSkills           []string `json:"preferred_skills"`
	PreferredMaxOfficeDays    *int     `json:"preferred_max_office_days"`
	PreferredLocation         string   `json:"preferred_location"`
	PreferredWorkingDays      string   `json:"preferred_working_days"`
}

// profileWeightsResponse is the JSON shape of fit-score weights.
type profileWeightsResponse struct {
	PreferredSkills int `json:"preferred_skills"`
	Salary          int `json:"salary"`
	Location        int `json:"location"`
	OfficeDays      int `json:"office_days"`
	WorkingDays     int `json:"working_days"`
}

// profileResponse is the JSON shape of a profile.
type profileResponse struct {
	ID             string                    `json:"id"`
	Name           string                    `json:"name"`
	SearchKeywords []string                  `json:"search_keywords"`
	Location       string                    `json:"location"`
	IsActive       bool                      `json:"is_active"`
	Skills         []string                  `json:"skills"`
	Seniority      string                    `json:"seniority"`
	Conditions     profileConditionsResponse `json:"conditions"`
	Weights        profileWeightsResponse    `json:"weights"`
}

func toProfileResponse(p profiles.Profile) profileResponse {
	kw := p.SearchKeywords
	if kw == nil {
		kw = []string{}
	}
	skills := p.Skills
	if skills == nil {
		skills = []string{}
	}

	c := p.Conditions
	reqSkills := c.DealBreakerRequiredSkills
	if reqSkills == nil {
		reqSkills = []string{}
	}
	prefSkills := c.PreferredSkills
	if prefSkills == nil {
		prefSkills = []string{}
	}
	var dct, drp *string
	if c.DealBreakerContractType != nil {
		s := string(*c.DealBreakerContractType)
		dct = &s
	}
	if c.DealBreakerRemotePolicy != nil {
		s := string(*c.DealBreakerRemotePolicy)
		drp = &s
	}

	return profileResponse{
		ID:             string(p.ID),
		Name:           p.Name,
		SearchKeywords: kw,
		Location:       p.Location,
		IsActive:       p.IsActive,
		Skills:         skills,
		Seniority:      string(p.Seniority),
		Conditions: profileConditionsResponse{
			DealBreakerContractType:   dct,
			DealBreakerRemotePolicy:   drp,
			DealBreakerSalaryMin:      c.DealBreakerSalaryMin,
			DealBreakerRequiredSkills: reqSkills,
			PreferredSkills:           prefSkills,
			PreferredMaxOfficeDays:    c.PreferredMaxOfficeDays,
			PreferredLocation:         c.PreferredLocation,
			PreferredWorkingDays:      string(c.PreferredWorkingDays),
		},
		Weights: profileWeightsResponse{
			PreferredSkills: p.Weights.PreferredSkills,
			Salary:          p.Weights.Salary,
			Location:        p.Weights.Location,
			OfficeDays:      p.Weights.OfficeDays,
			WorkingDays:     p.Weights.WorkingDays,
		},
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

// patchIdentityRequest is the request body for PATCH /api/profiles/{id}/identity.
type patchIdentityRequest struct {
	Skills    []string `json:"skills"`
	Seniority string   `json:"seniority"`
}

// PatchProfileIdentity handles PATCH /api/profiles/{id}/identity.
// Updates skills and seniority for the profile (manual edit path for P3-PR-3).
func PatchProfileIdentity(svc ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ProfileID(chi.URLParam(r, "id"))
		var body patchIdentityRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		p, err := svc.PatchIdentity(r.Context(), id, body.Skills, kernel.Seniority(body.Seniority))
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toProfileResponse(p))
	}
}

// putConditionsRequest is the request body for PUT /api/profiles/{id}/conditions.
type putConditionsRequest struct {
	DealBreakerContractType   *string  `json:"dealbreaker_contract_type"`
	DealBreakerRemotePolicy   *string  `json:"dealbreaker_remote_policy"`
	DealBreakerSalaryMin      *int64   `json:"dealbreaker_salary_min"`
	DealBreakerRequiredSkills []string `json:"dealbreaker_required_skills"`
	PreferredSkills           []string `json:"preferred_skills"`
	PreferredMaxOfficeDays    *int     `json:"preferred_max_office_days"`
	PreferredLocation         string   `json:"preferred_location"`
	PreferredWorkingDays      string   `json:"preferred_working_days"`
}

// PutProfileConditions handles PUT /api/profiles/{id}/conditions.
func PutProfileConditions(svc ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ProfileID(chi.URLParam(r, "id"))
		var body putConditionsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		c := profiles.ProfileConditions{
			DealBreakerSalaryMin:      body.DealBreakerSalaryMin,
			DealBreakerRequiredSkills: body.DealBreakerRequiredSkills,
			PreferredSkills:           body.PreferredSkills,
			PreferredMaxOfficeDays:    body.PreferredMaxOfficeDays,
			PreferredLocation:         body.PreferredLocation,
			PreferredWorkingDays:      kernel.WorkingDays(body.PreferredWorkingDays),
		}
		if body.DealBreakerContractType != nil {
			ct := kernel.ContractType(*body.DealBreakerContractType)
			c.DealBreakerContractType = &ct
		}
		if body.DealBreakerRemotePolicy != nil {
			rp := kernel.RemotePolicy(*body.DealBreakerRemotePolicy)
			c.DealBreakerRemotePolicy = &rp
		}
		p, err := svc.UpdateConditions(r.Context(), id, c)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toProfileResponse(p))
	}
}

// putWeightsRequest is the request body for PUT /api/profiles/{id}/weights.
type putWeightsRequest struct {
	PreferredSkills int `json:"preferred_skills"`
	Salary          int `json:"salary"`
	Location        int `json:"location"`
	OfficeDays      int `json:"office_days"`
	WorkingDays     int `json:"working_days"`
}

// PutProfileWeights handles PUT /api/profiles/{id}/weights.
// Validates that the five weights sum to 100.
func PutProfileWeights(svc ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := kernel.ProfileID(chi.URLParam(r, "id"))
		var body putWeightsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		weights := profiles.FitWeights{
			PreferredSkills: body.PreferredSkills,
			Salary:          body.Salary,
			Location:        body.Location,
			OfficeDays:      body.OfficeDays,
			WorkingDays:     body.WorkingDays,
		}
		p, err := svc.UpdateWeights(r.Context(), id, weights)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toProfileResponse(p))
	}
}
