package dashboard_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/g-trinh/job-tendencies/internal/app/dashboard"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// fakeDashboardQuery is an in-memory stub of DashboardQuery.
type fakeDashboardQuery struct {
	frequency []dashboard.SkillFrequencyEntry
	trend     []dashboard.SkillTrendEntry
	matches   []dashboard.MatchEntry
	stats     dashboard.Stats
	err       error
	// captures the last filter received for assertion.
	lastTrendFilter dashboard.SkillTrendFilter
}

func (f *fakeDashboardQuery) SkillFrequency(_ context.Context, _ kernel.ProfileID, _ dashboard.SkillFrequencyFilter) ([]dashboard.SkillFrequencyEntry, error) {
	return f.frequency, f.err
}

func (f *fakeDashboardQuery) SkillTrend(_ context.Context, _ kernel.ProfileID, filter dashboard.SkillTrendFilter) ([]dashboard.SkillTrendEntry, error) {
	f.lastTrendFilter = filter
	return f.trend, f.err
}

func (f *fakeDashboardQuery) TopMatches(_ context.Context, _ kernel.ProfileID, _ dashboard.MatchFilter) ([]dashboard.MatchEntry, error) {
	return f.matches, f.err
}

func (f *fakeDashboardQuery) Stats(_ context.Context, _ kernel.ProfileID) (dashboard.Stats, error) {
	return f.stats, f.err
}

// --- SkillFrequency ---

func TestService_SkillFrequency(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		stub    []dashboard.SkillFrequencyEntry
		stubErr error
		want    []dashboard.SkillFrequencyEntry
		wantErr bool
	}{
		{
			// AC P3-DA-1: returns ranked skill counts scoped to the active profile.
			name: "returns ranked skill counts",
			stub: []dashboard.SkillFrequencyEntry{
				{Skill: "Go", Count: 10},
				{Skill: "Docker", Count: 5},
			},
			want: []dashboard.SkillFrequencyEntry{
				{Skill: "Go", Count: 10},
				{Skill: "Docker", Count: 5},
			},
		},
		{
			name: "returns empty slice when no jobs exist",
			stub: []dashboard.SkillFrequencyEntry{},
			want: []dashboard.SkillFrequencyEntry{},
		},
		{
			name:    "propagates query error",
			stubErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			q := &fakeDashboardQuery{frequency: tc.stub, err: tc.stubErr}
			svc := dashboard.New(q)

			got, err := svc.SkillFrequency(context.Background(), "p-1", dashboard.SkillFrequencyFilter{})

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- SkillTrend ---

func TestService_SkillTrend_DefaultsBucketToWeek(t *testing.T) {
	t.Parallel()

	// AC P3-DA-2: default bucket is weeks (Open Question #3 resolved).
	q := &fakeDashboardQuery{trend: []dashboard.SkillTrendEntry{}}
	svc := dashboard.New(q)

	_, err := svc.SkillTrend(context.Background(), "p-1", dashboard.SkillTrendFilter{})

	require.NoError(t, err)
	assert.Equal(t, dashboard.TrendBucketWeek, q.lastTrendFilter.Bucket,
		"unset bucket must default to TrendBucketWeek")
}

func TestService_SkillTrend(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(24 * time.Hour)

	cases := []struct {
		name    string
		filter  dashboard.SkillTrendFilter
		stub    []dashboard.SkillTrendEntry
		stubErr error
		want    []dashboard.SkillTrendEntry
		wantErr bool
	}{
		{
			// AC P3-DA-2: returns per-period skill counts.
			name:   "returns per-period skill counts",
			filter: dashboard.SkillTrendFilter{Bucket: dashboard.TrendBucketWeek},
			stub: []dashboard.SkillTrendEntry{
				{Period: now, Skill: "Go", Count: 4},
				{Period: now, Skill: "Docker", Count: 2},
			},
			want: []dashboard.SkillTrendEntry{
				{Period: now, Skill: "Go", Count: 4},
				{Period: now, Skill: "Docker", Count: 2},
			},
		},
		{
			name:    "propagates query error",
			filter:  dashboard.SkillTrendFilter{Bucket: dashboard.TrendBucketWeek},
			stubErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			q := &fakeDashboardQuery{trend: tc.stub, err: tc.stubErr}
			svc := dashboard.New(q)

			got, err := svc.SkillTrend(context.Background(), "p-1", tc.filter)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- TopMatches ---

func TestService_TopMatches(t *testing.T) {
	t.Parallel()

	score := 0.85
	passes := true

	cases := []struct {
		name    string
		stub    []dashboard.MatchEntry
		stubErr error
		want    []dashboard.MatchEntry
		wantErr bool
	}{
		{
			// AC P3-DA-3: returns jobs ordered by weighted_score.
			name: "returns matches ordered by weighted score",
			stub: []dashboard.MatchEntry{
				{JobID: "j-1", Title: "Senior Go", WeightedScore: &score, PassesDealbreakers: &passes},
			},
			want: []dashboard.MatchEntry{
				{JobID: "j-1", Title: "Senior Go", WeightedScore: &score, PassesDealbreakers: &passes},
			},
		},
		{
			name: "returns empty list when no scored jobs exist",
			stub: []dashboard.MatchEntry{},
			want: []dashboard.MatchEntry{},
		},
		{
			name:    "propagates query error",
			stubErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			q := &fakeDashboardQuery{matches: tc.stub, err: tc.stubErr}
			svc := dashboard.New(q)

			got, err := svc.TopMatches(context.Background(), "p-1", dashboard.MatchFilter{})

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- Stats ---

func TestService_Stats(t *testing.T) {
	t.Parallel()

	avgSalary := 45000.0

	cases := []struct {
		name    string
		stub    dashboard.Stats
		stubErr error
		want    dashboard.Stats
		wantErr bool
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
			want: dashboard.Stats{
				Total:           100,
				NewToday:        5,
				NewThisWeek:     20,
				PctRemote:       0.35,
				AvgSalary:       &avgSalary,
				TopContractType: "CDI",
			},
		},
		{
			name: "returns zero stats when no jobs exist",
			stub: dashboard.Stats{},
			want: dashboard.Stats{},
		},
		{
			name:    "propagates query error",
			stubErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			q := &fakeDashboardQuery{stats: tc.stub, err: tc.stubErr}
			svc := dashboard.New(q)

			got, err := svc.Stats(context.Background(), "p-1")

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
