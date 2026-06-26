// Package scraping provides the json_api search fetcher and the Postgres
// repositories for the scrape-worker. The fetcher evaluates a declarative
// AdapterSpec against a board's JSON search endpoint — it never executes code.
package scraping

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// defaultFetchTimeout bounds each board HTTP request.
const defaultFetchTimeout = 20 * time.Second

// Fetcher fetches and parses search pages from a board's JSON API by evaluating an
// AdapterSpec. It satisfies app/scraping.SearchFetcher.
type Fetcher struct {
	client *http.Client
}

// NewFetcher constructs a json_api Fetcher with a bounded HTTP timeout.
func NewFetcher() *Fetcher {
	return &Fetcher{client: &http.Client{Timeout: defaultFetchTimeout}}
}

// FetchPage requests one search page and returns its result cards. For adapters with
// listing.fetch=detail_page it additionally fetches each listing URL as the raw payload;
// otherwise the search card itself is the raw payload (use_search_payload).
func (f *Fetcher) FetchPage(ctx context.Context, spec llm.AdapterSpec, target appscraping.ScrapeTarget, page int) ([]appscraping.Card, error) {
	endpoint := buildSearchURL(spec.Search.URLTemplate, target, page)

	body, err := f.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetching search page: %w", err)
	}

	cards, err := parseCards(body, spec)
	if err != nil {
		return nil, fmt.Errorf("parsing search cards: %w", err)
	}

	if spec.Listing.Fetch == llm.ListingFetchDetailPage {
		for i := range cards {
			detail, err := f.get(ctx, cards[i].ListingURL)
			if err != nil {
				return nil, fmt.Errorf("fetching listing detail %q: %w", cards[i].ListingURL, err)
			}
			cards[i].Raw = detail
		}
	}
	return cards, nil
}

// get performs a GET and returns the response body, failing on non-2xx status.
func (f *Fetcher) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d from %q", resp.StatusCode, rawURL)
	}
	return body, nil
}

// buildSearchURL substitutes {keywords}, {location} and {page} placeholders in the
// template, URL-escaping the profile-supplied values.
func buildSearchURL(template string, target appscraping.ScrapeTarget, page int) string {
	replacer := strings.NewReplacer(
		"{keywords}", url.QueryEscape(strings.Join(target.Keywords, " ")),
		"{location}", url.QueryEscape(target.Location),
		"{page}", strconv.Itoa(page),
	)
	return replacer.Replace(template)
}

// parseCards evaluates the spec's result_node_path and result_fields over a JSON
// response body, returning one Card per result entry. The raw payload of each card is
// the verbatim JSON of that entry (used directly for use_search_payload adapters).
func parseCards(body []byte, spec llm.AdapterSpec) ([]appscraping.Card, error) {
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("response body is not valid JSON")
	}

	fields := spec.Search.ResultFields
	nodes := gjson.GetBytes(body, normalizeJSONPath(spec.Search.ResultNodePath))
	urlPath := normalizeJSONPath(fields[llm.ResultFieldListingURL])
	postedPath := normalizeJSONPath(fields["posted_at"])
	externalPath := normalizeJSONPath(fields["external_id"])
	titlePath := normalizeJSONPath(fields["title"])
	companyPath := normalizeJSONPath(fields["company"])
	locationPath := normalizeJSONPath(fields["location"])

	var cards []appscraping.Card
	for _, node := range nodes.Array() {
		card := appscraping.Card{
			ListingURL: node.Get(urlPath).String(),
			ExternalID: node.Get(externalPath).String(),
			Title:      getField(node, titlePath),
			Company:    getField(node, companyPath),
			Location:   getField(node, locationPath),
			Raw:        []byte(node.Raw),
		}
		if postedPath != "" {
			if t := parseTime(node.Get(postedPath).String()); t != nil {
				card.PostedAt = t
			}
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// getField returns the string at path within node, or "" when the path is unset.
func getField(node gjson.Result, path string) string {
	if path == "" {
		return ""
	}
	return node.Get(path).String()
}

// normalizeJSONPath converts a JSONPath-style expression from an AdapterSpec into the
// gjson path syntax used to evaluate it: it strips a leading "$." and any "[*]" wildcards.
func normalizeJSONPath(path string) string {
	p := strings.TrimPrefix(path, "$.")
	p = strings.TrimPrefix(p, "$")
	p = strings.ReplaceAll(p, "[*]", "")
	return strings.Trim(p, ".")
}

// parseTime parses a board timestamp (RFC3339 or date-only), returning nil when empty
// or unrecognised.
func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
