---
name: software-development
description: >
  TRIGGER — load this skill BEFORE writing, editing, or reviewing any Go code:
  any .go file touched, a directory containing go.mod/go.sum, or commands like
  `go build`/`go test`/`go run`/`go vet`. Use this skill for all software
  implementation tasks in Go: writing new code,
  implementing a feature from an architecture document or ADR, structuring a
  package or module, Go-specific idioms and patterns, dependency injection wiring,
  testing strategy (unit, integration, contract), writing or reviewing tests,
  PR review, refactoring existing code, and identifying code smells. Trigger on
  phrases like: "implement this", "write the code for", "how do I structure this
  package", "review my code", "refactor this", "write a test for", "what's the
  Go way to", "how should I wire dependencies", "this code smells". Always reads
  architecture docs and ADRs before generating code for a new feature. Does NOT
  make architectural decisions — if the architecture is unclear or missing, flags
  this and directs the user to the software-architecture skill first.
---

# Software Development Skill

> **STOP — load these dependencies before reading further, in order:**
>
> 1. Invoke the Skill tool now: `Skill(skill: "base-guidelines")`
> 2. Invoke the Skill tool now: `Skill(skill: "go-guidelines")`
>
> Do not read past this line until both skills have been loaded.
> Then return here and continue with the next section.
>
> The following skills are loaded **at step 8 (tests) only**, not upfront:
> `testing-strategy` → `go-testing` — **at step 8, load these skills before writing any tests, never skip.**
>
> Run affected tests during task implementation. Run full tests and quality gates once during story finalization — see step 12.

---

## Scope of This Skill

This skill covers **how to implement** decisions that have already been made — not what to build.

| In scope | Out of scope |
|---|---|
| Implementing features from architecture docs | Architectural pattern selection |
| Testing strategy (unit, integration, contract) | Bounded context design |
| PR review & refactoring | Tech stack decisions |
| Implementation workflow | C4 diagrams, ADRs |
| | Go idioms & language rules → `go-guidelines` |

> **If the architecture is unclear, missing, or contradicted by the task:** stop, flag it
> explicitly, and direct the user to resolve it with the `software-architecture` skill before
> proceeding. Never invent architectural decisions silently.

---

## 0. Task Implementation and Story Finalization

> **This skill implements ONE tech-breakdown task per invocation.**
>
> When invoked from the task execution loop (via `software-architecture` Step 8):
> - Read the current story's tech-breakdown (`docs/feature/<slug>/tech-breakdown-<story-id>.md`).
> - Identify the **first unimplemented task** in it.
> - Implement that task only — domain types, application service, infra adapter, wiring, and tests.
> - Run only the smallest directly affected tests. Do not run full tests, quality gates, or commit while story tasks remain.
> - Stop and report: *"Task [ID] — [title] implemented. Affected tests pass. Full gates deferred until story completion. Ready for task [ID+1]: [title]?"*
> - Wait for explicit confirmation before implementing the next task.
> - After the final task, or when explicitly asked to finalize, run the full suite and quality gates once, then commit the story only when authorized.
>
> **Never implement multiple tasks in a single response.** If the tech-breakdown has 8 tasks, this skill runs 8 times — one task per run.

---

## 1. Before Writing Any Code

### Input Resolution

Before asking any questions, resolve what needs to be built:

1. **Look for a tech breakdown** — Check if `docs/feature/<slug>/tech-breakdown-<story-id>.md` exists (story-scoped), falling back to `docs/feature/<slug>/tech-breakdown.md` (feature-scoped).
   - If it does: use it as the authoritative task definition. Identify which task is next (first without a ✅ marker or explicit "done" confirmation). Read the current task, its directly named architecture/design document, and only linked ADRs or contracts required to implement that task. Do not load the entire feature directory.
   - If it does not: check whether the user's prompt describes a single, specific task clearly enough to proceed.
2. **Fall back to the user prompt** — If no tech breakdown exists but the prompt describes a single concrete task, use it directly.
3. **If neither exists** — Ask: *"What do you need me to implement? Please share the task description, design spec, or architecture document."*

---

### Read the Architecture Documents
For any non-trivial feature, before generating a single line of code:

1. If a tech breakdown was found above, read the architecture doc named in the task from `docs/feature/<slug>/`.
2. Otherwise ask: *"Is there an architecture document or ADR for this?"* If yes — read `docs/architecture/<id>.md` and the relevant `docs/adr/ADR-NNN.md` files. Extract from the `ai_context` block:
   - `must_not` constraints — these are hard rules, never violate them.
   - `affected contexts` and `affected components` — know what you're touching.
   - `open_questions` and `assumptions` — flag any that affect the implementation.
3. If no architecture doc exists for a significant feature — tell the user:
   *"There's no architecture document for this. For anything beyond a small isolated
   change, I'd recommend running through the software-architecture skill first.
   Should we do that, or proceed with explicit assumptions?"*

If proceeding with assumptions, **declare them** at the top of your response before any code.

