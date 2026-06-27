# Component: Profile

A search persona that scopes every job/dashboard/browser view and drives the scraper's
board-side filtering. Exactly one profile is active at a time.

## Properties

| Property       | Type      | Description                                          |
|----------------|-----------|------------------------------------------------------|
| ID             | ProfileID | Stable identifier assigned by the repository.        |
| Name           | string    | Human-readable persona name (required, non-empty).   |
| SearchKeywords | []string  | Keywords pushed into each board's search query.      |
| Location       | string    | Geographic search target.                            |
| IsActive       | bool      | True for the single active profile; false otherwise. |

## Exactly-one-active invariant

The `profile_single_active` partial unique index on `(is_active) WHERE is_active = true`
enforces at most one active profile at the database level. `Repository.Activate` switches
the active profile atomically: first a blanket deactivation, then activation of the target.

## API

| Method | Path                        | Description                              |
|--------|-----------------------------|------------------------------------------|
| GET    | /api/profiles               | List all profiles.                       |
| POST   | /api/profiles               | Create a new inactive profile.           |
| GET    | /api/profiles/{id}          | Get one profile.                         |
| PUT    | /api/profiles/{id}          | Update name, keywords, location.         |
| DELETE | /api/profiles/{id}          | Delete profile.                          |
| GET    | /api/active-profile         | Get the single active profile.           |
| PUT    | /api/active-profile         | Switch active profile (body: profile_id).|

## Notes

- `POST /api/profiles` always creates an **inactive** profile; use `PUT /api/active-profile` to switch.
- Identity, conditions, and fit_weights sub-resources are added in later tasks (P3-PR-2…6).
