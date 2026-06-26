---
name: frontend-development
description: >
  TRIGGER — load this skill BEFORE writing, editing, or reviewing any
  React/TypeScript frontend code: any .tsx/.jsx/.ts/.js file under a frontend
  package, or a package.json depending on react/next/vite/vue. Use this skill
  for all frontend implementation tasks: writing new components,
  implementing a feature from a design spec or architecture document, structuring
  a feature module, React/TypeScript idioms and patterns, state management, hook
  design, testing strategy (unit, component, e2e), writing or reviewing tests,
  PR review, and refactoring existing frontend code. Trigger on phrases like:
  "implement this component", "write the frontend for", "how do I structure this
  feature", "review my React code", "refactor this hook", "write a test for",
  "how do I handle this state", "this component is too big". Always reads design
  specs before generating code for a new feature. Does NOT make architectural
  decisions — if the tech stack or architecture is unclear, flags this first.
---

# Frontend Development Skill

> **Load foundations — in order:**
> 1. Read `base-guidelines` — DDD, SOLID, clean code, naming principles.
> 2. Read `frontend-guidelines` — Framework-agnostic frontend rules: TypeScript, component design, file structure, state management, accessibility.
> 3. Read `react-guidelines` — React-specific rules: hook design, React state APIs, data fetching, React naming, React code smells.
>
> The following skills are loaded at **step 9 (tests)** only, not upfront:
> `testing-strategy` → `frontend-testing`
> **At step 9, load these skills before writing any tests — never skip step 9.**
>
> Run affected tests during task implementation. Run the full suite and quality gates once during story finalization — see step 13.
>
> Return here after reading the three foundation skills.

---

## Scope of This Skill

This skill covers **how to implement** decisions that have already been made — not what to build or how to design it.

| In scope | Out of scope |
|---|---|
| Implementing components from design specs | UI/visual design decisions |
| Testing strategy (unit, component, e2e) | Architecture pattern selection |
| PR review & refactoring | Tech stack decisions |
| State management wiring | Design system definition |
| Implementation workflow | Wireframes, design specs → `wireframe`, `design-spec` |
| | React/TS idioms & patterns → `frontend-guidelines` |

> **If the design spec is unclear, missing, or contradicts the task:** stop, flag it
> explicitly, and direct the user to resolve it with the `design-spec` skill before
> proceeding. Never invent interaction behaviour or visual decisions silently.

---

## 0. Task Implementation and Story Finalization

> **This skill implements ONE tech-breakdown task per invocation.**
>
> When invoked from the task execution loop (via `software-architecture` Step 8):
> - Read the current story's tech-breakdown (`docs/feature/<slug>/tech-breakdown-<story-id>.md`),
>   falling back to `docs/feature/<slug>/tech-breakdown.md`.
> - Identify the **first unimplemented task** in it.
> - Implement that task only — types, components, hooks, data layer, and tests.
> - Run only directly affected tests, then commit the task. The user's request to implement an explicit task or workflow authorizes per-task commits unless they say not to commit. Do not run the full suite or quality gates while story tasks remain.
> - Stop and report: *"Task [ID] — [title] implemented, affected tests pass, and task commit is [hash]. Full gates deferred until story completion. Ready for task [ID+1]: [title]?"*
> - Wait for explicit confirmation before implementing the next task.
> - After the final task, or when explicitly asked to finalize, run the full suite and quality gates once. Do not create a bundle commit; task commits already exist.
>
> **Never implement multiple tasks in a single response.** If the tech-breakdown has 8 tasks, this skill runs 8 times — one task per run.

---

## 1. Before Writing Any Code

### Input Resolution

Before asking any questions, resolve what needs to be built:

1. **Look for a tech breakdown** — Check if `docs/feature/<slug>/tech-breakdown.md` exists.
   - If it does: use it as the authoritative task definition. Read the current task, its directly named design spec or wireframe, and only linked ADRs or contracts required to implement that task. Do not load the entire feature directory.
   - If it does not: check whether the user's prompt describes the task clearly enough to proceed.
2. **Fall back to the user prompt** — If no tech breakdown exists but the prompt is clear, use it directly.
3. **If neither exists** — Ask: *"What do you need me to implement? Please share the task description, design spec, or wireframe."*

---

### Read the Design Spec
For any feature involving UI, before generating a single line of code:

1. If a tech breakdown was found above, read the design spec named in the task from `docs/feature/<slug>/`.
2. Otherwise ask: *"Is there a design spec or wireframe for this?"* If yes — read it and extract:
   - **Component inventory** — every screen, component, and variant.
   - **States** — loading, empty, error, populated, disabled, hover, focus.
   - **Interactions** — what happens on each user action.
   - **Edge cases** — empty states, validation errors, permission boundaries.
3. If no spec exists for a significant feature — tell the user:
   *"There's no design spec for this. For anything beyond a trivial change, I'd
   recommend running through the design-spec skill first. Should we do that, or
   proceed with explicit assumptions?"*

If proceeding with assumptions, **declare them** at the top of your response before any code.

### Understand the Task Fully Before Coding
Ask clarifying questions before writing code when:
- The component tree or data flow is not clear.
- State ownership is ambiguous (local vs. shared vs. server).
- Error and empty states are not specified.
- Accessibility requirements are not stated.

Do not ask more than 2–3 questions at once. Ask the most important ones first.

