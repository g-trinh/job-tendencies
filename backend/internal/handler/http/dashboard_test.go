package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/app/dashboard"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// fakeDashboardSvc is an in-memory stub of handler.DashboardService.
type fakeDashboardSvc struct {
	frequency []dashboard.SkillFrequencyEntry
	trend     []dashboard.SkillTrendEntry
	matches   []dashboard.MatchEntry
	stats     dashboard.Stats
	err       error
}

func (f *fakeDashboardSvc) SkillFrequency(_ context.Context, _ kernel.ProfileID, _ dashboard.SkillFrequencyFilter) ([]dashboard.SkillFrequencyEntry, error) {
	return f.frequency, f.err
}

func (f *fakeDashboardSvc) SkillTrend(_ context.Context, _ kernel.ProfileID, _ dashboard.SkillTrendFilter) ([]dashboard.SkillTrendEntry, error) {
	return f.trend, f.err
}

func (f *fakeDashboardSvc) TopMatches(_ context.Context, _ kernel.ProfileID, _ dashboard.MatchFilter) ([]dashboard.MatchEntry, error) {
	return f.matches, f.err
}

func (f *fakeDashboardSvc) Stats(_ context.Context, _ kernel.ProfileID) (dashboard.Stats, error) {
	return f.stats, f.err
}

// newDashboardRouter builds a chi router wired with the given fake service and
// injects a fixed profile ID so scoped routes work without HTTP middleware.
func newDashboardRouter(svc *fakeDashboardSvc) *chi.Mux {
	r := handler.NewRouter(slog.Default())
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := handler.WithActiveProfileID(r.Context(), "p-1")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Get("/api/dashboard/skills/frequency", handler.GetDashboardSkillFrequency(svc))
	r.Get("/api/dashboard/skills/trend", handler.GetDashboardSkillTrend(svc))
	r.Get("/api/dashboard/matches", handler.GetDashboardMatches(svc))
	r.Get("/api/dashboard/stats", handler.GetDashboardStats(svc))
	return r
}

// --- GET /api/dashboard/skills/frequency ---

func TestGetDashboardSkillFrequency(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		stub       []dashboard.SkillFrequencyEntry
		stubErr    error
		query      string
		wantStatus int
		wantLen    int
	}{
		{
			// AC P3-DA-1: returns ranked skill counts scoped to the active profile.
			name: "returns ranked skill counts",
			stub: []dashboard.SkillFrequencyEntry{
				{Skill: "Go", Count: 10},
				{Skill: "Docker", Count: 5},
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "returns empty array when no skills",
			stub:       []dashboard.SkillFrequencyEntry{},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "returns 500 on service error",
			stubErr:    errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeDashboardSvc{frequency: tc.stub, err: tc.stubErr}
			r := newDashboardRouter(svc)

			url := "/api/dashboard/skills/frequency"
			if tc.query != "" {
				url += "?" + tc.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}

			var got []map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decoding response: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len = %d; want %d", len(got), tc.wantLen)
			}
			if tc.wantLen > 0 {
				if got[0]["skill"] == nil {
					t.Error("first entry missing 'skill' field")
				}
				if got[0]["count"] == nil {
					t.Error("first entry missing 'count' field")
				}
			}
		})
	}
}

// --- GET /api/dashboard/skills/trend ---

