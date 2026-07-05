---
ai_context:
  adr: "ADR-007"
  title: "Offset pagination for GET /api/jobs"
  status: "accepted"
  date: "2026-07-05"
  session: "docs/adr (standalone — no feature doc)"

  decision_type: "communication"

  affects:
    contexts: ["job-browser"]
    components: ["api", "db", "frontend"]

  chosen: "offset pagination (page/page_size) with a paginated response envelope, default-on"

  constraints_that_drove_this:
    - "Single-profile dataset is hundreds-to-low-thousands of rows; a full COUNT is cheap"
    - "User-selectable dynamic sort (date|salary, asc|desc) makes keyset/cursor pagination disproportionately complex"
    - "UI needs a total count and page numbers for table/cards; kanban needs the full tracked set"
    - "Endpoint is unversioned with a single consumer (frontend useJobs.ts), so a shape change is cheap to coordinate"

  rejected:
    - option: "cursor / keyset pagination"
      reason: "dynamic multi-column sort would need a composite cursor per sort mode; no total count for page UI; scale does not require it"
    - option: "opt-in pagination (bare array when page absent, envelope when present)"
      reason: "two response shapes for one endpoint; brittle for the shared frontend hook"
    - option: "no pagination / keep returning the full array"
      reason: "unbounded payload grows with the profile's job history; the reason this ADR exists"

  must_not:
    - "apply LIMIT/OFFSET or the total count in Go after the query — all filtering, counting and slicing happen in SQL"
    - "count with COUNT(*) — the list query is SELECT DISTINCT j.id, so the total must be COUNT(DISTINCT j.id) over the same filtered join"
    - "paginate the kanban view per column or build server-side status grouping"
    - "let page_size exceed the hard maximum (100)"

  open_questions: []
  assumptions:
    - "Tracked jobs (application_status != null, the only jobs the kanban renders) stay comfortably under the 100 page_size cap per profile"
---

# ADR-007 — Offset pagination for GET /api/jobs

## Status
Accepted — 2026-07-05

## Context
`GET /api/jobs` returns the full filtered, sorted list of jobs for the active profile as
a bare JSON array (`jobResponse[]`). Filtering and sorting are already pushed into a
single SQL query in `infra/jobs.Repository.ListByProfile` (skills, remote policy, contract
type, salary min/max, location, board, `since`, `confidence_min`, plus `sort`/`sort_dir`).
As a profile accumulates job history the payload grows unbounded, and the frontend renders
it in three view modes (table, cards, kanban) with no way to page.

The endpoint is unversioned (`/api/jobs`) and has exactly one consumer, the frontend
`useJobs` hook. This is a single-user application: the dataset is hundreds to low-thousands
of rows per profile. `confidence_min` is already applied **in SQL** (`WHERE
j.understanding_score >= $N`), not post-query in Go, so any pagination must count and slice
inside that same filtered query — never in application code. The list query also uses
`SELECT DISTINCT j.id` because of the `job_source` join fan-out, which constrains how the
total is counted.

## Goals
- Bound the `GET /api/jobs` response size with a sane default page.
- Return the total count so table/cards can show page numbers ("137 jobs, page 2 of 6").
- Keep the existing filter and sort query params unchanged.
- Keep the read path (ADR-005 CQRS-lite query service) as the only place that touches this.

## Non-Goals
- Cursor/keyset pagination and its stable-ordering guarantees.
- Per-column pagination for the kanban board.
- Versioning the endpoint or introducing an envelope for other list endpoints (they may
  adopt the same shape later, but that is out of scope here).

## Decision
**We will add offset pagination to `GET /api/jobs` with two new query params (`page`,
`page_size`), default-on, returning a paginated envelope instead of a bare array.**

Request params (added to the existing filter/sort set):

| Param | Type | Default | Bounds |
|---|---|---|---|
| `page` | int, 1-based | `1` | clamped to `>= 1` |
| `page_size` | int | `25` | clamped to `1..100` |

Out-of-range values are clamped, not rejected, so a stray `page_size=500` yields 100 rather
than a 400. `page` beyond the last page returns an empty `items` array with the real
`total` (the frontend can recover).

Response envelope:

```jsonc
{
  "items": [ /* jobResponse, unchanged shape */ ],
  "page": 2,
  "page_size": 25,
  "total": 137,        // COUNT(DISTINCT j.id) over the same filtered query
  "total_pages": 6     // ceil(total / page_size); 0 when total is 0
}
```

