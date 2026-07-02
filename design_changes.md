# Design Changes — Phase 6 static template

Static HTML/CSS design reference for Job Tendencies, built in `template/`. This is
the **design source of truth** for the follow-up visual pass on the React app. The
frontend team builds features in parallel; use this document to align markup
structure, tokens, component classes, and interaction decisions.

- **Pure HTML5 + modern CSS** (Grid, Flexbox, custom properties, `clamp()` fluid
  type, `@layer` cascade, `:focus-visible`, logical properties). No framework, no JS.
- **All UI copy in French.** Raw job listing text would render verbatim (never
  translated); enums/labels are French (CDI, Freelance, Remote, etc.).
- **Accessibility:** one `<main>`-equivalent region per screen, sequential headings,
  labelled inputs, `:focus-visible` rings via `--focus-ring`, WCAG-AA contrast,
  `prefers-reduced-motion` guards on the spinner/skeleton/shimmer.

## File tree

```
template/
  index.html                 # P6-UI-1 — design system + component inventory reference
  screens/
    dashboard.html           # F5
    job-browser.html         # F4
    profiles.html            # F2
    boards.html              # F1
    contacts.html            # F6
    pipeline.html            # pipeline run trigger + per-board status
  styles/
    main.css                 # @layer order + @import of the below
    tokens.css               # design tokens → CSS custom properties (:root)
    base.css                 # reset, element base, utilities, app shell (sidebar/topbar/page)
    components.css           # component inventory (one block per component)
```

## Design tokens (`styles/tokens.css`)

Mirror these names 1:1 in the React theme. Groups:

- **Color** — neutrals (`--color-bg/surface/surface-alt/surface-sunken/border/
  border-strong`), text (`--color-text/-muted/-subtle/-inverse`), brand
  (`--color-brand/-hover/-active/-soft/-ring`), semantic status
  (`success/warning/danger/info` each with a `-soft` companion), kanban stages
  (`--color-stage-saved/-applied/-interview/-offer/-rejected`), confidence scale
  (`--color-conf-low/-mid/-high`).
- **Typography** — `--font-sans`, `--font-mono`; sizes `xs → 2xl` (`lg/xl/2xl` are
  `clamp()` fluid); weights `regular/medium/semibold/bold`; line-heights
  `tight/snug/normal`.
- **Spacing** — `--space-1 … --space-8` (4px base).
- **Radii** — `sm/md/lg/xl/pill`. **Shadows** — `xs/sm/md/lg`.
- **Motion** — `--ease-standard`, `--duration-fast/base`.
- **Layout** — `--sidebar-width`, `--content-max`, `--focus-ring`.

## Component inventory (`styles/components.css`) — rendered on `index.html`

Buttons (`.btn` + `--primary/--secondary/--ghost/--danger/--sm/--loading`,
`:disabled`) · form fields (`.field`, `.input`, `.select`, `.textarea`,
`aria-invalid` error state, `.field__error/__hint`) · cards (`.card`, `.stat`) ·
badges (`.badge--neutral/brand/success/warning/danger/info`, `.badge__dot`) ·
confidence badge (`.conf.conf--low/mid/high`) · tags + tag-input (`.tag`,
`.tag-input`) · tables (`.table-wrap`, `.table`, `.th-sort[aria-sort]`) · tabs
(`role=tablist`/`.tab[aria-selected]`), segmented control (`.segmented
button[aria-pressed]`) · sliders (`.slider`, `.slider-row`) + sum-to-100 feedback
(`.weights-sum--ok/--off`) · toggle switch (`.toggle` + `.toggle__track`, native
checkbox) · modal (`.scrim`/`.modal`) & drawer (`.drawer`) · kanban column
(`.kanban`, `.kanban__col`, `.kanban__card`, `.kanban__stage-dot`) · banners
(`.banner--warning/danger/info/success`) · state blocks (`.state`,
`.skeleton`/shimmer, error banner) · progress (`.progress`/`.progress__bar`) ·
CSS bar chart (`.barchart`) and inline-SVG line chart (`.linechart`) placeholders ·
layout grids (`.grid-stats/-cards/-2`, `.layout-with-panel`) · filter panel
(`.filter-group`, `.check`) · job card (`.jobcard`, `.fit-score`).

Every screen ships **default + empty + loading + error** states.

## Screens → components / tokens / interaction notes

### Dashboard — `screens/dashboard.html` (F5)
- **Components:** stat cards (`.grid-stats` / `.stat` with up/down deltas), CSS
  `.barchart` (skills frequency), SVG `.linechart` (skills trend), match-alert rows
  (`.fit-score` circle + `.conf` badge), profile switcher in topbar.
- **Interaction notes:** charts are static placeholders — frontend renders them with
  **Recharts** (P6-FE-5). All sections scoped to the active profile (topbar `<select>`).
