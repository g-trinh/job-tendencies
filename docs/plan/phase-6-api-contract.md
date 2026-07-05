# Phase 6 API Contract (Frontend-consumed endpoints)

Source of truth: `backend/internal/handler/http/*.go`. Derived directly from handler code —
no invented fields. All endpoints scoped to a profile require the `X-Active-Profile: <profileID>`
header unless noted otherwise.

Status: IN PROGRESS — being filled in section by section.

## Route table (from `backend/cmd/api/main.go`)

All routes are mounted under `/api`. `/api/auth/login` and `/api/auth/logout` are public.
All other `/api` routes require a valid session cookie (`RequireAuth`) + CSRF check
(`RequireCSRF`). Routes under the innermost `guarded.Group` (jobs, dashboard) additionally
require `X-Active-Profile` header (`RequireActiveProfileMiddleware`).

```
POST   /api/auth/login
POST   /api/auth/logout
GET    /api/auth/me

GET    /api/boards
POST   /api/boards
PUT    /api/boards/{id}
DELETE /api/boards/{id}
GET    /api/boards/{id}/adapter
POST   /api/boards/{id}/adapter/generate
POST   /api/boards/{id}/adapter/approve
GET    /api/schedule
PUT    /api/schedule

GET    /api/profiles
POST   /api/profiles
GET    /api/profiles/{id}
PUT    /api/profiles/{id}
DELETE /api/profiles/{id}
GET    /api/active-profile
PUT    /api/active-profile
PATCH  /api/profiles/{id}/identity
POST   /api/profiles/{id}/identity/import
GET    /api/profiles/{id}/conditions   (alias of GET /api/profiles/{id})
PUT    /api/profiles/{id}/conditions
GET    /api/profiles/{id}/weights      (alias of GET /api/profiles/{id})
PUT    /api/profiles/{id}/weights

GET    /api/contacts
GET    /api/contacts/export.csv
POST   /api/contacts
GET    /api/contacts/{id}
PUT    /api/contacts/{id}
DELETE /api/contacts/{id}

POST   /api/pipeline/runs
GET    /api/pipeline/runs
GET    /api/pipeline/runs/{id}

# Requires X-Active-Profile in addition to auth:
GET    /api/jobs
GET    /api/jobs/{id}
GET    /api/jobs/{id}/original
PATCH  /api/jobs/{id}/application
POST   /api/jobs/{id}/reextract
GET    /api/dashboard/skills/frequency
GET    /api/dashboard/skills/trend
GET    /api/dashboard/matches
GET    /api/dashboard/stats
```

Note: `boards`, `schedule`, `profiles`, `contacts`, `pipeline/runs` are NOT gated by
`X-Active-Profile` at the router level. Frontend should check each handler for whether
active-profile is read from context/query implicitly (see per-feature sections below).

## Table of contents
- Auth / session
- Profiles (P6-FE-2)
- Boards (P6-FE-3)
- Job Browser (P6-FE-4)
- Dashboard (P6-FE-5)
- Contacts (P6-FE-6)
- Pipeline (P6-FE-7)
- Bugs found / fixed

---

## Error envelope (applies to all endpoints)

```jsonc
{ "error": "string" }   // human-readable message, safe to display or log
```
Status mapping (`backend/internal/handler/http/errors.go`): 404 not found, 400 invalid
input/validation, 409 conflict, 401 unauthorized, 500 otherwise (detail not leaked).

## Auth / session

Session is an httpOnly `__session` cookie (AES-GCM encrypted refresh token) — never
exposed to JS. Frontend must send `credentials: "include"` on every request and handle
401 globally (redirect to login). `X-Active-Profile` is a plain header, not a cookie.

| Method | Path | Request body | Response | Notes |
|---|---|---|---|---|
| POST | `/api/auth/login` | `{"email","password"}` | 200 `{"uid","email"}` + sets cookie | 400 missing fields, 401 bad credentials |
| POST | `/api/auth/logout` | — | 200 `{"status":"logged out"}` + clears cookie | |
| GET | `/api/auth/me` | — | 200 `{"uid","email"}` | 401 if no/invalid session |

