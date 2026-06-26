package llm

import "github.com/g-trinh/job-tendencies/internal/domain/kernel"

// ExtractedField wraps one extracted value with the model's self-reported confidence
// score for that field. Confidence is in [0, 100]; 0 means the field was absent or
// could not be reliably recognised.
type ExtractedField[T any] struct {
	Value      T                 `json:"value"`
	Confidence kernel.Confidence `json:"confidence"`
}

// Recruiter holds recruiter contact details extracted from a raw job listing.
// Fields are optional; absent details remain as zero-value strings.
type Recruiter struct {
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	LinkedInURL string `json:"linkedin_url,omitempty"`
	Phone       string `json:"phone,omitempty"`
}

// ExtractedListing is the structured output of a ListingExtractor.Extract call.
// Each field carries per-field confidence (0–100). Understanding measures overall
// parse quality for the whole listing (0–100) and maps to job.understanding_score.
//
// Salary values are whole euros (not centimes). Nil pointer means the field was absent
// (e.g. hidden salary); a zero-confidence field means the model saw something but could
// not interpret it.
type ExtractedListing struct {
	Skills       ExtractedField[[]string]            `json:"skills"`
	RemotePolicy ExtractedField[kernel.RemotePolicy] `json:"remote_policy"`
	// OfficeDays is the number of required on-site days per week (0 if not applicable).
	OfficeDays   ExtractedField[int]                 `json:"office_days"`
	ContractType ExtractedField[kernel.ContractType] `json:"contract_type"`
	WorkingDays  ExtractedField[kernel.WorkingDays]  `json:"working_days"`
	// SalaryMin and SalaryMax are in whole euros; nil means the salary was not published.
	SalaryMin ExtractedField[*int64]           `json:"salary_min"`
	SalaryMax ExtractedField[*int64]           `json:"salary_max"`
	Seniority ExtractedField[kernel.Seniority] `json:"seniority"`
	Recruiter ExtractedField[*Recruiter]       `json:"recruiter"`
	// Understanding is the overall parse-quality score for the whole listing (0–100).
	Understanding kernel.Understanding `json:"understanding"`
}
