---
name: ui-development
description: >
  Use this skill for implementing modern HTML & CSS templates and pages: semantic
  HTML5 structure, modern CSS (Grid, Flexbox, custom properties, container queries,
  cascade layers, clamp()-based fluid typography/spacing, :is/:where/:has,
  :focus-visible), responsive mobile-first layout, and accessible markup. Trigger on
  phrases like: "build this HTML template", "implement this page in HTML and CSS",
  "write the markup for", "make this responsive", "convert this design to HTML/CSS",
  "this needs modern CSS", "style this component with vanilla CSS". Reads
  design-system (tokens, component inventory) and design-spec (states, interactions,
  breakpoints) before generating any markup or styles. Does NOT cover framework
  component implementation (React/Vue/etc.) — see frontend-development. Does NOT
  make visual design decisions — see design-spec/design-system.
---

# UI Development Skill (HTML & CSS)

> **Load before coding:** `design-system` (tokens, component inventory) and
> `design-spec` (states, interactions, breakpoints, copy) for this page/feature.
> Tokens and states drive every decision below. Return here after reading them.

---

## Scope of This Skill

| In scope | Out of scope |
|---|---|
| Semantic HTML structure & document outline | Visual design decisions (colour, spacing, typography choices) → `design-system`, `design-spec` |
| Modern CSS layout (Grid, Flexbox, container queries) | Framework component implementation (React/Vue/etc.) → `frontend-development` |
| Design tokens → CSS custom properties | JS interactivity / state logic → `frontend-development`, `frontend-guidelines` |
| Responsive, mobile-first, fluid sizing | Design system definition → `design-system` |
| Accessible markup (WCAG AA) | Backend templating engine internals |
| Static template / partial composition | |

> **If the design spec or design system is missing for a non-trivial page:** stop,
> flag it, and direct the user to `design-spec` / `design-system` first. Never invent
> tokens, spacing, colours, or breakpoints silently.

---

## 0. Scope Guard — Confirm Before Touching Non-HTML/CSS Code

This skill's authority covers **markup (HTML/JSX/template syntax) and stylesheets
only**. In projects that mix markup with logic (React/Vue/Svelte components,
templating engines with control flow, etc.), stay inside that boundary.

### Stays in scope — no confirmation needed
- HTML tags, attributes, structure, and JSX/template markup that doesn't change
  component logic (adding/restructuring elements, changing `class`/`className`).
- CSS/SCSS/CSS Modules/styled-components *style blocks*.
- Design tokens and CSS custom properties.
- **Using existing JS/TS functions** — wiring an already-defined handler, util, or
  callback onto an element (`onClick={existingHandler}`) without writing or
  modifying its implementation.
- **Creating new JSX/template components made of plain HTML** — a new component
  file containing only markup and styling (no state, hooks, effects, or new logic).

### Requires stop + confirmation
Before editing or creating any of the following, **stop before making the edit**,
emit the warning below, and wait for explicit user confirmation:
- JS/TS logic: component state, hooks, props/types, event handlers, data fetching,
  business logic, utility functions.
- Any import/export change to a module's JS/TS surface — even if it's only needed
  to satisfy a markup/layout change (e.g. adding a prop to support a new layout).
- Build/config files (`vite.config.*`, `tailwind.config.*`, `package.json`, etc.).
- Backend templating *logic* (loops/conditionals that change data flow), as opposed
  to markup/classes inside template tags.

### The warning

Output exactly this, filled in, before making the edit:

```
🟡🟡🟡 =========================================== 🟡🟡🟡
   ⚠️  WARNING — OUTSIDE ui-development SCOPE  ⚠️
🟡🟡🟡 =========================================== 🟡🟡🟡

This change touches <file path(s)> (<concern, e.g. "React component logic —
adds a useState hook">), which is outside this skill's scope (HTML/JSX markup
and CSS only).

WHAT I'M ABOUT TO DO:
  - <specific change 1>
  - <specific change 2>

WHY IT'S NEEDED:
  <one-sentence reason>

🟡 Reply "yes" to proceed, "no" to skip it, or hand off to
   `frontend-development` for the logic change. 🟡
```

Do not proceed with the out-of-scope edit until the user responds. If declined,
implement the HTML/CSS-only parts and leave the rest as a handoff note for
`frontend-development`.

---

## 1. Before Writing Any Markup

### Input Resolution

1. Check `docs/feature/<slug>/` for a tech breakdown, design spec, wireframes, and
   design system reference — load all of them, plus anything they link to.
2. If none exist, ask: *"Is there a design spec, wireframe, or design system for
   this page?"* If yes, load it.
