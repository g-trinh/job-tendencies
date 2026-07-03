// Package scraping provides the search fetcher and the Postgres
// repositories for the scrape-worker. The fetcher evaluates a declarative
// AdapterSpec against a board's JSON search endpoint or HTML page — it never executes code.
package scraping

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
	"golang.org/x/time/rate"

	appscraping "github.com/g-trinh/job-tendencies/internal/app/scraping"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// defaultFetchTimeout bounds each board HTTP request.
const defaultFetchTimeout = 20 * time.Second

// defaultUserAgent identifies this scraper to board servers on every outgoing request.
// Some boards (e.g. RemoteOK) return 403 to requests with no User-Agent header at all;
// a descriptive, identifying UA is the correct default regardless. Boards running their
// own bot-detection at the edge may still block this UA — that is outside our control.
const defaultUserAgent = "job-tendencies-scraper/1.0 (+https://github.com/g-trinh/job-tendencies)"

// Fetcher fetches and parses search pages from a board's JSON API or HTML pages by
// evaluating an AdapterSpec. It satisfies app/scraping.SearchFetcher.
//
// Rate limiting is per board via an in-process golang.org/x/time/rate Limiter keyed
// by spec.Board. This is only correct while scrape-worker runs at max-instances=1 /
// concurrency=1; see tech_debt.md "Scrape-worker pinned to one instance".
//
// ponytail: rate limiter is in-process; do not raise max-instances without first
// externalising per-board dispatch to Cloud Tasks (deployment.md §4).
type Fetcher struct {
	client   *http.Client
	logger   *slog.Logger
	limiters sync.Map // map[string]*rate.Limiter (board name → limiter)
}

// NewFetcher constructs a Fetcher with a bounded HTTP timeout.
func NewFetcher(logger *slog.Logger) *Fetcher {
	return &Fetcher{client: &http.Client{Timeout: defaultFetchTimeout}, logger: logger}
}

// FetchPage requests one search page and returns its result cards. For adapters with
// listing.fetch=detail_page it additionally fetches each listing URL as the raw payload;
// otherwise the search card itself is the raw payload (use_search_payload).
//
// The fetch mode in the spec (json_api or html) controls parsing: json_api applies
// JSONPath selectors via gjson; html applies CSS selectors via goquery (ADR-004).
func (f *Fetcher) FetchPage(ctx context.Context, spec llm.AdapterSpec, target appscraping.ScrapeTarget, page int) ([]appscraping.Card, error) {
	lim := f.acquireLimiter(spec)
	endpoint := buildSearchURL(spec, target, page)

	var (
		cards []appscraping.Card
		err   error
	)

	switch spec.FetchMode {
	case llm.FetchModeHTML:
		body, fetchErr := f.getHTML(ctx, endpoint, lim)
		if fetchErr != nil {
			return nil, fmt.Errorf("fetching html search page: %w", fetchErr)
		}
		cards, err = parseHTMLCards(body, spec)
		if err != nil {
			return nil, fmt.Errorf("parsing html search cards: %w", err)
		}
	default: // json_api
		body, fetchErr := f.get(ctx, endpoint, lim)
		if fetchErr != nil {
			return nil, fmt.Errorf("fetching search page: %w", fetchErr)
		}
		cards, err = parseCards(body, spec)
		if err != nil {
			return nil, fmt.Errorf("parsing search cards: %w", err)
		}
	}

	cards = f.dropCardsWithoutURL(ctx, cards)

	if spec.Listing.Fetch == llm.ListingFetchDetailPage {
		for i := range cards {
			var detail []byte
			if spec.FetchMode == llm.FetchModeHTML {
				detail, err = f.getHTML(ctx, cards[i].ListingURL, lim)
			} else {
				detail, err = f.get(ctx, cards[i].ListingURL, lim)
			}
			if err != nil {
				return nil, fmt.Errorf("fetching listing detail %q: %w", cards[i].ListingURL, err)
			}
			cards[i].Raw = detail
		}
	}
	return cards, nil
}

// dropCardsWithoutURL removes cards whose listing URL is empty. An empty URL would flow
// through to the job browser as a broken link, and would fail the detail-page fetch, so
// such cards are skipped and logged rather than captured.
func (f *Fetcher) dropCardsWithoutURL(ctx context.Context, cards []appscraping.Card) []appscraping.Card {
	kept := cards[:0]
	for _, c := range cards {
		if c.ListingURL == "" {
			f.logger.WarnContext(ctx, "skipping search card with empty listing url",
				"title", c.Title, "company", c.Company)
			continue
		}
		kept = append(kept, c)
	}
	return kept
}

// acquireLimiter returns the per-board rate limiter for the spec, creating it on first
// use. Returns nil when the spec carries no rate limit (RatePerSecond == 0).
func (f *Fetcher) acquireLimiter(spec llm.AdapterSpec) *rate.Limiter {
	if spec.RatePerSecond <= 0 || spec.Board == "" {
		return nil
	}
	if lim, ok := f.limiters.Load(spec.Board); ok {
		return lim.(*rate.Limiter)
	}
	lim := rate.NewLimiter(rate.Limit(spec.RatePerSecond), 1)
	actual, _ := f.limiters.LoadOrStore(spec.Board, lim)
	return actual.(*rate.Limiter)
}

