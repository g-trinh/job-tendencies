package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/g-trinh/job-tendencies/internal/app/dashboard"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// DashboardService is the narrow handler-side interface for dashboard reads.
// Implemented by app/dashboard.Service.
type DashboardService interface {
	SkillFrequency(ctx context.Context, profileID kernel.ProfileID, filter dashboard.SkillFrequencyFilter) ([]dashboard.SkillFrequencyEntry, error)
	SkillTrend(ctx context.Context, profileID kernel.ProfileID, filter dashboard.SkillTrendFilter) ([]dashboard.SkillTrendEntry, error)
	TopMatches(ctx context.Context, profileID kernel.ProfileID, filter dashboard.MatchFilter) ([]dashboard.MatchEntry, error)
	Stats(ctx context.Context, profileID kernel.ProfileID) (dashboard.Stats, error)
}

// skillFrequencyResponse is the JSON shape of one ranked skill entry.
type skillFrequencyResponse struct {
	Skill string `json:"skill"`
	Count int    `json:"count"`
}

// skillTrendResponse is the JSON shape of one skill-trend data point.
type skillTrendResponse struct {
	Period string `json:"period"`
	Skill  string `json:"skill"`
	Count  int    `json:"count"`
}

// matchResponse is the JSON shape of one top-match job entry.
type matchResponse struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Company            string   `json:"company"`
	Location           string   `json:"location"`
	URL                string   `json:"url"`
	Skills             []string `json:"skills"`
	RemotePolicy       string   `json:"remote_policy"`
	ContractType       string   `json:"contract_type"`
	SalaryMin          *int64   `json:"salary_min"`
	SalaryMax          *int64   `json:"salary_max"`
	WeightedScore      *float64 `json:"weighted_score"`
	PassesDealbreakers *bool    `json:"passes_dealbreakers"`
}

// statsResponse is the JSON shape of the dashboard stats card.
type statsResponse struct {
	Total           int      `json:"total"`
	NewToday        int      `json:"new_today"`
	NewThisWeek     int      `json:"new_this_week"`
	PctRemote       float64  `json:"pct_remote"`
	AvgSalary       *float64 `json:"avg_salary"`
	TopContractType string   `json:"top_contract_type"`
}

// GetDashboardSkillFrequency handles GET /api/dashboard/skills/frequency.
// Returns ranked skill counts scoped to the active profile.
// Optional query param: n (default 20, max is server-side).
//
// AC P3-DA-1: ranked skill counts scoped to the active profile.
func GetDashboardSkillFrequency(svc DashboardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)

		filter := dashboard.SkillFrequencyFilter{}
		if n := r.URL.Query().Get("n"); n != "" {
			if v, err := strconv.Atoi(n); err == nil && v > 0 {
				filter.Limit = v
			}
		}

		entries, err := svc.SkillFrequency(r.Context(), profileID, filter)
		if err != nil {
			RespondError(w, r, err)
			return
		}

		out := make([]skillFrequencyResponse, 0, len(entries))
		for _, e := range entries {
			out = append(out, skillFrequencyResponse{Skill: e.Skill, Count: e.Count})
		}
		respond(w, http.StatusOK, out)
	}
}

// GetDashboardSkillTrend handles GET /api/dashboard/skills/trend.
// Returns per-period skill counts scoped to the active profile.
// Optional query params: bucket (week|month, default week), n (top-N skills, default 10).
//
// AC P3-DA-2: per-period skill counts; default bucket is week.
func GetDashboardSkillTrend(svc DashboardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)

		filter := dashboard.SkillTrendFilter{}
		if b := r.URL.Query().Get("bucket"); b != "" {
			filter.Bucket = dashboard.TrendBucket(b)
		}
		if n := r.URL.Query().Get("n"); n != "" {
			if v, err := strconv.Atoi(n); err == nil && v > 0 {
				filter.Limit = v
			}
		}

		entries, err := svc.SkillTrend(r.Context(), profileID, filter)
		if err != nil {
			RespondError(w, r, err)
			return
		}

		out := make([]skillTrendResponse, 0, len(entries))
		for _, e := range entries {
			out = append(out, skillTrendResponse{
				Period: e.Period.UTC().Format(time.RFC3339),
				Skill:  e.Skill,
				Count:  e.Count,
			})
		}
		respond(w, http.StatusOK, out)
	}
}

// GetDashboardMatches handles GET /api/dashboard/matches.
// Returns jobs ordered by weighted_score; dealbreaker-failed jobs are excluded.
// Optional query param: limit (default 20).
//
// AC P3-DA-3: ordered by weighted_score, dealbreaker-failed excluded from top.
func GetDashboardMatches(svc DashboardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)

		filter := dashboard.MatchFilter{}
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 {
				filter.Limit = v
			}
		}

		matches, err := svc.TopMatches(r.Context(), profileID, filter)
		if err != nil {
			RespondError(w, r, err)
			return
		}

		out := make([]matchResponse, 0, len(matches))
		for _, m := range matches {
			skills := m.Skills
			if skills == nil {
				skills = []string{}
			}
			out = append(out, matchResponse{
				ID:                 string(m.JobID),
				Title:              m.Title,
				Company:            m.Company,
				Location:           m.Location,
				URL:                m.URL,
				Skills:             skills,
				RemotePolicy:       string(m.RemotePolicy),
				ContractType:       string(m.ContractType),
				SalaryMin:          m.SalaryMin,
				SalaryMax:          m.SalaryMax,
				WeightedScore:      m.WeightedScore,
				PassesDealbreakers: m.PassesDealbreakers,
			})
		}
		respond(w, http.StatusOK, out)
	}
}

// GetDashboardStats handles GET /api/dashboard/stats.
// Returns aggregated stats cards scoped to the active profile:
// total, new_today, new_this_week, pct_remote, avg_salary, top_contract_type.
//
// AC P3-DA-4: all five stats scoped to the active profile.
func GetDashboardStats(svc DashboardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileID, _ := ActiveProfileID(r)

		stats, err := svc.Stats(r.Context(), profileID)
		if err != nil {
			RespondError(w, r, err)
			return
		}

		respond(w, http.StatusOK, statsResponse{
			Total:           stats.Total,
			NewToday:        stats.NewToday,
			NewThisWeek:     stats.NewThisWeek,
			PctRemote:       stats.PctRemote,
			AvgSalary:       stats.AvgSalary,
			TopContractType: stats.TopContractType,
		})
	}
}
