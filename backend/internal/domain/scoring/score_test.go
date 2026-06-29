package scoring_test

import (
	"math"
	"testing"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/scoring"
)

// helpers

func ptr[T any](v T) *T { return &v }

// fixture job used as a passing baseline across most dealbreaker cases.
func baseJob() scoring.JobFacts {
	return scoring.JobFacts{
		ContractType: kernel.ContractTypeCDI,
		RemotePolicy: kernel.RemotePolicyFullRemote,
		SalaryMin:    ptr(int64(60_000)),
		SalaryMax:    ptr(int64(75_000)),
		Skills:       []string{"Go", "PostgreSQL", "Docker"},
		Location:     "Paris, France",
		OfficeDays:   0,
		WorkingDays:  kernel.WorkingDaysFullTime,
	}
}

func baseConds() scoring.ProfileConditions {
	return scoring.ProfileConditions{
		DealBreakerContractType:   ptr(kernel.ContractTypeCDI),
		DealBreakerRemotePolicy:   ptr(kernel.RemotePolicyFullRemote),
		DealBreakerSalaryMin:      ptr(int64(50_000)),
		DealBreakerRequiredSkills: []string{"Go"},
	}
}

// ── P3-SC-1: dealbreaker gate ─────────────────────────────────────────────────

func TestEvaluateDealbreakers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		job  scoring.JobFacts
		cond scoring.ProfileConditions
		want bool
	}{
		{
			// AC P3-SC-1: job that matches every hard filter passes.
			name: "passes when all dealbreakers satisfied",
			job:  baseJob(),
			cond: baseConds(),
			want: true,
		},
		{
			// AC P3-SC-1: wrong contract type → fails.
			name: "fails when contract type does not match",
			job: func() scoring.JobFacts {
				j := baseJob()
				j.ContractType = kernel.ContractTypeFreelance
				return j
			}(),
			cond: baseConds(),
			want: false,
		},
		{
			// AC P3-SC-1: wrong remote policy → fails.
			name: "fails when remote policy does not match",
			job: func() scoring.JobFacts {
				j := baseJob()
				j.RemotePolicy = kernel.RemotePolicyOnSite
				return j
			}(),
			cond: baseConds(),
			want: false,
		},
		{
			// AC P3-SC-1: salary below minimum → fails.
			name: "fails when salary max is below minimum",
			job: func() scoring.JobFacts {
				j := baseJob()
				j.SalaryMax = ptr(int64(40_000))
				return j
			}(),
			cond: baseConds(),
			want: false,
		},
		{
			// AC P3-SC-1: unknown salary with a configured minimum → fails.
			name: "fails when salary is unknown and minimum is set",
			job: func() scoring.JobFacts {
				j := baseJob()
				j.SalaryMin = nil
				j.SalaryMax = nil
				return j
			}(),
			cond: baseConds(),
			want: false,
		},
		{
			// AC P3-SC-1: missing required skill → fails.
			name: "fails when required skill is absent from job",
			job: func() scoring.JobFacts {
				j := baseJob()
				j.Skills = []string{"Java", "PostgreSQL"}
				return j
			}(),
			cond: baseConds(),
			want: false,
		},
		{
			// AC P3-SC-1: required skills matched case-insensitively.
			name: "passes when required skill matches case-insensitively",
			job: func() scoring.JobFacts {
				j := baseJob()
				j.Skills = []string{"go", "postgresql"}
				return j
			}(),
			cond: baseConds(),
			want: true,
		},
		{
			// No dealbreakers configured → every job passes.
			name: "passes when no dealbreakers are set",
			job:  baseJob(),
			cond: scoring.ProfileConditions{},
			want: true,
		},
		{
			// Salary at exactly the minimum → passes.
			name: "passes when salary max equals the minimum exactly",
			job: func() scoring.JobFacts {
				j := baseJob()
				j.SalaryMax = ptr(int64(50_000))
				return j
			}(),
			cond: baseConds(),
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := scoring.EvaluateDealbreakers(tc.job, tc.cond)

			if got != tc.want {
				t.Errorf("EvaluateDealbreakers() = %v; want %v", got, tc.want)
			}
		})
	}
}

// ── P3-SC-2: weighted preference score ───────────────────────────────────────

