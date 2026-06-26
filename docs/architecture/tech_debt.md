# Tech Debt & Attention Points

Living register of accepted risks and deliberate constraints future developers could
silently break. Lower ceremony than an ADR.

## Declarative adapters are load-bearing — never execute LLM-generated code
Board adapters are **declarative specs** (JSON/JSONPath/CSS selectors + URL templates),
evaluated by a generic crawler. The LLM generates a spec, a human approves it; we never
`exec` LLM-written code.
- **Why it matters**: executing generated code reintroduces arbitrary-code-execution risk
  and forces recompiles.
- **Watch for**: any change that "upgrades" adapters to generated/executable code. Keep
  them declarative; validate the spec against a schema before approval.

## No headless browser — JSON-API-first (deferred headless bin)
`scrape-worker` is a lightweight Go HTTP/JSON client. The four named boards (WTTJ, Indeed,
LinkedIn, Glassdoor) have internal JSON APIs, so no Playwright/chromedp is shipped.
- **Trigger to revisit**: onboarding a board with **no findable JSON API**.
- **Then**: headless goes in a **separate** `cmd/headless-scrape-worker` (its own Cloud Run
  service, ~2 GB/2 vCPU, possibly `min-instances=1`) so the lean JSON path stays cheap.
  Evaluate proxy/maintenance cost against that board's value first.
- `// ponytail`: no browser until a board actually forces it; keep scrape-worker a plain
  HTTP client.

## JSON-API targeting is accepted reverse-engineering
Adapters prefer internal/undocumented JSON/GraphQL endpoints (`fetch_mode: json_api`).
- **Why it matters**: these endpoints can change without notice and may sit in tension with
  board ToS.
- **Watch for**: breakage is expected and handled by regenerating the adapter
  (human-reviewed). Do **not** add auth-bypass or credential-replay behavior.

## Scrape-worker pinned to one instance for rate limiting
Per-board rate limiting uses an in-process `x/time/rate` limiter, which is only correct
because `scrape-worker` runs at `max-instances=1`, `concurrency=1`.
- **Watch for**: raising `max-instances` without first externalizing the limiter. A board's
  rate limit is global to the board (shared across tenants), so horizontal scaling requires
  **Cloud Tasks per-board queues** — not a bigger in-process limiter.

## LLM self-reported confidence is a heuristic
`field_confidence` and `understanding_score` come from the extraction model, not a
calibrated probability.
- **Watch for**: treating thresholds as guarantees. They are UX filters/badges only.

## Incremental crawl depends on board `posted_at` quality
Pagination stops on a `posted_at` high-water-mark + ~36h overlap buffer; `safety_max_pages`
is only a guard.
- **Watch for**: a board with coarse/missing/garbage `posted_at` — the overlap buffer is the
  only guard against missing late-indexed posts, and the safety cap the only guard against
  runaway pagination. Re-check both when adding a board.

## Identity fields are card-captured — HTML-fallback boards may lack verbatim company/location
`title`/`company`/`location`/`url` are captured verbatim off the search card
(`result_fields`), not LLM-extracted, and feed the deterministic `fingerprint`.
- **Watch for**: an `html`-fallback board whose search cards don't expose a clean `company`
  or `location` (only present on the detail page, or absent). Then those identity fields are
  empty/degraded, which weakens cross-board dedup.
- **Then**: pull the field from the detail capture for that board, or accept reduced dedup
  precision for it — do **not** silently fall back to LLM-extracting identity fields, which
  reintroduces nondeterminism into `fingerprint`.

## Deferred: LinkedIn PDF re-import merge strategy
Re-importing a LinkedIn PDF currently assumes **overwrite** of identity. Merge-vs-overwrite
is deferred (per v0). Decide at build time before exposing re-import.

## Deferred Tier-1 security controls
Current posture is Tier 0 (single user, pre-launch). Deferred until MAU/paying users:
Cloud Armor/WAF, private-IP Cloud SQL + VPC connector, automated backups with tested
restore, IaC/image vulnerability scanning. See [infrastructure.md §6](infrastructure.md).
