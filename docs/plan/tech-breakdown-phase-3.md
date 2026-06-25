## Tech Breakdown: Phase 3 — Full backend contexts

**Design spec ref:** docs/v0.md
**Architecture ref:** overview.md §4/§6, pipeline.md, data-model.md, ADR-001, ADR-004
**Feature ref:** all of docs/feature/*/feature.md
**Plan ref:** docs/plan/development-plan.md (Phase 3)
**Teams:** Backend

Each task = domain → migrations → sqlc → app-service → http handlers → tests → quality gate
→ deploy-dev verify. Cross-context calls go through app-service interfaces only (ADR-001).

Sub-tracks: **A** (`profiles`, `boards`, `jobs`, `contacts`) are independent and parallel
after Phase 2. **B** (`scoring`, `scraping`, `extraction`, `dashboard`) join as deps land.

Test convention: domain/app logic → unit (table-driven `_test.go`); endpoints →
integration (httptest + dev Postgres); cross-context use cases → integration with the real
collaborating app-service.

---

### Track A — profiles (F2)

Refs for this group: profiles/feature.md, data-model.md (Profiles), overview.md §6 (Profiles)

#### P3-PR-1 — Implement profile CRUD + exactly-one-active + active-profile API

**Type:** Feature · **Owner:** Backend · **Dependencies:** Phase 2

**Description:** profile aggregate + CRUD (`/api/profiles`); enforce exactly one
`is_active`; `GET/PUT /api/active-profile`.
**Refs:** profiles/feature.md (active switch), data-model.md (profile), overview.md §6
**Acceptance Criteria:**
- Creating/activating a profile leaves exactly one active.
- `PUT /api/active-profile` switches the active profile.

#### P3-PR-2 — Implement POST /api/profiles/{id}/identity/import (single import)

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-PR-1, P1-BE-3

**Description:** Parse LinkedIn PDF (Claude document input or `ledongthuc/pdf`) → skills,
experience, seniority into an empty identity. **Single import only — no re-import endpoint**
(deferred).
**Refs:** profiles/feature.md (identity), overview.md open_questions + §6, tech_debt.md
**Acceptance Criteria:**
- Importing a PDF populates identity skills/experience/seniority for the profile.
- Endpoint rejects import when identity already populated (single-import guard).

#### P3-PR-3 — Implement skill list editing (PATCH identity/skills)

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-PR-2

**Description:** Add/remove/edit flat identity skills (no self-rating).
**Refs:** profiles/feature.md (manual skills), data-model.md (skill)
**Acceptance Criteria:** `PATCH /api/profiles/{id}/identity/skills` adds/removes a skill.

#### P3-PR-4 — Implement per-profile search config

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-PR-1

**Description:** Keywords + location stored per profile; consumed by the scraper target.
**Refs:** profiles/feature.md (search config), data-model.md (profile)
**Acceptance Criteria:** Updating keywords/location persists and re-scopes the scrape target.

#### P3-PR-5 — Implement conditions (dealbreakers + preferences)

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-PR-1

**Description:** `profile_conditions`: dealbreakers (contract_type, remote_policy, min_salary,
required_skills[]) + preferences (preferred_skills[], max_office_days, location_pref,
working_days).
**Refs:** profiles/feature.md (conditions), data-model.md (profile_conditions)
**Acceptance Criteria:** Conditions persist and are readable by the `scoring` app-service.

#### P3-PR-6 — Implement fit_weights with sum-to-100 validation

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-PR-1

**Description:** Per-profile `fit_weights` JSON; soft components must sum to 100%.
**Refs:** profiles/feature.md (fit-score weights table), data-model.md (fit_weights)
**Acceptance Criteria:** Saving weights whose soft components ≠ 100% is rejected.

---

### Track A — boards (F1)

Refs for this group: board-manager/feature.md, pipeline.md §1, data-model.md (Boards), ADR-004, tech_debt.md

#### P3-BO-1 — Implement board CRUD + enabled toggle + seed 4 boards

**Type:** Feature · **Owner:** Backend · **Dependencies:** Phase 2

**Description:** `/api/boards` CRUD incl. enabled toggle; seed WTTJ, Indeed, LinkedIn,
Glassdoor.
**Refs:** board-manager/feature.md, data-model.md (board)
**Acceptance Criteria:**
- CRUD works; four boards seeded.
- Disabling all boards is allowed (UI warns later — FE concern).

#### P3-BO-2 — Implement POST /api/boards/{id}/adapter/generate (LLM → declarative draft)

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-BO-1, P1-BE-3

**Description:** Given board URL + example page, call `AdapterGenerator` → declarative
`AdapterSpec` draft. **Never executable code.**
**Refs:** board-manager/feature.md (Option A), ADR-004, pipeline.md §1, tech_debt.md
**Acceptance Criteria:**
- Generates a draft AdapterSpec from an example page.
- Output contains no executable-code field.

#### P3-BO-3 — Validate AdapterSpec against schema

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-BO-2

**Description:** Schema validation for AdapterSpec (fetch_mode, url_template, param_map,
pagination, result fields, incremental) before approval.
**Refs:** pipeline.md §1, ADR-004 (validate before approval), tech_debt.md
**Acceptance Criteria:** Invalid specs are rejected with field-level errors.

#### P3-BO-4 — Implement adapter approve + get (draft → live)

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-BO-3

**Description:** `GET /api/boards/{id}/adapter`; `POST .../adapter/approve` promotes a
validated draft to the one live adapter (versioned).
**Refs:** board-manager/feature.md (review+approve), data-model.md (adapter status/version)
**Acceptance Criteria:** Approving makes exactly one adapter live; prior live is superseded.

#### P3-BO-5 — Implement global schedule GET/PUT (single row)

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-BO-1

**Description:** Single-row `schedule` (one global cron) exposed via `GET/PUT /api/schedule`.
**Refs:** board-manager/feature.md (one global schedule), data-model.md (schedule), infrastructure.md §5
**Acceptance Criteria:** `PUT /api/schedule` persists the cron string applied to Cloud Scheduler.

---

### Track A — jobs (F4)

Refs for this group: job-browser/feature.md, data-model.md (Jobs), overview.md §6 (Jobs)

#### P3-JO-1 — Implement the Job aggregate + repository + migrations

**Type:** Feature · **Owner:** Backend · **Dependencies:** Phase 2

**Description:** Job aggregate with structured fields + `field_confidence`,
`understanding_score`, `fingerprint`, `contact_id`, `first_seen/last_seen/expired_at`.
**Refs:** data-model.md (job), extraction-pipeline/feature.md (fields)
**Acceptance Criteria:** Job persists/loads with confidence JSON + understanding intact.

#### P3-JO-2 — Implement job_source ("found on") links

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-JO-1

**Description:** `job_source` rows linking a job to multiple boards/raw listings.
**Refs:** data-model.md (job_source), job-browser/feature.md (found on)
**Acceptance Criteria:** A job exposes its list of source boards.

#### P3-JO-3 — Implement GET /api/jobs with filters + sort

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-JO-1

**Description:** Filter-only listing (skills, remote_policy, contract_type, salary range,
location, board, date, confidence threshold); sort by date/fit/salary; scoped to active profile.
**Refs:** job-browser/feature.md (filters/sort), overview.md §6
**Acceptance Criteria:** Each filter narrows results; each sort orders correctly.

#### P3-JO-4 — Implement GET /api/jobs/{id} + /original

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-JO-1

**Description:** Job detail + link to the original posting.
**Refs:** job-browser/feature.md (link to original), overview.md §6
**Acceptance Criteria:** Detail returns fields + confidence badges data; `/original` resolves the source URL.

#### P3-JO-5 — Implement application kanban (PATCH /api/jobs/{id}/application)

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-JO-1

**Description:** `application` per (profile, job): Saved→Applied→Interview→Offer→Rejected.
**Refs:** job-browser/feature.md (kanban), data-model.md (application)
**Acceptance Criteria:** Status transitions persist per profile+job.

---

### Track A — contacts (F6)

Refs for this group: contacts-crm/feature.md, data-model.md (Contacts), overview.md §6 (Contacts)

#### P3-CO-1 — Implement Contact aggregate + CRUD + dedup

**Type:** Feature · **Owner:** Backend · **Dependencies:** Phase 2

**Description:** Contact CRUD; dedup by email|linkedin via `dedup_key`.
**Refs:** contacts-crm/feature.md (dedup), data-model.md (contact)
**Acceptance Criteria:** Upserting a contact with an existing email|linkedin merges, not duplicates.

#### P3-CO-2 — Implement tags + notes

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-CO-1

**Description:** Tags (`in-house`/`agency`, `responsive`/`ghosted`/`not-contacted`, custom) +
free-text notes.
**Refs:** contacts-crm/feature.md (tags)
**Acceptance Criteria:** Tags/notes persist and are filterable by tag.

#### P3-CO-3 — Implement GET /api/contacts/export.csv

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-CO-1

**Description:** CSV export of contacts.
**Refs:** contacts-crm/feature.md (CSV export), overview.md §6
**Acceptance Criteria:** Endpoint returns valid CSV with all contact fields + tags.

---

### Track B — scoring (F3 part) — needs jobs + profiles

Refs: pipeline.md §4, profiles/feature.md (weights), data-model.md (job_score)

#### P3-SC-1 — Implement the dealbreaker gate

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-JO-1, P3-PR-5

**Description:** Hard filters (contract type, remote policy, min salary, required skills) →
`passes_dealbreakers`.
**Refs:** pipeline.md §4, profiles/feature.md (dealbreakers)
**Acceptance Criteria:** A job failing any hard filter has `passes_dealbreakers=false`.

#### P3-SC-2 — Implement the weighted preference score + persist job_score

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-SC-1, P3-PR-6

**Description:** Weighted score (preferred-skills %, salary vs min, location, office days,
working days) per `fit_weights`; persist `job_score` per (job, profile) with
`component_breakdown`.
**Refs:** pipeline.md §4, profiles/feature.md (weights table), data-model.md (job_score)
**Acceptance Criteria:** `job_score.weighted_score` + breakdown match the weights for a fixture job.

---

### Track B — scraping (F3 part) — needs boards + profiles

Refs: pipeline.md §1/§2, data-model.md (Scraping), tech_debt.md (rate-limit, posted_at)

#### P3-SCR-1 — Implement the full two-phase crawl incl. html fallback

**Type:** Feature · **Owner:** Backend · **Dependencies:** P2-BE-3, P3-BO-4

**Description:** Generic crawler evaluating an AdapterSpec for both `json_api` and `html`
(goquery) fetch modes; search → detail.
**Refs:** pipeline.md §1 (fetch modes), tech_debt.md (no headless browser)
**Acceptance Criteria:** Both a json_api and an html adapter produce raw listings.

#### P3-SCR-2 — Implement board-side filtering via param_map

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-SCR-1, P3-PR-4

**Description:** Map active-profile keywords/location into board query params per `param_map`.
**Refs:** pipeline.md §1 (param_map)
**Acceptance Criteria:** Search URL carries mapped keywords/location for a fixture profile.

#### P3-SCR-3 — Implement incremental HWM + overlap buffer + safety cap

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-SCR-1

**Description:** Stop on per-(board,profile) `posted_at` high-water-mark minus ~36h overlap;
`safety_max_pages` guard.
**Refs:** pipeline.md §2, data-model.md (scrape_high_water_mark), tech_debt.md (posted_at quality)
**Acceptance Criteria:**
- First crawl (no HWM) runs to the safety cap, then sets the mark.
- Subsequent crawl stops at cutoff; overlap re-scan dedups via content_hash.

#### P3-SCR-4 — Implement the per-board rate limiter

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-SCR-1

**Description:** In-process `x/time/rate` limiter per board (correct only at
max-instances=1/concurrency=1).
**Refs:** deployment.md §4, tech_debt.md (single-instance pinning), ADR-003
**Acceptance Criteria:** Requests to one board are paced under its configured rate.

#### P3-SCR-5 — Implement scrape_run / scrape_run_board tracking + GET pipeline runs

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-SCR-1

**Description:** Record `scrape_run` + per-board `scrape_run_board` (status, counts, error);
expose `GET /api/pipeline/runs` + `/{id}`.
**Refs:** data-model.md (scrape_run, scrape_run_board), overview.md §6 (Pipeline), §9 (observability)
**Acceptance Criteria:** A run's per-board progress + counts are queryable.

---

### Track B — extraction (F3 part) — needs scraping + jobs + contacts

Refs: pipeline.md §3/§4, extraction-pipeline/feature.md, ADR-004, data-model.md

#### P3-EX-1 — Implement structured-output extraction schema

**Type:** Feature · **Owner:** Backend · **Dependencies:** P2-BE-4, P3-JO-1

**Description:** Full schema: each field `{value, confidence:0..100}` + top-level
`understanding:0..100`; persist to `job.field_confidence` / `understanding_score`.
**Refs:** pipeline.md §3, extraction-pipeline/feature.md (output fields), ADR-004
**Acceptance Criteria:** Extraction populates every output field with confidence + understanding.

#### P3-EX-2 — Implement cross-board dedup (fingerprint → merge + job_source)

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-EX-1, P3-JO-2

**Description:** Compute `fingerprint` (normalized title+company+location+salary); merge into
an existing job + append a `job_source` row.
**Refs:** pipeline.md §4 (dedup), data-model.md (fingerprint, job_source), job-browser/feature.md
**Acceptance Criteria:** Two boards' listings for the same role collapse to one job, two sources.

#### P3-EX-3 — Implement recruiter contact upsert

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-EX-1, P3-CO-1

**Description:** Upsert extracted recruiter into `contact` (dedup by email|linkedin); link
`job.contact_id`.
**Refs:** pipeline.md §4, contacts-crm/feature.md (auto-populated), data-model.md
**Acceptance Criteria:** A listing with recruiter fields creates/links a deduped contact.

#### P3-EX-4 — Trigger scoring after upsert

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-EX-2, P3-SC-2

**Description:** After job upsert, invoke the `scoring` app-service for the active profile.
**Refs:** pipeline.md §4 (scoring runs in extract-worker), overview.md §4 (context map)
**Acceptance Criteria:** A newly upserted job gets a `job_score` row.

#### P3-EX-5 — Handle extraction edge cases

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-EX-1

**Description:** Salary absent → null, confidence 0; hidden recruiter ("Easy Apply") → extract
visible, low understanding, flagged incomplete; FR + EN handled; raw never translated.
**Refs:** pipeline.md §3 (edge cases), extraction-pipeline/feature.md (edge cases)
**Acceptance Criteria:**
- Missing salary → null value, confidence 0.
- Hidden-recruiter listing flagged with low understanding; raw stored verbatim.

---

### Track B — dashboard (F5) — needs jobs + scoring + profiles

Refs: dashboard/feature.md, overview.md §6 (Dashboard), §4 (read-model)

#### P3-DA-1 — Implement GET /api/dashboard/skills/frequency

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-JO-3

**Description:** Top-N skills frequency across active-profile jobs.
**Refs:** dashboard/feature.md (skills frequency)
**Acceptance Criteria:** Returns ranked skill counts scoped to the active profile.

#### P3-DA-2 — Implement GET /api/dashboard/skills/trend

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-JO-3

**Description:** Skills frequency over time buckets (weeks/months).
**Refs:** dashboard/feature.md (skills trend)
**Acceptance Criteria:** Returns per-period skill counts.

#### P3-DA-3 — Implement GET /api/dashboard/matches

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-SC-2

**Description:** Jobs ranked by weighted fit score; exclude dealbreaker failures from top
matches.
**Refs:** dashboard/feature.md (match alerts), pipeline.md §4
**Acceptance Criteria:** Returns jobs ordered by `weighted_score`, dealbreaker-failed excluded from top.

#### P3-DA-4 — Implement GET /api/dashboard/stats

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-JO-3

**Description:** Stats cards: total scraped, new today/week, % remote, avg salary, top contract
type.
**Refs:** dashboard/feature.md (stats cards)
**Acceptance Criteria:** Returns all five stats scoped to the active profile.

---

### Dependency Graph

```
Phase 2 ─┬─ profiles: PR-1 → PR-2 → PR-3
         │            PR-1 → PR-4 / PR-5 / PR-6
         ├─ boards:   BO-1 → BO-2 → BO-3 → BO-4 ; BO-1 → BO-5
         ├─ jobs:     JO-1 → JO-2 / JO-3 / JO-4 / JO-5
         └─ contacts: CO-1 → CO-2 / CO-3

scoring:    JO-1 + PR-5 → SC-1 ; SC-1 + PR-6 → SC-2
scraping:   (P2-BE-3 + BO-4) → SCR-1 → SCR-2/3/4/5 ; SCR-2 needs PR-4
extraction: (P2-BE-4 + JO-1) → EX-1 → EX-2(+JO-2) / EX-3(+CO-1) / EX-5 ; EX-2 + SC-2 → EX-4
dashboard:  JO-3 → DA-1/DA-2/DA-4 ; SC-2 → DA-3
```

### Parallel tracks

- Track A's four contexts (profiles / boards / jobs / contacts) build fully in parallel.
- scoring starts once jobs + profiles land; scraping once boards + profiles land; extraction
  once scraping + jobs + contacts land; dashboard once jobs + scoring land.

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | PDF parse path: Claude document input vs `ledongthuc/pdf` | P3-PR-2 | Backend |
| 2 | Exact `fingerprint` normalization rules (salary bands? location canonicalization?) | P3-EX-2 | Backend/PM |
| 3 | Skills-trend bucket granularity default (weeks vs months) | P3-DA-2 | PM |
