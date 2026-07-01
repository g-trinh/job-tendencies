package scoring

import (
	"context"
	"fmt"

	appjobs "github.com/g-trinh/job-tendencies/internal/app/jobs"
	appscoringapp "github.com/g-trinh/job-tendencies/internal/app/scoring"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
	"github.com/g-trinh/job-tendencies/internal/domain/scoring"
)

// jobsServiceFacade is the narrow slice of the jobs application service consumed
// by the scoring adapters. It is defined here to avoid hard-coupling on the
// concrete *appjobs.Service type and to allow straightforward testing.
type jobsServiceFacade interface {
	GetJob(ctx context.Context, profileID kernel.ProfileID, id kernel.JobID) (appjobs.JobView, error)
}

// JobsAdapter adapts the jobs application service to the app/scoring.JobReader
// interface. It translates a JobView into the JobFacts subset consumed by the scorer.
type JobsAdapter struct {
	svc jobsServiceFacade
}

// NewJobsAdapter constructs a JobsAdapter over the given jobs service facade.
func NewJobsAdapter(svc jobsServiceFacade) *JobsAdapter {
	return &JobsAdapter{svc: svc}
}

// ReadJobForScoring fetches the job view from the jobs service and projects it
// into the scoring.JobFacts contract.
func (a *JobsAdapter) ReadJobForScoring(ctx context.Context, profileID kernel.ProfileID, jobID kernel.JobID) (scoring.JobFacts, error) {
	view, err := a.svc.GetJob(ctx, profileID, jobID)
	if err != nil {
		return scoring.JobFacts{}, fmt.Errorf("fetching job %q for scoring: %w", jobID, err)
	}
	return scoring.JobFacts{
		ContractType: view.ContractType,
		RemotePolicy: view.RemotePolicy,
		SalaryMin:    view.SalaryMin,
		SalaryMax:    view.SalaryMax,
		Skills:       view.Skills,
		Location:     view.Location,
		OfficeDays:   view.OfficeDays,
		WorkingDays:  view.WorkingDays,
	}, nil
}

// profilesServiceFacade is the narrow slice of the profiles application service
// consumed by the scoring adapters.
type profilesServiceFacade interface {
	ProfileByID(ctx context.Context, id kernel.ProfileID) (profiles.Profile, error)
}

// ProfilesAdapter adapts the profiles application service to the
// app/scoring.ProfileReader interface. It translates a Profile into the
// ProfileCriteria (conditions + weights) consumed by the scorer.
type ProfilesAdapter struct {
	svc profilesServiceFacade
}

// NewProfilesAdapter constructs a ProfilesAdapter over the given profiles service facade.
func NewProfilesAdapter(svc profilesServiceFacade) *ProfilesAdapter {
	return &ProfilesAdapter{svc: svc}
}

// ReadProfileForScoring fetches the profile from the profiles service and projects
// conditions and fit weights into the scoring.ProfileCriteria contract.
func (a *ProfilesAdapter) ReadProfileForScoring(ctx context.Context, profileID kernel.ProfileID) (scoring.ProfileCriteria, error) {
	p, err := a.svc.ProfileByID(ctx, profileID)
	if err != nil {
		return scoring.ProfileCriteria{}, fmt.Errorf("fetching profile %q for scoring: %w", profileID, err)
	}
	return scoring.ProfileCriteria{
		Conditions: scoring.ProfileConditions{
			DealBreakerContractType:   p.Conditions.DealBreakerContractType,
			DealBreakerRemotePolicy:   p.Conditions.DealBreakerRemotePolicy,
			DealBreakerSalaryMin:      p.Conditions.DealBreakerSalaryMin,
			DealBreakerRequiredSkills: p.Conditions.DealBreakerRequiredSkills,
			PreferredSkills:           p.Conditions.PreferredSkills,
			PreferredMaxOfficeDays:    p.Conditions.PreferredMaxOfficeDays,
			PreferredLocation:         p.Conditions.PreferredLocation,
			PreferredWorkingDays:      p.Conditions.PreferredWorkingDays,
		},
		Weights: scoring.FitWeights{
			PreferredSkills: p.Weights.PreferredSkills,
			Salary:          p.Weights.Salary,
			Location:        p.Weights.Location,
			OfficeDays:      p.Weights.OfficeDays,
			WorkingDays:     p.Weights.WorkingDays,
		},
	}, nil
}

// Verify adapters satisfy the consumer interfaces at compile time.
var (
	_ appscoringapp.JobReader     = (*JobsAdapter)(nil)
	_ appscoringapp.ProfileReader = (*ProfilesAdapter)(nil)
)