func TestComputeWeightedScore(t *testing.T) {
	t.Parallel()

	defaultWeights := scoring.FitWeights{
		PreferredSkills: 40,
		Salary:          25,
		Location:        15,
		OfficeDays:      10,
		WorkingDays:     10,
	}

	cases := []struct {
		name          string
		job           scoring.JobFacts
		cond          scoring.ProfileConditions
		weights       scoring.FitWeights
		wantScore     float64
		wantBreakdown scoring.ComponentBreakdown
	}{
		{
			// AC P3-SC-2: fixture job with known inputs → verify exact weighted_score +
			// breakdown. preferred_skills=2/3≈0.667, salary=1.0, location=1.0,
			// office_days=1.0, working_days=1.0
			// weighted = 0.667×0.40 + 1.0×0.25 + 1.0×0.15 + 1.0×0.10 + 1.0×0.10 = 0.867
			name: "weighted score matches fixture calculation",
			job: scoring.JobFacts{
				Skills:      []string{"Go", "Python"},
				SalaryMin:   ptr(int64(60_000)),
				Location:    "Paris",
				OfficeDays:  2,
				WorkingDays: kernel.WorkingDaysFullTime,
			},
			cond: scoring.ProfileConditions{
				PreferredSkills:        []string{"Go", "Python", "React"},
				DealBreakerSalaryMin:   ptr(int64(50_000)),
				PreferredLocation:      "Paris",
				PreferredMaxOfficeDays: ptr(3),
				PreferredWorkingDays:   kernel.WorkingDaysFullTime,
			},
			weights:   defaultWeights,
			wantScore: 2.0/3.0*0.40 + 1.0*0.25 + 1.0*0.15 + 1.0*0.10 + 1.0*0.10,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 2.0 / 3.0,
				Salary:          1.0,
				Location:        1.0,
				OfficeDays:      1.0,
				WorkingDays:     1.0,
			},
		},
		{
			// No preferred skills configured → component scores to 1.0.
			// Unset salary, location, office-days, working-days preferences also score 1.0.
			name: "preferred skills component is 1.0 when no preferences set",
			job: scoring.JobFacts{
				Skills: []string{"Go"},
			},
			cond:      scoring.ProfileConditions{},
			weights:   scoring.FitWeights{PreferredSkills: 100},
			wantScore: 1.0,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 1.0,
				Salary:          1.0, // no salary constraint → 1.0
				Location:        1.0, // no location pref → 1.0
				OfficeDays:      1.0, // no max office days → 1.0
				WorkingDays:     1.0, // no working-days pref → 1.0
			},
		},
		{
			// Zero preferred-skill matches → skills component 0.0.
			// Other unset preferences all return 1.0.
			name: "preferred skills component is 0.0 when no skills overlap",
			job: scoring.JobFacts{
				Skills: []string{"Java"},
			},
			cond: scoring.ProfileConditions{
				PreferredSkills: []string{"Go", "Python"},
			},
			weights:   scoring.FitWeights{PreferredSkills: 100},
			wantScore: 0.0,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 0.0,
				Salary:          1.0, // no salary constraint → 1.0
				Location:        1.0, // no location pref → 1.0
				OfficeDays:      1.0, // no max office days → 1.0
				WorkingDays:     1.0, // no working-days pref → 1.0
			},
		},
		{
			// Salary unknown → salary component 0.0.
			name: "salary component is 0.0 when job salary is unknown",
			job:  scoring.JobFacts{},
			cond: scoring.ProfileConditions{
				DealBreakerSalaryMin: ptr(int64(50_000)),
			},
			weights:   scoring.FitWeights{Salary: 100},
			wantScore: 0.0,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 1.0, // no preferred skills → 1.0
				Salary:          0.0,
				Location:        1.0, // no location pref → 1.0
				OfficeDays:      1.0, // no max office days → 1.0
				WorkingDays:     1.0, // no working-days pref → 1.0
			},
		},
		{
			// Job salary below min → proportional salary component.
			name: "salary component is proportional when below minimum",
			job:  scoring.JobFacts{SalaryMin: ptr(int64(40_000))},
			cond: scoring.ProfileConditions{
				DealBreakerSalaryMin: ptr(int64(50_000)),
			},
			weights:   scoring.FitWeights{Salary: 100},
			wantScore: 40_000.0 / 50_000.0,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 1.0, // no preferred skills → 1.0
				Salary:          40_000.0 / 50_000.0,
				Location:        1.0, // no location pref → 1.0
				OfficeDays:      1.0, // no max office days → 1.0
				WorkingDays:     1.0, // no working-days pref → 1.0
			},
		},
		{
			// Location mismatch → location component 0.0.
			name: "location component is 0.0 on mismatch",
			job:  scoring.JobFacts{Location: "Lyon"},
			cond: scoring.ProfileConditions{
				PreferredLocation: "Paris",
			},
			weights:   scoring.FitWeights{Location: 100},
			wantScore: 0.0,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 1.0, // no preferred skills → 1.0
				Salary:          1.0, // no salary constraint → 1.0
				Location:        0.0,
				OfficeDays:      1.0, // no max office days → 1.0
				WorkingDays:     1.0, // no working-days pref → 1.0
			},
		},
		{
			// Location match via case-insensitive substring → 1.0.
			name: "location component is 1.0 on case-insensitive match",
			job:  scoring.JobFacts{Location: "paris, france"},
			cond: scoring.ProfileConditions{
				PreferredLocation: "Paris",
			},
			weights:   scoring.FitWeights{Location: 100},
			wantScore: 1.0,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 1.0, // no preferred skills → 1.0
				Salary:          1.0, // no salary constraint → 1.0
				Location:        1.0,
				OfficeDays:      1.0, // no max office days → 1.0
				WorkingDays:     1.0, // no working-days pref → 1.0
			},
		},
		{
			// Office days exceed max → inverse-proportion score.
			name: "office days component is proportional when job exceeds max",
			job:  scoring.JobFacts{OfficeDays: 4},
			cond: scoring.ProfileConditions{
				PreferredMaxOfficeDays: ptr(2),
			},
			weights:   scoring.FitWeights{OfficeDays: 100},
			wantScore: 2.0 / 4.0,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 1.0, // no preferred skills → 1.0
				Salary:          1.0, // no salary constraint → 1.0
				Location:        1.0, // no location pref → 1.0
				OfficeDays:      2.0 / 4.0,
				WorkingDays:     1.0, // no working-days pref → 1.0
			},
		},
		{
			// Working days mismatch → 0.0.
			name: "working days component is 0.0 on mismatch",
			job:  scoring.JobFacts{WorkingDays: kernel.WorkingDaysPartTime},
			cond: scoring.ProfileConditions{
				PreferredWorkingDays: kernel.WorkingDaysFullTime,
			},
			weights:   scoring.FitWeights{WorkingDays: 100},
			wantScore: 0.0,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 1.0, // no preferred skills → 1.0
				Salary:          1.0, // no salary constraint → 1.0
				Location:        1.0, // no location pref → 1.0
				OfficeDays:      1.0, // no max office days → 1.0
				WorkingDays:     0.0,
			},
		},
		{
			// All preferences unset → every component is 1.0, score = 1.0.
			name: "score is 1.0 when no preferences are configured",
			job:  scoring.JobFacts{},
			cond: scoring.ProfileConditions{},
			weights: scoring.FitWeights{
				PreferredSkills: 40,
				Salary:          25,
				Location:        15,
				OfficeDays:      10,
				WorkingDays:     10,
			},
			wantScore: 1.0,
			wantBreakdown: scoring.ComponentBreakdown{
				PreferredSkills: 1.0,
				Salary:          1.0,
				Location:        1.0,
				OfficeDays:      1.0,
				WorkingDays:     1.0,
			},
		},
	}

	const epsilon = 1e-9

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotScore, gotBD := scoring.ComputeWeightedScore(tc.job, tc.cond, tc.weights)

			if math.Abs(gotScore-tc.wantScore) > epsilon {
				t.Errorf("WeightedScore = %.9f; want %.9f", gotScore, tc.wantScore)
			}
			if math.Abs(gotBD.PreferredSkills-tc.wantBreakdown.PreferredSkills) > epsilon {
				t.Errorf("Breakdown.PreferredSkills = %.9f; want %.9f", gotBD.PreferredSkills, tc.wantBreakdown.PreferredSkills)
			}
			if math.Abs(gotBD.Salary-tc.wantBreakdown.Salary) > epsilon {
				t.Errorf("Breakdown.Salary = %.9f; want %.9f", gotBD.Salary, tc.wantBreakdown.Salary)
			}
			if math.Abs(gotBD.Location-tc.wantBreakdown.Location) > epsilon {
				t.Errorf("Breakdown.Location = %.9f; want %.9f", gotBD.Location, tc.wantBreakdown.Location)
			}
			if math.Abs(gotBD.OfficeDays-tc.wantBreakdown.OfficeDays) > epsilon {
				t.Errorf("Breakdown.OfficeDays = %.9f; want %.9f", gotBD.OfficeDays, tc.wantBreakdown.OfficeDays)
			}
			if math.Abs(gotBD.WorkingDays-tc.wantBreakdown.WorkingDays) > epsilon {
				t.Errorf("Breakdown.WorkingDays = %.9f; want %.9f", gotBD.WorkingDays, tc.wantBreakdown.WorkingDays)
			}
		})
	}
}
