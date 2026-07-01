// Package scoring is the scoring bounded context. It owns the dealbreaker gate and
// the weighted preference score for a (job, profile) pair. The gate evaluates hard
// filters (contract type, remote policy, minimum salary, required skills); the
// weighted score aggregates soft-preference components per the profile's fit_weights.
// Both results are persisted in job_score and consumed by the job-browser and
// dashboard contexts.
package scoring

import (
	"strings"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// JobFacts is the subset of job data required by the scoring pipeline. It carries
// only the fields consumed by the dealbreaker gate and the weighted scorer; the
// scoring context never touches display or provenance fields.
type JobFacts struct {
	// ContractType is the employment contract category of the listing.
	ContractType kernel.ContractType
	// RemotePolicy is the remote-work policy advertised by the listing.
	RemotePolicy kernel.RemotePolicy
	// SalaryMin is the lower bound of the advertised salary range; nil when absent.
	SalaryMin *int64
	// SalaryMax is the upper bound of the advertised salary range; nil when absent.
	SalaryMax *int64
	// Skills is the list of technologies or competencies required by the listing.
	Skills []string
	// Location is the geographic location of the listing.
	Location string
	// OfficeDays is the number of required on-site days per week.
	OfficeDays int
	// WorkingDays is the weekly schedule of the listing.
	WorkingDays kernel.WorkingDays
}

// ProfileConditions is the subset of profile conditions required by the scoring
// pipeline. Dealbreaker fields gate jobs; preference fields feed the weighted score.
type ProfileConditions struct {
	// Dealbreakers — a job failing any of these is suppressed from top matches.
	DealBreakerContractType   *kernel.ContractType
	DealBreakerRemotePolicy   *kernel.RemotePolicy
	DealBreakerSalaryMin      *int64
	DealBreakerRequiredSkills []string

	// Preferences — soft inputs that produce the weighted score components.
	PreferredSkills        []string
	PreferredMaxOfficeDays *int
	PreferredLocation      string
	PreferredWorkingDays   kernel.WorkingDays
}

// FitWeights holds the per-profile integer percentage weights for the five soft
// scoring components. They are guaranteed to sum to 100 before they reach the scorer.
type FitWeights struct {
	PreferredSkills int
	Salary          int
	Location        int
	OfficeDays      int
	WorkingDays     int
}

// ProfileCriteria bundles the conditions and weights for a profile, providing
// everything the scorer needs in a single value from the ProfileReader.
type ProfileCriteria struct {
	Conditions ProfileConditions
	Weights    FitWeights
}

// ComponentBreakdown records each soft-scoring component's individual raw score
// (0.0–1.0). It is stored as JSON in job_score.component_breakdown and allows the
// dashboard to explain why a job scored as it did.
type ComponentBreakdown struct {
	PreferredSkills float64 `json:"preferred_skills"`
	Salary          float64 `json:"salary"`
	Location        float64 `json:"location"`
	OfficeDays      float64 `json:"office_days"`
	WorkingDays     float64 `json:"working_days"`
}

// JobScore is the fit score for a (job, profile) pair. PassesDealbreakers records
// the hard-filter gate result; WeightedScore is the 0.0–1.0 preference-weighted
// aggregate; Breakdown exposes each component for display.
type JobScore struct {
	// JobID identifies the scored job.
	JobID kernel.JobID
	// ProfileID identifies the profile the score was computed against.
	ProfileID kernel.ProfileID
	// PassesDealbreakers is false when the job fails at least one hard filter.
	// Jobs that fail are never surfaced in the top-matches dashboard view.
	PassesDealbreakers bool
	// WeightedScore is the aggregate preference score in [0, 1]. Only meaningful
	// when PassesDealbreakers is true, but always computed.
	WeightedScore float64
	// Breakdown holds each component's raw score (0–1) before weighting.
	Breakdown ComponentBreakdown
	// ScoredAt is when this score was last computed.
	ScoredAt time.Time
}

// EvaluateDealbreakers applies the four hard-filter rules and returns true only
// when the job satisfies all active dealbreakers. A nil dealbreaker field means
// "no constraint on this dimension". An unset salary (nil SalaryMax) fails the
// salary gate when a minimum is configured, because the offered salary cannot be
// confirmed.
//
// AC P3-SC-1: a job failing any hard filter → passes_dealbreakers=false.
func EvaluateDealbreakers(job JobFacts, conds ProfileConditions) bool {
	if conds.DealBreakerContractType != nil && job.ContractType != *conds.DealBreakerContractType {
		return false
	}
	if conds.DealBreakerRemotePolicy != nil && job.RemotePolicy != *conds.DealBreakerRemotePolicy {
		return false
	}
	if conds.DealBreakerSalaryMin != nil {
		if job.SalaryMax == nil || *job.SalaryMax < *conds.DealBreakerSalaryMin {
			return false
		}
	}
	for _, req := range conds.DealBreakerRequiredSkills {
		if !containsSkill(job.Skills, req) {
			return false
		}
	}
	return true
}

// ComputeWeightedScore computes the 0.0–1.0 aggregate preference score and the
// per-component breakdown for the given job and profile criteria.
//
// AC P3-SC-2: weighted_score + breakdown match the weights for a fixture job.
func ComputeWeightedScore(job JobFacts, conds ProfileConditions, weights FitWeights) (float64, ComponentBreakdown) {
	bd := ComponentBreakdown{
		PreferredSkills: preferredSkillsScore(job.Skills, conds.PreferredSkills),
		Salary:          salaryScore(job.SalaryMin, conds.DealBreakerSalaryMin),
		Location:        locationScore(job.Location, conds.PreferredLocation),
		OfficeDays:      officeDaysScore(job.OfficeDays, conds.PreferredMaxOfficeDays),
		WorkingDays:     workingDaysScore(job.WorkingDays, conds.PreferredWorkingDays),
	}

	weighted := bd.PreferredSkills*float64(weights.PreferredSkills)/100.0 +
		bd.Salary*float64(weights.Salary)/100.0 +
		bd.Location*float64(weights.Location)/100.0 +
		bd.OfficeDays*float64(weights.OfficeDays)/100.0 +
		bd.WorkingDays*float64(weights.WorkingDays)/100.0

	return weighted, bd
}

// preferredSkillsScore returns the fraction of preferred skills present in the
// job's skill list (case-insensitive). Returns 1.0 when no preferred skills are set.
func preferredSkillsScore(jobSkills, preferred []string) float64 {
	if len(preferred) == 0 {
		return 1.0
	}
	matched := 0
	for _, p := range preferred {
		if containsSkill(jobSkills, p) {
			matched++
		}
	}
	return float64(matched) / float64(len(preferred))
}

// salaryScore returns 1.0 when the job's minimum salary meets the profile minimum.
// Returns 0.0 when the job salary is unknown. Returns a proportion in (0,1) when
// the job salary is below the configured minimum.
func salaryScore(jobSalaryMin *int64, profileMin *int64) float64 {
	if profileMin == nil {
		return 1.0
	}
	if jobSalaryMin == nil {
		return 0.0
	}
	if *jobSalaryMin >= *profileMin {
		return 1.0
	}
	return float64(*jobSalaryMin) / float64(*profileMin)
}

// locationScore returns 1.0 when the job location contains or is contained by the
// preferred location (case-insensitive). Returns 1.0 when no preference is set.
func locationScore(jobLocation, preferred string) float64 {
	if preferred == "" {
		return 1.0
	}
	jl := strings.ToLower(jobLocation)
	pl := strings.ToLower(preferred)
	if strings.Contains(jl, pl) || strings.Contains(pl, jl) {
		return 1.0
	}
	return 0.0
}

// officeDaysScore returns 1.0 when the job's office-day requirement is at or below
// the profile's maximum. Returns a inverse-proportion score when the job exceeds
// the maximum. Returns 1.0 when no maximum is set.
func officeDaysScore(jobOfficeDays int, maxOfficeDays *int) float64 {
	if maxOfficeDays == nil {
		return 1.0
	}
	if jobOfficeDays <= *maxOfficeDays {
		return 1.0
	}
	return float64(*maxOfficeDays) / float64(jobOfficeDays)
}

// workingDaysScore returns 1.0 when the job's working days match the preference
// exactly, or when no preference is set.
func workingDaysScore(jobWorkingDays, preferred kernel.WorkingDays) float64 {
	if preferred == "" {
		return 1.0
	}
	if jobWorkingDays == preferred {
		return 1.0
	}
	return 0.0
}

// containsSkill reports whether skill appears in the list (case-insensitive).
func containsSkill(skills []string, skill string) bool {
	for _, s := range skills {
		if strings.EqualFold(s, skill) {
			return true
		}
	}
	return false
}