// get performs a GET with Accept: application/json and returns the response body,
// failing on non-2xx status. If lim is non-nil it waits for a token before sending.
func (f *Fetcher) get(ctx context.Context, rawURL string, lim *rate.Limiter) ([]byte, error) {
	return f.fetch(ctx, rawURL, "application/json", lim)
}

// getHTML performs a GET with Accept: text/html and returns the response body,
// failing on non-2xx status. If lim is non-nil it waits for a token before sending.
func (f *Fetcher) getHTML(ctx context.Context, rawURL string, lim *rate.Limiter) ([]byte, error) {
	return f.fetch(ctx, rawURL, "text/html,application/xhtml+xml", lim)
}

// fetch performs a GET with the given Accept header, enforces the per-board rate
// limiter when provided, and returns the response body.
func (f *Fetcher) fetch(ctx context.Context, rawURL, accept string, lim *rate.Limiter) ([]byte, error) {
	if lim != nil {
		if err := lim.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter wait: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", defaultUserAgent)

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

// buildSearchURL constructs the board search URL by substituting {param} placeholders
// using spec.Search.ParamMap to resolve profile fields, plus the special {page}
// placeholder from the pagination config. ParamMap entries map board parameter names to
// profile field paths (e.g. "keywords" → "profile.search.keywords").
func buildSearchURL(spec llm.AdapterSpec, target appscraping.ScrapeTarget, page int) string {
	pairs := make([]string, 0, (len(spec.Search.ParamMap)+1)*2)
	for param, fieldPath := range spec.Search.ParamMap {
		pairs = append(pairs, "{"+param+"}", resolveProfileField(fieldPath, target))
	}
	pairs = append(pairs, "{page}", strconv.Itoa(page))
	return strings.NewReplacer(pairs...).Replace(spec.Search.URLTemplate)
}

// resolveProfileField maps a profile field path from spec.Search.ParamMap to its
// current value on the active ScrapeTarget.
func resolveProfileField(fieldPath string, target appscraping.ScrapeTarget) string {
	switch fieldPath {
	case "profile.search.keywords":
		return url.QueryEscape(strings.Join(target.Keywords, " "))
	case "profile.search.location":
		return url.QueryEscape(target.Location)
	default:
		return ""
	}
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

// parseHTMLCards evaluates the spec's result_node_path (CSS selector) and result_fields
// (CSS selector expressions) over an HTML response body, returning one Card per matched
// node. Field expressions support the "selector@attr" syntax to extract an attribute
// value; without "@attr" the element's trimmed text content is used.
func parseHTMLCards(body []byte, spec llm.AdapterSpec) ([]appscraping.Card, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing html: %w", err)
	}

	fields := spec.Search.ResultFields
	urlExpr := fields[llm.ResultFieldListingURL]
	postedExpr := fields["posted_at"]
	extIDExpr := fields["external_id"]
	titleExpr := fields["title"]
	companyExpr := fields["company"]
	locationExpr := fields["location"]

	var cards []appscraping.Card
	doc.Find(spec.Search.ResultNodePath).Each(func(_ int, node *goquery.Selection) {
		raw, _ := node.Html()
		card := appscraping.Card{
			ListingURL: selectorValue(node, urlExpr),
			ExternalID: selectorText(node, extIDExpr),
			Title:      selectorText(node, titleExpr),
			Company:    selectorText(node, companyExpr),
			Location:   selectorText(node, locationExpr),
			Raw:        []byte(raw),
		}
		if postedExpr != "" {
			if t := parseTime(selectorText(node, postedExpr)); t != nil {
				card.PostedAt = t
			}
		}
		cards = append(cards, card)
	})
	return cards, nil
}

// selectorValue evaluates a CSS selector expression against node. Expressions of the
// form "selector@attr" return the attribute's value; plain selectors return trimmed text.
func selectorValue(node *goquery.Selection, expr string) string {
	if expr == "" {
		return ""
	}
	if idx := strings.LastIndex(expr, "@"); idx != -1 {
		sel := expr[:idx]
		attr := expr[idx+1:]
		matched := node.Find(sel)
		if matched.Length() == 0 {
			matched = node.Filter(sel)
		}
		val, _ := matched.First().Attr(attr)
		return strings.TrimSpace(val)
	}
	return selectorText(node, expr)
}

// selectorText selects elements by CSS selector within node and returns their combined
// trimmed text. An empty selector returns an empty string.
func selectorText(node *goquery.Selection, sel string) string {
	if sel == "" {
		return ""
	}
	matched := node.Find(sel)
	if matched.Length() == 0 {
		matched = node.Filter(sel)
	}
	return strings.TrimSpace(matched.First().Text())
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
