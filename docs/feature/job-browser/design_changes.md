# Job Browser — Design Changes (append-only)

## P2-FE-1 — Jobs list page (walking skeleton)

**Date:** 2026-06-26 · **Author:** fe-phase2

Implemented the minimal jobs list page (`frontend/src/features/jobs/`) mounted at `/`.
Phase 2 is deliberately thin: list only, no table/card toggle, filters, sort, or kanban
(those are Phase 3 per `docs/plan/tech-breakdown-phase-2.md`).

### API contract assumed (until P2-BE-5 lands)

`GET /api/jobs` (scoped via `X-Active-Profile`) → `JobSummary[]`, snake_case:
`id, title, company, location, url, contract_type, remote_policy, seniority, skills[],`
`salary_min, salary_max, understanding_score, sources[], first_seen, last_seen`.

`GET /api/active-profile` → `{ "id": "<uuid>", ... }` (FE reads `.id` only).

Enum string values mirror the backend `kernel` package exactly
(`contract_type`, `remote_policy`, `seniority`). Contract proposed to `be-phase2`.

### Notable decisions / flags

- **i18n alignment:** `frontend/src/i18n/fr.ts` enum keys were corrected to match the
  backend `kernel` enums. `job.contract.*` now uses `interim` (was `internship`/
  `apprenticeship`); `job.remote.*` now uses `on_site`/`hybrid`/`full_remote` (was
  `none`/`partial`/`full`). Added `job.seniority.*` and `job.working_days.*`. Without this
  the FR labels would have silently fallen back to raw enum keys.
- **Raw display fields not yet modelled backend-side:** `title`, `company`, `location`,
  `url` are required by the list but are absent from `ExtractedListing` and
  `data-model.job`. They must live on the job row (captured from the raw search result).
  Flagged to `be-phase2`.
- **Active-profile bootstrap:** `ActiveProfileProvider` now fetches `/api/active-profile`
  on mount and seeds the context id; scoped queries stay disabled until it resolves.
- **Fixture fallback:** `useJobs` resolves from `frontend/src/features/jobs/fixtures.ts`
  when `VITE_USE_FIXTURES=true`, so the page renders locally before P2-BE-5 ships.
  Defaults off; deployed dev and prod always use the real endpoint.

### Contract reconciliation with be-phase2 (2026-06-27)

`be-phase2` confirmed the stable `GET /api/jobs` shape; the FE was realigned:

- **`sources` is an array of objects**, not strings — `{ source_url }` (the FE consumes
  only `source_url` for Phase 2; `board_id` uuid / `raw_listing_id` are not displayed,
  and a board-name lookup for "found on: WTTJ" is deferred to Phase 3).
- **Original posting link** now comes from `sources[0].source_url`; the top-level `url`
  field was dropped.
- **`title`/`company`/`location` are optional** in the FE and rendered conditionally.
  They are not on the job row yet — `be-phase2` escalated to the architect whether to
  capture them verbatim from the search card (recommended) vs adding them to the LLM
  schema. The card heading falls back to "Voir l'offre" when `title` is absent.
- **`working_days`** is now displayed (FR), alongside contract/remote/seniority.
- **`first_seen`/`last_seen`** dropped from the FE contract — not needed until the list
  shows dates; `be-phase2` will omit them from the response for now.
- `GET /api/active-profile` confirmed as `{ id, name, search_keywords[], location }`;
  the FE reads `.id` only.

---

## P3-FE-1 — Phase 3 data layer (types, fixtures, hooks)

**Date:** 2026-06-27 · **Author:** fe-phase3

Extended the data layer to carry all Phase 3 fields. No UI changes in this entry.

### API contracts assumed (pending P3-JO-3 / P3-JO-4 / P3-JO-5 backend confirmation)

**`GET /api/jobs` — extended query params (P3-JO-3)**

All new params are optional; omitted params are not sent:

| Param | Type | Description |
|-------|------|-------------|
| `skills` | `string[]` | Filter by skill (multi-value) |
| `remote_policy` | enum string | `on_site \| hybrid \| full_remote` |
| `contract_type` | enum string | `cdi \| cdd \| freelance \| interim` |
| `salary_min` | integer | Minimum salary in whole euros |
| `salary_max` | integer | Maximum salary in whole euros |
| `location` | string | Free-text location substring |
| `board_id` | UUID | Filter to one board's jobs |
| `since` | ISO-8601 date | Only jobs first seen after this date |
| `confidence_min` | 0–100 | Minimum `understanding_score` threshold |
| `sort` | `date \| fit \| salary` | Sort field (default: `date`) |
| `sort_dir` | `asc \| desc` | Sort direction (default: `desc`) |

**`GET /api/jobs` — extended response fields (Phase 3 additions to each item)**

```json
{
  "application_status": "saved | applied | interview | offer | rejected | null",
  "fit_score": 87,
  "sources": [{ "board_id": "<uuid>", "source_url": "https://…", "board_name": "Welcome to the Jungle" }],
  "first_seen": "2026-06-20T10:00:00Z"
}
```

