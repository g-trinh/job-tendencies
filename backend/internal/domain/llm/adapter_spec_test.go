package llm

import (
	"encoding/json"
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

// remotiveSeedSpecJSON and arbeitnowSeedSpecJSON are byte-identical to the adapter.spec
// JSONB payloads seeded by migration 00017_public_board_seed.sql. Keeping them here and
// validating them guards against a malformed seed failing at scrape time instead of CI
// (mirrors the "verified against schema before going live" rule in ADR-004).
const remotiveSeedSpecJSON = `{
  "board": "remotive",
  "fetch_mode": "json_api",
  "search": {
    "url_template": "https://remotive.com/api/remote-jobs?search=go",
    "method": "GET",
    "param_map": {},
    "pagination": {"kind": "query_param", "param": "page", "start": 1},
    "result_node_path": "$.jobs",
    "result_fields": {
      "listing_url": "$.url",
      "title": "$.title",
      "company": "$.company_name",
      "location": "$.candidate_required_location",
      "posted_at": "$.publication_date",
      "external_id": "$.id"
    }
  },
  "listing": {"fetch": "use_search_payload", "raw_capture": "$"},
  "incremental": {"cursor_field": "posted_at", "overlap_buffer": "36h", "safety_max_pages": 1}
}`

const arbeitnowSeedSpecJSON = `{
  "board": "arbeitnow",
  "fetch_mode": "json_api",
  "search": {
    "url_template": "https://www.arbeitnow.com/api/job-board-api?page={page}",
    "method": "GET",
    "param_map": {},
    "pagination": {"kind": "query_param", "param": "page", "start": 1},
    "result_node_path": "$.data",
    "result_fields": {
      "listing_url": "$.url",
      "title": "$.title",
      "company": "$.company_name",
      "location": "$.location",
      "posted_at": "$.created_at",
      "external_id": "$.slug"
    }
  },
  "listing": {"fetch": "use_search_payload", "raw_capture": "$"},
  "incremental": {"cursor_field": "posted_at", "overlap_buffer": "36h", "safety_max_pages": 5}
}`

// remoteokSeedSpecJSON is byte-identical to the adapter.spec JSONB payload seeded by
// migration 00018_remoteok_board_seed.sql.
const remoteokSeedSpecJSON = `{
  "board": "remoteok",
  "fetch_mode": "json_api",
  "search": {
    "url_template": "https://remoteok.com/api",
    "method": "GET",
    "param_map": {},
    "pagination": {"kind": "query_param", "param": "page", "start": 1},
    "result_node_path": "$.#(id!=)#",
    "result_fields": {
      "listing_url": "$.url",
      "title": "$.position",
      "company": "$.company",
      "location": "$.location",
      "posted_at": "$.date",
      "external_id": "$.id"
    }
  },
  "listing": {"fetch": "use_search_payload", "raw_capture": "$"},
  "incremental": {"cursor_field": "posted_at", "overlap_buffer": "36h", "safety_max_pages": 1}
}`

func TestSeededPublicBoardAdapterSpecs_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		json string
	}{
		{name: "remotive seed spec (migration 00017)", json: remotiveSeedSpecJSON},
		{name: "arbeitnow seed spec (migration 00017)", json: arbeitnowSeedSpecJSON},
		{name: "remoteok seed spec (migration 00018)", json: remoteokSeedSpecJSON},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var spec AdapterSpec
			if err := json.Unmarshal([]byte(tc.json), &spec); err != nil {
				t.Fatalf("unmarshalling seed spec: %v", err)
			}

			if err := spec.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
		})
	}
}
