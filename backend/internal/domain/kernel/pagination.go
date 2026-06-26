package kernel

import "fmt"

const (
	// DefaultPageSize is the number of items returned per page when no limit is specified.
	DefaultPageSize = 20
	// MaxPageSize is the upper bound on items per page enforced by all list endpoints.
	MaxPageSize = 100
)

// PageFilter specifies which page of results to return. Use DefaultPageFilter to
// get sensible defaults and Validate to enforce business rules before querying.
type PageFilter struct {
	// Page is 1-based. Values < 1 are treated as 1 by Offset.
	Page int
	// Limit is the maximum number of items per page. Capped to [1, MaxPageSize].
	Limit int
}

// DefaultPageFilter returns a PageFilter with the standard defaults.
func DefaultPageFilter() PageFilter {
	return PageFilter{Page: 1, Limit: DefaultPageSize}
}

// Offset returns the zero-based row offset for use in SQL OFFSET clauses.
func (p PageFilter) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// Validate returns a ValidationError if the filter has out-of-range values.
func (p PageFilter) Validate() error {
	if p.Page < 1 {
		return &ValidationError{Field: "page", Message: "must be >= 1"}
	}
	if p.Limit < 1 || p.Limit > MaxPageSize {
		return &ValidationError{
			Field:   "limit",
			Message: fmt.Sprintf("must be between 1 and %d", MaxPageSize),
		}
	}
	return nil
}

// SortOrder specifies sort direction for list queries.
type SortOrder string

const (
	// SortOrderAsc sorts ascending (oldest or lowest first).
	SortOrderAsc SortOrder = "asc"
	// SortOrderDesc sorts descending (newest or highest first).
	SortOrderDesc SortOrder = "desc"
)

// ParseSortOrder parses a SortOrder from a raw string, returning an error when the
// value is not "asc" or "desc".
func ParseSortOrder(s string) (SortOrder, error) {
	switch SortOrder(s) {
	case SortOrderAsc, SortOrderDesc:
		return SortOrder(s), nil
	default:
		return "", fmt.Errorf("unknown sort order %q; valid values: asc, desc", s)
	}
}
