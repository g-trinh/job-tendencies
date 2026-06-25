---
ai_context:
  need: "Master development plan: scaffold + implement Job Tendencies end-to-end"
  strategy: "Walking skeleton first, then full contexts in dependency order, frontend wired per-context behind a static template/ track"
  dev_model: "Pure cloud dev — all binaries run on Cloud Run dev with real Pub/Sub push. No local worker run, no local messaging shim. Inner loop = build image + deploy to dev. Dev infra env is built up front (not Phase 6)."
  deliverable: "This master plan + per-feature atomic tickets produced by the tech-breakdown skill"
  agents: ["architect", "backend-developer", "frontend-developer", "ui-integrator", "cloud-ops"]
  related:
    - "docs/architecture/overview.md"
    - "docs/architecture/pipeline.md"
    - "docs/architecture/deployment.md"
    - "docs/architecture/infrastructure.md"
    - "docs/adr/ADR-001..004"
---

# Job Tendencies — Master Development Plan

This plan sequences the build so development agents can scaffold and implement the whole
application without ordering surprises. It is grounded in the architecture docs and ADRs.

Build strategy (decided): **walking skeleton first**, then full contexts in dependency
order, with frontend wired per-context behind an early static `template/` track.

**Dev model (decided): pure cloud.** Every binary runs on **Cloud Run dev** with **real
Pub/Sub push** — there is no local worker run and no local messaging shim. The inner loop
is *build image → deploy to dev → test in cloud*. Consequence: the **dev infra environment
is built up front** (Phase 1, by `cloud-ops`) because the walking skeleton already needs
real Pub/Sub topics + Cloud Run services. The `messaging` port has exactly **one** adapter
(GCP Pub/Sub).

## 0. Conventions for this plan

- **Owner agent** per task: `backend-developer` (Go in `backend/`), `frontend-developer`
  (React in `frontend/`), `ui-integrator` (static `template/`), `cloud-ops` (`infra/`),
  `architect` (open design questions only).
- **DoD (every backend task):** domain has no outward imports; cross-context calls go
  through app-service interfaces only (ADR-001); migrations + sqlc queries committed; unit
  tests via `go-testing` mapping acceptance criteria; quality gate green
  (`go-vet`, `go-golangci-lint`, `go-goimports`, `go-govulncheck`); `slog` at boundaries.
  Worker behavior verified by **deploying to dev** and exercising the real Pub/Sub path.
- **DoD (every frontend task):** matches `template/` reference; French via i18n dict (raw
  listing text verbatim); server state via TanStack Query keyed on active-profile id;
  tests via `frontend-testing`.
- **Repo root layout:** `backend/` (one Go module), `frontend/` (Vite SPA),
  `template/` (static HTML/CSS reference), `infra/` (OpenTofu), `docs/`.

## 1. Dependency graph (drives ordering)

Contexts and their build dependencies (from `overview.md` §4):

```
llm (port)        ← boards, extraction
profiles (core)   ← scoring, scraping, dashboard          [needs llm for PDF import]
boards            ← scraping                                [needs llm for adapter gen]
jobs (core)       ← scoring, extraction, dashboard
contacts          ← extraction
scoring (core)    : depends on jobs + profiles
scraping          : depends on boards + profiles
extraction (core) : depends on scraping + jobs + contacts + llm
dashboard         : depends on jobs + scoring + profiles
```

Binary composition (thin mains, ADR-001 / `deployment.md` §3):
- `cmd/api`: profiles, boards, jobs, dashboard, contacts + `messaging.Publisher`.
- `cmd/scrape-worker`: scraping + boards/profiles readers + GCS writer + publisher.
- `cmd/extract-worker`: extraction + scoring + GCS reader + `llm` client.

**Parallelizable once the skeleton exists:** `profiles`, `boards`, `jobs`, `contacts` are
mutually independent → four parallel tracks. `scoring`/`scraping`/`extraction`/`dashboard`
join as their deps land.

## 2. Phase plan

