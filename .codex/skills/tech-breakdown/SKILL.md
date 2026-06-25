---
name: tech-breakdown
description: >
  Use this skill to translate product design specs and architecture decisions
  into small, atomic implementation tasks for frontend and backend development
  teams. Trigger on phrases like: "break this down into tasks", "create dev
  tasks from the spec", "what do we need to implement", "split this into
  tickets", "task breakdown", "create implementation tasks", "what should
  the dev team build". Requires a design spec (from design-spec skill) and/or
  an architecture document or ADR (from software-architecture skill) as input.
  Output is consumed directly by software-development and frontend-development.
---

## Tech Breakdown

Translate design specs and architecture decisions into small, implementable
tasks for frontend and backend development teams.

> **Scope rule**: When invoked from the `software-architecture` task execution loop (Step 8),
> this skill breaks down **one user story at a time** — not the full feature.
> The output covers only the story passed to it. Write to
> `docs/feature/<slug>/tech-breakdown-<story-id>.md` (not the feature-level file).
> The feature-level `tech-breakdown.md` is only appropriate for a standalone invocation
> outside the loop.

## Input Resolution

This skill follows **design-spec** and **software-architecture** in the workflow. Before asking questions:

1. **Look for preceding outputs** — Check if the following documents exist and load all that are found:
   - `docs/feature/<slug>/design-spec.md` (from design-spec skill)
   - `docs/feature/<slug>/architecture.md` (from software-architecture skill)
   - If invoked from the task loop: the scoped architecture note from `software-architecture` Step 8a for this specific story.
2. **Fall back to user prompt** — If neither document exists but the user provided spec or architecture content, use that.
3. **If no input available** — Ask: *"Please share the design spec and/or architecture document for the feature you want to break down into tasks."*

---

### Questions phase

Ask the user these questions (all at once, numbered):

1. **Inputs** — Share the design spec and/or architecture document (ADR, C4
   diagram, or architecture notes). Paste or link both if available.
2. **Scope** — Are there any screens, flows, or components explicitly out of
   scope for this breakdown?
3. **Teams** — Which teams are involved? (e.g. frontend only, backend only,
   both) Are there any platform constraints (e.g. mobile web, desktop only)?
4. **Granularity** — How small should tasks be? (e.g. "1–2 hour tasks",
   "half-day tasks", "story-point sized")
5. **Dependencies** — Are there any external dependencies or blockers already
   known (e.g. API not ready, design token updates pending)?

Wait for the user's answers before proceeding.

---

### Steps

1. **Inventory inputs** — Extract every screen, component, API endpoint,
   data model, and architectural decision from the provided documents.
   List them without interpreting yet.

2. **Identify teams** — Classify each item by owner:
   - `Frontend` — UI components, interactions, state management, routing, accessibility, animations
   - `Backend` — API endpoints, data models, business logic, persistence, auth, background jobs
   - `Mobile` — native mobile screens or platform-specific work
   - `Full-stack` — vertical slices owned end-to-end by one person
   - `Design` — design work, design-token updates, or spec clarifications
   - `QA` — dedicated test-writing or QA validation tasks

   Also assign a **type** to each task:
   - `Feature` — new user-facing behaviour
   - `Chore` — setup, scaffolding, refactor, dependency update
   - `Spike` — time-boxed investigation or proof-of-concept
   - `Bug` — defect fix

3. **Detect dependencies** — Map which tasks must complete before others can
   start. A frontend task that calls an API depends on the backend task that
   implements it. Make these explicit.

4. **Split into atomic tasks** — Break each item into the smallest independently
   testable unit. One task = one PR. Apply these rules:
   - A task must be completable without waiting on another in-progress task
   - A task must have a single clear acceptance criterion
   - If "and" appears in the task title, split it

5. **Assign acceptance criteria** — For each task, write one measurable
   acceptance criterion (Given/When/Then or a binary check).

6. **Order by dependency** — Sequence tasks so that blocked tasks appear after
   their blockers. Flag any tasks that can run in parallel.

---

### Output format

```
## Tech Breakdown: <Feature Name>

**Design spec ref:** <link or "pasted above">
**Architecture ref:** <link or "pasted above">
**Teams:** Frontend / Backend / Both

---

### Tasks

---

#### <ID> — <Title>

**Type:** Feature | Chore | Spike | Bug
**Owner:** Frontend | Backend | Mobile | Full-stack | Design | QA
**Dependencies:** <ID>, <ID> or —

**Description:**
<1–3 sentence description of what needs to be built and why>

**Acceptance Criteria:**
- <binary check or Given/When/Then>
- <binary check or Given/When/Then>

---

*(repeat for each task)*

---

### Dependency Graph

```
BE-1 → BE-2 → FE-1 → FE-2
                ↑
            BE-1 (shared)
```

---

### Parallel tracks

Tasks that can be built concurrently:
- BE-1 and FE-2 (no shared dependency)

---

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | <question> | BE-1, FE-1 | BE/PM |
```

---

### Ticket guidance

When generating the task list:
- **Prefer vertical slices** — thin end-to-end slices over horizontal layers where possible
- **Split by meaningful boundaries** — API, UI, state management, tests each warrant their own task
- **Include at minimum:**
  - Setup / scaffolding tasks (if the feature requires new infra, packages, or boilerplate)
  - UI implementation tasks — one per screen or flow
  - API / data tasks — one per endpoint or data model change
  - Analytics / instrumentation tasks (if the spec calls for tracking events)
  - Accessibility tasks (if scope goes beyond per-ticket a11y criteria)
  - Testing tasks — unit, integration, and/or e2e as appropriate
  - Release tasks — feature flags, staged rollout, kill-switch wiring when relevant

---

### Output location

| Invocation context | Output file |
|---|---|
| Standalone (outside the task loop) | `docs/feature/<slug>/tech-breakdown.md` |
| From `software-architecture` task loop (Step 8) | `docs/feature/<slug>/tech-breakdown-<story-id>.md` |

`<story-id>` is the story's identifier from `user-stories.md` (e.g. `US-01`, `US-02`).
Example: `docs/feature/user-profile/tech-breakdown-US-03.md`

Commit — load the `git` skill:
```bash
git add <the output file from the table above>
git commit -m "feat(<slug>): add tech breakdown"
```

---

### Tips

- **Backend first** — Identify API contracts before writing frontend tasks.
  Frontend tasks that call an API should reference the exact endpoint and
  payload shape from the architecture doc.
- **Keep acceptance criteria measurable** — Each criterion should be a binary check or a Given/When/Then statement. If the list grows beyond 4–5 items, consider splitting the task.
- **Name tasks as verbs** — "Implement POST /users endpoint", not "Users API".
  The name should describe what gets built, not the area of the system.
- **Flag uncertainty** — If a task's scope is unclear from the spec, add an
  open question rather than guessing. Assumptions hidden in tasks become bugs.
- **Edge cases are tasks too** — Empty states, error states, and loading states
  from the design spec should each appear as explicit tasks, not assumed to be
  part of the "happy path" task.
- **Contract tasks** — When frontend and backend must agree on an API shape,
  create a shared contract task (e.g. "Define and agree on POST /orders
  request/response schema") that both teams depend on.
