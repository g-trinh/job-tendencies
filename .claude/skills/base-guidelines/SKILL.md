---
name: base-guidelines
description: >
  SHARED FOUNDATION — never triggered directly. This skill is a dependency
  imported by other skills (software-architecture, software-development).
  It contains universal engineering principles: DDD, clean code, KISS, SOLID,
  YAGNI, naming conventions, layering rules, and project structure conventions.
  Do NOT use standalone. Trigger only via a parent skill that explicitly says
  "first, read base-guidelines/SKILL.md".
---

# Base Engineering Guidelines

> **This file is a shared library.** It is read by other skills, never used alone.
> After reading this file, return to the parent skill that directed you here.

---

## 1. Core Design Philosophies

### KISS — Keep It Simple, Stupid
- Prefer the simplest solution that correctly solves the problem.
- Complexity must be justified by a concrete requirement, not anticipated future needs.
- If you can explain a solution in one sentence, it is probably the right one.

### YAGNI — You Aren't Gonna Need It
- Do not implement features, abstractions, or flexibility that have no current use case.
- Generalization is earned through duplication first (Rule of Three: abstract only after the third occurrence).

### DRY — Don't Repeat Yourself
- Every piece of knowledge must have a single, unambiguous, authoritative representation.
- DRY applies to *knowledge*, not necessarily to code — two similar lines are not always a DRY violation.

### Separation of Concerns
- Each module, class, or function should have one clearly defined responsibility.
- Business logic must never leak into infrastructure (DB, HTTP, I/O) layers.
- Infrastructure must never contain domain decisions.

---

## 2. SOLID Principles

| Principle | Rule |
|---|---|
| **S**ingle Responsibility | A class/module changes for one reason only |
| **O**pen/Closed | Open for extension, closed for modification |
| **L**iskov Substitution | Subtypes must be substitutable for their base types |
| **I**nterface Segregation | Many specific interfaces > one general-purpose interface |
| **D**ependency Inversion | Depend on abstractions, not concretions |

Apply SOLID proportionally — a 20-line utility script does not need a DI container.

---

## 3. Domain-Driven Design (DDD)

### Ubiquitous Language
- Use the exact vocabulary of the domain experts in code: class names, method names, variable names, module names.
- Never mix technical jargon with domain terms in the same name (e.g., `UserRepository` is fine; `UserDataAccessObject` leaks infrastructure).
- Maintain a glossary per bounded context if the domain is complex.

### Bounded Contexts
- A bounded context is a linguistic boundary: the same word can mean different things in different contexts.
- Each bounded context owns its own model; never share domain objects across contexts.
- Cross-context communication happens through well-defined contracts (events, DTOs, APIs) — never through shared databases or shared domain classes.

### Building Blocks

| Block | Rule |
|---|---|
| **Entity** | Has identity (`id`), mutable state, lifecycle |
| **Value Object** | No identity, immutable, compared by value |
| **Aggregate** | Cluster of entities/VOs with one root; enforce invariants inside the aggregate |
| **Domain Service** | Stateless operation that doesn't belong to a single entity |
| **Repository** | Write side: load/save whole aggregates, returns domain objects not DB rows. Split off a read model (query→DTOs) for list/search; never serve UI queries through the aggregate repository |
| **Domain Event** | Something that happened in the domain; past tense, immutable |
| **Factory** | Encapsulates complex construction logic |

### Aggregate Rules
- Only reference other aggregates by ID, never by object reference.
- All mutations go through the aggregate root.
- Keep aggregates small; large aggregates are a design smell.

---

## 4. Clean Code

### Naming
- Names should reveal intent. If you need a comment to explain a name, rename it.
- Variables: noun or noun phrase (`userEmail`, `orderTotal`).
- Functions/methods: verb or verb phrase (`calculateTax`, `sendWelcomeEmail`).
- Booleans: question form (`isActive`, `hasPermission`, `canEdit`).
- Avoid abbreviations unless universally known (`id`, `url`, `http` are fine; `usrMgr` is not).
- Avoid generic names: `data`, `info`, `manager`, `handler`, `util`, `helper` — be specific.

### Functions
- A function does one thing, and does it well.
- Maximum recommended length: ~20 lines. If it grows longer, extract.
- Arguments: 0–2 ideal, 3 acceptable, 4+ is a smell — consider a parameter object.
- No side effects in functions that appear to only compute/query.
- Command-Query Separation: a function either changes state *or* returns a value, not both.

### Comments
- Prefer self-documenting code over comments.
- Comments are for *why*, not *what* (the code already says what).
- Delete commented-out code; use version control instead.
- Exception: public API documentation comments are always appropriate.

### Error Handling
- Use exceptions (or typed errors) for exceptional cases, not control flow.
- Fail fast: validate inputs at boundaries, not deep inside business logic.
- Never swallow exceptions silently.
- Error messages must be actionable: say what went wrong and, where possible, how to fix it.

---

## 5. Project Structure Conventions (Go)

### Directory Layout

