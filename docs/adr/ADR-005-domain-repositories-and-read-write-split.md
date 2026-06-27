---
ai_context:
  decision: "Aggregate repository interfaces live in the domain layer; reads and writes are split (CQRS-lite)"
  chosen: "domain-side aggregate repository interfaces (write side); app-side query services returning read DTOs (read side) that bypass the aggregate repository and query storage directly; aggregates are persistence-ignorant"
  rejected: ["Go consumer-interface convention (repository interface in the app package that uses it)", "single repository serving both reads and writes", "aggregates that persist themselves"]
  must:
    - "Aggregate repository interfaces are declared in internal/domain/<aggregate>; implementations stay in internal/infra/<aggregate>"
    - "The job-browser read path uses an app-side query service returning a read DTO, not the Job aggregate repository"
    - "Aggregates never persist themselves; the application service decides when to save"
  must_not:
    - "route read models through the write-side aggregate repository"
    - "give a domain aggregate a reference to its repository or any infra dependency"
  parent: "docs/architecture/overview.md"
  supersedes_convention: "go-guidelines §3 (interfaces in the consumer package) for aggregate repositories only"
---

# ADR-005 — Domain-side repositories and read/write split

## Status
Accepted — 2026-06-27

## Context
Phase 1 placed every aggregate repository interface in its `internal/app/<context>`
package, following Go's "define interfaces in the consumer package" convention. As the
backend grew this scattered each aggregate's persistence contract across the app layer,
let read and write concerns share one interface, and made the job-browser read path
depend on the same port used to create jobs. We want the persistence contract to belong
to the aggregate it serves, and the high-volume read path to evolve independently of the
write model.

## Decision
1. **Aggregate repository interfaces move to the domain layer.** Each aggregate's
   repository interface is declared in `internal/domain/<aggregate>` and named in domain
   language; the Postgres implementation stays in `internal/infra/<aggregate>` and
   satisfies it implicitly. This is a deliberate departure from Go's consumer-interface
   convention: the persistence contract is part of the aggregate's definition, not the
   use case's. Capability ports that are not aggregate repositories — `llm`,
   `blobstore`, `messaging` — keep their existing domain placement and are unaffected.

2. **Read/write split (CQRS-lite).** The *write* side is the aggregate repository
   (loads/saves whole aggregates), declared in the domain. The *read* side for the job
   browser is an app-side query service (`app/jobs`) that returns a read DTO
   (`JobView`), bypasses the Job aggregate repository, and may query storage directly.
   The read query port stays app-side (consumer convention) because it is a use-case
   projection, not an aggregate contract. The HTTP response shape is unchanged.

3. **Persistence ignorance.** Aggregates never persist themselves and hold no reference
   to a repository or any infrastructure. The application service owns the decision of
   when to save.

4. **One canonical write port per aggregate.** Duplicated write ports are collapsed: the
   former `app/extraction.JobWriter` and the job read repository are no longer two
   overlapping interfaces — `Create` is the Job aggregate's single write port in
   `domain/jobs`. Where two genuinely distinct use cases touch one aggregate with
   disjoint method sets (raw-listing capture vs. downstream extraction), interface
   segregation is preserved with two narrow domain ports (`RawListingRepository`,
   `RawListingSource`) over the same aggregate rather than one fat interface.

## Alternatives considered
- **Keep interfaces in the consumer (app) package** — rejected: scatters an aggregate's
  persistence contract across use cases and couples the read path to the write port.
- **One repository for reads and writes** — rejected: the job browser is read-heavy and
  must evolve its projections without touching the write model.
- **Self-persisting aggregates** — rejected: drags infrastructure into the domain and
  breaks persistence ignorance.

## Consequences
- (+) Each aggregate owns its persistence contract in one place, in domain language.
- (+) Read models evolve independently of the write model.
- (+) Aggregates stay infrastructure-free and unit-testable.
- (−) Departs from idiomatic Go interface placement; new contributors must learn the
  rule (recorded here and in the architecture skill).
- (−) A downstream pipeline stage that consumes an upstream aggregate's domain read port
  (extraction reading `scraping.RawListingSource`) imports that aggregate's domain
  package; the anti-corruption mapping is kept in the consuming app service.

## Implementation constraints
- DO declare aggregate repository interfaces in `internal/domain/<aggregate>`.
- DO keep implementations in `internal/infra/<aggregate>`, satisfying the domain
  interface implicitly with a compile-time assertion.
- DO return read DTOs from the app-side query service; never the write aggregate repo.
- DO keep `llm`, `blobstore`, `messaging` capability ports where they are.
- DO NOT give a domain aggregate a reference to its repository or any infra dependency.
- DO NOT route read models through the write-side aggregate repository.
