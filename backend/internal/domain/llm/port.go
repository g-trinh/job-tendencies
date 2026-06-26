package llm

import "context"

// AdapterGenerator generates a declarative scraping AdapterSpec from a job board URL
// and an example page response. The returned spec is data — not executable code.
// It must be reviewed and approved by the user before the scraper evaluates it.
//
// boardURL is the search or listing URL the user wants to scrape.
// exampleResponse is the raw HTML or JSON returned by that URL, used as the basis
// for selector/JSONPath generation.
type AdapterGenerator interface {
	GenerateAdapter(ctx context.Context, boardURL string, exampleResponse string) (*AdapterSpec, error)
}

// ListingExtractor extracts structured fields from a raw job listing payload
// (HTML or JSON) captured verbatim from a job board. Each returned field carries
// a per-field confidence score (0–100); the listing as a whole carries an
// understanding score (0–100).
//
// raw is the full response body stored in GCS by the scrape-worker.
type ListingExtractor interface {
	Extract(ctx context.Context, raw string) (*ExtractedListing, error)
}