---

## 2. Frontend Implementation Rules

> Read `frontend-guidelines` for framework-agnostic rules:
> component design, TypeScript usage, file & folder structure, state management
> concepts, naming, accessibility, and general code smells.
> Read `react-guidelines` for React-specific rules:
> hook design, React state APIs, data fetching, and React code smells.
> Return here after reading both.

---

## 3. Testing Strategy

> Read `testing-strategy` and `frontend-testing`
> for the full testing rules: AC → test case mapping, test type selection, naming convention,
> unit/component/e2e patterns, MSW handler organisation, Page Object Model, and the AC coverage gate.
> These skills are loaded at step 9. Return here after reading them.

---

## 4. Refactoring

### When to Refactor
Refactor when:
- A component exceeds ~150 lines and contains multiple logical sections.
- The same JSX pattern appears in a third place (Rule of Three).
- A `useEffect` is compensating for poorly modelled state.
- A test is hard to write because the component mixes data fetching and rendering.
- A name no longer matches what the component or hook does.

Never mix refactoring and feature work in the same commit.

### Common Frontend Smells & Fixes

> The full smell catalogue is in `frontend-guidelines` §8.
> Consult it when a smell is frontend-specific (hook misuse, prop drilling, type escapes, etc.).

---

## 5. PR Review Checklist

When reviewing or self-reviewing frontend code, verify in this order:

**Correctness**
- [ ] Does the component render correctly for all states from the design spec?
- [ ] Are all loading, empty, and error states handled?
- [ ] Do all interactive elements work via keyboard, not just mouse?
- [ ] Are form inputs associated with labels?

**Design**
- [ ] Is business logic in hooks, not in component bodies?
- [ ] Is state owned at the lowest correct level?
- [ ] Does the component cross a feature boundary without going through the feature's barrel?
- [ ] Are there any prop-drilling chains longer than 2 levels?

**Clarity**
- [ ] Do component and hook names reveal intent without requiring comments?
- [ ] Is there dead code or commented-out JSX?
- [ ] Are there magic strings or numbers that should be constants or enums?

**Tests**
- [ ] Is every rendered state covered by at least one component test?
- [ ] Are queries in tests by role/label, not by class or test-id?
- [ ] Are network requests mocked with MSW, not with module mocks?

**TypeScript & idioms**
> Verify against the full rules in `frontend-guidelines`. Key items:
- [ ] No `any` types?
- [ ] No `as` casts without a runtime guard and justification?
- [ ] Custom hooks follow the rules of hooks?
- [ ] `useEffect` dependencies are complete and correct?
- [ ] Named exports used throughout (no default exports on components)?

---

## 6. Implementation Workflow

For any non-trivial task, follow this sequence. Do not skip steps.

```
1. Read design spec & architecture docs      → extract component inventory, states, interactions
2. Declare assumptions (if any)              → state them before writing any code
3. Identify affected features/modules        → map the task to the folder structure
4. Define types/interfaces first             → props, state shapes, API response types
5. Implement leaf components (bottom-up)     → pure presentational components first
6. Implement hooks & data layer              → fetching, mutations, local state logic
7. Compose into feature components           → wire leaves + hooks into feature-level views
8. Connect to routing / page layout          → integrate into the app shell
9. Write tests — **load `testing-strategy` → `frontend-testing` now, then**:
      a. Map ACs to test cases           → every AC must have at least one test
      b. Write unit tests                → hooks + utility functions
      c. Write component tests           → all states, interactions, edge cases
      d. Write e2e tests                 → AC-level journeys only (Playwright)
      e. Run AC coverage gate            → confirm every AC is covered
10. Update component specs              → for each domain component created or changed,
      create/update `docs/components/<component>.md` (see `base-guidelines` §7).
11. Run only directly affected frontend tests for this task.
12. Load `git`, stage only files changed for this task, commit the task, and verify
      with `git status`. The user's request to implement an explicit task or workflow
      authorizes per-task commits unless they say not to commit.
      One explicit user task outside a workflow, or one tech-breakdown task inside a
      workflow, equals one commit.
13. If story tasks remain, STOP and report the task complete. Do not run the full suite
      or quality gates. Wait for confirmation before the next task.
14. After the final story task, or when explicitly asked to finalize, run once:
      a. `rtk vitest run`
      b. `rtk npx eslint frontend/`
      c. `rtk npx tsc --noEmit`
    Fix findings yourself and rerun only failed checks.
15. STOP and report finalization status, files changed, and commit status.
```

At step 4–7, if you discover the design spec is missing something significant
(an undocumented state, a missing edge case, a conflicting interaction), **stop and
surface it** before continuing. Do not invent UI decisions silently.

---

## 7. What This Skill Does NOT Cover

| Concern | Covered by |
|---|---|
| Visual design, colour, spacing, typography | `design-spec`, `design-system` |
| Wireframes and user flows | `wireframe` |
| Tech stack & framework decisions | `software-architecture` |
| DDD building blocks & SOLID rules | `base-guidelines` |
| React/TS idioms & component patterns | `frontend-guidelines` |
| AC → test case mapping, test type selection | `testing-strategy` |
| Frontend test patterns (component, hook, e2e, MSW) | `frontend-testing` |
| Linting, formatting, import order | `frontend-eslint` |
| TypeScript strict type checking | `frontend-tsc` |
