## Tech Breakdown: Phase 1 — Dev infra + ports + http base

**Design spec ref:** docs/v0.md
**Architecture ref:** overview.md §5/§8/§9, infrastructure.md, deployment.md, ADR-002, ADR-003, ADR-004
**Plan ref:** docs/plan/development-plan.md (Phase 1)
**Teams:** Infra (cloud-ops), Backend

Pure-cloud dev model → the **dev** OpenTofu environment is stood up now; the skeleton
(Phase 2) runs on Cloud Run dev with real Pub/Sub push.

---

### Tasks

---

#### P1-IN-1 — Scaffold OpenTofu repo + dev environment + GCS state

**Type:** Chore
**Owner:** Infra
**Dependencies:** —

**Description:**
`infra/` with `modules/` + `environments/dev`, GCS state backend (separate state per env),
`README.md` recording provider/region/state/security tier.

**Refs:** infrastructure.md §1/§2/§6 (Tier 0)

**Acceptance Criteria:**
- `tofu fmt -recursive` + `tofu validate` pass for `environments/dev`.
- Dev state lives in its own GCS bucket/prefix; region `europe-west9`.

---

#### P1-IN-2 — `database` module (dev Cloud SQL Postgres)

**Type:** Chore
**Owner:** Infra
**Dependencies:** P1-IN-1

**Description:**
Cloud SQL Postgres 16 instance + `job_tendencies` db + IAM DB users per worker SA;
IAM auth on; no public IP; backups on.

**Refs:** infrastructure.md §3/§5 (database), ADR-002

**Acceptance Criteria:**
- `tofu plan` (dev) shows the instance, db, and IAM users; no public IP.

---

#### P1-IN-3 — `blobstore` module (GCS raw bucket)

**Type:** Chore
**Owner:** Infra
**Dependencies:** P1-IN-1

**Description:**
GCS bucket for raw HTML/JSON, uniform bucket-level access, no public access.

**Refs:** infrastructure.md §3 (blobstore), ADR-002

**Acceptance Criteria:**
- `tofu plan` (dev) shows a private bucket with uniform access.

---

#### P1-IN-4 — `secrets` module (Claude key)

**Type:** Chore
**Owner:** Infra
**Dependencies:** P1-IN-1

**Description:**
Secret Manager secret for the Claude API key + accessor bindings; secret value never in
`.tf`/`.tfvars`/state.

**Refs:** infrastructure.md §3/§6, ADR-004

**Acceptance Criteria:**
- `tofu plan` (dev) shows the secret + accessor IAM; no secret material in code/state.

---

#### P1-IN-5 — `pubsub` module (topics, push subs, DLQs)

**Type:** Chore
**Owner:** Infra
**Dependencies:** P1-IN-1

**Description:**
`scrape.tick` + `listing.extract` topics, OIDC push subscriptions, `*.dlq` dead-letter
topics + subscriptions, retry policy.

**Refs:** infrastructure.md §3/§5 (pubsub), ADR-003

**Acceptance Criteria:**
- `tofu plan` (dev) shows both topics, push subs with OIDC, and DLQs (max 5 attempts).

---

#### P1-IN-6 — `cloud-run-service` module ×3 + per-binary SAs + IAM

**Type:** Chore
**Owner:** Infra
**Dependencies:** P1-IN-2, P1-IN-3, P1-IN-4, P1-IN-5

**Description:**
Reusable Cloud Run service module instantiated for api/scrape/extract; per-binary SA;
least-privilege IAM; OIDC push invoker; scrape-worker max-instances=1/concurrency=1; no
`allUsers`.

**Refs:** infrastructure.md §4/§5 (cloud-run-service, IAM), deployment.md §2/§4

**Acceptance Criteria:**
- `tofu plan` (dev) shows 3 services + 3 SAs; only push-auth SA has `run.invoker`.
- scrape-worker pinned to 1 instance / concurrency 1.

---

#### P1-IN-7 — `scheduler` module (wired, paused)

**Type:** Chore
**Owner:** Infra
**Dependencies:** P1-IN-5

**Description:**
Cloud Scheduler job targeting `scrape.tick`, `Europe/Paris`; cron may stay paused until
Phase 8.

**Refs:** infrastructure.md §5 (scheduler), deployment.md §1

**Acceptance Criteria:**
- `tofu plan` (dev) shows the scheduler job publishing to `scrape.tick`.

---

#### P1-BE-1 — Implement the shared kernel

**Type:** Chore
**Owner:** Backend
**Dependencies:** —

**Description:**
Typed IDs, money/salary VO, enums (`contract_type`, `remote_policy`, `working_days`,
`seniority`, `application_status`), confidence/understanding VOs, domain errors,
pagination/filter DTOs. Keep minimal — no god package.

**Refs:** data-model.md (enums/fields), overview.md §9, extraction-pipeline/feature.md