3. If neither exists for a non-trivial page, tell the user there's no spec and offer
   to run `design-system` / `design-spec` first, or proceed with explicit assumptions.

If proceeding with assumptions, **declare them** at the top of your response before
any code.

> **One task at a time.** If a tech breakdown exists (`docs/feature/<slug>/tech-breakdown.md`
> or `tech-breakdown-<story-id>.md`), implement the **first unimplemented task** only —
> markup, styles, responsive states, accessibility checklist. After step 10 of the
> workflow below passes, stop and report: *"Task [ID] — [title] done. Checklist
> passed. Ready for task [ID+1]: [title]?"* Wait for explicit confirmation before
> implementing the next task. Never implement multiple tasks in a single response.

### Extract Tokens & Inventory

From `design-system`:
- Colour, typography, spacing, radius, shadow, and motion tokens → map 1:1 to CSS
  custom properties (same names where possible).
- Component inventory & variants → map to template partials and class names.

From `design-spec`:
- Breakpoints (or derive from content if not specified).
- States per component (default / hover / focus / disabled / loading / error / empty).
- Copy, interactions, and accessibility notes.

---

## 2. HTML Guidelines

- Use semantic landmarks: `<header>`, `<nav>`, `<main>`, `<article>`, `<section>`,
  `<aside>`, `<footer>`. Every page has exactly one `<main>`.
