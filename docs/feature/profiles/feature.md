# F2: Unified Profiles (identity + search + conditions)

**Purpose:** Define who you are, what you're looking for, and what makes a job a match.

A Profile merges identity + search + matching conditions. One LinkedIn PDF (one identity snapshot) can spawn multiple profiles differing only in search target + conditions.

- Profile A: "Go Backend Paris — CDI 65k+"
- Profile B: "Go Backend Remote Europe — Freelance 500€/day"

Both share imported skills/seniority; differ in search keywords, location, dealbreakers, preferences, fit-score weights. One profile **active** at a time. Fast switch (dropdown/tabs). Switching re-scopes Dashboard, Job Browser, scraper target.

## Identity (from LinkedIn PDF import)
- Import LinkedIn PDF export → auto-extract: skills (flat list, no self-rating), experience, seniority.
- Manual add/remove/edit skills after import.
- Re-import behavior TBD (overwrite vs merge) — decide during build.

## Search config (per profile)
- Keywords (free text, e.g. "backend golang engineer").
- Location: city / country / region / remote scope (e.g. Paris, France, Remote Europe).

## Conditions (per profile)

### Dealbreakers (hard filters — failing job hidden/flagged, never in top matches)
- Contract type (e.g. CDI only).
- Remote policy (e.g. required remote).
- Min salary (annual gross).
- Required skills (must have).

### Preferences (soft scoring)
- Preferred skills (nice to have).
- Max office days per week.
- Location preference (e.g. Paris preferred, France OK).
- Working days (e.g. full-time preferred).

## Fit-score weights (per profile, user-configurable)
User tunes weights in UI. Starting defaults (soft components sum = 100%):

| Component | Default weight | Type |
|-----------|---------------|------|
| Dealbreaker skills | pass/fail | gate |
| Dealbreaker contract type | pass/fail | gate |
| Dealbreaker remote policy | pass/fail | gate |
| Dealbreaker min salary | pass/fail | gate |
| Preferred skills match % | 40% | score |
| Salary vs minimum | 25% | score |
| Location preference | 15% | score |
| Office days vs max | 10% | score |
| Working days match | 10% | score |

Dealbreakers gate first; preferences produce the weighted score.