- `application_status`: `null` when the job is not yet tracked for this profile.
- `fit_score`: `null` until the scoring pipeline runs (P3-SC-2).
- `sources`: board-name display deferred in Phase 2 is now included.
- `first_seen`: re-introduced (was dropped in Phase 2 reconciliation) for date sort/filter display.

**`GET /api/jobs/{id}` — new endpoint (P3-JO-4)**

Extends the `GET /api/jobs` item with:
```json
{
  "description": "Full raw job text as scraped.",
  "field_confidence": { "contract_type": 95, "remote_policy": 88, … },
  "contact_id": "<uuid> | null",
  "last_seen": "2026-06-27T08:00:00Z",
  "expired_at": "<iso-date> | null"
}
```

**`GET /api/jobs/{id}/original` (P3-JO-4)**

Not consumed directly by the FE — the original posting URL is embedded in the job
detail as the top-level `url` field (established in Phase 2). If the backend provides
a redirect, the FE links to `url` directly and does not call `/original`.

**`PATCH /api/jobs/{id}/application` — new endpoint (P3-JO-5)**

Request: `{ "status": "saved | applied | interview | offer | rejected" }`
Response: `{ "status": "<enum>", "updated_at": "<iso-date>" }`

The first status in the lifecycle is `saved` (not `to_apply` — the FR key
`application.status.saved` was added to `fr.ts` to match). `to_apply` and
`abandoned` keys are retained in `fr.ts` for backward compatibility only.

---

## P3-FE-2 — Filters, sort, view toggle, table view

**Date:** 2026-06-27 · **Author:** fe-phase3

### Components added

- **`JobFiltersBar`** — controlled filter bar with all 8 filter params + sort field/direction.
  Skills entered as a comma-separated string, split on blur. Board select hardcodes the 4
  boards seeded by P3-BO-1; a dynamic `/api/boards` fetch is deferred to the board-manager
  FE story.
- **`ViewToggle`** — card/table toggle using `aria-pressed` button group.
- **`JobsTable`** — dense table view; job title links to `/jobs/:id` detail page.

### JobsPage changes

- `JobCard` heading now links to `/jobs/:id` (detail page) instead of the external
  `url`. The external link is shown separately as "Offre originale".
- Page manages `filters: JobFilters` and `view: View` as local state; both are passed
  to `useJobs(filters)` and `ViewToggle`.

---

## P3-FE-3 — Job detail page + confidence badges

**Date:** 2026-06-27 · **Author:** fe-phase3

### Components added

- **`ConfidenceBadge`** — inline badge rendering a per-field score with a confidence
  tier label. Tiers: high ≥ 70, medium 40–69, low < 40. `data-tier` attribute exposes
  the tier for CSS without relying on colour assertions in tests.
- **`JobDetailPage`** (`/jobs/:id`) — full detail view: description, per-field
  confidence badges, source boards with links, first/last seen dates, expiry notice,
  fit score, and application status selector.
- **`ApplicationStatusSelector`** — dropdown for updating a job's application status
  via `PATCH /api/jobs/{id}/application`. Shows "Sauvegarder cette offre" when status
  is null (first save), "Statut de candidature" when already tracked.

---

## P3-FE-4 — Application kanban

**Date:** 2026-06-27 · **Author:** fe-phase3

### Components added

- **`KanbanBoard`** — 5 columns (Sauvegardé → Candidature envoyée → Entretien →
  Offre reçue → Refusé). Each column groups jobs by `applicationStatus`. Jobs with
  `applicationStatus: null` are excluded. Each card has prev/next navigation buttons
  that call `useApplicationMutation`.
- **`KanbanPage`** (`/kanban`) — fetches all jobs via `useJobs()` (no filters),
  hands them to `KanbanBoard`. Grouping is client-side; a server-side
  `application_status` filter was not added to P3-JO-3 scope since the kanban data
  set is bounded by the active profile's tracked jobs.

### Notable decisions

- The kanban does not use drag-and-drop (deferred); users advance/revert one step at
  a time via labelled buttons.
- `back-link` from kanban to the list is "← Toutes les offres"; from detail to list
  is "← Retour aux offres".

---

### Final contract — architect ruling implemented (2026-06-27)

`be-phase2` landed the final `GET /api/jobs` contract; the FE was finalized to match:

- **Identity fields `title`/`company`/`location`/`url` are guaranteed on every row**,
  captured verbatim from the search card during scraping (NOT LLM-extracted), so a
  low-understanding extraction still yields a browsable, linkable card. The posting link
  now uses the top-level `url`; the heading still falls back to "Voir l'offre" defensively.
  `company`/`location` may be empty for HTML-fallback boards and render conditionally.
- **Enum fields can be the empty string `""`** (not `null`) when the LLM could not
  determine them. The card now skips empty enums via a truthiness guard instead of
  rendering the raw i18n key — covered by a regression test.
- **`sources`** (array of `{ board_id, raw_listing_id, source_url }`) is no longer needed
  by the FE for the posting link (top-level `url` supersedes it); board-name display is
  deferred to Phase 3, so `sources` is dropped from the FE types for now.
