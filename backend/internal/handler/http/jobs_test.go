package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	jobs           map[kernel.JobID]appjobs.JobView
	err            error
	apps           map[kernel.JobID]kernel.ApplicationStatus
	reextractCalls []kernel.JobID
	reextractErr   error
	total          int // override for the reported total; 0 means "use len(jobs)"
	lastFilter     appjobs.JobListFilter
}

func newFakeJobService() *fakeJobService {
	return &fakeJobService{
		jobs: make(map[kernel.JobID]appjobs.JobView),
		apps: make(map[kernel.JobID]kernel.ApplicationStatus),
	}
}

func (f *fakeJobService) ListJobs(_ context.Context, _ kernel.ProfileID, filter appjobs.JobListFilter) (appjobs.JobListResult, error) {
	f.lastFilter = filter
	if f.err != nil {
		return appjobs.JobListResult{}, f.err
	}
	out := make([]appjobs.JobView, 0, len(f.jobs))
	for _, j := range f.jobs {
		out = append(out, j)
	}
	total := len(out)
	if f.total != 0 {
		total = f.total
	}
	page, pageSize := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = len(out)
	}
	start := (page - 1) * pageSize
	if start > len(out) {
		start = len(out)
	}
	end := start + pageSize
	if end > len(out) {
		end = len(out)
	}
	return appjobs.JobListResult{Items: out[start:end], Total: total}, nil
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

func (f *fakeJobService) ReextractJob(_ context.Context, _ kernel.ProfileID, id kernel.JobID) error {
	if f.reextractErr != nil {
		return f.reextractErr
	}
	f.reextractCalls = append(f.reextractCalls, id)
	return nil
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
	r.Post("/api/jobs/{id}/reextract", handler.PostJobReextract(svc))
	return r
}

// TestPostJobReextract_PublishesAndReturns202 verifies P5-4: a successful reextract
// request delegates to the service with the job id and returns 202 Accepted (the
// actual re-extraction happens asynchronously in extract-worker).
func TestPostJobReextract_PublishesAndReturns202(t *testing.T) {
	t.Parallel()

	svc := newFakeJobService()
	seedJob(svc)
	r := newJobRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/jobs/j-1/reextract", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d; want 202 (body: %s)", rec.Code, rec.Body.String())
	}
	if len(svc.reextractCalls) != 1 || svc.reextractCalls[0] != "j-1" {
		t.Fatalf("reextractCalls = %v, want [j-1]", svc.reextractCalls)
	}
}

// TestPostJobReextract_NotFoundPropagates verifies a service-level not-found error
// (job has no sources visible to this profile) surfaces as 404, not a 202.
func TestPostJobReextract_NotFoundPropagates(t *testing.T) {
	t.Parallel()

	svc := newFakeJobService()
	svc.reextractErr = &kernel.NotFoundError{Kind: "job", ID: "missing"}
	r := newJobRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/jobs/missing/reextract", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404 (body: %s)", rec.Code, rec.Body.String())
	}
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

	var resp struct {
		Items []struct {
			Sources []struct {
				BoardName string `json:"board_name"`
			} `json:"sources"`
		} `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected at least one job")
	}
	if got := resp.Items[0].Sources[0].BoardName; got != "WTTJ" {
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

			var resp struct {
				Items []map[string]any `json:"items"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decoding: %v", err)
			}
			if len(resp.Items) == 0 {
				t.Fatal("expected one job")
			}
			item := resp.Items[0]
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

// AC: include_expired defaults to false (excludes expired jobs) and only becomes true
// when the query param explicitly parses as boolean true; absent, "false", or garbage
// values all default to false rather than a 400 (mirrors the ADR-007 clamping style).
func TestListJobs_IncludeExpiredParsing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "absent defaults to false", query: "", want: false},
		{name: "explicit true", query: "?include_expired=true", want: true},
		{name: "explicit false", query: "?include_expired=false", want: false},
		{name: "garbage defaults to false", query: "?include_expired=notabool", want: false},
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

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; want 200 (body: %s)", rec.Code, rec.Body.String())
			}
			if got := svc.lastFilter.IncludeExpired; got != tc.want {
				t.Errorf("IncludeExpired = %v; want %v", got, tc.want)
			}
		})
	}
}

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

