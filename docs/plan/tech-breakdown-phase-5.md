## Tech Breakdown: Phase 5 — Pipeline hardening

**Design spec ref:** docs/v0.md (dataflow)
**Architecture ref:** pipeline.md §4/§5, deployment.md §1, ADR-003
**Feature ref:** extraction-pipeline/feature.md, job-browser/feature.md (dedup/expiry)
**Plan ref:** docs/plan/development-plan.md (Phase 5)
**Teams:** Backend

---

### Tasks

---

#### P5-1 — Prove end-to-end idempotency

**Type:** Chore · **Owner:** Backend · **Dependencies:** Phase 3 (scraping, extraction)

**Description:** Verify `content_hash` (raw) + `fingerprint` (job) + upsert + Pub/Sub message
id make redelivery a no-op.
**Refs:** pipeline.md §5, ADR-003 (idempotency)
**Acceptance Criteria:**
- Replaying the same `scrape.tick` and `listing.extract` produces zero duplicate raw/jobs.

---

#### P5-2 — Implement DLQ + retry/backoff handling

**Type:** Feature · **Owner:** Backend · **Dependencies:** P1-BE-8

**Description:** Confirm push retry/backoff; poison messages land in `*.dlq` after max
attempts; handlers return correct ack/nack codes.
**Refs:** pipeline.md §5, infrastructure.md §5 (dead_letter_policy), deployment.md §1
**Acceptance Criteria:**
- A handler returning 5xx is retried; a permanently failing message reaches the DLQ.

---

#### P5-3 — Implement job expiry marking

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-SCR-5, P3-JO-1

**Description:** Jobs not seen in a later run of the same board → `expired_at`; data retained.
**Refs:** pipeline.md §5 (expiry), job-browser/feature.md (expired), data-model.md (job.expired_at)
**Acceptance Criteria:**
- A job absent from a board's subsequent run is marked expired; its row is retained.

---

#### P5-4 — Implement POST /api/jobs/{id}/reextract

**Type:** Feature · **Owner:** Backend · **Dependencies:** P3-EX-1

**Description:** Re-publish `listing.extract` for a job's retained raw to reprocess with an
improved extractor.
**Refs:** pipeline.md §5 (re-extraction), overview.md §6 (Pipeline)
**Acceptance Criteria:**
- The endpoint re-publishes `listing.extract` and the job is re-extracted from retained raw.

---

#### P5-5 — Add Batch API option for scheduled bulk extraction

**Type:** Feature · **Owner:** Backend · **Dependencies:** P1-BE-3, P3-EX-1

**Description:** For scheduled (non-user-facing) runs, route bulk extraction through the
Anthropic Batch API (≈50% cost). Config-gated.
**Refs:** pipeline.md §3 (Batch API), ADR-004
**Acceptance Criteria:**
- Scheduled bulk runs use Batch when enabled; on-demand path stays synchronous.

---

#### P5-6 — Write cross-worker integration tests

**Type:** Chore · **Owner:** Backend · **Dependencies:** P5-1

**Description:** Integration test exercising scrape → extract → dedup → score → job visible,
plus one dev pipeline run.
**Refs:** development-plan.md §4 (testing), pipeline.md (full flow)
**Acceptance Criteria:**
- The integration suite passes; a dev run yields a scored, browsable job.

---

### Dependency Graph

```
Phase 3 → P5-1 → P5-6
         P5-2
         P5-3
         P5-4
         P5-5
```

### Parallel tracks

- P5-2, P5-3, P5-4, P5-5 are independent of each other (all post-Phase-3).
- P5-1 gates P5-6.

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | Batch API acceptable latency for the scheduled cron window | P5-5 | PM |
