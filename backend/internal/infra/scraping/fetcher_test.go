package scraping

import (
	"testing"

	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// wttjSpec returns a spec matching the seeded WTTJ json_api adapter result_fields.
func wttjSpec() llm.AdapterSpec {
	return llm.AdapterSpec{
		Search: llm.SearchConfig{
			URLTemplate:    "https://api.wttj.co/v2/search?query={keywords}&aroundQuery={location}&page={page}",
			ResultNodePath: "$.jobs[*]",
			ResultFields: map[string]string{
				"listing_url": "$.url",
				"title":       "$.name",
				"company":     "$.organization.name",
				"location":    "$.office.city",
				"posted_at":   "$.published_at",
				"external_id": "$.id",
			},
		},
	}
}

func TestParseCards(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		body      string
		wantLen   int
		wantFirst appscraping.Card
		wantErr   bool
	}{
		{
			name: "parses identity fields verbatim from a wttj card",
			body: `{"jobs":[{"url":"https://wttj/jobs/1","name":"Go Engineer",` +
				`"organization":{"name":"Acme"},"office":{"city":"Paris"},` +
				`"published_at":"2026-06-20T10:00:00Z","id":"ext-1"}]}`,
			wantLen: 1,
			wantFirst: appscraping.Card{
				ListingURL: "https://wttj/jobs/1",
				ExternalID: "ext-1",
				Title:      "Go Engineer",
				Company:    "Acme",
				Location:   "Paris",
			},
		},
		{
			name:    "returns empty for a response with no result node",
			body:    `{"jobs":[]}`,
			wantLen: 0,
		},
		{
			name:    "errors on non-JSON body",
			body:    `<html>nope</html>`,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cards, err := parseCards([]byte(tc.body), wttjSpec())

			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseCards() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCards() error = %v", err)
			}
			if len(cards) != tc.wantLen {
				t.Fatalf("parseCards() len = %d, want %d", len(cards), tc.wantLen)
			}
			if tc.wantLen == 0 {
				return
			}
			got := cards[0]
			if got.ListingURL != tc.wantFirst.ListingURL || got.ExternalID != tc.wantFirst.ExternalID ||
				got.Title != tc.wantFirst.Title || got.Company != tc.wantFirst.Company ||
				got.Location != tc.wantFirst.Location {
				t.Fatalf("parseCards() first = %+v, want %+v", got, tc.wantFirst)
			}
			if got.PostedAt == nil {
				t.Fatalf("parseCards() first.PostedAt = nil, want parsed time")
			}
			if len(got.Raw) == 0 {
				t.Fatalf("parseCards() first.Raw is empty, want verbatim card JSON")
			}
		})
	}
}

func TestBuildSearchURL(t *testing.T) {
	t.Parallel()

	target := appscraping.ScrapeTarget{Keywords: []string{"go", "backend"}, Location: "Paris"}
	got := buildSearchURL(
		"https://api.wttj.co/v2/search?query={keywords}&aroundQuery={location}&page={page}",
		target, 2)

	want := "https://api.wttj.co/v2/search?query=go+backend&aroundQuery=Paris&page=2"
	if got != want {
		t.Fatalf("buildSearchURL() = %q, want %q", got, want)
	}
}

func TestNormalizeJSONPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
	}{
		{"$.jobs[*]", "jobs"},
		{"$.organization.name", "organization.name"},
		{"$.url", "url"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := normalizeJSONPath(tc.in); got != tc.want {
			t.Fatalf("normalizeJSONPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