- **States:** empty ("Pas encore de données" + launch CTA), skeleton, error banner.

### Job Browser — `screens/job-browser.html` (F4)
- **Components:** view-mode `.segmented` (Tableau / Cartes / Kanban), sort `<select>`,
  `.layout-with-panel` filter sidebar (`.filter-group` + `.check`, filters only — no
  free-text search per spec), `.table` with `.th-sort`, `.jobcard` card grid, `.kanban`
  five-column board, `.conf` confidence + understanding badges, "Trouvée sur : …"
  dedup line, `.badge--danger` "Expirée" + `.jobcard--expired` dimming, original-link
  `<a rel="noreferrer">`.
- **DESIGN DECISIONS (Open Questions from tech breakdown — frontend chose the simpler
  options):**
  1. **Kanban = status `<select>` dropdown, not drag-and-drop.** Each row/card exposes
     a status `<select>` (Sauvegardée → Postulé → Entretien → Offre → Rejetée). The
     kanban board is a read/visual grouping; status changes happen via the dropdown.
     Frontend wires this as an optimistic mutation (P6-FE-4).
  2. **Single global confidence slider, not per-field thresholds.** One
     `.slider` ("Seuil de confiance minimum") above the results filters the whole list.
     Per-field confidence is display-only (badges on each job).
- **States:** empty (no match + reset filters), skeleton rows, error banner.

### Profiles — `screens/profiles.html` (F2)
- **Components:** profile switcher + "Actif" badge in topbar, `.tabs`
  (Identité / Recherche / Conditions / Poids), PDF import (`<input type=file>` styled
  as `.btn` label), skills `.tag-input` editor, `<fieldset>` groups for dealbreakers
  vs preferences, weight `.slider-row` set + `.weights-sum--ok/--off` sum-to-100
  feedback.
- **Interaction notes:** weights show a **soft warning** when the sum ≠ 100% (does not
  block save) — `.weights-sum--off`. Dealbreakers gate; preferences feed the weighted
  score. Skills list is flat (no self-rating).
- **States:** empty (no profile → import PDF), loading (PDF extraction skeleton),
  error (import failed).

### Boards — `screens/boards.html` (F1)
- **Components:** global schedule card (`<select>` + `.toggle`), CRUD `.table` with
  per-row enabled `.toggle` and adapter-status badges (Prêt / Brouillon à valider /
  Aucun), 3-step adapter flow (`generate → review draft → approve`) with read-only
  `<pre>` adapter draft, warning `.banner--warning` when all boards disabled.
- **Interaction notes:** one global schedule (not per-board). User never hand-writes
  adapter code — the draft `<pre>` is read-only; actions are Régénérer / Rejeter /
  Approuver. "Générer l'adaptateur" appears on boards without one.
- **States:** empty (no sources), loading (adapter generation skeleton), error.

### Contacts — `screens/contacts.html` (F6)
- **Components:** CSV export `.btn`, tag filter `<select>`, `.table` (name, company,
  contact, jobs posted, tags, notes), `.tag` chips, edit drawer preview
  (`.drawer`-style card with tag-input + notes textarea), "Incomplet" badge for Easy
  Apply / masked-email contacts.
- **Interaction notes:** contacts auto-populate from extraction (dedup by email /
  LinkedIn). Manual add/edit via the drawer. Tags: `in-house`/`agency`,
  `responsive`/`ghosted`/`not-contacted`, plus custom.
- **States:** empty (no contacts), skeleton, error banner.

### Pipeline — `screens/pipeline.html`
- **Components:** run-trigger `.btn` in topbar, current-run card with **per-board**
  `.progress` bars + status badges (Terminé / Extraction… / En attente / Échec),
  run-history `.table` (trigger = Planifié/Manuel, statut = Réussi/Partiel/Échec).
- **Interaction notes:** frontend polls `GET /api/pipeline/runs` via TanStack Query
  (P6-FE-7); progress bars and badges update live until completion. Per-board failure
  (e.g. rate-limit) is shown with `.progress__bar--danger` without failing the whole run.
- **States:** empty (no runs), loading (polling skeleton), error (status fetch failed,
  auto-retry).

## Notes for the frontend visual pass

- Class names are BEM-ish; port the **token values** first, then match component
  structure. Do not invent tokens not present in `tokens.css` — flag gaps instead.
- The sidebar/topbar/page shell (`.app`, `.app__sidebar`, `.nav`, `.topbar`, `.page`)
  is duplicated per static screen for standalone viewing; in React it becomes one
  layout component.
- Charts (`.barchart`, `.linechart`) are token-driven placeholders — replace with
  Recharts but keep the same color tokens (`--color-brand`, `--color-brand-soft`).
- Responsive: shell collapses to single column under 720px; filter panel stacks under
  860px; kanban scrolls horizontally.
