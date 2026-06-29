## Tech Breakdown: Phase 4 — Authentication (backend-proxied, single-user)

**Design spec ref:** docs/v0.md (dataflow)
**Architecture ref:** deployment.md §5 (auth row), infrastructure.md (security tier), overview.md §6
**Plan ref:** docs/plan/development-plan.md (Phase 4)
**Teams:** Backend, Frontend, Infra (cloud-ops)

API edge auth, inserted before the backend-contexts breadth so every later API surface is
built behind a real guard, and the SPA can call the deployed backend without leaving the API
unauthenticated.

**Hard constraint:** the **frontend NEVER calls Firebase / Identity Platform directly.** All
identity calls are **proxied through the backend** (BFF). No Firebase client SDK, no Firebase
config in the SPA, no ID tokens in browser JS. The browser only ever holds an **httpOnly**
session cookie it cannot read.

**Mechanism:** Identity Platform (Firebase Auth's GCP tier), **server-side only**. SPA →
`POST /api/auth/*` → backend → Identity Platform REST. **Scope: single-user.** Keep the
multi-tenant seam (active-profile param); no `tenant_id` model yet.

---

### Tasks

---

#### P4-IN-1 — Enable Identity Platform + store credentials in Secret Manager

**Type:** Chore
**Owner:** Infra (cloud-ops)
**Dependencies:** Phase 1 (dev infra)

**Description:**
Enable Identity Platform on the dev project via OpenTofu (`google_identity_platform_config`,
email/password provider). Store the IdP **API key** and any admin credential as Secret
Manager secrets; grant the `api` runtime SA `secretAccessor`. Create the single application
user out of band (documented, not in tf/state).

**Refs:** infrastructure.md (Secret Manager pattern), README (secret-version-out-of-band)

**Acceptance Criteria:**
- `tofu plan` clean; Identity Platform enabled with email/password.
- `api` service reads the IdP API key from Secret Manager (never in tfvars/state).
- One user exists and can be authenticated.

---

#### P4-BE-1 — Server-side Identity Platform client

**Type:** Feature
**Owner:** Backend
**Dependencies:** P4-IN-1

**Description:**
A backend client wrapping the Identity Platform REST API: `signInWithPassword`, token
refresh, and **ID-token verification** (verify signature against Google JWKS + `aud`/`iss`/
`exp`). Config (API key, project id) from env/Secret Manager. No domain logic in `main`.

**Refs:** ADR-001 (ports/adapters), config.go pattern

**Acceptance Criteria:**
- Given valid credentials, the client returns a verified ID token + refresh token.
- Invalid credentials/expired tokens are surfaced as typed errors (no panic).
- Token verification rejects wrong `aud`/`iss`/expired/forged signatures (unit-tested).

---

#### P4-BE-2 — Auth endpoints: login / logout / me

**Type:** Feature
**Owner:** Backend
**Dependencies:** P4-BE-1

**Description:**
`POST /api/auth/login` (email+password → IdP via P4-BE-1 → set **httpOnly, Secure,
SameSite=Strict** session cookie carrying the session/refresh material); `POST /api/auth/logout`
(clear cookie + revoke); `GET /api/auth/me` (current user from the session, 401 if none).
The browser receives **no token in the body** — only the cookie.

**Refs:** overview.md §6, deployment.md §5

**Acceptance Criteria:**
- Successful login sets an httpOnly cookie and returns the user (no token in JSON).
- `GET /api/auth/me` returns the user when the cookie is valid, 401 otherwise.
- Logout invalidates the session; subsequent `/api/auth/me` → 401.

---

#### P4-BE-3 — Auth middleware guarding /api + CSRF protection

**Type:** Feature
**Owner:** Backend
**Dependencies:** P4-BE-2

**Description:**
Middleware that verifies the session cookie on the `/api` subrouter (composes with the
existing `requireActiveProfile`); returns 401 when absent/invalid; refreshes the IdP token
transparently when near expiry. Add CSRF protection appropriate to cookie auth
(double-submit token or `SameSite=Strict` + origin check). Auth routes (`/api/auth/login`)
are exempt from the session guard.

**Refs:** handler/http/router.go (middleware pattern), middleware_test.go

**Acceptance Criteria:**
- Unauthenticated request to a guarded `/api` route → 401; authenticated → 200.
- A state-changing request without a valid CSRF token is rejected.
- `/healthz` and `/api/auth/login` remain reachable unauthenticated.

---

#### P4-FE-1 — French login screen + route guard (no direct Firebase)

**Type:** Feature
**Owner:** Frontend
**Dependencies:** P4-BE-2

**Description:**
A login screen (French) posting email/password to `POST /api/auth/login`. App-level route
guard gates the app on `GET /api/auth/me`; a logout action calls `POST /api/auth/logout`.
**No Firebase SDK, no Firebase config** — the SPA only talks to its own `/api`. The cookie is
set/sent automatically; the SPA never reads a token.

**Refs:** overview.md §7, react-guidelines

**Acceptance Criteria:**
- Unauthenticated user sees the login screen; valid login reveals the app.
- 401 from any `/api` call redirects to login.
- No network call from the browser targets a Firebase/Identity Platform domain.

---

#### P4-1 — Deploy auth to dev + verify end-to-end; open the API behind the guard

**Type:** Chore
**Owner:** Full-stack
**Dependencies:** P4-BE-3, P4-FE-1, P4-IN-1

**Description:**
Deploy the auth-enabled `api` + SPA to dev. The `api` service becomes `allUsers`-invocable
(`allow_public_invoker`) **because the app now enforces auth**, so the Firebase Hosting `/api`
rewrite resolves. Verify login → app → logout end-to-end.

**Refs:** deployment.md §5, development-plan.md (Phase 4 exit), infra cloud-run-service module

**Acceptance Criteria:**
- On dev: SPA login works through the Hosting `/api` rewrite; unauthenticated `/api` → 401.
- No Firebase call originates from the browser; session is an httpOnly cookie.

---

### Dependency Graph

```
P4-IN-1 → P4-BE-1 → P4-BE-2 → P4-BE-3 ┐
                         └→ P4-FE-1 ──┴→ P4-1
```

### Parallel tracks

- P4-FE-1 can stub against a fixture `/api/auth/me` until P4-BE-2 lands.
- P4-IN-1 (infra) runs concurrently with FE login-screen scaffolding.

### Open Questions

| # | Question | Blocking tasks | Owner |
|---|----------|----------------|-------|
| 1 | Session model: opaque server session vs storing the IdP refresh token in the cookie | P4-BE-2 | Backend |
| 2 | CSRF strategy: double-submit token vs SameSite=Strict + origin check only | P4-BE-3 | Backend |
| 3 | Single-user provisioning: console-created vs a one-shot admin script | P4-IN-1 | User/cloud-ops |
