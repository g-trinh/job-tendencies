## Tech Breakdown: Phase 2 — Walking skeleton (vertical slice, on dev)

**Design spec ref:** docs/v0.md (dataflow)
**Architecture ref:** pipeline.md §1/§2/§3, deployment.md §1, overview.md §6, ADR-003, ADR-004
**Feature ref:** board-manager/feature.md, profiles/feature.md, extraction-pipeline/feature.md, job-browser/feature.md
**Plan ref:** docs/plan/development-plan.md (Phase 2)
**Teams:** Backend, Frontend

Thinnest end-to-end path on Cloud Run dev. **Deliberately skips** adapter generation,
dedup/merge, contacts, scoring, dashboard, full filters/kanban, PDF import, expiry,
reextract — those are Phase 3.

---

### Tasks

---

#### P2-BE-1 — Seed one board with a pre-written approved AdapterSpec + GET /api/boards

**Type:** Feature
**Owner:** Backend
**Dependencies:** Phase 1

**Description:**
Seed WTTJ + a hand-written, approved `json_api` AdapterSpec (skip LLM generation here);
expose `GET /api/boards`.

**Refs:** board-manager/feature.md, pipeline.md §1 (AdapterSpec), data-model.md (board/adapter)

**Acceptance Criteria:**
- `GET /api/boards` returns WTTJ with an approved adapter.
- AdapterSpec validates against the spec schema.

---

#### P2-BE-2 — Provision one default active profile (min)

**Type:** Feature
**Owner:** Backend
**Dependencies:** Phase 1

**Description:**
A single default profile (hardcoded keywords/location) marked active; resolvable via
`X-Active-Profile`. No PDF import, conditions, or weights yet.

**Refs:** profiles/feature.md, overview.md §6 (active-profile), data-model.md (profile)

**Acceptance Criteria:**
- `GET /api/active-profile` returns the default profile id.
- Scraper target reads keywords/location from it.

---

#### P2-BE-3 — Implement the json_api crawl loop with HWM, raw→GCS, publish

**Type:** Feature
**Owner:** Backend
**Dependencies:** P2-BE-1, P2-BE-2

**Description:**
scrape-worker crawl loop (`json_api` only): fetch search page, capture raw → GCS with
`content_hash`, set/read high-water-mark, publish `listing.extract` per new raw listing.

**Refs:** pipeline.md §2 (HWM algorithm), data-model.md (raw_listing, scrape_high_water_mark)

**Acceptance Criteria:**
- A run stores raw blobs in GCS and publishes one `listing.extract` per new listing.
- Re-run within the overlap window skips by `content_hash` (no duplicate publish).

---

#### P2-BE-4 — Extract a raw listing into one job (min)

**Type:** Feature
**Owner:** Backend
**Dependencies:** P2-BE-3, P1-BE-3

**Description:**
extract-worker handles `listing.extract`: load raw from GCS, call Claude structured extract,
create one `job` row. Skip dedup/merge, contacts, scoring.

**Refs:** pipeline.md §3, extraction-pipeline/feature.md, ADR-004

**Acceptance Criteria:**
- A `listing.extract` message produces one `job` with structured fields + understanding.
- Raw is never translated.

---

#### P2-BE-5 — Implement GET /api/jobs and GET /api/jobs/{id} (min)

**Type:** Feature
**Owner:** Backend
**Dependencies:** P2-BE-4

**Description:**
Minimal jobs read API: list + detail, scoped to active profile. No filters/sort yet.

**Refs:** job-browser/feature.md, overview.md §6 (Jobs)

**Acceptance Criteria:**
- `GET /api/jobs` returns the created job; `GET /api/jobs/{id}` returns its detail.

---

#### P2-BE-6 — Trigger pipeline via POST /api/pipeline/runs

**Type:** Feature
**Owner:** Backend
**Dependencies:** P2-BE-3

**Description:**
`POST /api/pipeline/runs` publishes `scrape.tick` (on-demand); both workers consume via real
OIDC push. Same path scheduled runs will use.

**Refs:** pipeline.md §6 (triggers), overview.md §6 (Pipeline), ADR-003

**Acceptance Criteria:**
- `POST /api/pipeline/runs` returns a run id and publishes `scrape.tick`.
- scrape-worker receives the push and starts a crawl.

---

#### P2-FE-1 — Render a jobs list page against the dev API

**Type:** Feature
**Owner:** Frontend
**Dependencies:** P2-BE-5

**Description:**
One React page listing jobs from `GET /api/jobs`, keyed on active-profile id.

**Refs:** job-browser/feature.md, overview.md §7

**Acceptance Criteria:**
- The page lists jobs returned by the dev API; structured enums shown in French.

---

#### P2-1 — Deploy the skeleton to dev and verify end-to-end

**Type:** Chore
**Owner:** Full-stack
**Dependencies:** P2-BE-6, P2-BE-4, P2-FE-1

**Description:**
Deploy all three binaries to Cloud Run dev; run an on-demand pipeline; confirm a job appears
in the browser. Proves Pub/Sub + GCS + Claude + DB + OIDC wiring.

**Refs:** deployment.md §1, development-plan.md (Phase 2 exit)

**Acceptance Criteria:**
- On dev: `POST /api/pipeline/runs` → scrape → extract → a job visible via the SPA.

---

### Dependency Graph

```
P2-BE-1 ┐
P2-BE-2 ┴→ P2-BE-3 → P2-BE-4 → P2-BE-5 → P2-FE-1 ┐
                  └→ P2-BE-6 ───────────────────┴→ P2-1
```

### Parallel tracks

- P2-BE-1 and P2-BE-2 build concurrently.
- P2-FE-1 can stub against a fixture until P2-BE-5 lands.

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | WTTJ JSON search endpoint + param names for the seed adapter | P2-BE-1, P2-BE-3 | Backend (reverse-engineer) |