| Phase | Name | Output | Gate to exit |
|---|---|---|---|
| 0 | Scaffold & toolchain | Buildable repo, empty binaries boot, images build | `make build`/`make test` pass; 3 images build |
| 1 | Dev infra + ports + http base | Cloud Run dev + Pub/Sub dev live; llm/messaging/blobstore/DB adapters | A deployed `/healthz` on dev; Pub/Sub push reaches a worker |
| 2 | **Walking skeleton** | 1 board → on-demand run → scrape → extract → 1 job, **on dev** | End-to-end run on Cloud Run dev produces a visible job |
| 3 | Full backend contexts | All 8 contexts complete behind app-services | Per-context DoD; API surface (`overview.md` §6) live on dev |
| 4 | Pipeline hardening | Idempotency, DLQ, expiry, reextract, dedup, scoring | Integration tests + dev pipeline run green |
| 5 | Frontend | All 6 feature UIs in React, French | Each feature wired to dev API; FE tests green |
| 6 | Infra: prod + hardening | prod env, full IAM, Scheduler, SPA hosting | `tofu plan` clean both envs; prod deployable |
| 7 | E2E & polish | Scheduled run, badges/thresholds, CSV, security tier-0 | Full scheduled pipeline verified end-to-end |

`template/` (ui-integrator) runs as a **parallel track from Phase 1** — no backend
dependency. **Infra is split**: the **dev environment** is built in Phase 1 (the skeleton
needs it); **prod + remaining hardening** is Phase 6.

---

## Phase 0 — Scaffold & toolchain

**Owner:** backend-developer.

- Repo skeleton: `backend/` Go module; `internal/{domain,app,infra,handler}`;
  `cmd/{api,scrape-worker,extract-worker}/main.go` (boot + `/healthz` only);
  `migrations/`.
- Toolchain: `chi`, `pgx`+`sqlc` (`sqlc.yaml`), `goose`, `slog`, env config loader,
  golangci-lint/goimports/govulncheck configs.
- **Containerization:** Dockerfile per binary; `make image-<bin>` build + push to Artifact
  Registry; a `make deploy-dev-<bin>` (`gcloud run deploy`) for the pure-cloud inner loop.
- `Makefile`: `build`, `test`, `lint`, `migrate`, `sqlc`, `image-*`, `deploy-dev-*`.
- Frontend scaffold: `frontend/` Vite + React 18 + TS, Router, TanStack Query, RHF+Zod,
  Recharts, Axios; `ActiveProfileProvider` + `X-Active-Profile` fetch wrapper stubs; i18n
  dict skeleton. (FE dev stays local against the deployed dev API.)

**Exit:** `make build`/`make test` pass; three images build and push.

## Phase 1 — Dev infra + ports + http base

Two parallel sub-tracks; Phase 2 needs both.

**1a — Dev infra (cloud-ops).** Stand up the **dev** OpenTofu environment now (per
`infrastructure.md`), enough to deploy + run the skeleton:
- `database` (dev Cloud SQL Postgres, IAM auth), `blobstore` (GCS raw bucket),
  `secrets` (Claude key), `pubsub` (`scrape.tick` + `listing.extract` topics, push subs,
  DLQs), `cloud-run-service` ×3 (api/scrape/extract + per-binary SAs + OIDC push invoker),
  `scheduler` (wired but cron can stay paused until Phase 7).
- Separate GCS state for dev. scrape-worker max-instances=1/concurrency=1. No public
  DB/bucket; authenticated Cloud Run; OIDC Pub/Sub push. `apply` only on explicit
  per-action confirmation.

**1b — Ports + http base (backend-developer; consult `architect` on shared-kernel
boundary).**
- **Shared kernel** (minimal): typed IDs, money/salary, enums (`contract_type`,
  `remote_policy`, `working_days`, `seniority`, `application_status`),
  confidence/understanding VOs, domain errors, pagination/filter DTOs.
- **`llm` port** (`internal/domain/llm`): `AdapterGenerator.GenerateAdapter`,
  `ListingExtractor.Extract` (ADR-004); Claude impl in `internal/infra/llm` (Anthropic Go
  SDK; prompt caching on stable system prompt + schema; model id configurable, default
  `claude-opus-4-8`).
- **`messaging` port**: **single** GCP Pub/Sub adapter — `Publisher` + OIDC push-handler
  decode. No local shim (pure cloud).
- **`blobstore` port**: GCS adapter.
- **DB**: pgx pool via Cloud SQL connector (IAM auth); goose runner in `make migrate`.
- **`handler/http` base**: chi router; middleware (slog, active-profile resolver from
  `X-Active-Profile`, error→HTTP mapping); worker Pub/Sub push handler scaffold with **OIDC
  verification**.

**Exit:** a deployed dev `/healthz` responds; a test publish to `scrape.tick` reaches the
scrape-worker via real OIDC push; Claude adapter does a smoke extraction on a fixture.