func TestGetDashboardSkillTrend(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(24 * time.Hour)

	cases := []struct {
		name       string
		stub       []dashboard.SkillTrendEntry
		stubErr    error
		query      string
		wantStatus int
		wantLen    int
	}{
		{
			// AC P3-DA-2: returns per-period skill counts.
			name: "returns per-period skill counts",
			stub: []dashboard.SkillTrendEntry{
				{Period: now, Skill: "Go", Count: 4},
				{Period: now, Skill: "Kubernetes", Count: 2},
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "returns empty array when no data",
			stub:       []dashboard.SkillTrendEntry{},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "accepts bucket=month query param",
			stub:       []dashboard.SkillTrendEntry{{Period: now, Skill: "Go", Count: 10}},
			query:      "bucket=month",
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "returns 500 on service error",
			stubErr:    errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeDashboardSvc{trend: tc.stub, err: tc.stubErr}
			r := newDashboardRouter(svc)

			url := "/api/dashboard/skills/trend"
			if tc.query != "" {
				url += "?" + tc.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}

			var got []map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decoding response: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len = %d; want %d", len(got), tc.wantLen)
			}
			if tc.wantLen > 0 {
				if got[0]["period"] == nil {
					t.Error("first entry missing 'period' field")
				}
				if got[0]["skill"] == nil {
					t.Error("first entry missing 'skill' field")
				}
				if got[0]["count"] == nil {
					t.Error("first entry missing 'count' field")
				}
			}
		})
	}
}

// --- GET /api/dashboard/matches ---

func TestGetDashboardMatches(t *testing.T) {
	t.Parallel()

	score := 0.85
	passes := true

	cases := []struct {
		name       string
		stub       []dashboard.MatchEntry
		stubErr    error
		wantStatus int
		wantLen    int
	}{
		{
			// AC P3-DA-3: returns jobs ordered by weighted_score,
			// dealbreaker-failed excluded from top.
			name: "returns scored jobs ordered by weighted score",
			stub: []dashboard.MatchEntry{
				{
					JobID: "j-1", Title: "Senior Go Dev",
					WeightedScore: &score, PassesDealbreakers: &passes,
					Skills: []string{"Go"},
				},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "returns empty array when no scored jobs",
			stub:       []dashboard.MatchEntry{},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "returns 500 on service error",
			stubErr:    errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeDashboardSvc{matches: tc.stub, err: tc.stubErr}
			r := newDashboardRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/matches", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}

			var got []map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decoding response: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len = %d; want %d", len(got), tc.wantLen)
			}
			if tc.wantLen > 0 {
				if got[0]["id"] == nil {
					t.Error("first entry missing 'id' field")
				}
				if got[0]["weighted_score"] == nil {
					t.Error("first entry missing 'weighted_score' field")
				}
				if got[0]["passes_dealbreakers"] == nil {
					t.Error("first entry missing 'passes_dealbreakers' field")
				}
			}
		})
	}
}

// --- GET /api/dashboard/stats ---

func TestGetDashboardStats(t *testing.T) {
	t.Parallel()

	avgSalary := 45000.0

	cases := []struct {
		name             string
		stub             dashboard.Stats
		stubErr          error
		wantStatus       int
		wantTotal        float64
		wantContractType string
	}{
		{
			// AC P3-DA-4: returns all five stats scoped to the active profile.
			name: "returns all five stats",
			stub: dashboard.Stats{
				Total:           100,
				NewToday:        5,
				NewThisWeek:     20,
				PctRemote:       0.35,
				AvgSalary:       &avgSalary,
				TopContractType: "CDI",
			},
			wantStatus:       http.StatusOK,
			wantTotal:        100,
			wantContractType: "CDI",
		},
		{
			name:       "returns zero stats when profile has no jobs",
			stub:       dashboard.Stats{},
			wantStatus: http.StatusOK,
			wantTotal:  0,
		},
		{
			name:       "returns 500 on service error",
			stubErr:    errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeDashboardSvc{stats: tc.stub, err: tc.stubErr}
			r := newDashboardRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/stats", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}

			var got map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decoding response: %v", err)
			}

			if got["total"] != tc.wantTotal {
				t.Errorf("total = %v; want %v", got["total"], tc.wantTotal)
			}
			if tc.wantContractType != "" {
				if got["top_contract_type"] != tc.wantContractType {
					t.Errorf("top_contract_type = %v; want %v", got["top_contract_type"], tc.wantContractType)
				}
			}
			// Verify all five fields are present in the response.
			for _, field := range []string{"total", "new_today", "new_this_week", "pct_remote", "top_contract_type"} {
				if _, ok := got[field]; !ok {
					t.Errorf("response missing field %q", field)
				}
			}
		})
	}
}
