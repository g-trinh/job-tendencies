package llm

import "github.com/g-trinh/job-tendencies/internal/domain/kernel"

// ExtractedIdentity holds the professional identity fields parsed from a LinkedIn PDF
// export. It is the return type of the IdentityExtractor port and is passed to the
// profiles application service to populate the profile's identity on first import.
type ExtractedIdentity struct {
	// Skills is the flat list of technical and professional skills found in the PDF.
	Skills []string
	// RawExperience is the verbatim work experience block extracted from the PDF.
	RawExperience string
	// Seniority is the career level inferred from titles and years of experience.
	Seniority kernel.Seniority
}
