package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// JobReader lists and fetches job views scoped to a profile. Implemented by
// app/jobs.Service (the read side, ADR-005).
type JobReader interface {
	ListJobs(ctx context.Context, profileID kernel.ProfileID) ([]appjobs.JobView, error)
	GetJob(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (appjobs.JobView, error)
}

// jobSourceResponse is the JSON shape of a job's source linkage.
type jobSourceResponse struct {
	BoardID      string `json:"board_id"`
	RawListingID string `json:"raw_listing_id"`
	SourceURL    string `json:"source_url"`
}

// jobResponse is the JSON shape of a job returned by the jobs endpoints. Structured
// enums are returned as their machine values; the frontend renders them in French.
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
	Sources            []jobSourceResponse `json:"sources"`
}

// ListJobs handles GET /api/jobs, returning jobs scoped to the active profile.
func ListJobs(reader JobReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, ok := ActiveProfileID(r)
		if !ok {
			RespondError(w, r, &kernel.ValidationError{Field: activeProfileHeader, Message: "header is required"})
			return
		}

		list, err := reader.ListJobs(r.Context(), profileID)
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
		profileID, ok := ActiveProfileID(r)
		if !ok {
			RespondError(w, r, &kernel.ValidationError{Field: activeProfileHeader, Message: "header is required"})
			return
		}

		id := kernel.JobID(chi.URLParam(r, "id"))
		job, err := reader.GetJob(r.Context(), profileID, id)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		respond(w, http.StatusOK, toJobResponse(job))
	}
}

func toJobResponse(j appjobs.JobView) jobResponse {
	sources := make([]jobSourceResponse, 0, len(j.Sources))
	for _, s := range j.Sources {
		sources = append(sources, jobSourceResponse{
			BoardID:      string(s.BoardID),
			RawListingID: string(s.RawListingID),
			SourceURL:    s.SourceURL,
		})
	}
	return jobResponse{
		ID:                 string(j.ID),
		Title:              j.Title,
		Company:            j.Company,
		Location:           j.Location,
		URL:                j.URL,
		Skills:             j.Skills,
		RemotePolicy:       string(j.RemotePolicy),
		OfficeDays:         j.OfficeDays,
		ContractType:       string(j.ContractType),
		WorkingDays:        string(j.WorkingDays),
		SalaryMin:          j.SalaryMin,
		SalaryMax:          j.SalaryMax,
		Seniority:          string(j.Seniority),
		FieldConfidence:    j.FieldConfidence,
		UnderstandingScore: j.UnderstandingScore.Int(),
		Sources:            sources,
	}
}
