package llm

import (
	"errors"
	"testing"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// validSpec returns a structurally valid AdapterSpec matching the seeded WTTJ adapter.
func validSpec() AdapterSpec {
	return AdapterSpec{
		Board:     "welcometothejungle",
		FetchMode: FetchModeJSONAPI,
		Search: SearchConfig{
			URLTemplate:    "https://api.wttj.co/v2/search?query={keywords}&page={page}",
			Method:         "GET",
			Pagination:     PaginationConfig{Kind: PaginationKindQueryParam, Param: "page", Start: 1},
			ResultNodePath: "$.jobs[*]",
			ResultFields:   map[string]string{ResultFieldListingURL: "$.url", "posted_at": "$.published_at"},
		},
		Listing:     ListingConfig{Fetch: ListingFetchUseSearchPayload, RawCapture: "full_response"},
		Incremental: IncrementalConfig{CursorField: "posted_at", OverlapBuffer: "36h", SafetyMaxPages: 20},
	}
}

func TestAdapterSpec_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		mutate  func(s *AdapterSpec)
		wantErr error
	}{
		{
			name:   "accepts a valid json_api spec",
			mutate: func(*AdapterSpec) {},
		},
		{
			name:    "rejects empty board",
			mutate:  func(s *AdapterSpec) { s.Board = "" },
			wantErr: kernel.ErrInvalidInput,
		},
		{
			name:    "rejects unknown fetch mode",
			mutate:  func(s *AdapterSpec) { s.FetchMode = "graphql" },
			wantErr: kernel.ErrInvalidInput,
		},
		{
			name:    "rejects missing url template",
			mutate:  func(s *AdapterSpec) { s.Search.URLTemplate = "" },
			wantErr: kernel.ErrInvalidInput,
		},
		{
			name:    "rejects missing result node path",
			mutate:  func(s *AdapterSpec) { s.Search.ResultNodePath = "" },
			wantErr: kernel.ErrInvalidInput,
		},
		{
			name:    "rejects missing listing_url result field",
			mutate:  func(s *AdapterSpec) { delete(s.Search.ResultFields, ResultFieldListingURL) },
			wantErr: kernel.ErrInvalidInput,
		},
		{
			name:    "rejects unknown pagination kind",
			mutate:  func(s *AdapterSpec) { s.Search.Pagination.Kind = "offset" },
			wantErr: kernel.ErrInvalidInput,
		},
		{
			name:    "rejects unknown listing fetch",
			mutate:  func(s *AdapterSpec) { s.Listing.Fetch = "scrape" },
			wantErr: kernel.ErrInvalidInput,
		},
		{
			name:    "rejects invalid overlap buffer",
			mutate:  func(s *AdapterSpec) { s.Incremental.OverlapBuffer = "later" },
			wantErr: kernel.ErrInvalidInput,
		},
		{
			name:    "rejects non-positive safety max pages",
			mutate:  func(s *AdapterSpec) { s.Incremental.SafetyMaxPages = 0 },
			wantErr: kernel.ErrInvalidInput,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := validSpec()
			tc.mutate(&spec)

			err := spec.Validate()

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Validate() = %v, want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}
