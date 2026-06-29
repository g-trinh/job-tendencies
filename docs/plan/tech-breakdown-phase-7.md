## Tech Breakdown: Phase 7 — Infra: prod + hardening

**Design spec ref:** docs/v0.md
**Architecture ref:** infrastructure.md, deployment.md §2/§5, ADR-002, ADR-003
**Plan ref:** docs/plan/development-plan.md (Phase 7)
**Teams:** Infra (cloud-ops)

Dev env already exists (Phase 1). This phase adds prod + finishes Scheduler, SPA hosting,
and IAM hardening. `apply` only on explicit per-action user confirmation.

---

### Tasks

---

#### P7-IN-1 — Stand up environments/prod with separate GCS state

**Type:** Chore · **Owner:** Infra · **Dependencies:** Phase 1 modules

**Description:** Thin `environments/prod` calling the same modules with prod tfvars; own GCS
state bucket/prefix.
**Refs:** infrastructure.md §1/§2 (separate state per env), §7
**Acceptance Criteria:** `tofu validate` + `tofu plan` (prod) clean; state isolated from dev.

#### P7-IN-2 — Apply prod sizing tfvars (Cloud SQL REGIONAL, deletion protection)

**Type:** Chore · **Owner:** Infra · **Dependencies:** P7-IN-1

**Description:** Prod sizing: REGIONAL availability, deletion protection on, prod db tier,
extract-worker max ~5.
**Refs:** infrastructure.md §5 (availability_type, deletion_protection), deployment.md §2
**Acceptance Criteria:** Prod plan shows REGIONAL Cloud SQL + deletion protection true.

#### P7-IN-3 — Finalize least-privilege IAM across both envs

**Type:** Chore · **Owner:** Infra · **Dependencies:** P7-IN-1

**Description:** Verify per-binary SA roles match infrastructure.md §4; no admin/editor; no
`allUsers` invoker; OIDC push only.
**Refs:** infrastructure.md §4/§6, deployment.md §4
**Acceptance Criteria:** Each SA holds only its listed roles; workers invokable only by the push-auth SA.

#### P7-IN-4 — Enable the Cloud Scheduler global cron

**Type:** Feature · **Owner:** Infra · **Dependencies:** P1-IN-7, P3-BO-5

**Description:** Activate the Scheduler job with the global cron from app config,
`Europe/Paris`, publishing `scrape.tick`.
**Refs:** infrastructure.md §5 (scheduler), board-manager/feature.md (global schedule), pipeline.md §6
**Acceptance Criteria:** The scheduled cron fires `scrape.tick` and a scrape run starts.

#### P7-IN-5 — Provision SPA static hosting + deploy

**Type:** Chore · **Owner:** Infra · **Dependencies:** Phase 6 build

**Description:** Host the built SPA on GCS+CDN or Firebase Hosting; wire CI deploy.
**Refs:** deployment.md §2 (static assets, no server compute)
**Acceptance Criteria:** The production SPA is served over HTTPS and reaches the prod API.

---

### Dependency Graph

```
Phase 1 modules → P7-IN-1 → P7-IN-2 / P7-IN-3
P1-IN-7 + P3-BO-5 → P7-IN-4
Phase 6 build → P7-IN-5
```

### Parallel tracks

- P7-IN-2, P7-IN-3 independent after P7-IN-1.
- P7-IN-4 and P7-IN-5 independent of the prod-sizing tasks.

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | SPA host choice: GCS+CDN vs Firebase Hosting | P7-IN-5 | User/cloud-ops |
| 2 | ~~API edge auth for prod~~ — **resolved: built in Phase 4** (backend-proxied Identity Platform, single-user). This phase only carries prod wiring of the Phase 4 auth. | — | — |
| 3 | Default global cron expression | P7-IN-4 | User/PM |
