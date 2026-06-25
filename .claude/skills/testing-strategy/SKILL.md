---
name: testing-strategy
description: >
  DEPENDENCY — loaded by go-testing and frontend-testing. Defines how to map
  acceptance criteria to test cases, select the right test level for each
  scenario, and name tests as specifications. Never triggered directly. Covers
  the "what to test and why" — not the "how" (that is in go-testing or
  frontend-testing).
---

# Testing Strategy

> **This file is a dependency of `go-testing` and `frontend-testing`.** It is never used standalone.
> After reading this file, return to the skill that loaded it and continue from where you left off.

---

## 1. Core Principle: Tests Are Specifications

Every test is a machine-executable specification of a requirement.
A failing test means a broken requirement — not just broken code.
Name tests so that a failing run tells you *which requirement failed*, not *which function failed*.

---

## 2. Acceptance Criteria → Test Cases

Before writing any test, map requirements to test cases explicitly.

### The mapping process

1. Read the user story or feature spec. Extract every acceptance criterion (AC).
2. For each AC, determine the lowest test level that can verify it (see §3).
3. Write one or more test cases per AC. Name them to mirror the AC.
4. Track coverage: every AC must map to at least one test. Flag uncovered ACs before closing the task.

### AC traceability rule

Tests that directly verify ACs must reference the AC in their name or a comment:

```
// AC: user cannot place an order with an empty cart
"returns error when cart is empty"
```

If you cannot trace a test to a requirement or a known edge case, question whether it should exist.

---

## 3. Test Type Selection

### Decision table

| Scenario | Test type |
|---|---|
| Pure function, domain rule, value object invariant | Unit |
| Application service use case (no real I/O) | Unit (with fakes) |
| Module or component public API, multiple collaborators | Integration / Component |
| Repository, HTTP client, message adapter against real infra | Integration (infra) |
| User-visible outcome described in an AC | Acceptance |
| Contract between two bounded contexts or services | Contract |

### When NOT to write a test

- Constructor with no logic (no branch, no validation).
- Trivial getter or formatter with no conditional path.
- Framework glue with no business logic.
- Code that is already covered transitively by a higher-level test and adds no new scenario.

Adding a test for every line of code dilutes the signal. Coverage is a floor, not a goal.

---

## 4. Naming Convention

Tests read as specifications — subject, condition, expected outcome.

**Pattern**: `"<subject> <condition> <outcome>"`

Good names:
- `"order service returns error when cart is empty"`
- `"payment handler redirects to confirmation on success"`
- `"inventory checker rejects reservation when stock is zero"`

Bad names:
- `"TestPlaceOrder"` — no condition, no outcome
- `"test1"` — meaningless
- `"error case"` — no subject or condition

For AC-driven tests, the name should be paraphraseable as the AC itself.

---

## 5. Test Doubles

Use the simplest double that gets the job done. Prefer in order:

1. **Fake** — a working, simplified in-memory implementation. Best for domain and application layer tests. Write it once, reuse across the test suite.
2. **Stub** — hardcoded return values. Use for simple, single-path scenarios.
3. **Spy / Mock** — records calls and verifies them. Use only when the *interaction itself* (not the result) is what the test asserts.

Never mock what you own at the unit level — use a fake instead. Mocks of internal interfaces make tests brittle and resist refactoring.

---

## 6. AC Coverage Gate

Before marking a task done, verify:

- [ ] Every AC has at least one automated test.
- [ ] Every test name is traceable to an AC or a documented edge case.
- [ ] No AC is tested only by a manual step ("manually verified").
- [ ] Tests at the wrong level are flagged (e.g. an e2e test covering something a unit test could cover at lower cost).

If an AC cannot be automated (e.g. visual regression, accessibility audit), document why and state what manual check replaces it.