### Understand the Task Fully Before Coding
Ask clarifying questions before writing code when:
- The requirement has ambiguous boundaries (what's in / out of scope).
- The feature touches multiple bounded contexts.
- The data flow is not clear.
- Error handling expectations are not stated.

Do not ask more than 2–3 questions at once. Ask the most important ones first.

---

## 2. Go Implementation Rules

> Read `go-guidelines` skill for the full Go rules.
> That skill covers: package design, structs & constructors, interfaces, dependency
> injection, context propagation, concurrency, error handling, and Go-specific smells.
> Return here after reading it.

---

## 3. Testing Strategy

> Read `testing-strategy` skill and `go-testing` skill
> for the full testing rules: AC → test case mapping, test type selection, naming convention,
> the test cases format, unit/integration/contract/acceptance patterns, and the AC coverage gate.
> These skills are loaded at step 8. Return here after reading them.

---

## 4. Refactoring

### When to Refactor
Refactor when:
- A function exceeds ~20 lines and can be extracted cleanly.
- The same logic appears in a third place (Rule of Three).
- A name no longer reflects what the code does.
- A test is hard to write because the code has too many dependencies.
- A change in one place requires changes in many unrelated places.

Never refactor and add features in the same commit. Separate the refactor (behaviour-preserving) from the feature change.

### Common Go Smells & Fixes

> The full Go smell catalogue is in `go-guidelines` §8.
> Consult it when a smell is Go-specific (goroutine lifecycle, context propagation, interface placement, etc.).

---

## 5. PR Review Checklist

When reviewing or self-reviewing code, verify in this order:

**Correctness**
- [ ] Does the code do what the ticket / ADR requires?
- [ ] Are all error paths handled and wrapped with context?
- [ ] Are all `ctx.Err()` checks present in loops and I/O chains?
- [ ] Are there any data races (goroutines accessing shared state without synchronisation)?

**Design**
- [ ] Does the code respect the layering rule (`handler → app → domain`)?
- [ ] Is business logic in the domain layer, not in handlers or repositories?
- [ ] Does the new code cross a bounded context boundary directly? If so, is that intended and via a defined contract?
- [ ] Does it violate any `must_not` constraint from the relevant ADR?

**Clarity**
- [ ] Do names reveal intent without requiring a comment to explain them?
- [ ] Is there dead code or commented-out code?
- [ ] Are there any magic numbers or strings that should be named constants?

**Tests**
- [ ] Is the new behaviour covered by at least one unit test?
- [ ] Are table-driven tests used where there are multiple cases?
- [ ] Do the tests assert behaviour, not internal state?
- [ ] Are integration tests tagged with `//go:build integration`?

**Go idioms**
> Verify against the full rules in `go-guidelines`. Key items:
- [ ] Interfaces defined in the consumer package?
- [ ] Constructors validate inputs and return errors for invalid state?
- [ ] `context.Context` is the first parameter of all I/O functions?
- [ ] All goroutines are owned and their lifecycles managed?

---

## 6. Implementation Workflow

For any non-trivial task, follow this sequence. Do not skip steps.

```
1. Read architecture docs & ADRs         → extract constraints, must_nots, affected contexts
2. Declare assumptions (if any)          → state them before writing code
3. Identify the affected packages        → map the task to the Go project structure
4. Write the domain types first          → entities, value objects, domain events, errors
5. Write the application service         → use case orchestration, no infra dependencies
6. Write the infra adapter(s)            → repository impl, HTTP client, message publisher
7. Wire in cmd/                          → connect concrete types via constructors
8. Write tests — **load `testing-strategy` → `go-testing` now, then**:
      a. Map ACs to test cases           → every AC must have at least one test
      b. Write unit tests                → domain + application layer
      c. Write integration tests         → infra adapters only (if needed)
      d. Write contract tests            → bounded context boundaries (if needed)
      e. Run AC coverage gate            → confirm every AC is covered
9. Update component specs               → for each domain component created or changed,
      create/update `docs/components/<component>.md` (see `base-guidelines` §7).
10. Run the smallest directly affected Go test package(s) for this task.
11. If story tasks remain, STOP and report the task complete. Do not run full tests,
      quality gates, or commit. Wait for confirmation before the next task.
12. After the final story task, or when explicitly asked to finalize, run once:
      a. `rtk goimports -w backend/`
      b. `rtk go test ./...`
      c. `rtk go vet ./...`
      d. `rtk golangci-lint run ./...`
      e. `rtk govulncheck ./...`
    Fix findings yourself and rerun only failed checks.
13. Commit the complete story only when explicitly authorized. Load `git`, stage only
      story files, commit once, and verify with `git status`.
14. STOP and report finalization status, files changed, and commit status.
```

At step 4–6, if you discover the architecture docs are missing something significant
(an undocumented boundary, a missing contract, a conflicting constraint), **stop and
surface it** before continuing. Do not paper over architecture gaps with code.

---

## 7. What This Skill Does NOT Cover

| Concern | Covered by |
|---|---|
| Architectural pattern selection | `software-architecture` |
| Bounded context design | `software-architecture` |
| C4 diagrams, ADRs | `software-architecture` |
| Tech stack & infrastructure decisions | `software-architecture` |
| DDD building blocks & SOLID rules | `base-guidelines` |
| Go project structure & package layout | `base-guidelines` |
| Go idioms, patterns & language rules | `go-guidelines` |
| AC → test case mapping, test type selection | `testing-strategy` |
| Go test patterns (test cases format, fakes, integration, contract) | `go-testing` |
| Formatting & import organisation | `go-goimports` |
| Built-in static analysis | `go-vet` |
| Linting & code smell detection | `go-golangci-lint` |
| Vulnerability scanning | `go-govulncheck` |
