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
	ListJobs(ctx context.Context, profileID kernel.ProfileID, filter appjobs.JobListFilter) (appjobs.JobListResult, error)
	GetJob(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (appjobs.JobView, error)
}

// JobApplicationUpdater handles the per-(profile,job) kanban write. Implemented by
// app/jobs.Service.
type JobApplicationUpdater interface {
	SetApplicationStatus(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID, status kernel.ApplicationStatus) (appjobs.ApplicationResult, error)
}

// JobReextractor re-publishes listing.extract for a job's retained raw listings.
// Implemented by app/jobs.Service (P5-4).
type JobReextractor interface {
	ReextractJob(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID) error
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

// jobListResponse is the paginated envelope returned by GET /api/jobs (ADR-007),
// replacing the previous bare jobResponse[] array.
type jobListResponse struct {
	Items      []jobResponse `json:"items"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	Total      int           `json:"total"`
	TotalPages int           `json:"total_pages"`
}

// defaultPageSize and maxPageSize/minPage implement the ADR-007 clamping rules for
// page/page_size: out-of-range values are clamped, never rejected with a 400.
const (
	defaultJobsPage     = 1
	defaultJobsPageSize = 25
	maxJobsPageSize     = 100
)

// ListJobs handles GET /api/jobs, returning a paginated envelope of jobs scoped to
// the active profile (ADR-007) with optional filter and sort query parameters:
// skills[], remote_policy, contract_type, salary_min, salary_max, location,
// board_id, since, confidence_min, sort (date|salary), sort_dir (asc|desc),
// page, page_size.
func ListJobs(reader JobReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)
		filter := parseJobListFilter(r)

		result, err := reader.ListJobs(r.Context(), profileID, filter)
		if err != nil {
			RespondError(w, r, err)
			return
		}

		items := make([]jobResponse, 0, len(result.Items))
		for _, j := range result.Items {
			items = append(items, toJobResponse(j))
		}
		respond(w, http.StatusOK, jobListResponse{
			Items:      items,
			Page:       filter.Page,
			PageSize:   filter.PageSize,
			Total:      result.Total,
			TotalPages: totalPages(result.Total, filter.PageSize),
		})
	}
}

// totalPages computes ceil(total/pageSize), returning 0 when total is 0 (ADR-007).
func totalPages(total, pageSize int) int {
	if total == 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
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

// PostJobReextract handles POST /api/jobs/{id}/reextract. It re-publishes
// listing.extract for the job's retained raw listing(s) so extract-worker reprocesses
// them (e.g. after an extractor improvement). Returns 202 Accepted since the actual
// re-extraction happens asynchronously in extract-worker.
func PostJobReextract(reextractor JobReextractor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)
		id := kernel.JobID(chi.URLParam(r, "id"))

		if err := reextractor.ReextractJob(r.Context(), profileID, id); err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusAccepted, map[string]string{"status": "re-extraction queued"})
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

	f.Page = clampPage(q.Get("page"))
	f.PageSize = clampPageSize(q.Get("page_size"))
	return f
}

// clampPage parses the page query param and clamps it to >= 1 (ADR-007). A missing or
// invalid value defaults to page 1 rather than a 400.
func clampPage(raw string) int {
	v, err := strconv.Atoi(raw)
	if err != nil || v < defaultJobsPage {
		return defaultJobsPage
	}
	return v
}

// clampPageSize parses the page_size query param and clamps it to 1..100 (ADR-007): a
// missing or unparseable value defaults to 25, an out-of-range value is clamped to the
// nearest bound (e.g. page_size=500 yields 100) rather than a 400.
func clampPageSize(raw string) int {
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultJobsPageSize
	}
	if v < 1 {
		return 1
	}
	if v > maxJobsPageSize {
		return maxJobsPageSize
	}
	return v
}
