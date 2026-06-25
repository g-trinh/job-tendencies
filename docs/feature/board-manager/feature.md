# F1: Board Manager

**Purpose:** Manage which job boards get scraped.

## In scope
- CRUD on board sources. Each board: name, base URL, enabled/disabled toggle.
- **One global scrape schedule** for all boards (not per-board).
- Default seed boards: Welcome to the Jungle, Indeed, LinkedIn, Glassdoor.
- On-demand + scheduled runs.
- Rate limiting per board.
- UI warns if all boards disabled (= no scraping).

## Adapter generation (Option A — LLM-generated)
- User supplies board URL + example search/listing page.
- LLM analyzes HTML structure, generates scraper adapter code.
- User reviews + approves before it goes live.
- User never hand-writes adapter code.

## Out of scope (v1)
- Auto-discovering boards.
- Scraper health monitoring.
