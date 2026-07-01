package llm

import (
	"encoding/json"

	domainllm "github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// ParseExtractedListingForTest exposes the internal parseExtractedListing function
// for use in package-external tests without requiring a live Claude API call.
func ParseExtractedListingForTest(raw json.RawMessage) (*domainllm.ExtractedListing, error) {
	return parseExtractedListing(raw)
}

// ParseExtractedIdentityForTest exposes the internal parseExtractedIdentity function
// for use in package-external tests without requiring a live Claude API call.
func ParseExtractedIdentityForTest(raw json.RawMessage) (*domainllm.ExtractedIdentity, error) {
	return parseExtractedIdentity(raw)
}