## Phase 2 — Walking skeleton (vertical slice, on dev)

**Owner:** backend-developer (single track — integration de-risk on real Pub/Sub).

Thinnest end-to-end path, deployed to Cloud Run dev:

- **boards (min):** seed WTTJ + one **pre-written approved** AdapterSpec (skip LLM
  generation here); `GET /api/boards`.
- **profiles (min):** one default active profile, hardcoded keywords/location.
- **scraping (min):** crawl loop (`pipeline.md` §2) for `json_api` mode; HWM set/read; raw
  → GCS; `content_hash`; publish `listing.extract`.
- **extraction (min):** load raw → Claude structured extract → create **one** `job` (skip
  dedup/contacts/scoring).
- **jobs (min):** create + `GET /api/jobs` + `GET /api/jobs/{id}`.
- **messaging:** `POST /api/pipeline/runs` publishes `scrape.tick`; both workers consume via
  real push.
- **frontend (min):** one page listing jobs from the dev API.

**Deliberately skipped (Phase 3):** adapter generation, dedup/fingerprint, job_source merge,
contacts upsert, scoring, dashboard, full filters/kanban, PDF import, expiry, reextract.

**Exit:** an on-demand run on **Cloud Run dev** scrapes WTTJ, extracts via Claude, and a job
appears in the browser — proving Pub/Sub + GCS + Claude + DB + OIDC end-to-end before
breadth.

## Phase 3 — Full backend contexts

**Owner:** backend-developer (parallel tracks). Each task = domain → migrations → sqlc →
app-service → http handlers → `go-testing` → quality gate → deploy-dev verify.

**Track A (parallel after Phase 2):**

- **3a profiles (core):** identity PDF import (Claude document input or `ledongthuc/pdf`);
  skills CRUD; search config; conditions (dealbreakers + preferences); `fit_weights` (JSON,
  soft sum=100%); multi-profile + exactly-one-active; `GET/PUT /api/active-profile`.
  **v1 = single import only** (`POST .../identity/import` populates an empty identity).
  **Re-import stays deferred** (v0.md, overview open_questions) — no re-import endpoint
  ships in v1; decide overwrite-vs-merge at build time *if* re-import is ever added, noting
  one identity is shared across every profile spawned from that PDF.
- **3b boards:** full CRUD + enabled toggle; `POST .../adapter/generate` (LLM →
  **declarative** AdapterSpec, ADR-004/tech_debt — never executable code) → draft;
  `GET .../adapter`; `POST .../adapter/approve` with **schema validation**; single-row
  `GET/PUT /api/schedule`; seed 4 boards (WTTJ, Indeed, LinkedIn, Glassdoor).
- **3c jobs (core):** Job aggregate (structured fields + `field_confidence`,
  `understanding_score`, `fingerprint`, `contact_id`, `first_seen/last_seen/expired_at`);
  `job_source`; kanban (`PATCH .../application`); filters + sort (`GET /api/jobs`);
  `GET /api/jobs/{id}/original`.
- **3d contacts:** Contact aggregate; CRUD; dedup by email|linkedin (`dedup_key`); tags
  (`in-house`/`agency`, `responsive`/`ghosted`/`not-contacted`, custom); notes;
  `GET /api/contacts/export.csv`.

**Track B (after deps land):**

- **3e scoring (core)** [jobs+profiles]: dealbreaker gate (`passes_dealbreakers`) → weighted
  preference score per `fit_weights` (`component_breakdown` JSON); persist `job_score` per
  `(job_id, profile_id)`.
- **3f scraping** [boards+profiles]: full two-phase crawl; `fetch_mode` json_api/html
  (goquery); `param_map` board-side filter; incremental HWM + ~36h overlap; `content_hash`;
  `safety_max_pages`; per-board `x/time/rate` limiter (max-instances=1);
  `scrape_run`/`scrape_run_board` tracking.
- **3g extraction (core)** [scraping+jobs+contacts]: structured schema
  (`{value,confidence:0..100}` per field + `understanding:0..100`); dedup `fingerprint` →
  merge + append `job_source`; contact upsert; trigger scoring; edge cases (salary
  null→conf 0; hidden recruiter→low understanding, flagged); FR/EN, **raw never
  translated**.
- **3h dashboard** [jobs+scoring+profiles]: read-model — skills frequency, skills trend,
  matches by weighted fit, stats cards. Scoped to active profile.

**Exit:** every endpoint in `overview.md` §6 live on dev; per-context DoD met.

