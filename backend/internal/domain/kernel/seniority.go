package kernel

import "fmt"

// Seniority describes the experience level expected for a job listing.
type Seniority string

const (
	// SeniorityEntry is an entry-level position (débutant / junior, 0–2 years).
	SeniorityEntry Seniority = "entry"
	// SeniorityMid is a mid-level position (confirmé, 2–5 years).
	SeniorityMid Seniority = "mid"
	// SenioritySenior is a senior position (senior, 5+ years).
	SenioritySenior Seniority = "senior"
	// SeniorityLead is a lead or principal role.
	SeniorityLead Seniority = "lead"
	// SeniorityExec is an executive or C-level role.
	SeniorityExec Seniority = "exec"
)

var validSeniorities = map[Seniority]bool{
	SeniorityEntry:  true,
	SeniorityMid:    true,
	SenioritySenior: true,
	SeniorityLead:   true,
	SeniorityExec:   true,
}

// ParseSeniority parses a Seniority from a raw string, returning an error
// if the value is not recognised.
func ParseSeniority(s string) (Seniority, error) {
	sn := Seniority(s)
	if !validSeniorities[sn] {
		return "", fmt.Errorf("unknown seniority %q; valid values: entry, mid, senior, lead, exec", s)
	}
	return sn, nil
}

// IsValid reports whether s is a known Seniority value.
func (s Seniority) IsValid() bool { return validSeniorities[s] }