```
<module-root>/               ← go.mod lives here (e.g. github.com/org/service)
├── cmd/
│   └── <entrypoint>/        ← main packages only; one per binary (e.g. cmd/api/, cmd/worker/)
│       └── main.go
├── internal/                ← private application code; not importable by outside modules
│   ├── domain/              ← Entities, Value Objects, Aggregates, Domain Services, Repository interfaces
│   │   └── <context>/       ← one package per bounded context (e.g. internal/domain/order/)
│   ├── app/                 ← Use cases / Application Services (orchestrate domain, no business logic)
│   │   └── <context>/
│   ├── infra/               ← Repository implementations, DB, HTTP clients, external services
│   │   └── <context>/
│   └── handler/             ← HTTP handlers, gRPC servers, CLI commands, event consumers
│       └── <context>/
├── pkg/                     ← Reusable packages safe to import by external modules (only if truly generic)
├── migrations/              ← SQL migration files
└── config/                  ← Configuration structs and loaders (no business logic)
```

> Prefer `internal/` over `pkg/` by default. Only promote to `pkg/` when an explicit external consumer exists.

### Dependency Rule
- Dependencies always point **inward**: `handler → app → domain`.
- `domain` imports nothing from `app`, `infra`, or `handler` — zero outward dependencies.
- `infra` implements interfaces declared in `domain` (Go interfaces are satisfied implicitly).
- `cmd/` is the composition root: it wires dependencies together and must not contain business logic.

### Package Naming
- Package names are lowercase, single words, no underscores: `order`, `payment`, `infra`.
- Package name = last segment of import path; avoid stutter (`order.Order` is fine, `orderservice.OrderService` is not — rename the package).
- One package per bounded context per layer (e.g. `internal/domain/order`, `internal/app/order`).
- Avoid `util`, `common`, `shared`, `helpers` — name packages after what they provide, not what they are.

### File Naming
- All filenames: `snake_case.go`.
- Name files after the primary concept they contain: `order.go`, `order_repository.go`, `create_order.go`.
- Test files co-located with source: `order_test.go`.
- Group by concept, not by type — don't create `entities.go`, `interfaces.go` mega-files.

### Interface Conventions
- Define interfaces **in the package that uses them**, not the package that implements them.
- Keep interfaces small: prefer one-method interfaces (`Reader`, `Storer`) over large contracts.
- Repository interfaces live in `domain/<context>`; implementations live in `infra/<context>`.

### Error Handling
- Always handle errors explicitly; never use `_` to discard an error without a documented reason.
- Wrap errors with context using `fmt.Errorf("creating order: %w", err)`.
- Define sentinel errors (`var ErrNotFound = errors.New(...)`) or typed errors in `domain` for cases callers must handle differently.
- Panic only for unrecoverable programmer errors (e.g. nil pointer on startup config); never for expected runtime failures.

### Module Boundaries
- Everything under `internal/` is private to the module — Go enforces this at compile time.
- Exported identifiers (capital letter) within `internal/` are still package-private to the module.
- The public surface of a package is defined by its exported symbols; document all of them with a Go doc comment (`// TypeName ...`).

---

## 6. Code Quality Gates

Before any code is considered done:

- [ ] Does it follow the ubiquitous language of the bounded context?
- [ ] Is business logic in the domain layer, not in controllers or DB queries?
- [ ] Are there no cross-context domain object references (only IDs or DTOs)?
- [ ] Do names reveal intent without needing a comment?
- [ ] Does each function/method do exactly one thing?
- [ ] Are all error cases handled explicitly?
- [ ] Is there no dead code or commented-out code?
- [ ] Does the dependency flow obey the layering rule (inward only)?

---

## 7. Component Specs — Keep the Domain Reference Up to Date

`docs/components/<component>.md` is the **living reference** of the domain's implemented
components/concepts — a project-wide glossary of "what exists and how it behaves right now".
It is distinct from `docs/feature/<slug>/` (per-feature working docs: architecture, tech
breakdowns, todos — historical/in-progress) and from `docs/architecture/` /  `docs/adr/`
(decisions and rationale).

### When to update
After implementing or changing a domain component (entity, aggregate, core business/domain
concept), create or update its `docs/components/<component>.md` in the **same commit** as
the implementation. One file per component — not per feature.

### Format
```
# Component: <Name>

<One-line description of what this is.>

## Properties

| Property | Type | Description |
|----------|------|-------------|
| ...      | ...  | ...         |

## <Flow / Rules / States — whatever sections fit the component>

## Notes

- <Implementation detail, constraint, or invariant worth flagging>
```

- Cross-link related components with relative links: `[member](member.md)`.
- Describe the **implemented** state only — no proposals, no TODOs, no open questions.
  If something is planned but not built, it does not belong here yet.
- Keep it short: a domain reference, not a design doc. If a section needs paragraphs of
  rationale, that belongs in the architecture doc/ADR, not here.

---

## 8. What This Skill Does NOT Cover

The following concerns are **out of scope** for this base skill.
They are handled by the parent skills that import this file:

| Concern | Covered by |
|---|---|
| Choosing architectural patterns (CQRS, event-driven, hexagonal…) | `software-architecture` |
| Writing ADRs, C4 diagrams, tech stack decisions | `software-architecture` |
| Code generation, language-specific idioms | `software-development` |
| Testing strategy, PR review checklists | `software-development` |
| Refactoring patterns | `software-development` |

---

> **Done reading.** Return to the parent skill that directed you here and continue with its instructions.
