---
name: Design System
description: >
  Use this skill to define, audit, or extend a product's design system: documenting
  tokens (color, typography, spacing), inventorying components and their variants/states,
  establishing usage rules, and flagging gaps when a feature requires components that
  don't exist yet. Trigger on phrases like: "what components do we have", "define our
  design system", "audit our UI components", "what tokens should we use", "do we have
  a component for X", "extend the design system with". This skill is a shared foundation
  — wireframe and design-spec reference it before doing their work. Run this skill once
  per product, then update it incrementally as new components are added.
---

## Design System

Establish or extend the design system that all product design work builds upon.
This document becomes the shared reference for wireframe and design-spec skills.

### Questions phase

Ask the user these questions (all at once, numbered):

1. **Existing system** — Do you already have a design system? (Figma library, Storybook, component library like shadcn/MUI/Radix, or a custom one?) If yes, describe it briefly.
2. **Platform** — What platform(s) are you designing for? (Web, iOS, Android, or all?)
3. **Scope** — What do you need right now: audit what exists, define tokens only, document specific components, or define from scratch?
4. **Feature context** — Is there a specific feature driving this? If so, share the feature spec so I can flag gaps immediately.

Wait for the user's answers before proceeding.

---

### Steps

1. **Establish tokens** — Define or document the foundational design tokens:
   - Color: brand, semantic (success/warning/error/info), neutral scale, surface/background
   - Typography: font families, size scale, weight, line-height, letter-spacing
   - Spacing: base unit and scale (4px or 8px base recommended)
   - Radius: none / sm / md / lg / full
   - Shadows / elevation levels
   - Motion: duration scale, easing presets

2. **Inventory components** — For each component, document:
   - Name (match the codebase or design tool exactly)
   - Variants (e.g. Button: primary / secondary / ghost / destructive)
   - States (default / hover / focus / disabled / loading / error)
   - Props / configuration that affect appearance
   - Usage rules (when to use vs. not use)
   - Known accessibility requirements

3. **Identify gaps** — If a feature spec was provided, cross-reference it against the component inventory and flag every UI element that has no existing component.

4. **Propose new components** — For each gap, describe the new component needed: name, variants, states, and which existing component it most resembles (for reference).

---

### Output format

```
## Design System: <Product Name>

---

### Tokens

#### Color
| Token | Value | Usage |
|-------|-------|-------|
| color.brand.primary | #... | Primary actions, links |
| color.semantic.success | #... | Confirmation, positive state |
| color.semantic.error | #... | Errors, destructive actions |
| color.neutral.100 | #... | Backgrounds |
| ... | | |

#### Typography
| Token | Value |
|-------|-------|
| font.size.sm | 12px |
| font.size.base | 16px |
| font.size.lg | 20px |
| font.weight.regular | 400 |
| font.weight.semibold | 600 |

#### Spacing
Base unit: 4px
Scale: 4 / 8 / 12 / 16 / 24 / 32 / 48 / 64

#### Radius
none / sm (4px) / md (8px) / lg (16px) / full (9999px)

---

### Component Inventory

#### <ComponentName>
**Variants:** primary · secondary · ghost
**States:** default · hover · focus · disabled · loading
**Usage:** Use for all primary user actions. One primary button per screen section.
**Do not use:** As a navigation element — use Link instead.

#### <ComponentName>
...

---

### Gaps (feature: <Feature Name>)

| UI Element Needed | Closest Existing | Action |
|-------------------|-----------------|--------|
| Inline error tooltip | Tooltip (no error variant) | Extend Tooltip with error variant |
| Tag / chip with dismiss | — | New component: Tag |
```

---

### After output

This is a project-wide document (not feature-scoped). Ask the user: "Save this to `docs/design/design-system.md`?" Wait for confirmation, then create `docs/design/` if needed. If the file already exists, update it in place — merge new tokens/components/gaps rather than overwriting.

Commit — load the `git` skill:
```bash
git add docs/design/design-system.md
git commit -m "infra(design-system): update design system reference"
```

---

### Tips

- Tokens first, components second — components are built from tokens. Without consistent tokens, components drift.
- Name components after what they *are*, not what they look like (`StatusBadge` not `GreenPill`).
- Document the "do not use" rule for each component — it prevents misuse as much as the "use when" rule.
- If a gap requires a new component, flag whether it should be built generically (goes into the system) or as a one-off (stays in the feature).
- Keep this document in the repo (`docs/design/design-system.md`) so engineers and designers share the same source of truth.
