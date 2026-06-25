---
ai_context:
  decision: "Scheduled pipeline runs as separate worker binaries triggered by Cloud Scheduler via Pub/Sub push to Cloud Run"
  chosen: "Cloud Scheduler -> Pub/Sub (scrape.tick, listing.extract) -> Cloud Run services (push, OIDC)"
  rejected: ["in-process goroutine workers + SQLite job queue", "Cloud Run Jobs as primary shape", "Cloud Tasks as the trigger", "external broker (Redis/RabbitMQ)"]
  must:
    - "Scrape and extract run in their own binaries (cmd/scrape-worker, cmd/extract-worker)"
    - "Pub/Sub is the queue: at-least-once, retries, dead-letter topics"
    - "Every stage idempotent: content_hash (raw) + fingerprint (job) + upsert + message id"
    - "Scrape pins max-instances=1 so the in-process per-board rate limiter stays correct"
  must_not:
    - "Run scheduled scrape/extract as goroutines inside the API process"
    - "Raise scrape-worker max-instances without externalizing per-board rate limiting"
  parent: "docs/architecture/overview.md"
---

# ADR-003 — Cloud-scheduled separate worker binaries

## Status
Accepted — 2026-06-25

## Context
Scraping is rate-limited and slow; extraction is LLM-bound. Both must run off the API
request path and be triggerable on a global schedule. The user requires scheduled work to
run as **separate binaries** on cloud infrastructure (GCP), not goroutines in the API
process, and the design must scale from single-user to multi-tenant without a rewrite.

## Decision
**Cloud Scheduler → Pub/Sub → Cloud Run (push, OIDC).** A global Cloud Scheduler cron
publishes `scrape.tick`; a push subscription invokes `scrape-worker` (Cloud Run service),
which fetches listings and publishes one `listing.extract` message per new raw listing; a
second push subscription invokes `extract-worker`, which runs LLM extraction + dedup +
scoring. On-demand runs publish the same `scrape.tick` topic from the API — one path, two
triggers. Pub/Sub provides the queue semantics (retries, dead-letter topics). Scrape-worker
is pinned to `max-instances=1` so its in-process per-board `x/time/rate` limiter remains
authoritative.

## Alternatives considered
- **In-process goroutine workers + SQLite job queue** — rejected: violates the
  separate-binary requirement; couples API and pipeline scaling; SQLite can't back multiple
  processes (ADR-002).
- **Cloud Run Jobs (Scheduler → jobs:run) as the primary shape** — rejected: good for one
  monolithic run-to-completion batch, but weak for per-listing fan-out, per-message retry,
  and decoupling scrape from extract. Pub/Sub keeps the clean seam.
- **Cloud Tasks as the trigger** — rejected for triggering, but earmarked for per-board
  rate limiting at multi-tenant scale.
- **External broker (Redis/RabbitMQ)** — rejected: operational overhead; Pub/Sub is managed
  and free at this volume.

## Consequences
- (+) Pipeline decoupled from API; scales to zero when idle; autoscales for multi-tenant
  fan-out with no code change.
- (+) Reliability (retries, DLQ, idempotency) provided by the platform.
- (−) At-least-once delivery means every consumer must be idempotent (it is).
- (−) Per-board rate limiting is correct only while scrape-worker is single-instance; going
  multi-tenant requires moving to Cloud Tasks per-board queues.

## Implementation constraints
- DO make every stage idempotent: `content_hash` (raw), `fingerprint` (job), upsert
  semantics, Pub/Sub message id.
- DO keep scheduled and on-demand runs on the same `scrape.tick` path.
- DO authenticate Pub/Sub push with OIDC; Cloud Run services require auth (no public
  invoker).
- DO NOT run scheduled scrape/extract inside the API process.
- DO NOT raise scrape-worker `max-instances` without first externalizing per-board rate
  limiting to Cloud Tasks.
