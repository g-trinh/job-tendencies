---
ai_context:
  decision: "PostgreSQL (Cloud SQL) as the datastore; GCS for raw HTML/JSON"
  chosen: "Cloud SQL Postgres for structured data (Postgres container for local dev); GCS bucket for raw payloads"
  rejected: ["SQLite (single-file, local-disk)", "Litestream", "Turso/distributed SQLite", "raw HTML on local filesystem"]
  must:
    - "All binaries connect to the same networked Postgres"
    - "Raw HTML/JSON stored in GCS, referenced by path from raw_listing"
    - "Repository interfaces live in domain; Postgres impls in infra"
  must_not:
    - "Use SQLite or any embedded/local-disk DB for the deployed path"
    - "Store raw payloads on a worker's local filesystem (ephemeral on Cloud Run)"
  parent: "docs/architecture/overview.md"
---

# ADR-002 — PostgreSQL (Cloud SQL) + GCS for raw storage

## Status
Accepted — 2026-06-25

## Context
The pipeline runs as separate stateless worker binaries on Cloud Run (ADR-003).
`api`, `scrape-worker`, and `extract-worker` are distinct processes that read and write
shared state concurrently, and Cloud Run filesystems are ephemeral and per-instance. The
product is not excluded from multi-user later.

## Decision
Use **Cloud SQL PostgreSQL** as the single datastore for all structured data; connect via
the Cloud SQL Go connector with IAM DB auth. Store raw scraped payloads (HTML/JSON) in a
**GCS bucket**, referenced by path from `raw_listing`. Local development uses a Postgres
container (not SQLite) to avoid SQL-dialect drift.

## Alternatives considered
- **SQLite** — rejected. It is a single-file, single-writer, local-disk embedded DB. There
  is no shared filesystem across Cloud Run instances, no concurrent multi-process writers,
  and the file has nowhere durable to live. It only fit the abandoned single-local-binary
  design.
- **Litestream** (SQLite + streaming replication) — rejected: still single-writer; does not
  solve concurrent multi-process writes from separate worker binaries.
- **Turso / distributed SQLite** — rejected: adds a vendor and still has single-writer
  semantics for the pipeline; Cloud SQL is the lower-risk managed Postgres.
- **Raw HTML on local FS** — rejected: ephemeral, not shared, lost on instance recycle.

## Consequences
- (+) Concurrent access from multiple binaries; real transactions; managed backups.
- (+) Multi-user is additive (`tenant_id` columns + row scoping), not a migration.
- (+) Raw payloads cheap and durable in GCS; kept for re-extraction.
- (−) A networked DB is the one always-on cost (smallest tier for single user) and adds
  connection/latency considerations versus an in-process DB. Acceptable.

## Implementation constraints
- DO access Postgres through `pgx`/`sqlc`; keep repository interfaces in `domain`, impls in
  `infra`.
- DO store raw payloads in GCS and reference them by path; keep raw verbatim, never
  translated.
- DO add `tenant_id` to scoped tables at the multi-user boundary rather than re-modelling.
- DO NOT introduce SQLite or any local-disk DB on the deployed path.
- DO NOT write raw payloads to a worker's local filesystem.
