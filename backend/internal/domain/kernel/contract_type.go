package kernel

import "fmt"

// ContractType classifies the employment contract of a job listing.
// Values match the French employment categories surfaced in the UI.
type ContractType string

const (
	// ContractTypeCDI is a permanent open-ended contract (Contrat à durée indéterminée).
	ContractTypeCDI ContractType = "cdi"
	// ContractTypeCDD is a fixed-term contract (Contrat à durée déterminée).
	ContractTypeCDD ContractType = "cdd"
	// ContractTypeFreelance is a freelance or self-employed engagement.
	ContractTypeFreelance ContractType = "freelance"
	// ContractTypeInterim is a temporary staffing contract (travail temporaire).
	ContractTypeInterim ContractType = "interim"
)

var validContractTypes = map[ContractType]bool{
	ContractTypeCDI:       true,
	ContractTypeCDD:       true,
	ContractTypeFreelance: true,
	ContractTypeInterim:   true,
}

// ParseContractType parses a ContractType from a raw string, returning an error
// if the value is not recognised.
func ParseContractType(s string) (ContractType, error) {
	ct := ContractType(s)
	if !validContractTypes[ct] {
		return "", fmt.Errorf("unknown contract type %q; valid values: cdi, cdd, freelance, interim", s)
	}
	return ct, nil
}

// IsValid reports whether c is a known ContractType value.
func (c ContractType) IsValid() bool { return validContractTypes[c] }
