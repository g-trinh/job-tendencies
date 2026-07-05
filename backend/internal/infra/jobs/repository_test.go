package jobs

import (
	"strings"
	"testing"

	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// TestBuildJobListFilterQuery_SharedWhereClause verifies the ADR-007 load-bearing rule
// that the list and count queries in ListByProfile are built from the identical
// WHERE-clause fragment (no copy-paste drift): buildJobListFilterQuery is the single
// place conditions and args are assembled, and both queries consume its output as-is.
func TestBuildJobListFilterQuery_SharedWhereClause(t *testing.T) {
	t.Parallel()

	confidenceMin := 70
	filter := appjobs.JobListFilter{ConfidenceMin: &confidenceMin, Page: 2, PageSize: 25}

	fq := buildJobListFilterQuery(kernel.ProfileID("p-1"), filter)

	if !strings.Contains(fq.whereClause, "j.understanding_score >=") {
		t.Fatalf("whereClause = %q; want confidence_min applied in SQL (ADR-007 must_not: never post-filter in Go)", fq.whereClause)
	}
	// The confidence_min bind arg must be present alongside the profile scoping arg —
	// count and list queries pass the exact same fq.args to their WHERE clause.
	if len(fq.args) != 2 {
		t.Fatalf("args = %v; want 2 args (profile id + confidence_min)", fq.args)
	}
	if fq.args[0] != "p-1" || fq.args[1] != confidenceMin {
		t.Fatalf("args = %v; want [p-1 %d]", fq.args, confidenceMin)
	}
}

// TestBuildJobListFilterQuery_ConditionsPerFilterField verifies each optional filter
// field contributes its own WHERE fragment, and is absent when the field is unset.
func TestBuildJobListFilterQuery_ConditionsPerFilterField(t *testing.T) {
	t.Parallel()

	remotePolicy := "hybrid"
	boardID := "b-1"

	cases := []struct {
		name       string
		filter     appjobs.JobListFilter
		wantInSQL  []string
		wantNotSQL []string
	}{
		{
			name:       "no optional filters",
			filter:     appjobs.JobListFilter{},
			wantInSQL:  []string{"rl.profile_id = $1"},
			wantNotSQL: []string{"js.board_id", "j.remote_policy", "j.understanding_score"},
		},
		{
			name:      "remote policy and board filters",
			filter:    appjobs.JobListFilter{RemotePolicy: remotePolicy, BoardID: boardID},
			wantInSQL: []string{"j.remote_policy = $2", "js.board_id = $3"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fq := buildJobListFilterQuery(kernel.ProfileID("p-1"), tc.filter)

			for _, want := range tc.wantInSQL {
				if !strings.Contains(fq.whereClause, want) {
					t.Errorf("whereClause = %q; want it to contain %q", fq.whereClause, want)
				}
			}
			for _, notWant := range tc.wantNotSQL {
				if strings.Contains(fq.whereClause, notWant) {
					t.Errorf("whereClause = %q; want it NOT to contain %q", fq.whereClause, notWant)
				}
			}
		})
	}
}

// TestBuildJobListQuery_TieBreakAndPagination verifies the list query's deterministic
// ORDER BY tie-break and LIMIT/OFFSET wiring (ADR-007): rows never straddle two pages
// under equal sort keys, and paging math is (page-1)*page_size.
func TestBuildJobListQuery_TieBreakAndPagination(t *testing.T) {
	t.Parallel()

	fq := buildJobListFilterQuery(kernel.ProfileID("p-1"), appjobs.JobListFilter{})

	query, args := buildJobListQuery(fq, 25, 3)

	if !strings.Contains(query, "SELECT DISTINCT j.id") {
		t.Errorf("query = %q; want SELECT DISTINCT j.id (job_source join fans out)", query)
	}
	if !strings.Contains(query, "ORDER BY j.first_seen DESC, j.id DESC") {
		t.Errorf("query = %q; want a deterministic `, j.id DESC` tie-break after the sort column", query)
	}
	if !strings.Contains(query, "LIMIT $2 OFFSET $3") {
		t.Errorf("query = %q; want LIMIT/OFFSET placeholders appended after the shared filter args", query)
	}
	wantArgs := []any{"p-1", 25, 50} // offset = (page 3 - 1) * page_size 25
	if len(args) != len(wantArgs) || args[1] != wantArgs[1] || args[2] != wantArgs[2] {
		t.Errorf("args = %v; want %v", args, wantArgs)
	}
}

// TestBuildJobCountQuery_UsesCountDistinctOverSharedWhere verifies the ADR-007
// must_not rule: the total is COUNT(DISTINCT j.id), never COUNT(*), computed over the
// exact same WHERE clause the list query uses — never a separately-built condition set.
func TestBuildJobCountQuery_UsesCountDistinctOverSharedWhere(t *testing.T) {
	t.Parallel()

	confidenceMin := 80
	filter := appjobs.JobListFilter{ConfidenceMin: &confidenceMin}
	fq := buildJobListFilterQuery(kernel.ProfileID("p-1"), filter)

	countQuery := buildJobCountQuery(fq)
	listQuery, _ := buildJobListQuery(fq, 25, 1)

	if !strings.Contains(countQuery, "COUNT(DISTINCT j.id)") {
		t.Fatalf("countQuery = %q; want COUNT(DISTINCT j.id)", countQuery)
	}
	if strings.Contains(countQuery, "COUNT(*)") {
		t.Fatalf("countQuery = %q; must not use COUNT(*) (over-counts multi-source jobs)", countQuery)
	}
	if countQuery == listQuery {
		t.Fatalf("count and list queries must differ in SELECT/ORDER/LIMIT, not just be equal")
	}
	// Both queries must be built from the identical WHERE fragment — assert the exact
	// substring appears in both rather than re-deriving it, so the test fails the
	// moment the two queries' WHERE clauses drift apart.
	whereFragment := "WHERE " + fq.whereClause
	if !strings.Contains(countQuery, whereFragment) {
		t.Errorf("countQuery = %q; want it to contain %q", countQuery, whereFragment)
	}
	if !strings.Contains(listQuery, whereFragment) {
		t.Errorf("listQuery = %q; want it to contain %q", listQuery, whereFragment)
	}
}

// TestBuildJobListFilterQuery_OrderColumn verifies the sort-mode → ORDER BY column
// mapping (date is the default, salary opts into the salary column).
func TestBuildJobListFilterQuery_OrderColumn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		sort         string
		sortDir      string
		wantOrderCol string
		wantOrderDir string
	}{
		{name: "defaults to first_seen desc", wantOrderCol: "j.first_seen", wantOrderDir: "DESC"},
		{name: "salary sort", sort: "salary", wantOrderCol: "j.salary_min NULLS LAST", wantOrderDir: "DESC"},
		{name: "asc direction", sortDir: "asc", wantOrderCol: "j.first_seen", wantOrderDir: "ASC"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fq := buildJobListFilterQuery(kernel.ProfileID("p-1"),
				appjobs.JobListFilter{Sort: tc.sort, SortDir: tc.sortDir})

			if fq.orderCol != tc.wantOrderCol {
				t.Errorf("orderCol = %q; want %q", fq.orderCol, tc.wantOrderCol)
			}
			if fq.orderDir != tc.wantOrderDir {
				t.Errorf("orderDir = %q; want %q", fq.orderDir, tc.wantOrderDir)
			}
		})
	}
}
