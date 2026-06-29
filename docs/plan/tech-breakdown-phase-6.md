## Tech Breakdown: Phase 6 — Frontend

**Design spec ref:** docs/v0.md, template/ (static reference, built here)
**Architecture ref:** overview.md §7 (frontend architecture), §6 (API surface), §9 (language)
**Feature ref:** all of docs/feature/*/feature.md
**Plan ref:** docs/plan/development-plan.md (Phase 6)
**Teams:** UI (ui-integrator), Frontend

`template/` is built early (parallel from Phase 1, no backend dep). React features wire once
each context's dev API exists. All UI French; raw listing text verbatim.

---

### Tasks

---

#### P6-UI-1 — Define design system + static component inventory in template/

**Type:** Chore · **Owner:** UI · **Dependencies:** —

**Description:** Tokens (color, type, spacing) + component inventory as static HTML/CSS in
`template/`. Does not touch `frontend/`.
**Refs:** overview.md §7, frontend-design guidance
**Acceptance Criteria:** `template/` renders the token set + base components.

#### P6-UI-2 — Build static HTML/CSS screens for all six features + pipeline

**Type:** Chore · **Owner:** UI · **Dependencies:** P6-UI-1

**Description:** Static screens (French) for profiles, boards, job browser, dashboard,
contacts, pipeline run/status → drives `design_changes.md`.
**Refs:** all feature.md, v0.md (UI surfaces)
**Acceptance Criteria:** Each feature has a static reference screen incl. empty/loading/error states.

---

#### P6-FE-1 — Build the app shell (provider, i18n, fetch wrapper, charts)

**Type:** Feature · **Owner:** Frontend · **Dependencies:** P0-8, P3-PR-1

**Description:** Full `ActiveProfileProvider`; `X-Active-Profile` injection; active-profile in
every Query cache key; `setActiveProfile` PUTs `/api/active-profile`; French i18n dict;
Recharts setup; routing per feature.
**Refs:** overview.md §7, §6 (active-profile)
**Acceptance Criteria:**
- Switching profile re-scopes all server state (cache keys include the id).
- Enums render French; raw text verbatim.

#### P6-FE-2 — Build the Profiles UI

**Type:** Feature · **Owner:** Frontend · **Dependencies:** P6-FE-1, P3-PR-1…6

**Description:** PDF import, skills editor, search config, conditions, weights sliders
(sum-to-100 feedback), profile switcher.
**Refs:** profiles/feature.md, overview.md §6 (Profiles)
**Acceptance Criteria:** Create/edit/activate a profile; import a PDF; weights warn unless soft sum=100%.

#### P6-FE-3 — Build the Boards UI (incl. adapter review/approve)

**Type:** Feature · **Owner:** Frontend · **Dependencies:** P6-FE-1, P3-BO-1…5

**Description:** Board CRUD, enabled toggles (warn if all disabled), schedule editor, adapter
generate → review draft → approve.
**Refs:** board-manager/feature.md, overview.md §6 (Boards)
**Acceptance Criteria:** Generate, review, and approve an adapter; UI warns when all boards disabled.

#### P6-FE-4 — Build the Job Browser

**Type:** Feature · **Owner:** Frontend · **Dependencies:** P6-FE-1, P3-JO-1…5

**Description:** Table + card modes; filter-only panel; sort; kanban
(Saved→Applied→Interview→Offer→Rejected) with optimistic updates; confidence/understanding
badges; "found on: …"; expired marker; original link.
**Refs:** job-browser/feature.md, overview.md §6/§7 (optimistic kanban)
**Acceptance Criteria:** Filters/sort work; kanban drag persists optimistically; badges + "found on" + expired render.

#### P6-FE-5 — Build the Dashboard

**Type:** Feature · **Owner:** Frontend · **Dependencies:** P6-FE-1, P3-DA-1…4

**Description:** Skills frequency bar, skills trend line, match alerts, stats cards (Recharts).
**Refs:** dashboard/feature.md, overview.md §6 (Dashboard)
**Acceptance Criteria:** All four dashboard sections render from the dev API, scoped to active profile.

#### P6-FE-6 — Build the Contacts UI

**Type:** Feature · **Owner:** Frontend · **Dependencies:** P6-FE-1, P3-CO-1…3

**Description:** Contacts table, tags, notes, manual add/edit, CSV export download.
**Refs:** contacts-crm/feature.md, overview.md §6 (Contacts)
**Acceptance Criteria:** Edit tags/notes; add a contact; download CSV.

#### P6-FE-7 — Build pipeline trigger + run-status polling

**Type:** Feature · **Owner:** Frontend · **Dependencies:** P6-FE-1, P2-BE-6, P3-SCR-5

**Description:** On-demand run button + run status via TanStack Query polling of
`GET /api/pipeline/runs`.
**Refs:** overview.md §6/§7 (polling pipeline runs), pipeline.md §6
**Acceptance Criteria:** Triggering a run shows live per-board progress until completion.

---

### Dependency Graph

```
P6-UI-1 → P6-UI-2   (parallel to everything, from Phase 1)

P0-8 ─┐
P3-PR-1 ┴→ P6-FE-1 ─┬→ P6-FE-2 (needs profiles API)
                    ├→ P6-FE-3 (needs boards API)
                    ├→ P6-FE-4 (needs jobs API)
                    ├→ P6-FE-5 (needs dashboard API)
                    ├→ P6-FE-6 (needs contacts API)
                    └→ P6-FE-7 (needs pipeline API)
```

### Parallel tracks

- `template/` track (P6-UI-*) runs independently of all backend work.
- After P6-FE-1, the six feature UIs build in parallel as their APIs land.

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | Kanban interaction: drag-and-drop vs status dropdown | P6-FE-4 | Design |
| 2 | Confidence-threshold control: per-field vs single global slider | P6-FE-4 | Design/PM |