The `JobListFilter` gains `Page`/`PageSize` (or a small `Page` value object) and the query
port returns items plus the total. `ORDER BY` stays as-is and gains a deterministic tie-break
(`, j.id DESC`) so the same row never straddles two pages under equal sort keys. The total
is `COUNT(DISTINCT j.id)` over the identical `FROM/JOIN/WHERE`, and the page slice adds
`LIMIT $n OFFSET $m` after the `ORDER BY`. Filtering (including `confidence_min`) is untouched
— it already lives in the `WHERE` clause, so counting and slicing sit strictly downstream of it.

**Kanban stays fetch-all.** The kanban board only renders jobs with a non-null
`application_status` (the actively tracked pipeline), which is inherently a small subset.
The frontend fetches the kanban dataset with `page_size=100` (the cap) and does not paginate
per column. We deliberately do not build server-side status grouping or per-column paging —
that is gold-plating at this scale. If a profile ever exceeds 100 tracked jobs the frontend
can page through client-side, but this is not expected.

## Considered Alternatives

### Option A — Offset pagination, default-on envelope *(chosen)*
`page`/`page_size` with a `{items, page, page_size, total, total_pages}` envelope, always returned.

**Pros**
- Trivial to implement over the existing single filtered query (`LIMIT/OFFSET` + one `COUNT`).
- Free total count enables page-number UI, which the table/cards views want.
- Works unchanged across all four current sort modes.
- One response shape for the endpoint — simple for the shared `useJobs` hook.

**Cons**
- Deep pages get slower (`OFFSET` scans skipped rows) — irrelevant at this row count.
- Rows can shift between pages if data changes mid-browse — acceptable for a personal tracker.

### Option B — Cursor / keyset pagination
Opaque cursor encoding the last row's sort key(s) + id; `WHERE (sort_key, id) < (...)`.

**Why rejected**: the endpoint offers user-selectable multi-column sort (date|salary ×
asc|desc), so each sort mode needs its own composite cursor and boundary logic; it yields no
total count for a page-number UI; and it buys scale benefits the dataset will never need.

### Option C — Opt-in pagination (bare array unless `page` is sent)
Return the legacy array when `page` is absent, the envelope when present.

**Why rejected**: two response shapes for one endpoint forces the shared frontend hook to
branch on shape, and invites drift; a single-consumer unversioned endpoint has no reason to
preserve the old shape.

## Thinking Process
The instinct at this scale is "don't paginate at all", but an unbounded list that grows with
job history is the concrete problem, so some bound is warranted. Cursor pagination is the
textbook "correct" answer, but it collides head-on with the existing dynamic sort and the
UI's desire for a total/page count — it would add real complexity for a few-thousand-row
personal app, a clear YAGNI violation. Offset with a cheap `COUNT(DISTINCT j.id)` gives the
table/cards exactly what they need. The only genuine wrinkle was the kanban view: paginating
it per column sounded principled but the board only ever shows the small tracked subset, so
fetch-all (capped at the max page size) is both simpler and correct. Default-on with one
envelope shape won over opt-in because the endpoint is unversioned with a single consumer —
there is nothing to stay backward-compatible with except the frontend, which changes in lockstep.

## Consequences

### Positive
- Bounded, predictable payloads with a default page of 25.
- Page-number UX for table/cards from the returned `total`/`total_pages`.
- No change to filtering/sorting logic or the ADR-005 read/write split.

### Negative / Trade-offs
- Breaking response shape (array → envelope); the frontend `useJobs` hook and its
  `JobSummaryDto[]` typing must change in the same change set.
- Offset performance degrades on deep pages (immaterial here).

### Risks
- A forgotten `DISTINCT` in the count query would over-count jobs with multiple sources.
  Mitigated by the `must_not` rule: count is `COUNT(DISTINCT j.id)` over the same join.
- Non-deterministic ordering under equal sort keys could duplicate/skip a row across pages.
  Mitigated by the `, j.id DESC` tie-break.

## Implementation Constraints
- **DO** add `page`/`page_size` to `JobListFilter` and return `(items, total)` from the read
  query port; keep it on the read side (ADR-005), never the write aggregate repo.
- **DO** compute the total as `COUNT(DISTINCT j.id)` over the identical `FROM/JOIN/WHERE`,
  and slice with `LIMIT/OFFSET` after `ORDER BY … , j.id DESC`.
- **DO** clamp `page` to `>= 1` and `page_size` to `1..100`, defaulting to `1` / `25`.
- **DO** return the envelope `{items, page, page_size, total, total_pages}` from the handler.
- **DO** have the kanban view request `page_size=100` and page client-side if ever needed.
- **DO NOT** apply `LIMIT/OFFSET` or count rows in Go after the query.
- **DO NOT** use `COUNT(*)` for the total.
- **DO NOT** paginate the kanban per column or add server-side status grouping.

## Related
- **Depends on**: ADR-005 (domain repositories and read/write split) — pagination lives on
  the app-side read query, returning DTOs, not the Job aggregate write repository.
