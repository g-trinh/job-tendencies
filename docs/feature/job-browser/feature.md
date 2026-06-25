# F4: Job Browser

**Purpose:** Browse, filter, manage scraped jobs. Scoped to active profile.

## In scope
- View modes: table (dense) + card (rich).
- **Filters only** (no free-text search): skills, remote policy, contract type, salary range, location, board source, date, confidence threshold.
- Sort by: date, fit score, salary.
- Application kanban: **Saved → Applied → Interview → Offer → Rejected**.
- Each job links to original posting.
- Confidence + understanding badges on each job.

## Edge cases
- Duplicate across boards → merged into one entry, shows "found on: WTTJ, Indeed".
- Job removed from board → marked "expired", data retained.