All `guarded` routes (everything except login/logout) require: valid session cookie
(`RequireAuth`) + CSRF header (`RequireCSRF` — check `middleware.go` for exact header
name if implementing CSRF token flow; frontend must fetch/send it, not documented here
as it's outside Phase 6 feature scope, already built in Phase 4).

---

## Profiles (P6-FE-2)

Source: `backend/internal/handler/http/profiles.go`. All routes require auth (session cookie
+ CSRF); NOT gated by `X-Active-Profile` header (profiles endpoints operate on explicit
`{id}` path params, or resolve "the" active profile server-side for `/active-profile`).

### Shared shape: `profileResponse`
```jsonc
{
  "id": "string",
  "name": "string",
  "search_keywords": ["string"],       // never null, [] if empty
  "location": "string",
  "is_active": true,
  "skills": ["string"],                // never null, [] if empty
  "seniority": "string",               // raw kernel.Seniority verbatim (not French-mapped by backend)
  "raw_experience": "string",
  "conditions": {
    "dealbreaker_contract_type": "string|null",   // raw kernel.ContractType
    "dealbreaker_remote_policy": "string|null",   // raw kernel.RemotePolicy
    "dealbreaker_salary_min": 0,                  // int64|null
    "dealbreaker_required_skills": ["string"],    // never null
    "preferred_skills": ["string"],               // never null
    "preferred_max_office_days": 0,               // int|null
    "preferred_location": "string",
    "preferred_working_days": "string"            // raw kernel.WorkingDays
  },
  "weights": {
    "preferred_skills": 0,
    "salary": 0,
    "location": 0,
    "office_days": 0,
    "working_days": 0
  }
}
```
Enum fields (`seniority`, `dealbreaker_contract_type`, `dealbreaker_remote_policy`,
`preferred_working_days`) are raw backend enum strings — frontend i18n dict must map
these to French labels; they are not pre-translated by the backend.

### Endpoints

| Method | Path | Request body | Response |
|---|---|---|---|
| GET | `/api/active-profile` | — | 200 `profileResponse` |
| PUT | `/api/active-profile` | `{"profile_id":"string"}` (400 if missing) | 200 `profileResponse` |
| GET | `/api/profiles` | — | 200 `profileResponse[]` |
| POST | `/api/profiles` | `{"name","search_keywords":[string],"location"}` | 201 `profileResponse` |
| GET | `/api/profiles/{id}` | — | 200 `profileResponse` |
| PUT | `/api/profiles/{id}` | same as POST body | 200 `profileResponse` |
| DELETE | `/api/profiles/{id}` | — | 204 no body |
| PATCH | `/api/profiles/{id}/identity` | `{"skills":[string],"seniority":"string"}` | 200 `profileResponse` |
| POST | `/api/profiles/{id}/identity/import` | `multipart/form-data`, field name `file` (PDF); 400 if missing/empty; 409 if identity already populated (single-import guard) | 200 `profileResponse` |
| GET | `/api/profiles/{id}/conditions` | — (alias of GET profile) | 200 `profileResponse` (full profile, not just conditions) |
| PUT | `/api/profiles/{id}/conditions` | see `putConditionsRequest` below | 200 `profileResponse` |
| GET | `/api/profiles/{id}/weights` | — (alias of GET profile) | 200 `profileResponse` (full profile, not just weights) |
| PUT | `/api/profiles/{id}/weights` | `{"preferred_skills","salary","location","office_days","working_days"}` all `int` | 200 `profileResponse` |

`putConditionsRequest` body:
```jsonc
{
  "dealbreaker_contract_type": "string|null",
  "dealbreaker_remote_policy": "string|null",
  "dealbreaker_salary_min": 0,
  "dealbreaker_required_skills": ["string"],
  "preferred_skills": ["string"],
  "preferred_max_office_days": 0,
  "preferred_location": "string",
  "preferred_working_days": "string"
}
```

Notes for frontend:
- The GET conditions/weights aliases return the **full** profile object, not a scoped
  subset — frontend must read `.conditions`/`.weights` off the full response.
- Weights sum-to-100 validation: handler does NOT itself reject non-100 sums in the code
  read so far — the "sum-to-100 soft warning" from the AC is a frontend-side UX check
  unless `svc.UpdateWeights` enforces it (see app/profiles service — not read here;
  treat as advisory, verify no 4xx surprises during integration).
- Errors use the standard `RespondError` envelope (see Auth section below for shape).

---

## Boards (P6-FE-3)

Source: `backend/internal/handler/http/boards.go`. All routes require auth only (no
`X-Active-Profile`) — boards/schedule are global, not per-profile.

### Shared shape: `boardResponse`
```jsonc
{
  "id": "string",
  "name": "string",
  "base_url": "string",
  "enabled": true,
  "adapter": { /* adapterResponse */ } | null   // null if board has no approved adapter yet
}
```

### `adapterResponse`
```jsonc
{
  "id": "string",
  "status": "string",      // raw boards.AdapterStatus (e.g. draft/approved) — verbatim, not French
  "fetch_mode": "string",  // raw llm.FetchMode
  "version": 0,
  "spec": {                 // llm.AdapterSpec — verbatim technical config, always raw text (never French)
    "board": "string",
    "fetch_mode": "string",
    "search": { "url_template", "method", "body_template", "param_map": {}, "pagination": {...}, "result_node_path", "result_fields": {} },
    "listing": { /* board-specific listing selectors */ },
    "incremental": { /* incremental-crawl config */ },
    "rate_per_second": 0.0
  }
}
```
Full nested `AdapterSpec` field list: see `backend/internal/domain/llm/adapter_spec.go`.
Frontend should render this as a read-only JSON/code viewer during adapter review — do
not attempt to build bespoke form fields for every nested key without confirming with
architect first.

### `scheduleResponse`
```jsonc
{ "cron": "string" }   // raw cron expression, no French rendering
```

### Endpoints

| Method | Path | Request body | Response |
|---|---|---|---|
| GET | `/api/boards` | — | 200 `boardResponse[]` |
| POST | `/api/boards` | `{"name","base_url","enabled":bool|null}` (enabled ignored on create — service defaults new boards to enabled) | 201 `boardResponse` (`adapter` is `null`) |
| PUT | `/api/boards/{id}` | `{"name","base_url","enabled":bool|null}` (`enabled` defaults to `true` when omitted/null — **frontend must always send explicit `enabled` on toggle**) | 200 `boardResponse` (`adapter` is `null` in this response even if the board has one — see bug note) |
| DELETE | `/api/boards/{id}` | — | 204 no body |
| GET | `/api/boards/{id}/adapter` | — | 200 `adapterResponse` (most recent: draft or approved) |
| POST | `/api/boards/{id}/adapter/generate` | `{"example_response":"string"}` (raw HTML/JSON of a sample search/listing page) | 201 `adapterResponse` (new draft) |
| POST | `/api/boards/{id}/adapter/approve` | — | 200 `adapterResponse` (promotes latest draft to approved; 400 with field errors if spec invalid) |
| GET | `/api/schedule` | — | 200 `scheduleResponse` |
| PUT | `/api/schedule` | `{"cron":"string"}` | 200 `scheduleResponse` |

Notes for frontend:
- "Warn if all boards disabled" (AC) is a frontend-only concern — compute from
  `GET /api/boards` list, no dedicated backend endpoint.
- Adapter generate → review → approve is 3 calls: `POST generate` to create a draft
  (or `GET adapter` to fetch the latest one), render for review, then `POST approve`
  with no body to promote it.
- Note: `PostBoard`/`PutBoard` responses always have `adapter: null` (by construction —
  the service returns `boards.Board`, not `boards.BoardView`), even if the board already
  has an approved adapter. Not a bug per se, but frontend should re-fetch `GET /api/boards`
  (or `GET /api/boards/{id}/adapter`) after a PUT if it needs the adapter state, rather
  than relying on the PUT response.

---

## Job Browser (P6-FE-4)

Source: `backend/internal/handler/http/jobs.go`. All routes require auth AND
`X-Active-Profile` header (innermost `guarded.Group`).

### Shared shape: `jobResponse`
```jsonc
{
  "id": "string",
  "title": "string",
  "company": "string",
  "location": "string",
  "url": "string",
  "skills": ["string"],                 // never null
  "remote_policy": "string",            // raw kernel.RemotePolicy — French-render in FE
  "office_days": 0,
  "contract_type": "string",            // raw kernel.ContractType — French-render in FE
  "working_days": "string",             // raw kernel.WorkingDays — French-render in FE
  "salary_min": 0,                      // int64|null
  "salary_max": 0,                      // int64|null
  "seniority": "string",                // raw kernel.Seniority — French-render in FE
  "field_confidence": { "fieldName": 0 }, // map[string]int, never null; per-field confidence score
  "understanding_score": 0,             // int (from UnderstandingScore.Int())
  "description": "string",              // raw listing text, render verbatim (NOT French-translated)
  "contact_id": "string|null",
  "first_seen": "RFC3339 string",
  "last_seen": "RFC3339 string",
  "expired_at": "RFC3339 string|null",
  "application_status": "string|null",  // raw kernel.ApplicationStatus — French-render in FE; null = not yet applied/tracked
  "fit_score": 0.0,                     // float64|null
  "sources": [
    { "board_id": "string", "source_url": "string", "board_name": "string" }
  ]
}
```
"found on: …" UI element = render from `sources[].board_name` (+ `source_url` as link).
Confidence/understanding badges = `field_confidence` (per-field) and
`understanding_score` (overall). Expired marker = `expired_at != null`.

### `GET /api/jobs` response envelope (ADR-007)

**Breaking change**: `GET /api/jobs` no longer returns a bare `jobResponse[]` array. It
returns a paginated envelope, default-on (no opt-in — see ADR-007):

```jsonc
{
  "items": [ /* jobResponse[], unchanged shape */ ],
  "page": 2,
  "page_size": 25,
  "total": 137,        // total rows across the whole filtered result, COUNT(DISTINCT j.id)
  "total_pages": 6     // ceil(total / page_size); 0 when total is 0
}
```

Two new query params, added to the existing filter/sort set:

| Param | Type | Default | Bounds |
|---|---|---|---|
| `page` | int, 1-based | `1` | clamped to `>= 1` |
| `page_size` | int | `25` | clamped to `1..100` |

Out-of-range or unparseable values are **clamped, never rejected** — `page=-5` becomes
`1`, `page_size=500` becomes `100`, `page_size=abc` falls back to the default `25`. A
`page` beyond the last page returns `items: []` with the real `total` (not `0`), so the
frontend can recover (e.g. clamp back to the last valid page).

The kanban view fetches with `page_size=100` (the max) and does not paginate per column
— it only renders the small tracked (`application_status != null`) subset (ADR-007).

### Endpoints

| Method | Path | Query params / body | Response |
|---|---|---|---|
| GET | `/api/jobs` | `skills[]` (repeatable), `remote_policy`, `contract_type`, `salary_min`, `salary_max`, `location`, `board_id`, `since` (RFC3339), `confidence_min` (int), `sort` (`date`\|`salary`), `sort_dir` (`asc`\|`desc`), `page` (int, default 1), `page_size` (int, default 25, max 100) | 200 paginated envelope (see above) |
| GET | `/api/jobs/{id}` | — | 200 `jobResponse` |
| GET | `/api/jobs/{id}/original` | — | 302 redirect to `job.url`; 400 if job has no URL | Use as `<a href>` target, not `fetch` |
| PATCH | `/api/jobs/{id}/application` | `{"status":"string"}` — must be a valid `kernel.ApplicationStatus` (kanban column value: Saved/Applied/Interview/Offer/Rejected — check `kernel.ParseApplicationStatus` for exact accepted string values) | 200 `{"status":"string","updated_at":"RFC3339 string"}` |
| POST | `/api/jobs/{id}/reextract` | — | 202 `{"status":"re-extraction queued"}` (async; actual data updates asynchronously via extract-worker, not in this response) |

Notes for frontend:
- Kanban optimistic update: PATCH `/api/jobs/{id}/application` returns just `{status,
  updated_at}`, not the full job — frontend should optimistically update its own cache
  entry for that job's `application_status`, and roll back on error.
- `sort` only supports `date`/`salary` per the query parser; there is no server-side sort
  by fit score, confidence, or title — client-side sort those columns if needed.
- `useJobs` must be updated to read `.items` (list) and `.page/.page_size/.total/.total_pages`
  (pagination controls) instead of treating the response as an array (ADR-007).
