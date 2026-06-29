// Package dashboard contains the dashboard read use cases: skills frequency,
// skills trend, top matches, and stats cards. All queries are scoped to the
// active profile. The dashboard is a read-model context (ADR-001) and never
// writes to the domain; it delegates every query to the DashboardQuery port
// implemented by infra/dashboard.
package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// TrendBucket is the time granularity for the skills-trend query.
//
// Default is TrendBucketWeek (Open Question #3 resolved: weeks are chosen
// because a typical active job search spans weeks, making per-week deltas more
// actionable than per-month aggregates and less noisy than per-day counts).
type TrendBucket string

const (
	// TrendBucketWeek groups skill counts by ISO calendar week (server default).
	TrendBucketWeek TrendBucket = "week"
	// TrendBucketMonth groups skill counts by calendar month.
	TrendBucketMonth TrendBucket = "month"
)

// SkillFrequencyEntry is one ranked skill with its occurrence count across
// all jobs in the active profile's listing set.
type SkillFrequencyEntry struct {
	Skill string
	Count int
}

// SkillTrendEntry records the frequency of one skill within one time bucket.
type SkillTrendEntry struct {
	Period time.Time
	Skill  string
	Count  int
}

// MatchEntry is one scored job in the top-matches list. WeightedScore and
// PassesDealbreakers are nil for jobs that have not yet been scored.
type MatchEntry struct {
	JobID              kernel.JobID
	Title              string
	Company            string
	Location           string
	URL                string
	Skills             []string
	RemotePolicy       kernel.RemotePolicy
	ContractType       kernel.ContractType
	SalaryMin          *int64
	SalaryMax          *int64
	WeightedScore      *float64
	PassesDealbreakers *bool
}

// Stats holds the five dashboard aggregated metrics scoped to the active profile.
type Stats struct {
	// Total is the count of distinct jobs scraped for this profile.
	Total int
	// NewToday is the count of jobs first seen on today's UTC calendar date.
	NewToday int
	// NewThisWeek is the count of jobs first seen within the current ISO week.
	NewThisWeek int
	// PctRemote is the fraction (0–1) of jobs advertising full-remote policy.
	PctRemote float64
	// AvgSalary is the mean salary_min across jobs where it is non-null.
	// Nil when no job in the profile set has salary data.
	AvgSalary *float64
	// TopContractType is the most frequently advertised contract type.
	// Empty when no jobs exist.
	TopContractType string
}

// SkillFrequencyFilter controls top-N truncation for the frequency endpoint.
// A zero Limit falls back to the server default of 20 skills.
type SkillFrequencyFilter struct {
	Limit int
}

// SkillTrendFilter controls bucket granularity and the number of top skills
// included per bucket.
type SkillTrendFilter struct {
	// Bucket is the time granularity; defaults to TrendBucketWeek when unset.
	Bucket TrendBucket
	// Limit is the number of top skills to include across all buckets.
	// 0 uses the server default of 10 skills.
	Limit int
}

// MatchFilter controls the maximum number of top matches returned.
// A zero Limit falls back to the server default of 20 matches.
type MatchFilter struct {
	Limit int
}

// DashboardQuery is the consumer interface for dashboard read queries. It is
// defined here (consumer package, ADR-001) and satisfied by infra/dashboard.
// All methods are scoped to the active profile.
type DashboardQuery interface {
	// SkillFrequency returns the top-N skills ranked by occurrence count.
	SkillFrequency(ctx context.Context, profileID kernel.ProfileID, filter SkillFrequencyFilter) ([]SkillFrequencyEntry, error)
	// SkillTrend returns per-bucket skill counts for the most frequent skills.
	SkillTrend(ctx context.Context, profileID kernel.ProfileID, filter SkillTrendFilter) ([]SkillTrendEntry, error)
	// TopMatches returns jobs ordered by weighted_score (desc), excluding jobs
	// that fail the dealbreaker gate.
	TopMatches(ctx context.Context, profileID kernel.ProfileID, filter MatchFilter) ([]MatchEntry, error)
	// Stats returns the five aggregated stats-card metrics.
	Stats(ctx context.Context, profileID kernel.ProfileID) (Stats, error)
}

// Service exposes dashboard read use cases to the HTTP API layer.
type Service struct {
	query DashboardQuery
}

// New constructs a dashboard Service wired with the given query adapter.
func New(query DashboardQuery) *Service {
	return &Service{query: query}
}

// SkillFrequency returns top-N skills by occurrence count across the profile's jobs.
func (s *Service) SkillFrequency(ctx context.Context, profileID kernel.ProfileID, filter SkillFrequencyFilter) ([]SkillFrequencyEntry, error) {
	out, err := s.query.SkillFrequency(ctx, profileID, filter)
	if err != nil {
		return nil, fmt.Errorf("skill frequency for profile %q: %w", profileID, err)
	}
	return out, nil
}

// SkillTrend returns per-bucket skill counts for the active profile's jobs.
// When filter.Bucket is empty it defaults to TrendBucketWeek.
func (s *Service) SkillTrend(ctx context.Context, profileID kernel.ProfileID, filter SkillTrendFilter) ([]SkillTrendEntry, error) {
	if filter.Bucket == "" {
		filter.Bucket = TrendBucketWeek
	}
	out, err := s.query.SkillTrend(ctx, profileID, filter)
	if err != nil {
		return nil, fmt.Errorf("skill trend for profile %q: %w", profileID, err)
	}
	return out, nil
}

// TopMatches returns jobs ranked by weighted_score, excluding dealbreaker failures.
// Only jobs with a job_score row (i.e. that have been scored) are returned.
func (s *Service) TopMatches(ctx context.Context, profileID kernel.ProfileID, filter MatchFilter) ([]MatchEntry, error) {
	out, err := s.query.TopMatches(ctx, profileID, filter)
	if err != nil {
		return nil, fmt.Errorf("top matches for profile %q: %w", profileID, err)
	}
	return out, nil
}

// Stats returns the aggregated dashboard metrics for the active profile.
func (s *Service) Stats(ctx context.Context, profileID kernel.ProfileID) (Stats, error) {
	out, err := s.query.Stats(ctx, profileID)
	if err != nil {
		return Stats{}, fmt.Errorf("stats for profile %q: %w", profileID, err)
	}
	return out, nil
}
