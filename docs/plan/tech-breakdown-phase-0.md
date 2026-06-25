## Tech Breakdown: Phase 0 — Scaffold & toolchain

**Design spec ref:** docs/v0.md
**Architecture ref:** docs/architecture/overview.md §5/§8, docs/architecture/deployment.md §3, ADR-001, ADR-002
**Plan ref:** docs/plan/development-plan.md (Phase 0)
**Teams:** Backend, Frontend

---

### Tasks

---

#### P0-1 — Scaffold repo layout and Go module

**Type:** Chore
**Owner:** Backend
**Dependencies:** —

**Description:**
Create `backend/` Go module with `internal/{domain,app,infra,handler}`,
`cmd/{api,scrape-worker,extract-worker}/`, and `migrations/`. Root dirs `frontend/`,
`template/`, `infra/`, `docs/`.

**Refs:** overview.md §5 (code layout), ADR-001 (modular monolith, thin mains)

**Acceptance Criteria:**
- `go build ./...` succeeds on an empty module.
- Directory tree matches overview.md §5.

---

#### P0-2 — Add three binary skeletons with /healthz

**Type:** Chore
**Owner:** Backend
**Dependencies:** P0-1

**Description:**
Each `cmd/<bin>/main.go` boots an HTTP server exposing `/healthz` and nothing else
(no domain logic).

**Refs:** deployment.md §3, ADR-001 (DO keep main thin)

**Acceptance Criteria:**
- Each binary starts and returns 200 on `GET /healthz`.
- No `internal/domain` or `internal/app` import in any `main` yet beyond wiring stubs.

---

#### P0-3 — Wire core backend dependencies

**Type:** Chore
**Owner:** Backend
**Dependencies:** P0-1

**Description:**
Add `chi`, `pgx`, `sqlc` (`sqlc.yaml`), `goose`, `slog`, and an env-based config loader.

**Refs:** overview.md §8 (tech stack), ADR-002 (pgx/sqlc)

**Acceptance Criteria:**
- `go mod tidy` resolves; `sqlc generate` runs against an empty schema without error.
- Config loader reads required env vars and fails fast when missing.

---

#### P0-4 — Add Go quality-gate configs

**Type:** Chore
**Owner:** Backend
**Dependencies:** P0-1

**Description:**
Add config for `golangci-lint`, `goimports`, `go vet`, `govulncheck` matching the
project quality gate.

**Refs:** development-plan.md §4 (quality gate)

**Acceptance Criteria:**
- `golangci-lint run`, `go vet ./...`, `govulncheck ./...` all run green on the skeleton.

---

#### P0-5 — Containerize each binary + Artifact Registry push

**Type:** Chore
**Owner:** Backend
**Dependencies:** P0-2

**Description:**
One Dockerfile per binary; build + push to Artifact Registry. Required because the dev
inner loop is **pure cloud** (deploy to Cloud Run dev to test).

**Refs:** development-plan.md (dev model: pure cloud), deployment.md §2

**Acceptance Criteria:**
- Three images build and push to Artifact Registry.
- Image entrypoint runs the correct binary and serves `/healthz`.

---

#### P0-6 — Author Makefile targets

**Type:** Chore
**Owner:** Backend
**Dependencies:** P0-3, P0-5

**Description:**
`Makefile` with `build`, `test`, `lint`, `migrate`, `sqlc`, `image-<bin>`,
`deploy-dev-<bin>` (`gcloud run deploy`).

**Refs:** development-plan.md (Phase 0)

**Acceptance Criteria:**
- `make build` and `make test` pass.
- `make image-api` builds+pushes; `make deploy-dev-api` deploys (manual run, documented).

---

#### P0-7 — Scaffold the React SPA

**Type:** Chore
**Owner:** Frontend
**Dependencies:** —

**Description:**
`frontend/` Vite + React 18 + TS with React Router, TanStack Query, React Hook Form + Zod,
Recharts, Axios.

**Refs:** overview.md §7/§8 (frontend architecture + stack)

**Acceptance Criteria:**
- `npm run build` and `npm run dev` succeed; a placeholder route renders.

---

#### P0-8 — Stub ActiveProfileProvider, X-Active-Profile wrapper, i18n dict

**Type:** Chore
**Owner:** Frontend
**Dependencies:** P0-7

**Description:**
Context provider holding active-profile id, a fetch wrapper injecting `X-Active-Profile`,
and an empty French i18n dict skeleton.

**Refs:** overview.md §7 (ActiveProfileProvider, X-Active-Profile, i18n), §9 (language)

**Acceptance Criteria:**
- Outbound requests carry `X-Active-Profile` when an id is set.
- i18n dict resolves a sample enum key to French.

---

### Dependency Graph

```
P0-1 → P0-2 → P0-5 → P0-6
  └──→ P0-3 ──────────┘
  └──→ P0-4
P0-7 → P0-8
```

### Parallel tracks

- Backend (P0-1…P0-6) and Frontend (P0-7, P0-8) run concurrently.

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | Artifact Registry repo name + GCP project id for dev | P0-5 | User/cloud-ops |
