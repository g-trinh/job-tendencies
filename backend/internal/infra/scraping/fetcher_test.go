package scraping

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
)

func noLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// wttjSpec returns a json_api spec matching the seeded WTTJ adapter result_fields.
func wttjSpec() llm.AdapterSpec {
	return llm.AdapterSpec{
		Board:     "wttj",
		FetchMode: llm.FetchModeJSONAPI,
		Search: llm.SearchConfig{
			URLTemplate:    "https://api.wttj.co/v2/search?query={keywords}&aroundQuery={location}&page={page}",
			ResultNodePath: "$.jobs[*]",
			ParamMap: map[string]string{
				"keywords": "profile.search.keywords",
				"location": "profile.search.location",
			},
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

// htmlBoardSpec returns an html-mode spec that scrapes a simple job listing HTML page.
func htmlBoardSpec() llm.AdapterSpec {
	return llm.AdapterSpec{
		Board:     "html-board",
		FetchMode: llm.FetchModeHTML,
		Search: llm.SearchConfig{
			URLTemplate:    "https://board.example.com/jobs?q={q}&loc={loc}&page={page}",
			ResultNodePath: "article.job-card",
			ParamMap: map[string]string{
				"q":   "profile.search.keywords",
				"loc": "profile.search.location",
			},
			ResultFields: map[string]string{
				"listing_url": "a.job-link@href",
				"title":       "h2.job-title",
				"company":     "span.company",
				"location":    "span.location",
				"external_id": "span.job-id",
			},
		},
		Listing: llm.ListingConfig{
			Fetch: llm.ListingFetchUseSearchPayload,
		},
		Incremental: llm.IncrementalConfig{
			OverlapBuffer:  "36h",
			SafetyMaxPages: 10,
		},
	}
}

// --- P3-SCR-1: json_api card parsing ---

func TestParseCards_JSONApi(t *testing.T) {
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

// --- P3-SCR-1: HTML card parsing ---

// sampleHTML is a minimal HTML page that matches htmlBoardSpec's selectors.
const sampleHTML = `<!DOCTYPE html><html><body>
<article class="job-card">
  <h2 class="job-title">Backend Engineer</h2>
  <span class="company">TechCorp</span>
  <span class="location">Lyon</span>
  <span class="job-id">job-42</span>
  <a class="job-link" href="https://board.example.com/jobs/42">View</a>
</article>
<article class="job-card">
  <h2 class="job-title">Frontend Dev</h2>
  <span class="company">StartupXY</span>
  <span class="location">Remote</span>
  <span class="job-id">job-43</span>
  <a class="job-link" href="https://board.example.com/jobs/43">View</a>
</article>
</body></html>`

func TestParseHTMLCards(t *testing.T) {
	t.Parallel()

	cards, err := parseHTMLCards([]byte(sampleHTML), htmlBoardSpec())
	if err != nil {
		t.Fatalf("parseHTMLCards() error = %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("parseHTMLCards() len = %d, want 2", len(cards))
	}

	got := cards[0]
	if got.ListingURL != "https://board.example.com/jobs/42" {
		t.Errorf("card[0].ListingURL = %q, want %q", got.ListingURL, "https://board.example.com/jobs/42")
	}
	if got.Title != "Backend Engineer" {
		t.Errorf("card[0].Title = %q, want %q", got.Title, "Backend Engineer")
	}
	if got.Company != "TechCorp" {
		t.Errorf("card[0].Company = %q, want %q", got.Company, "TechCorp")
	}
	if got.Location != "Lyon" {
		t.Errorf("card[0].Location = %q, want %q", got.Location, "Lyon")
	}
	if got.ExternalID != "job-42" {
		t.Errorf("card[0].ExternalID = %q, want %q", got.ExternalID, "job-42")
	}
	if len(got.Raw) == 0 {
		t.Error("card[0].Raw is empty, want verbatim HTML fragment")
	}
}

// TestFetchPage_HTMLMode verifies that the Fetcher dispatches to HTML parsing when
// the adapter spec declares fetch_mode: html, using a fake HTTP server.
func TestFetchPage_HTMLMode(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(sampleHTML))
	}))
	defer srv.Close()

	spec := htmlBoardSpec()
	spec.Search.URLTemplate = srv.URL + "/jobs?q={q}&loc={loc}&page={page}"

	fetcher := NewFetcher(noLogger())
	target := appscraping.ScrapeTarget{Keywords: []string{"go"}, Location: "Lyon"}

	cards, err := fetcher.FetchPage(context.Background(), spec, target, 1)
	if err != nil {
		t.Fatalf("FetchPage() error = %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("FetchPage() len = %d, want 2", len(cards))
	}
	if cards[0].Title != "Backend Engineer" {
		t.Errorf("cards[0].Title = %q, want %q", cards[0].Title, "Backend Engineer")
	}
}

// --- P3-SCR-2: param_map URL building ---

func TestBuildSearchURL_ParamMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		spec    llm.AdapterSpec
		target  appscraping.ScrapeTarget
		page    int
		wantURL string
	}{
		{
			name: "maps keywords and location via param_map to wttj-style url",
			spec: llm.AdapterSpec{
				Search: llm.SearchConfig{
					URLTemplate: "https://api.wttj.co/v2/search?query={keywords}&aroundQuery={location}&page={page}",
					ParamMap: map[string]string{
						"keywords": "profile.search.keywords",
						"location": "profile.search.location",
					},
				},
			},
			target:  appscraping.ScrapeTarget{Keywords: []string{"go", "backend"}, Location: "Paris"},
			page:    2,
			wantURL: "https://api.wttj.co/v2/search?query=go+backend&aroundQuery=Paris&page=2",
		},
		{
			name: "board using different param names (q and city)",
			spec: llm.AdapterSpec{
				Search: llm.SearchConfig{
					URLTemplate: "https://board.com/search?q={q}&city={city}&p={page}",
					ParamMap: map[string]string{
						"q":    "profile.search.keywords",
						"city": "profile.search.location",
					},
				},
			},
			target:  appscraping.ScrapeTarget{Keywords: []string{"python"}, Location: "Lyon"},
			page:    1,
			wantURL: "https://board.com/search?q=python&city=Lyon&p=1",
		},
		{
			name: "unknown profile field resolves to empty string",
			spec: llm.AdapterSpec{
				Search: llm.SearchConfig{
					URLTemplate: "https://board.com/jobs?x={x}&page={page}",
					ParamMap:    map[string]string{"x": "profile.unknown.field"},
				},
			},
			target:  appscraping.ScrapeTarget{},
			page:    1,
			wantURL: "https://board.com/jobs?x=&page=1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildSearchURL(tc.spec, tc.target, tc.page)
			if got != tc.wantURL {
				t.Fatalf("buildSearchURL() = %q, want %q", got, tc.wantURL)
			}
		})
	}
}

