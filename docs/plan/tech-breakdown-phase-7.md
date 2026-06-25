## Tech Breakdown: Phase 7 — E2E & polish

**Design spec ref:** docs/v0.md
**Architecture ref:** overview.md §9, pipeline.md, infrastructure.md §6 (Tier 0), deployment.md
**Feature ref:** all of docs/feature/*/feature.md
**Plan ref:** docs/plan/development-plan.md (Phase 7)
**Teams:** Backend, Frontend, Infra

---

### Tasks

---

#### P7-1 — Verify a full scheduled run end-to-end (dev then prod)

**Type:** Chore · **Owner:** Full-stack · **Dependencies:** P6-IN-4

**Description:** Cloud Scheduler fires `scrape.tick` → scrape → extract → score → jobs
browsable, on dev then prod.
**Refs:** pipeline.md §6, deployment.md §1
**Acceptance Criteria:** A scheduled (not on-demand) run yields scored, browsable jobs in both envs.

#### P7-2 — Verify multi-board dedup

**Type:** Chore · **Owner:** Backend · **Dependencies:** P3-EX-2

**Description:** Same role on two boards collapses into one job with multiple `job_source`
rows.
**Refs:** pipeline.md §4, job-browser/feature.md (found on), data-model.md
**Acceptance Criteria:** A cross-board duplicate shows one job, "found on: WTTJ, Indeed".

#### P7-3 — Verify confidence/understanding badges + threshold filtering

**Type:** Chore · **Owner:** Full-stack · **Dependencies:** P3-EX-1, P5-FE-4

**Description:** Badges render from stored scores; the confidence-threshold filter narrows
results.
**Refs:** overview.md §9 (confidence/understanding), job-browser/feature.md, tech_debt.md (heuristic)
**Acceptance Criteria:** Badges match stored values; threshold filter excludes below-threshold jobs.

#### P7-4 — Verify expiry, CSV export, and French rendering across surfaces

**Type:** Chore · **Owner:** Full-stack · **Dependencies:** P4-3, P3-CO-3, P5-FE-*

**Description:** Expired jobs marked + retained; CSV export valid; all structured enums shown
in French while raw text stays verbatim.
**Refs:** job-browser/feature.md (expiry), contacts-crm/feature.md (CSV), overview.md §9 (language)
**Acceptance Criteria:** Expired badge shows; CSV opens cleanly; no raw text is translated.

#### P7-5 — Run quality gates + verify Tier-0 security posture

**Type:** Chore · **Owner:** Backend + Infra · **Dependencies:** all

**Description:** Green `go vet` / golangci-lint / goimports / govulncheck; confirm Tier-0:
secrets only in Secret Manager, authenticated Cloud Run + OIDC push, backups on, no public
DB/bucket.
**Refs:** development-plan.md §4, infrastructure.md §6 (Tier 0), overview.md §9 (security)
**Acceptance Criteria:**
- All Go quality gates pass.
- No secret in code/state; no public DB/bucket; Cloud Run requires auth; backups enabled.

---

### Dependency Graph

```
P6-IN-4 → P7-1
P3-EX-2 → P7-2
(P3-EX-1 + P5-FE-4) → P7-3
(P4-3 + P3-CO-3 + P5-FE-*) → P7-4
all → P7-5
```

### Parallel tracks

- P7-1…P7-4 are independent verifications; run concurrently.
- P7-5 is the final gate after the rest.

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | When to flip from private/IAP to public API edge (Tier-1 trigger) | P7-5 | User |
