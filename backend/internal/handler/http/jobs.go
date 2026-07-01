package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// JobReader lists and fetches job views scoped to a profile. Implemented by
// app/jobs.Service (the read side, ADR-005).
type JobReader interface {
	ListJobs(ctx context.Context, profileID kernel.ProfileID, filter appjobs.JobListFilter) ([]appjobs.JobView, error)
	GetJob(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (appjobs.JobView, error)
}

// JobApplicationUpdater handles the per-(profile,job) kanban write. Implemented by
// app/jobs.Service.
type JobApplicationUpdater interface {
	SetApplicationStatus(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID, status kernel.ApplicationStatus) (appjobs.ApplicationResult, error)
}

// jobSourceResponse is the JSON shape of a job's source linkage per the FE contract:
// {board_id, source_url, board_name}.
type jobSourceResponse struct {
	BoardID   string `json:"board_id"`
	SourceURL string `json:"source_url"`
	BoardName string `json:"board_name"`
}

// jobResponse is the JSON shape of a job returned by both the list and detail endpoints.
// Structured enums are returned as their machine values; the frontend renders them in French.
type jobResponse struct {
	ID                 string              `json:"id"`
	Title              string              `json:"title"`
	Company            string              `json:"company"`
	Location           string              `json:"location"`
	URL                string              `json:"url"`
	Skills             []string            `json:"skills"`
	RemotePolicy       string              `json:"remote_policy"`
	OfficeDays         int                 `json:"office_days"`
	ContractType       string              `json:"contract_type"`
	WorkingDays        string              `json:"working_days"`
	SalaryMin          *int64              `json:"salary_min"`
	SalaryMax          *int64              `json:"salary_max"`
	Seniority          string              `json:"seniority"`
	FieldConfidence    map[string]int      `json:"field_confidence"`
	UnderstandingScore int                 `json:"understanding_score"`
	Description        string              `json:"description"`
	ContactID          *string             `json:"contact_id"`
	FirstSeen          string              `json:"first_seen"`
	LastSeen           string              `json:"last_seen"`
	ExpiredAt          *string             `json:"expired_at"`
	ApplicationStatus  *string             `json:"application_status"`
	FitScore           *float64            `json:"fit_score"`
	Sources            []jobSourceResponse `json:"sources"`
}

// ListJobs handles GET /api/jobs, returning jobs scoped to the active profile with
// optional filter and sort query parameters:
// skills[], remote_policy, contract_type, salary_min, salary_max, location,
// board_id, since, confidence_min, sort (date|salary), sort_dir (asc|desc).
func ListJobs(reader JobReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)
		filter := parseJobListFilter(r)

		list, err := reader.ListJobs(r.Context(), profileID, filter)
		if err != nil {
			RespondError(w, r, err)
			return
		}

		out := make([]jobResponse, 0, len(list))
		for _, j := range list {
			out = append(out, toJobResponse(j))
		}
		respond(w, http.StatusOK, out)
	}
}

// GetJob handles GET /api/jobs/{id}, returning one job scoped to the active profile.
func GetJob(reader JobReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)
		id := kernel.JobID(chi.URLParam(r, "id"))
		job, err := reader.GetJob(r.Context(), profileID, id)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toJobResponse(job))
	}
}

// GetJobOriginal handles GET /api/jobs/{id}/original, redirecting to the source URL.
func GetJobOriginal(reader JobReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)
		id := kernel.JobID(chi.URLParam(r, "id"))
		job, err := reader.GetJob(r.Context(), profileID, id)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		if job.URL == "" {
			RespondError(w, r, &kernel.ValidationError{Field: "url", Message: "no source URL available"})
			return
		}
		http.Redirect(w, r, job.URL, http.StatusFound)
	}
}

// PatchJobApplication handles PATCH /api/jobs/{id}/application.
// Body: {"status":"<value>"}. Returns {"status","updated_at"}.
func PatchJobApplication(updater JobApplicationUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)
		id := kernel.JobID(chi.URLParam(r, "id"))

		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "body", Message: "invalid JSON"})
			return
		}
		status, err := kernel.ParseApplicationStatus(body.Status)
		if err != nil {
			RespondError(w, r, &kernel.ValidationError{Field: "status", Message: err.Error()})
			return
		}

		result, err := updater.SetApplicationStatus(r.Context(), profileID, id, status)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, map[string]string{
			"status":     string(result.Status),
			"updated_at": result.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
}

func toJobResponse(j appjobs.JobView) jobResponse {
	sources := make([]jobSourceResponse, 0, len(j.Sources))
	for _, s := range j.Sources {
		sources = append(sources, jobSourceResponse{
			BoardID:   string(s.BoardID),
			SourceURL: s.SourceURL,
			BoardName: s.BoardName,
		})
	}

	skills := j.Skills
	if skills == nil {
		skills = []string{}
	}
	if j.FieldConfidence == nil {
		j.FieldConfidence = map[string]int{}
	}

	var expiredAt *string
	if j.ExpiredAt != nil {
		s := j.ExpiredAt.UTC().Format(time.RFC3339)
		expiredAt = &s
	}
	var appStatus *string
	if j.ApplicationStatus != nil {
		s := string(*j.ApplicationStatus)
		appStatus = &s
	}

	return jobResponse{
		ID:                 string(j.ID),
		Title:              j.Title,
		Company:            j.Company,
		Location:           j.Location,
		URL:                j.URL,
		Skills:             skills,
		RemotePolicy:       string(j.RemotePolicy),
		OfficeDays:         j.OfficeDays,
		ContractType:       string(j.ContractType),
		WorkingDays:        string(j.WorkingDays),
		SalaryMin:          j.SalaryMin,
		SalaryMax:          j.SalaryMax,
		Seniority:          string(j.Seniority),
		FieldConfidence:    j.FieldConfidence,
		UnderstandingScore: j.UnderstandingScore.Int(),
		Description:        j.Description,
		ContactID:          j.ContactID,
		FirstSeen:          j.FirstSeen.UTC().Format(time.RFC3339),
		LastSeen:           j.LastSeen.UTC().Format(time.RFC3339),
		ExpiredAt:          expiredAt,
		ApplicationStatus:  appStatus,
		FitScore:           j.FitScore,
		Sources:            sources,
	}
}

// parseJobListFilter reads query parameters into a JobListFilter.
func parseJobListFilter(r *http.Request) appjobs.JobListFilter {
	q := r.URL.Query()
	f := appjobs.JobListFilter{
		Skills:       q["skills[]"],
		RemotePolicy: q.Get("remote_policy"),
		ContractType: q.Get("contract_type"),
		Location:     q.Get("location"),
		BoardID:      q.Get("board_id"),
		Sort:         q.Get("sort"),
		SortDir:      q.Get("sort_dir"),
	}
	if s := q.Get("salary_min"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			f.SalaryMin = &v
		}
	}
	if s := q.Get("salary_max"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			f.SalaryMax = &v
		}
	}
	if s := q.Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			f.Since = &t
		}
	}
	if s := q.Get("confidence_min"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			f.ConfidenceMin = &v
		}
	}
	// Normalise sort_dir to lower-case for the repo comparison.
	f.SortDir = strings.ToLower(f.SortDir)
	return f
}
