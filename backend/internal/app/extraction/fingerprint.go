package extraction

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode"
)

// computeFingerprint returns a deterministic, hex-encoded SHA-256 dedup key for a job
// listing. It is derived from the normalized title, company, and city — salary is
// excluded because advertised ranges vary between boards and extraction runs, making it
// an unreliable dedup signal.
//
// Normalization applied: lowercase, collapse internal whitespace, trim. Location is
// reduced to the first segment before the first comma so "Paris, France" and
// "Paris, Île-de-France" collapse to the same city token "paris".
//
// Example:
//
//	computeFingerprint("  Go Engineer ", "Acme Corp", "Paris, France")
//	// → sha256("go engineer|acme corp|paris") as hex string
func computeFingerprint(title, company, location string) string {
	key := normalize(title) + "|" + normalize(company) + "|" + normalizeCity(location)
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", sum)
}

// normalize lowercases s, collapses all internal whitespace runs to a single space,
// and trims leading/trailing whitespace.
func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.FieldsFunc(s, unicode.IsSpace), " ")
}

// normalizeCity extracts the city token from a location string by taking the segment
// before the first comma, then applying the same whitespace normalization.
//
// Examples: "Paris, France" → "paris", "Lyon (69)" → "lyon (69)",
// "Remote" → "remote".
func normalizeCity(location string) string {
	if idx := strings.IndexByte(location, ','); idx >= 0 {
		location = location[:idx]
	}
	return normalize(location)
}