// jobListEnvelope mirrors the ADR-007 response shape for decoding in tests.
type jobListEnvelope struct {
	Items      []map[string]any `json:"items"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	Total      int              `json:"total"`
	TotalPages int              `json:"total_pages"`
}

// seedJobs adds n distinct jobs to the fake service.
func seedJobs(svc *fakeJobService, n int) {
	for i := 0; i < n; i++ {
		id := kernel.JobID(fmt.Sprintf("j-%d", i))
		svc.jobs[id] = appjobs.JobView{
			ID:        id,
			Title:     "Go Engineer",
			Company:   "Acme",
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
		}
	}
}

// AC: GET /api/jobs returns the ADR-007 paginated envelope, not a bare array.

func TestListJobs_ReturnsPaginatedEnvelope(t *testing.T) {
	t.Parallel()

	svc := newFakeJobService()
	seedJobs(svc, 3)
	r := newJobRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp jobListEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding envelope: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Errorf("items = %d; want 3", len(resp.Items))
	}
	if resp.Page != 1 {
		t.Errorf("page = %d; want default 1", resp.Page)
	}
	if resp.PageSize != 25 {
		t.Errorf("page_size = %d; want default 25", resp.PageSize)
	}
	if resp.Total != 3 {
		t.Errorf("total = %d; want 3", resp.Total)
	}
	if resp.TotalPages != 1 {
		t.Errorf("total_pages = %d; want 1", resp.TotalPages)
	}
}

// AC: page and page_size are clamped rather than rejected with a 400 (ADR-007).

func TestListJobs_ClampsPageAndPageSize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		query        string
		wantPage     int
		wantPageSize int
	}{
		{name: "page below 1 clamps to 1", query: "?page=-5", wantPage: 1, wantPageSize: 25},
		{name: "page non-numeric defaults to 1", query: "?page=abc", wantPage: 1, wantPageSize: 25},
		{name: "page_size above 100 clamps to 100", query: "?page_size=500", wantPage: 1, wantPageSize: 100},
		{name: "page_size below 1 clamps to 1", query: "?page_size=0", wantPage: 1, wantPageSize: 1},
		{name: "page_size non-numeric defaults to 25", query: "?page_size=abc", wantPage: 1, wantPageSize: 25},
		{name: "valid page and page_size pass through", query: "?page=2&page_size=10", wantPage: 2, wantPageSize: 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newFakeJobService()
			seedJobs(svc, 30)
			r := newJobRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/jobs"+tc.query, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; want 200 (body: %s)", rec.Code, rec.Body.String())
			}

			var resp jobListEnvelope
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decoding envelope: %v", err)
			}
			if resp.Page != tc.wantPage {
				t.Errorf("page = %d; want %d", resp.Page, tc.wantPage)
			}
			if resp.PageSize != tc.wantPageSize {
				t.Errorf("page_size = %d; want %d", resp.PageSize, tc.wantPageSize)
			}
		})
	}
}

// AC: a page past the end returns empty items with the real total (ADR-007), not an
// error, so the frontend can recover (e.g. clamp back to the last page).

func TestListJobs_PagePastEnd_ReturnsEmptyItemsWithTrueTotal(t *testing.T) {
	t.Parallel()

	svc := newFakeJobService()
	seedJobs(svc, 3)
	r := newJobRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs?page=5&page_size=25", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp jobListEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding envelope: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("items = %d; want 0 (page past the end)", len(resp.Items))
	}
	if resp.Total != 3 {
		t.Errorf("total = %d; want 3 (the true total, not 0)", resp.Total)
	}
}

// AC: total_pages is ceil(total/page_size), and 0 when the profile has no jobs at all.

func TestListJobs_TotalPagesIsZeroWhenNoJobs(t *testing.T) {
	t.Parallel()

	svc := newFakeJobService()
	r := newJobRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp jobListEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding envelope: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("total = %d; want 0", resp.Total)
	}
	if resp.TotalPages != 0 {
		t.Errorf("total_pages = %d; want 0 when total is 0", resp.TotalPages)
	}
}