**Acceptance Criteria:**
- Enums reject invalid values; money VO unit-tested for parse/format.
- `domain` imports nothing outward.

---

#### P1-BE-2 — Define the `llm` domain port

**Type:** Feature
**Owner:** Backend
**Dependencies:** P1-BE-1

**Description:**
`internal/domain/llm` interfaces: `AdapterGenerator.GenerateAdapter(boardURL, example)` and
`ListingExtractor.Extract(raw)`. Data types only (declarative AdapterSpec, ExtractedListing).

**Refs:** ADR-004, pipeline.md §1/§3, board-manager/feature.md, extraction-pipeline/feature.md

**Acceptance Criteria:**
- Interfaces compile with no SDK import in `domain`.
- AdapterSpec is declarative data (no executable-code field).

---

#### P1-BE-3 — Implement the Claude `llm` infra adapter

**Type:** Feature
**Owner:** Backend
**Dependencies:** P1-BE-2

**Description:**
`internal/infra/llm` using the Anthropic Go SDK; prompt caching on stable system prompt +
schema; structured-output extraction `{value,confidence}` + `understanding`; model id
configurable, default `claude-opus-4-8`.

**Refs:** ADR-004, pipeline.md §3 (model selection, caching)

**Acceptance Criteria:**
- Smoke `Extract` against a fixture returns per-field confidence + understanding.
- Model id read from config (defaults to `claude-opus-4-8`).

---

#### P1-BE-4 — Implement `messaging` port + Pub/Sub adapter

**Type:** Feature
**Owner:** Backend
**Dependencies:** P1-BE-1

**Description:**
Single GCP Pub/Sub adapter: `Publisher` + OIDC push-message decode. No local shim
(pure-cloud dev).

**Refs:** ADR-003, deployment.md §1, infrastructure.md §5

**Acceptance Criteria:**
- Publisher publishes to a topic; push decoder parses a Pub/Sub push envelope.

---

#### P1-BE-5 — Implement `blobstore` port + GCS adapter

**Type:** Feature
**Owner:** Backend
**Dependencies:** P1-BE-1

**Description:**
`blobstore` port + GCS adapter to store/load raw payloads by path; raw stored verbatim.

**Refs:** ADR-002, data-model.md (raw_listing.raw_ref), extraction-pipeline/feature.md (storage)

**Acceptance Criteria:**
- Write then read of a raw blob round-trips byte-identical against the dev bucket.

---

#### P1-BE-6 — Wire DB pool + migration runner

**Type:** Chore
**Owner:** Backend
**Dependencies:** P1-BE-1, P1-IN-2

**Description:**
pgx pool via Cloud SQL connector (IAM auth); goose runner behind `make migrate`.

**Refs:** ADR-002, overview.md §8

**Acceptance Criteria:**
- A local process connects to dev Cloud SQL via IAM and runs a no-op migration.

---

#### P1-BE-7 — Build the http base + middleware

**Type:** Chore
**Owner:** Backend
**Dependencies:** P1-BE-1

**Description:**
chi router; middleware: slog request log, active-profile resolver from `X-Active-Profile`,
domain-error→HTTP mapping.

**Refs:** overview.md §6/§7/§9, deployment.md §3

**Acceptance Criteria:**
- A scoped route rejects a missing `X-Active-Profile` with a 400; errors map to status codes.

---

#### P1-BE-8 — Scaffold the worker Pub/Sub push handler + OIDC verify

**Type:** Feature
**Owner:** Backend
**Dependencies:** P1-BE-4, P1-BE-7

**Description:**
Tiny HTTP route per worker that accepts a Pub/Sub push, verifies the OIDC token, decodes the
message, and dispatches to the (stub) app-service.

**Refs:** ADR-003 (OIDC push), infrastructure.md §5

**Acceptance Criteria:**
- A push with a valid OIDC token is accepted; an invalid/absent token is rejected 401/403.

---

### Dependency Graph

```
P1-IN-1 → P1-IN-2 ┐
        → P1-IN-3 ┤
        → P1-IN-4 ┼→ P1-IN-6
        → P1-IN-5 ┘ → P1-IN-7

P1-BE-1 → P1-BE-2 → P1-BE-3
        → P1-BE-4 → P1-BE-8
        → P1-BE-5
        → P1-BE-6 (also needs P1-IN-2)
        → P1-BE-7 → P1-BE-8
```

### Parallel tracks

- Infra track (P1-IN-*) and backend ports track (P1-BE-1…5, 7) run concurrently.
- P1-BE-6 and P1-BE-8 are the join points (need infra up).

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | GCP project id(s) + billing for dev | all P1-IN-* | User/cloud-ops |
| 2 | Claude API key provisioned into Secret Manager | P1-IN-4, P1-BE-3 | User |
