package scoring_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/g-trinh/job-tendencies/internal/app/scoring"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	domscoring "github.com/g-trinh/job-tendencies/internal/domain/scoring"
)

// ── test doubles ──────────────────────────────────────────────────────────────

type fakeJobReader struct {
	facts domscoring.JobFacts
	err   error
}

func (f *fakeJobReader) ReadJobForScoring(_ context.Context, _ kernel.ProfileID, _ kernel.JobID) (domscoring.JobFacts, error) {
	return f.facts, f.err
}

type fakeProfileReader struct {
	criteria domscoring.ProfileCriteria
	err      error
}

func (f *fakeProfileReader) ReadProfileForScoring(_ context.Context, _ kernel.ProfileID) (domscoring.ProfileCriteria, error) {
	return f.criteria, f.err
}

type fakeRepo struct {
	scores map[string]domscoring.JobScore
	err    error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{scores: make(map[string]domscoring.JobScore)}
}

func (f *fakeRepo) Upsert(_ context.Context, score domscoring.JobScore) error {
	if f.err != nil {
		return f.err
	}
	f.scores[string(score.JobID)+":"+string(score.ProfileID)] = score
	return nil
}

func (f *fakeRepo) FindByJobAndProfile(_ context.Context, jobID kernel.JobID, profileID kernel.ProfileID) (domscoring.JobScore, error) {
	s, ok := f.scores[string(jobID)+":"+string(profileID)]
	if !ok {
		return domscoring.JobScore{}, &kernel.NotFoundError{Kind: "job_score", ID: string(jobID)}
	}
	return s, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T { return &v }

const epsilon = 1e-9

// ── ScoreJob tests ────────────────────────────────────────────────────────────

func TestService_ScoreJob(t *testing.T) {
	t.Parallel()

	defaultWeights := domscoring.FitWeights{
		PreferredSkills: 40,
		Salary:          25,
		Location:        15,
		OfficeDays:      10,
		WorkingDays:     10,
	}

	cases := []struct {
		name              string
		job               domscoring.JobFacts
		criteria          domscoring.ProfileCriteria
		jobReaderErr      error
		profileReaderErr  error
		repoErr           error
		wantPasses        bool
		wantWeightedScore *float64 // nil = skip weighted-score assertion
		wantErr           bool
	}{
		{
			// AC P3-SC-1 + P3-SC-2: passes all dealbreakers → score computed and persisted.
			// preferred_skills = 2/3, salary = 1.0, location = 1.0, office = 1.0, wd = 1.0
			name: "scores and persists when job passes all dealbreakers",
			job: domscoring.JobFacts{
				ContractType: kernel.ContractTypeCDI,
				RemotePolicy: kernel.RemotePolicyFullRemote,
				SalaryMin:    ptr(int64(60_000)),
				SalaryMax:    ptr(int64(75_000)),
				Skills:       []string{"Go", "Python"},
				Location:     "Paris",
				OfficeDays:   2,
				WorkingDays:  kernel.WorkingDaysFullTime,
			},
			criteria: domscoring.ProfileCriteria{
				Conditions: domscoring.ProfileConditions{
					DealBreakerContractType:   ptr(kernel.ContractTypeCDI),
					DealBreakerRemotePolicy:   ptr(kernel.RemotePolicyFullRemote),
					DealBreakerSalaryMin:      ptr(int64(50_000)),
					DealBreakerRequiredSkills: []string{"Go"},
					PreferredSkills:           []string{"Go", "Python", "React"},
					PreferredLocation:         "Paris",
					PreferredMaxOfficeDays:    ptr(3),
					PreferredWorkingDays:      kernel.WorkingDaysFullTime,
				},
				Weights: defaultWeights,
			},
			wantPasses:        true,
			wantWeightedScore: ptr(2.0/3.0*0.40 + 1.0*0.25 + 1.0*0.15 + 1.0*0.10 + 1.0*0.10),
		},
		{
			// AC P3-SC-1: wrong contract type → passes_dealbreakers = false.
			// Weighted score is still computed: no preferences set → all components 1.0.
			name: "passes_dealbreakers is false when contract type does not match",
			job: domscoring.JobFacts{
				ContractType: kernel.ContractTypeFreelance,
				SalaryMax:    ptr(int64(75_000)),
				Skills:       []string{"Go"},
			},
			criteria: domscoring.ProfileCriteria{
				Conditions: domscoring.ProfileConditions{
					DealBreakerContractType: ptr(kernel.ContractTypeCDI),
				},
				Weights: domscoring.FitWeights{PreferredSkills: 100},
			},
			wantPasses:        false,
			wantWeightedScore: ptr(1.0), // no preferred skills → 1.0
		},
		{
			// AC P3-SC-1: missing required skill → fails.
			// Weighted score still computed: no preferred skills set → 1.0.
			name: "passes_dealbreakers is false when required skill is absent",
			job: domscoring.JobFacts{
				Skills:    []string{"Java"},
				SalaryMax: ptr(int64(60_000)),
			},
			criteria: domscoring.ProfileCriteria{
				Conditions: domscoring.ProfileConditions{
					DealBreakerRequiredSkills: []string{"Go"},
				},
				Weights: domscoring.FitWeights{PreferredSkills: 100},
			},
			wantPasses:        false,
			wantWeightedScore: ptr(1.0), // no preferred skills → 1.0
		},
		{
			// AC P3-SC-1: nil salary → fails salary gate; salary component = 0.0.
			name: "passes_dealbreakers is false when job has no salary and minimum is set",
			job: domscoring.JobFacts{
				Skills: []string{"Go"},
			},
			criteria: domscoring.ProfileCriteria{
				Conditions: domscoring.ProfileConditions{
					DealBreakerSalaryMin: ptr(int64(50_000)),
				},
				Weights: domscoring.FitWeights{Salary: 100},
			},
			wantPasses:        false,
			wantWeightedScore: ptr(0.0), // unknown salary → salary component 0.0
		},
		{
			// job reader error is propagated.
			name:         "returns error when job reader fails",
			jobReaderErr: errors.New("job not found"),
			wantErr:      true,
		},
		{
			// profile reader error is propagated.
			name:             "returns error when profile reader fails",
			profileReaderErr: errors.New("profile unavailable"),
			wantErr:          true,
		},
		{
			// repository error is propagated.
			name:    "returns error when repository upsert fails",
			repoErr: errors.New("db write error"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepo()
			repo.err = tc.repoErr

			svc := scoring.New(
				&fakeJobReader{facts: tc.job, err: tc.jobReaderErr},
				&fakeProfileReader{criteria: tc.criteria, err: tc.profileReaderErr},
				repo,
			)

			got, err := svc.ScoreJob(context.Background(), "job-1", "profile-1")

			if tc.wantErr {
				if err == nil {
					t.Fatal("want error; got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.PassesDealbreakers != tc.wantPasses {
				t.Errorf("PassesDealbreakers = %v; want %v", got.PassesDealbreakers, tc.wantPasses)
			}
			if tc.wantWeightedScore != nil {
				if math.Abs(got.WeightedScore-*tc.wantWeightedScore) > epsilon {
					t.Errorf("WeightedScore = %.9f; want %.9f", got.WeightedScore, *tc.wantWeightedScore)
				}
			}
			// Verify the result was persisted.
			stored, fetchErr := repo.FindByJobAndProfile(context.Background(), "job-1", "profile-1")
			if fetchErr != nil {
				t.Fatalf("score was not persisted: %v", fetchErr)
			}
			if stored.PassesDealbreakers != got.PassesDealbreakers {
				t.Errorf("stored PassesDealbreakers = %v; want %v", stored.PassesDealbreakers, got.PassesDealbreakers)
			}
		})
	}
}
