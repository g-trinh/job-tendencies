package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// fakeJobService is an in-memory fake of handler.JobReader and
// handler.JobApplicationUpdater.
type fakeJobService struct {
	jobs map[kernel.JobID]appjobs.JobView
	err  error
	apps map[kernel.JobID]kernel.ApplicationStatus
}

func newFakeJobService() *fakeJobService {
	return &fakeJobService{
		jobs: make(map[kernel.JobID]appjobs.JobView),
		apps: make(map[kernel.JobID]kernel.ApplicationStatus),
	}
}

func (f *fakeJobService) ListJobs(_ context.Context, _ kernel.ProfileID, _ appjobs.JobListFilter) ([]appjobs.JobView, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]appjobs.JobView, 0, len(f.jobs))
	for _, j := range f.jobs {
		out = append(out, j)
	}
	return out, nil
}

func (f *fakeJobService) GetJob(_ context.Context, _ kernel.ProfileID, id kernel.JobID) (appjobs.JobView, error) {
	if f.err != nil {
		return appjobs.JobView{}, f.err
	}
	j, ok := f.jobs[id]
	if !ok {
		return appjobs.JobView{}, &kernel.NotFoundError{Kind: "job", ID: string(id)}
	}
	return j, nil
}

func (f *fakeJobService) SetApplicationStatus(_ context.Context, _ kernel.ProfileID, id kernel.JobID, status kernel.ApplicationStatus) (appjobs.ApplicationResult, error) {
	if f.err != nil {
		return appjobs.ApplicationResult{}, f.err
	}
	if _, ok := f.jobs[id]; !ok {
		return appjobs.ApplicationResult{}, &kernel.NotFoundError{Kind: "job", ID: string(id)}
	}
	f.apps[id] = status
	return appjobs.ApplicationResult{Status: status, UpdatedAt: time.Now()}, nil
}

func newJobRouter(svc *fakeJobService) *chi.Mux {
	r := handler.NewRouter(slog.Default())
	// Middleware injects a fake profile id so scoped routes work without HTTP middleware.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := handler.WithActiveProfileID(r.Context(), "p-1")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Get("/api/jobs", handler.ListJobs(svc))
	r.Get("/api/jobs/{id}", handler.GetJob(svc))
	r.Get("/api/jobs/{id}/original", handler.GetJobOriginal(svc))
	r.Patch("/api/jobs/{id}/application", handler.PatchJobApplication(svc))
	return r
}

func seedJob(svc *fakeJobService) appjobs.JobView {
	j := appjobs.JobView{
		ID:      "j-1",
		Title:   "Go Engineer",
		Company: "Acme",
		URL:     "https://example.com/jobs/1",
		Skills:  []string{"go"},
		Sources: []appjobs.JobSourceView{
			{BoardID: "b-1", SourceURL: "https://wttj.co/j1", BoardName: "WTTJ"},
		},
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
	}
	svc.jobs["j-1"] = j
	return j
}

// AC: sources shape includes board_name per FE contract.

func TestListJobs_SourcesHaveBoardName(t *testing.T) {
	t.Parallel()

	svc := newFakeJobService()
	seedJob(svc)
	r := newJobRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp []struct {
		Sources []struct {
			BoardName string `json:"board_name"`
		} `json:"sources"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) == 0 {
		t.Fatal("expected at least one job")
	}
	if got := resp[0].Sources[0].BoardName; got != "WTTJ" {
		t.Errorf("board_name = %q; want %q", got, "WTTJ")
	}
}

// AC: list item includes application_status (null when not set) and first_seen.

func TestListJobs_ApplicationStatusAndFirstSeen(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		appStatus         *string // pre-seeded application status, nil = not set
		wantAppStatusNull bool
	}{
		{
			name:              "application_status is null when no application",
			wantAppStatusNull: true,
		},
		{
			name:              "application_status is present when set",
			appStatus:         ptr("saved"),
			wantAppStatusNull: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newFakeJobService()
			j := seedJob(svc)
			if tc.appStatus != nil {
				s := kernel.ApplicationStatus(*tc.appStatus)
				svc.jobs["j-1"] = func() appjobs.JobView {
					j.ApplicationStatus = &s
					return j
				}()
			}
			r := newJobRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; want 200", rec.Code)
			}

			var resp []map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decoding: %v", err)
			}
			if len(resp) == 0 {
				t.Fatal("expected one job")
			}
			item := resp[0]
			_, hasFirstSeen := item["first_seen"]
			if !hasFirstSeen {
				t.Error("first_seen missing from response")
			}
			appStatusVal, hasAppStatus := item["application_status"]
			if !hasAppStatus {
				t.Error("application_status missing from response")
			}
			if tc.wantAppStatusNull && appStatusVal != nil {
				t.Errorf("application_status = %v; want null", appStatusVal)
			}
			if !tc.wantAppStatusNull && appStatusVal == nil {
				t.Error("application_status = null; want a value")
			}
		})
	}
}

// AC: GET /api/jobs/{id}/original redirects to the job's URL.

func TestGetJobOriginal_Redirects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		id         string
		wantStatus int
		wantLoc    string
	}{
		{
			name:       "redirects to source URL",
			id:         "j-1",
			wantStatus: http.StatusFound,
			wantLoc:    "https://example.com/jobs/1",
		},
		{
			name:       "returns 404 for unknown job",
			id:         "unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newFakeJobService()
			seedJob(svc)
			r := newJobRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/jobs/"+tc.id+"/original", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantLoc != "" {
				if got := rec.Header().Get("Location"); got != tc.wantLoc {
					t.Errorf("Location = %q; want %q", got, tc.wantLoc)
				}
			}
		})
	}
}

// AC: PATCH /api/jobs/{id}/application returns {status, updated_at} on valid status.

func TestPatchJobApplication(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		id         string
		body       string
		wantStatus int
		wantAppSt  string
	}{
		{
			name:       "sets application status to saved",
			id:         "j-1",
			body:       `{"status":"saved"}`,
			wantStatus: http.StatusOK,
			wantAppSt:  "saved",
		},
		{
			name:       "returns 400 for invalid status",
			id:         "j-1",
			body:       `{"status":"unknown"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns 404 for unknown job",
			id:         "no-job",
			body:       `{"status":"applied"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "returns 400 for missing status field",
			id:         "j-1",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newFakeJobService()
			seedJob(svc)
			r := newJobRouter(svc)

			req := httptest.NewRequest(http.MethodPatch, "/api/jobs/"+tc.id+"/application",
				strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantAppSt != "" {
				var resp map[string]string
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("decoding: %v", err)
				}
				if got := resp["status"]; got != tc.wantAppSt {
					t.Errorf("status = %q; want %q", got, tc.wantAppSt)
				}
				if resp["updated_at"] == "" {
					t.Error("updated_at missing from response")
				}
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

// TestListJobs_FilterQueryParams verifies that filter params are parsed without error.
func TestListJobs_FilterQueryParams(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{
			name:       "returns 200 with no filters",
			query:      "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns 200 with skills filter",
			query:      "?skills[]=go&skills[]=postgres",
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns 200 with salary filter",
			query:      "?salary_min=50000&salary_max=100000",
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns 200 with sort params",
			query:      "?sort=salary&sort_dir=asc",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newFakeJobService()
			seedJob(svc)
			r := newJobRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/jobs"+tc.query, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}
