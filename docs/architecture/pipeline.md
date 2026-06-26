# Scrape + Extraction Pipeline

The pipeline is **asynchronous**, driven by Pub/Sub between separate worker binaries.
Stages: scrape → extract → dedup → score. See
[ADR-003](../adr/ADR-003-cloud-scheduled-worker-binaries.md) and
[ADR-004](../adr/ADR-004-llm-port-and-model-selection.md).

## 1. Defining the pages to scrape — the AdapterSpec

Scraping is a **two-phase crawl** (search/index pages → listing detail) expressed as
**declarative config**, not code. The LLM generates the spec from a user-supplied example;
the user reviews and approves it before it goes live. Per board, the spec records its
**fetch mode**: `json_api` (preferred — target the board's internal JSON/GraphQL search
endpoint) or `html` (fallback — CSS selectors over rendered HTML). No headless browser is
in scope (see `tech_debt.md`).

```yaml
board: welcometothejungle
fetch_mode: json_api          # json_api (preferred) | html (fallback)
search:
  url_template: "https://api.wttj.co/v2/search?query={keywords}&aroundQuery={location}&page={page}"
  method: GET                 # or POST + body_template for GraphQL
  param_map:                  # board-side filtering: profile field -> board query param
    keywords: profile.search.keywords
    location: profile.search.location
  pagination: { kind: query_param, param: page, start: 1 }
  result_node_path: "$.jobs[*]"          # JSONPath (json_api) or CSS selector (html)
  result_fields:
    listing_url: "$.url"
    posted_at:   "$.published_at"         # captured board-side, no detail fetch needed
    external_id: "$.id"                    # stable id for dedup when present
    title:    "$.name"                     # identity fields, captured verbatim off the card
    company:  "$.organization.name"
    location: "$.office.city"
listing:
  fetch: detail_page          # or: use_search_payload when search returns the full listing
  raw_capture: full_response  # raw JSON/HTML stored verbatim in GCS, never translated
incremental:
  cursor_field: posted_at
  overlap_buffer: "36h"       # re-scan window so late-indexed posts aren't missed
  safety_max_pages: 20        # SAFETY cap only, not the primary stop condition
```

Key decisions baked into the spec:
- **JSON-API first, HTML fallback** — recorded per board in `fetch_mode`. JSON endpoints
  are more stable than rendered HTML and avoid needing a browser.
- **Board-side filtering** — keywords + location pushed into the search URL via
  `param_map`. Params differ per board; mapping them is the adapter's job.
- **Incremental by `posted_at`** — pagination stops on age, not a fixed page count
  (see §2).

## 2. Scrape flow (high-water-mark incrementality)

The primary stop condition is the per-`(board, profile)` **high-water-mark** (most recent
`posted_at` from the previous run), with a `safety_max_pages` guard so bad data can't
paginate forever.

```
scrapeBoardForProfile(board, profile):
  hwm    = highWaterMark(board.id, profile.id)        // nil on first-ever crawl
  cutoff = hwm == nil ? nil : hwm - adapter.overlap_buffer   // ~36h overlap
  page   = adapter.search.pagination.start
  newest = nil

  loop:
    cards = fetchSearchPage(adapter, profile, page)    // board-side keyword/location filter
    if cards empty: break
    for card in cards:
      newest = max(newest, card.posted_at)
      if cutoff != nil and card.posted_at < cutoff:
        goto done                                      // incremental stop
      raw = fetchAndCapture(card.listing_url)          // verbatim -> GCS, content_hash
      if seen(content_hash): continue                  // idempotent overlap
      store(raw_listing); publish("listing.extract", raw_listing.id)
    page++
    if page - start >= adapter.safety_max_pages: break // SAFETY only
  done:
    if newest != nil: setHighWaterMark(board.id, profile.id, newest)
```

- **First-ever crawl** (`hwm == nil`): no cutoff → crawl to the safety cap, then set the
  mark. Incremental thereafter.
- **Overlap buffer** (~36h): re-scans recent postings so late-indexed jobs aren't missed;
  `content_hash` dedup makes the overlap skip-not-duplicate.
- **Rate limiting** is per board, enforced inside scrape-worker (see
  [deployment.md §rate-limiting](deployment.md)).

## 3. LLM extraction

Each `listing.extract` message → extract-worker loads the raw payload from GCS and calls
Claude through the `llm` port (`internal/domain/llm`, implemented in `internal/infra/llm`).

- **Identity fields not extracted**: `title`/`company`/`location`/`url` are captured
  verbatim off the search card (§1 `result_fields`) and carried on `raw_listing` → `job`;
  the LLM only produces the structured/enum fields below.
- **Structured output**: a JSON schema where every field is `{value, confidence:0..100}`
  plus a top-level `understanding:0..100` (overall parse quality). Confidence/understanding
  are produced by the model and stored on `job.field_confidence` / `job.understanding_score`.
- **Model**: default `claude-opus-4-8`; configurable to Sonnet 4.6 / Haiku 4.5 as cost
  levers. Prompt caching on the stable system prompt + schema; raw text after the cache
  breakpoint. Scheduled bulk runs may use the Batch API (50% cost) since latency isn't
  user-facing.
- **Edge cases**: salary absent → field null, confidence 0. Hidden recruiter ("Easy Apply")
  → extract what's visible, low understanding, flagged incomplete. FR and EN both handled;
  **raw never translated**, structured enums displayed in French by the frontend.

## 4. Dedup + scoring

- **Dedup** (in extract-worker): compute a `fingerprint` (normalized
  title+company+location+salary). Match an existing `job` → merge, append a `job_source`
  row ("found on: WTTJ, Indeed"). Recruiter upserted into `contact` (dedup by
  email/linkedin).
- **Scoring** (`scoring` context, runs in extract-worker after upsert): **dealbreaker gate
  first** — fail any hard filter (contract type, remote policy, min salary, required skills)
  → `passes_dealbreakers=false`, never surfaced in top matches. Then the **weighted
  preference score** (preferred-skills match %, salary vs minimum, location preference,
  office days vs max, working days) per `fit_weights`. Result stored per
  `(job_id, profile_id)` in `job_score`.

## 5. Reliability — retries & idempotency

- **Pub/Sub** gives at-least-once delivery, automatic retry with backoff, and dead-letter
  topics for poison messages. No custom queue.
- **Idempotency** comes from `content_hash` (raw), `fingerprint` (job), upsert semantics,
  and Pub/Sub message ids — redelivery is safe and produces no duplicates.
- **Re-extraction**: raw is retained in GCS, so `POST /api/jobs/{id}/reextract` re-publishes
  `listing.extract` to reprocess when extraction improves.
- **Expiry**: jobs not seen in a subsequent run of the same board are marked `expired_at`;
  data is retained.

## 6. Triggers

Scheduled and on-demand runs share one path: Cloud Scheduler publishes `scrape.tick` on the
global cron; the API publishes the same topic for on-demand runs. The scrape-worker handles
both identically.
