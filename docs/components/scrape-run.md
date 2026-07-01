# Component: ScrapeRun

One pipeline execution (scheduled cron or on-demand API trigger), recorded for observability and status polling.

## Properties

| Property | Type | Description |
|----------|------|-------------|
| ID | ScrapeRunID | Stable identifier |
| ProfileID | ProfileID | The profile this run was scoped to |
| Trigger | string | `on_demand` or `scheduled` |
| Status | string | `queued` → `running` → `done` \| `error` |
| CreatedAt | time.Time | When the run was recorded |
| FinishedAt | *time.Time | When the run reached its terminal state (nil if still running) |
| Boards | []ScrapeRunBoard | Per-board progress entries (populated by GetRun only) |

## ScrapeRunBoard

| Property | Type | Description |
|----------|------|-------------|
| ID | ScrapeRunBoardID | Stable identifier |
| RunID | ScrapeRunID | Parent run |
| BoardID | BoardID | Board that was crawled |
| Status | string | `running` → `done` \| `error` |
| PagesFetched | int | Number of search pages fetched |
| ListingsCaptured | int | Number of genuinely new raw listings stored (not deduped) |
| Error | string | Non-empty when the board crawl failed |
| StartedAt | *time.Time | When this board's crawl began |
| FinishedAt | *time.Time | When this board's crawl ended |

## Lifecycle

1. `CreateRun` (app/pipeline) inserts with `status=queued`, publishes `scrape.tick`.
2. `MarkRunning` (infra/pipeline via RunTracker) transitions to `running`; creates a new scheduled run if no run_id in the message.
3. For each board: `TrackBoard` opens a `scrape_run_board` row; `FinishBoard` records counts and error.
4. `FinishRun` sets terminal status (`done` or `error`) and `finished_at`.

## Endpoints

- `POST /api/pipeline/runs` — create an on-demand run, returns run_id
- `GET /api/pipeline/runs` — list recent runs (newest first, limit 50)
- `GET /api/pipeline/runs/{id}` — run with per-board breakdown

## Notes

- The [RunTracker](../architecture/pipeline.md) interface (`app/scraping`) is satisfied by `infra/pipeline.Repository`; pass `nil` for no-op tracking.
- Overlap re-scans within the HWM buffer show `listings_captured` lower than `pages_fetched * cards_per_page` — that is expected; deduped cards are counted as pages fetched but not listed as captured.