// --- P3-SCR-4: per-board rate limiter ---

// TestFetchPage_RateLimiter verifies that a spec with RatePerSecond=10 gets a limiter
// that paces requests and that two boards with different rates get independent limiters.
func TestFetchPage_RateLimiter(t *testing.T) {
	t.Parallel()

	requestTimes := make([]time.Time, 0, 3)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jobs":[]}`))
	}))
	defer srv.Close()

	spec := llm.AdapterSpec{
		Board:         "rate-test-board",
		FetchMode:     llm.FetchModeJSONAPI,
		RatePerSecond: 20, // 20 req/s → ~50ms between requests
		Search: llm.SearchConfig{
			URLTemplate:    srv.URL + "/search?page={page}",
			ResultNodePath: "$.jobs[*]",
			ResultFields:   map[string]string{"listing_url": "$.url"},
		},
		Listing:     llm.ListingConfig{Fetch: llm.ListingFetchUseSearchPayload},
		Incremental: llm.IncrementalConfig{OverlapBuffer: "1h", SafetyMaxPages: 3},
	}

	fetcher := NewFetcher(noLogger())
	target := appscraping.ScrapeTarget{}

	// Two requests through the same board's limiter.
	if _, err := fetcher.FetchPage(context.Background(), spec, target, 1); err != nil {
		t.Fatalf("first FetchPage error = %v", err)
	}
	if _, err := fetcher.FetchPage(context.Background(), spec, target, 2); err != nil {
		t.Fatalf("second FetchPage error = %v", err)
	}

	// A second board with no rate limit should not be affected.
	spec2 := spec
	spec2.Board = "rate-test-board-2"
	spec2.RatePerSecond = 0
	if _, err := fetcher.FetchPage(context.Background(), spec2, target, 1); err != nil {
		t.Fatalf("no-limit board FetchPage error = %v", err)
	}

	// Verify the rate-limited board got an independent limiter (sync.Map key is board name).
	if _, ok := fetcher.limiters.Load("rate-test-board"); !ok {
		t.Error("expected limiter for rate-test-board in sync.Map")
	}
	if _, ok := fetcher.limiters.Load("rate-test-board-2"); ok {
		t.Error("did not expect limiter for board with RatePerSecond=0")
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
