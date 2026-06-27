package profiles

import (
	"fmt"
	"time"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// Identity is the LinkedIn-imported snapshot for a profile. It holds the
// professional facts (seniority, raw experience text, flat skill list) extracted
// from a LinkedIn PDF export. One identity per profile; nil until the import runs.
type Identity struct {
	// ProfileID is the owning profile.
	ProfileID kernel.ProfileID
	// Seniority is the extracted seniority level.
	Seniority kernel.Seniority
	// RawExperience is the raw experience block from the PDF.
	RawExperience string
	// Skills is the flat, user-editable skill list (no self-rating).
	Skills []string
	// ImportedAt is when the PDF was imported; nil before the first import.
	ImportedAt *time.Time
}

// ProfileConditions holds per-profile dealbreakers (hard filters) and preferences
// (soft-scoring inputs) used by the scoring pipeline. Dealbreakers gate jobs (a
// failing job is hidden); preferences drive the weighted fit score.
type ProfileConditions struct {
	// Dealbreakers — a job failing any of these is hidden in the browser.
	DealBreakerContractType   *kernel.ContractType
	DealBreakerRemotePolicy   *kernel.RemotePolicy
	DealBreakerSalaryMin      *int64
	DealBreakerRequiredSkills []string

	// Preferences — soft inputs that produce the weighted fit score.
	PreferredSkills        []string
	PreferredMaxOfficeDays *int
	PreferredLocation      string
	PreferredWorkingDays   kernel.WorkingDays
}

// FitWeights holds the user-configured weights for the scoring pipeline's soft
// components. The five weights must sum to exactly 100.
//
// Example:
//
//	w, err := profiles.NewFitWeights(40, 25, 15, 10, 10)
type FitWeights struct {
	PreferredSkills int // % weight for skills match
	Salary          int // % weight for salary vs minimum
	Location        int // % weight for location preference
	OfficeDays      int // % weight for office days vs max
	WorkingDays     int // % weight for working days match
}

// NewFitWeights constructs FitWeights and validates that the five soft components
// sum to exactly 100. All values must be non-negative.
func NewFitWeights(preferredSkills, salary, location, officeDays, workingDays int) (FitWeights, error) {
	for _, v := range []int{preferredSkills, salary, location, officeDays, workingDays} {
		if v < 0 {
			return FitWeights{}, &kernel.ValidationError{
				Field:   "weights",
				Message: "all weight values must be non-negative",
			}
		}
	}
	total := preferredSkills + salary + location + officeDays + workingDays
	if total != 100 {
		return FitWeights{}, &kernel.ValidationError{
			Field:   "weights",
			Message: fmt.Sprintf("soft component weights must sum to 100; got %d", total),
		}
	}
	return FitWeights{
		PreferredSkills: preferredSkills,
		Salary:          salary,
		Location:        location,
		OfficeDays:      officeDays,
		WorkingDays:     workingDays,
	}, nil
}

// DefaultFitWeights returns the starting weights from the feature specification
// (40/25/15/10/10, sums to 100).
func DefaultFitWeights() FitWeights {
	return FitWeights{
		PreferredSkills: 40,
		Salary:          25,
		Location:        15,
		OfficeDays:      10,
		WorkingDays:     10,
	}
}

// Validate reports an error when the weights do not sum to 100.
func (w FitWeights) Validate() error {
	sum := w.PreferredSkills + w.Salary + w.Location + w.OfficeDays + w.WorkingDays
	if sum != 100 {
		return fmt.Errorf("fit weights must sum to 100; got %d", sum)
	}
	return nil
}
