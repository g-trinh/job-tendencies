# Component: Board

A job board source that Job Tendencies can scrape. Each board has a declarative
scraping adapter (draft → approved lifecycle). Exactly one approved adapter per board.

## Properties

| Property | Type    | Description                                                    |
|----------|---------|----------------------------------------------------------------|
| ID       | BoardID | Stable identifier.                                             |
| Name     | string  | Human-readable board name (required, non-empty).               |
| BaseURL  | string  | Board's public base URL (required, non-empty).                 |
| Enabled  | bool    | When false the board is excluded from scrape runs.             |

## Seeded boards

Four boards are seeded with stable IDs:

| ID suffix | Name                  | Base URL                             |
|-----------|-----------------------|--------------------------------------|
| …0001     | Welcome to the Jungle | https://www.welcometothejungle.com   |
| …0002     | Indeed                | https://www.indeed.com               |
| …0003     | LinkedIn              | https://www.linkedin.com/jobs        |
| …0004     | Glassdoor             | https://www.glassdoor.com/Job        |

## API

| Method | Path             | Description                                       |
|--------|------------------|---------------------------------------------------|
| GET    | /api/boards      | List all boards with their approved adapter.      |
| POST   | /api/boards      | Create a new board (enabled by default).          |
| PUT    | /api/boards/{id} | Update name, base_url, enabled (includes toggle). |
| DELETE | /api/boards/{id} | Delete board.                                     |

## Notes

- Disabling all boards is allowed; the UI warns the user when no boards are enabled.
- Adapter generation (P3-BO-2..4) and the global schedule (P3-BO-5) are later tasks.