- Heading hierarchy is sequential and meaningful — one `<h1>` per page, no skipped
  levels (don't pick headings for their default font size).
- `<button>` for actions, `<a>` for navigation. Never attach click handlers to
  `<div>`/`<span>` — they're invisible to keyboard and screen readers.
- Every form input has an associated `<label>` (via `for`/`id` or wrapping). Group
  related inputs with `<fieldset>`/`<legend>`. Use correct `type` and `autocomplete`
  attributes.
- Images: meaningful `alt` text for content images, `alt=""` for decorative ones.
  Use `<picture>`/`srcset`/`sizes` for responsive images, not CSS background-image
  for content images.
- ARIA fills gaps native HTML doesn't cover — it never replaces correct semantics.
  If you reach for `role="button"`, ask whether `<button>` would just work.

---

## 3. CSS Guidelines — Modern CSS

### Organisation

- Use `@layer` to make the cascade explicit and predictable:

```css
@layer reset, tokens, base, components, utilities;
```

- **tokens**: design tokens as custom properties on `:root`, named to mirror
  `design-system` exactly (e.g. `--color-brand-primary`, `--space-4`,
  `--font-size-lg`, `--radius-md`).
- **base**: element defaults (typography, body, links).
- **components**: one block per component, scoped to a single class.
- **utilities**: small single-purpose helpers (`.sr-only`, `.visually-hidden`).

### Layout

- **Grid** for two-dimensional / page-level layout (page shell, card grids, form
  layouts with aligned columns).
- **Flexbox** for one-dimensional component layout (toolbars, button groups, nav
  items).
- Avoid floats and absolute-positioning layout hacks — Grid/Flexbox cover them.

### Responsive & Fluid

- Mobile-first: write the base (smallest) layout first, add complexity with
  `min-width` media queries.
- Use **container queries** (`@container`) for components whose layout should
  respond to their container, not the viewport (cards in a sidebar vs. main column).
- Use `clamp(min, preferred, max)` for fluid typography and spacing scales instead
  of stepping every value at every breakpoint.
- Use **logical properties** (`margin-inline`, `padding-block`, `inset-inline-start`)
  instead of physical ones (`margin-left`, `top`) so layouts hold up under different
  writing modes / RTL.

### Animations

- **CSS/HTML before JS, always.** Reach for `transition`, `@keyframes`,
  `animation`, `:hover`/`:focus-visible`/`:checked`/`:target`, `@starting-style`,
  and the View Transitions API first.
- Only fall back to JS-driven animation when the effect genuinely cannot be
  expressed in CSS (e.g. sequencing tied to async app state) — and that falls
  under the scope guard above (confirm first).

### Selectors & Specificity

- `:is()` / `:where()` to deduplicate selector lists without raising specificity
  (`:where()` for zero-specificity grouping, `:is()` when the group should still
  carry specificity).
- `:has()` for parent/sibling-based state styling (e.g. style a `<label>` based on
  its `<input>`'s state) instead of JS class toggling where purely presentational.
- `:focus-visible` (not bare `:focus`) for keyboard focus rings — never remove focus
  outlines without a replacement that's at least as visible.
- No ID selectors for styling. No `!important` except a documented, isolated
  override with a comment explaining why.
- One class per element for component styling; utility classes are the documented
  exception.

### Naming Convention

Pick one and apply it consistently across the project (document the choice if not
already documented):
- **BEM** (`.card`, `.card__title`, `.card--featured`), or
- **Utility-first** within the `utilities` layer for spacing/layout helpers, with
  BEM-style component classes for anything stateful.

---

## 4. File & Folder Structure

```
templates/
  pages/
    home.html
    product.html
  partials/
    header.html
    footer.html
    card.html
styles/
  tokens.css         ← :root custom properties, mirrors design-system
  reset.css
  base.css
  components/
    card.css
    nav.css
  utilities.css
  main.css           ← @layer order + @import of the above
assets/
  images/
  fonts/
```

- One stylesheet per component, named after the component.
- `tokens.css` is the single source of truth for values — components reference
  tokens, they don't redeclare raw values.

---

## 5. Responsive & Accessibility Checklist

- [ ] Layout works from ~320px wide up to large desktop without horizontal scroll.
- [ ] Text remains readable (no clipping/overlap) at 200% browser zoom.
- [ ] Every interactive element is reachable and operable via keyboard alone, in a
      logical tab order.
- [ ] Focus is visible at every step (`:focus-visible` styling present).
- [ ] Colour contrast meets WCAG AA (4.5:1 body text, 3:1 large text/icons).
- [ ] All states from the design spec are implemented (hover, focus, disabled,
      loading, error, empty) — not just the default state.
- [ ] Reduced motion respected: wrap non-essential animation in
      `@media (prefers-reduced-motion: no-preference)`.

---

## 6. PR Review Checklist

**Correctness**
- [ ] Does the markup render correctly for every state in the design spec?
- [ ] Are all breakpoints from the spec (or content-derived ones) handled?
- [ ] Does every interactive element work via keyboard, not just mouse?

**HTML**
- [ ] Semantic landmarks and correct heading hierarchy?
- [ ] Forms: labels, fieldsets, correct input types?
- [ ] No click handlers on non-interactive elements?

**CSS**
- [ ] All values come from tokens (`tokens.css`) — no hardcoded colours, spacing,
      or font sizes that duplicate a token?
- [ ] No ID selectors, no undocumented `!important`?
- [ ] Layout uses Grid/Flexbox appropriately, no positioning hacks?
- [ ] `@layer` order respected — new rules placed in the correct layer?

**Clarity**
- [ ] Class names reveal intent (component/part/modifier), no `div1`/`wrapper2`
      style names?
- [ ] Dead/commented-out markup or styles removed?

---

## 7. Implementation Workflow

For any non-trivial page or template, follow this sequence. Do not skip steps.

```
1. Read design spec & design system            → extract tokens, component inventory, states, breakpoints
2. Declare assumptions (if any)                 → state them before writing any code
3. Define/update tokens.css                     → map design tokens to CSS custom properties
4. Build semantic HTML structure (no styling)   → landmarks, heading order, forms, content
5. Compose partials                             → break the page into reusable template fragments
6. Apply layout (Grid/Flexbox, mobile-first)    → page shell first, then components
7. Style components                             → one stylesheet per component, token-driven
8. Add responsive refinements                   → container queries, clamp(), breakpoints
9. Implement all states from the spec           → hover, focus, disabled, loading, error, empty
10. Run the accessibility & responsive checklist → fix findings before presenting the code
11. Commit — load the `git` skill and commit only the files produced by this task:
      git add <files changed in this task only — never git add -A or git add .>
      git commit -m "(type)(slug): message"
    Verify clean state with git status after committing.
12. STOP — report completion and wait:
      "Task [ID] — [title] done. Checklist ✅. Committed.
       Files changed: [list].
       Next task: [ID+1] — [title]. Ready to proceed?"
    Do not implement the next task until the user confirms.
```

At step 4–7, if you discover the design spec is missing something significant (an
undocumented state, a missing breakpoint, a conflicting token), **stop and surface
it** before continuing. Do not invent visual decisions silently.

---

## 8. What This Skill Does NOT Cover

| Concern | Covered by |
|---|---|
| Visual design, colour, spacing, typography decisions | `design-system`, `design-spec` |
| Wireframes and user flows | `wireframe` |
| Component states, interactions, copy, edge cases | `design-spec` |
| Framework component implementation (React/Vue/etc.) | `frontend-development` |
| JS interactivity, state management | `frontend-development`, `frontend-guidelines` |
| DDD building blocks & SOLID rules | `base-guidelines` |
| Frontend test patterns | `frontend-testing` |
