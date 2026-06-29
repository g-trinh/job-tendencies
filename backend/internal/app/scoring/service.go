// Package scoring contains the scoring application service. It orchestrates the
// fit-score pipeline: read job facts, read profile criteria, run the dealbreaker
// gate and the weighted scorer (domain/scoring), then persist the result. The
// scoring context is consumed by the extraction worker (P3-EX-4) and the dashboard.
// No HTTP handler is exposed — scoring is an internal pipeline step, not a public
// CRUD endpoint (pipeline.md §4, ADR-001).
package scoring

import (
	"context"
	"fmt"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/scoring"
)

// JobReader is the narrow consumer interface for the job data required by the
// scoring pipeline. It is defined here (consumer package) and satisfied by an
// adapter over the jobs application service (ADR-001, consumer-side interface).
type JobReader interface {
	// ReadJobForScoring returns the job facts needed to run the gate and scorer.
	// profileID is required because the underlying read is profile-scoped.
	ReadJobForScoring(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID) (scoring.JobFacts, error)
}

// ProfileReader is the narrow consumer interface for the profile criteria required
// by the scoring pipeline. Defined here and satisfied by an adapter over the
// profiles application service (ADR-001).
type ProfileReader interface {
	// ReadProfileForScoring returns the conditions and weights for a profile.
	ReadProfileForScoring(ctx context.Context, profileID kernel.ProfileID) (scoring.ProfileCriteria, error)
}

// Service orchestrates the scoring pipeline: read inputs, run domain logic, persist.
type Service struct {
	jobReader     JobReader
	profileReader ProfileReader
	repo          scoring.Repository
}

// New constructs a scoring Service wired with the given reader adapters and repository.
func New(jr JobReader, pr ProfileReader, repo scoring.Repository) *Service {
	return &Service{jobReader: jr, profileReader: pr, repo: repo}
}

// ScoreJob computes and persists the fit score for a (job, profile) pair. It runs
// the dealbreaker gate and the weighted preference scorer, then upserts the result
// into job_score. The operation is idempotent: re-scoring the same pair overwrites
// the previous result.
func (s *Service) ScoreJob(ctx context.Context, jobID kernel.JobID, profileID kernel.ProfileID) (scoring.JobScore, error) {
	job, err := s.jobReader.ReadJobForScoring(ctx, profileID, jobID)
	if err != nil {
		return scoring.JobScore{}, fmt.Errorf("reading job %q for scoring: %w", jobID, err)
	}

	profile, err := s.profileReader.ReadProfileForScoring(ctx, profileID)
	if err != nil {
		return scoring.JobScore{}, fmt.Errorf("reading profile %q for scoring: %w", profileID, err)
	}

	passes := scoring.EvaluateDealbreakers(job, profile.Conditions)
	weighted, breakdown := scoring.ComputeWeightedScore(job, profile.Conditions, profile.Weights)

	score := scoring.JobScore{
		JobID:              jobID,
		ProfileID:          profileID,
		PassesDealbreakers: passes,
		WeightedScore:      weighted,
		Breakdown:          breakdown,
		ScoredAt:           time.Now().UTC(),
	}

	if err := s.repo.Upsert(ctx, score); err != nil {
		return scoring.JobScore{}, fmt.Errorf("persisting score for job %q profile %q: %w", jobID, profileID, err)
	}
	return score, nil
}

// GetScore returns the stored fit score for a (job, profile) pair. Returns a
// kernel.NotFoundError when the pair has not been scored yet.
func (s *Service) GetScore(ctx context.Context, jobID kernel.JobID, profileID kernel.ProfileID) (scoring.JobScore, error) {
	sc, err := s.repo.FindByJobAndProfile(ctx, jobID, profileID)
	if err != nil {
		return scoring.JobScore{}, fmt.Errorf("fetching score for job %q profile %q: %w", jobID, profileID, err)
	}
	return sc, nil
}
