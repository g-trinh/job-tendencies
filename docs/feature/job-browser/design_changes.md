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
