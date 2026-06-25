# ADR-<NNN>: <Short Title>

> **File**: `docs/adr/ADR-<NNN>-<slug>.md`
> **Date**: YYYY-MM-DD
> **Status**: `proposed` | `accepted` | `deprecated` | `superseded`
> **Superseded by**: *(ADR-NNN link if applicable)*
> **Session**: *(link to the Architecture Session doc that produced this ADR)*

---

<!--
  ╔══════════════════════════════════════════════════════════════╗
  ║  AI CONTEXT BLOCK                                            ║
  ║  Machine-readable. Keep this block at the top of the file.  ║
  ╚══════════════════════════════════════════════════════════════╝

  Instructions for AI consumers:
  - Read `constraints` and `chosen` first — they are the minimum
    context needed to implement this decision correctly.
  - `rejected` options are documented to prevent re-opening closed
    debates. Do not propose them again unless `open_questions` or
    `assumptions` are resolved.
  - `affects` lists the bounded contexts and components this
    decision directly constrains. Use it to scope impact analysis.
  - `must_not` is a hard constraint list for the implementation
    layer — treat every item as a non-negotiable rule.
-->

```yaml
ai_context:
  adr: "ADR-NNN"
  title: ""
  status: "proposed"           # proposed | accepted | deprecated | superseded
  date: "YYYY-MM-DD"
  session: ""                  # id of the parent architecture doc (ticket-id-or-short-slug)

  decision_type: ""            # architecture-style | communication | data-storage |
                               # bounded-context | cross-cutting | tech-stack | security

  affects:                     # bounded contexts and components constrained by this decision
    contexts: []               # ["OrderContext", "PaymentContext"]
    components: []             # ["api", "worker", "db"]

  chosen: ""                   # the option that was selected (short label)

  constraints_that_drove_this: # the specific constraints that made this option win
    - ""

  rejected:                    # options that were considered and why they lost
    - option: ""
      reason: ""

  must_not:                    # hard rules for the implementation layer — non-negotiable
    - ""

  open_questions: []           # unresolved issues that could affect this decision
  assumptions: []              # declared but unvalidated premises
```

---

## Context

*What is the situation forcing this decision? What problem needs to be solved?
Include the business and technical constraints that exist. 2–4 paragraphs.*

---

## Goals

*What this decision must achieve. Each goal should be specific and verifiable.*

- Goal 1
- Goal 2

## Non-Goals

*What this decision explicitly does NOT address.*

- Non-goal 1

---

## Decision

**We will: <state the decision clearly and unambiguously in one sentence>.**

*Expand with 1–3 paragraphs explaining the chosen approach in enough detail that
an implementer does not need to ask clarifying questions.*

---

## Considered Alternatives

### Option A — <Name> *(chosen)*

*Describe the option. How does it work? What does it require?*

**Pros**
- …

**Cons**
- …

---

### Option B — <Name>

*Describe the option.*

**Pros**
- …

**Cons**
- …

**Why rejected**: *One clear sentence tied to a specific constraint or quality priority.*

---

### Option C — <Name> *(if applicable)*

**Why rejected**: …

---

## Thinking Process

*Retrace how this decision was reached. Write as a short narrative — 1–3 paragraphs.
Cover: what the initial instinct was, what tensions were surfaced during the session,
what the user confirmed or pushed back on, and what tipped the decision toward the
chosen option.*

---

## Consequences

### Positive
- …

### Negative / Trade-offs
- …

### Risks
- …

---

## Implementation Constraints

*Rules the development team and AI tools must follow when implementing this decision.
These are derived from the `must_not` block in the AI context and should be stated
in positive + negative form.*

- **DO**: …
- **DO NOT**: …

---

## Follow-up Actions

- [ ] `[owner]` Action — *due: YYYY-MM-DD*

---

## Related

- **Architecture doc**: [PROJ-123 or short-slug](../architecture/ticket-id-or-short-slug.md)
- **Depends on**: ADR-NNN
- **Required by**: ADR-NNN