## Phase 4 — Pipeline hardening

**Owner:** backend-developer.

- Idempotency proof: `content_hash` + `fingerprint` + upsert + Pub/Sub message id
  (ADR-003); redelivery → no duplicates.
- DLQ + retry/backoff verified on dev; poison-message path.
- Expiry: jobs unseen in a later run of the same board → `expired_at`, retained.
- `POST /api/jobs/{id}/reextract` re-publishes `listing.extract`.
- Batch API option for scheduled bulk extraction (50% cost; latency not user-facing).
- Cross-worker integration tests + a dev pipeline run.

## Phase 5 — Frontend

**`template/` track (ui-integrator)** — from Phase 1, no backend dep: design-system tokens
+ component inventory; French strings; static HTML/CSS for all 6 features + pipeline
run/status → drives `design_changes.md`. **Does not touch `frontend/`.**

**React track (frontend-developer)** — each feature wired once its dev API exists:

- App shell: ActiveProfileProvider; `X-Active-Profile` injection; active-profile in every
  Query cache key; `setActiveProfile` PUTs `/api/active-profile`; i18n dict (enums→FR, raw
  verbatim); Recharts.
- profiles UI (PDF import, skills editor, search config, conditions, weights, switcher).
- boards UI (CRUD, schedule editor, adapter review + approve).
- job browser (table + card, filter-only panel, sort, kanban with optimistic updates,
  confidence/understanding badges, "found on: …", expired marker, original link).
- dashboard (skills frequency bar, trend line, match alerts, stats cards).
- contacts (table, tags, notes, CSV, manual add/edit).
- pipeline (on-demand trigger + run status polling).

**Exit:** all features wired to the dev API; `frontend-testing` green per feature.

## Phase 6 — Infra: prod + hardening

**Owner:** cloud-ops. Dev env already exists (Phase 1); this phase adds prod and finishes.

- `environments/prod` with **separate GCS state**; sizing via tfvars
  (REGIONAL Cloud SQL, deletion protection).
- Finalize per-binary least-privilege IAM; OIDC push; no public DB/bucket; backups on.
- Cloud Scheduler cron enabled (global schedule from app config), `Europe/Paris`.
- SPA static hosting (GCS+CDN or Firebase Hosting) + CI deploy.
- Workflow: `tofu fmt -recursive` + `validate` both envs → `plan` (reviewed) → **stop**;
  `apply` only on explicit per-action confirmation.

## Phase 7 — E2E & polish

- Full **scheduled** run via Cloud Scheduler (dev then prod) end-to-end.
- Multi-board dedup verified (one job, multiple `job_source`).
- Confidence/understanding badges + threshold filters.
- Expiry, CSV export, French rendering verified across surfaces.
- Final quality gates + Tier-0 security verification (secrets only in Secret Manager,
  authenticated Cloud Run + OIDC, backups on, no public DB/bucket).

---

## 3. Open decisions resolved in this plan

| Decision | Resolution | Source |
|---|---|---|
| Dev model | **Pure cloud** — all binaries on Cloud Run dev, real Pub/Sub push; dev infra built in Phase 1 | this plan |
| LinkedIn PDF re-import | **Deferred** — v1 ships single import only, no re-import endpoint; overwrite-vs-merge decided at build time if ever added | v0.md, overview open_questions |
| Adapters | **Declarative spec, schema-validated, never executed code** | ADR-004, tech_debt.md |
| Default extraction model | `claude-opus-4-8`, configurable to Sonnet 4.6 / Haiku 4.5 | ADR-004 |

## 4. Cross-cutting tracks (run throughout)

- **Testing:** `go-testing` / `frontend-testing`, acceptance-criteria-mapped.
- **Quality gate:** vet, golangci-lint, goimports, govulncheck (Go); pre-PR.
- **Observability:** `slog` at boundaries; `scrape_run` rows + DLQ as audit trail.
- **Language:** raw stored verbatim, never translated; structured enums shown in French.
- **Multi-user seam:** keep active-profile id an explicit parameter everywhere; `tenant_id`
  stays additive later (deployment.md §5) — do not hardcode a single profile.

## 5. Next step

Run the **tech-breakdown** skill per feature (F1–F6) + the pipeline, turning each Phase-3/5
task into atomic backend/frontend tickets the `backend-developer` and `frontend-developer`
agents consume. Order matches the dependency graph: profiles → boards → jobs → contacts →
scoring → scraping → extraction → dashboard, with the `template/` track started in parallel.
