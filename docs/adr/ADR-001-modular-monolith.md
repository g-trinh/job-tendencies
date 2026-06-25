---
ai_context:
  decision: "Modular monolith codebase, deployed as multiple binaries sharing one internal/"
  chosen: "Single Go module; bounded contexts as packages; cmd/api + cmd/scrape-worker + cmd/extract-worker"
  rejected: ["microservices with separate repos/modules", "single binary with in-process goroutine workers"]
  must:
    - "Modules map 1:1 to bounded contexts and communicate through app-service interfaces"
    - "Each cmd/<bin>/main.go is a thin composition root, no domain logic"
    - "All binaries import the same internal/ packages — never fork/duplicate domain code"
  must_not:
    - "Access another context's repository or DB tables directly"
    - "Share domain objects across contexts (pass IDs and DTOs)"
  parent: "docs/architecture/overview.md"
---

# ADR-001 — Modular monolith codebase, multi-binary deployment

## Status
Accepted — 2026-06-25

## Context
Single-user job intelligence tool today, not excluded from multi-user later. Six features
map onto bounded contexts. Scheduled scrape/extract work must run separately from the API
process (off the request path, independently triggerable by Cloud Scheduler), but the team
is one person and the domain is shared across all of it.

## Decision
One Go module, modular monolith codebase. Bounded contexts are packages under `internal/`
(`domain`/`app`/`infra` per context). Deploy as **three thin binaries** that import the
same packages: `cmd/api`, `cmd/scrape-worker`, `cmd/extract-worker`. Each `main` wires only
the dependencies its binary needs. Contexts talk through application-service interfaces;
domain objects never cross context boundaries.

## Alternatives considered
- **Microservices (separate repos/modules per context)** — rejected: operational and
  cognitive cost with zero benefit for a one-person, single-user system; would force
  network contracts and distributed-transaction handling we don't need.
- **Single binary with goroutine workers** — rejected: the user requires scheduled work to
  run as separate binaries triggered by Cloud Scheduler, off the API process; also couples
  scaling of API and pipeline.

## Consequences
- (+) Shared domain, no duplication; extracting a context to its own service later is
  mechanical, not a rewrite.
- (+) Each binary scales and deploys independently (Cloud Run services).
- (−) All binaries share a build and a module version; a domain change rebuilds all three.
  Acceptable at this scale.

## Implementation constraints
- DO keep `cmd/<bin>/main.go` thin: open resources, construct app-services, wire handlers.
- DO call other contexts through their app-service interface only.
- DO NOT import another context's `infra`/repository or query its tables directly.
- DO NOT pass domain entities across contexts — use IDs and DTOs.
