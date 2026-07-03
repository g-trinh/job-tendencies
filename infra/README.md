# Infrastructure

OpenTofu / GCP. Design of record: [docs/architecture/infrastructure.md](../docs/architecture/infrastructure.md).
No `tofu apply` is run without explicit per-action user confirmation.

## Cloud Provider

**Provider**: GCP (user-confirmed)
**Primary region**: `europe-west9` (Paris) — single-region, zonal where cheap.
**Why**: chosen in the architecture phase (ADR-002/003/004); managed Cloud SQL + Pub/Sub +
Cloud Run + Secret Manager + Cloud Scheduler fit the scheduled scrape→extract pipeline with
scale-to-zero cost for a single user.

## State Backend

GCS, **separate state per environment** — never shared, never workspaces for prod-vs-dev.

**dev**: GCS bucket `job-tendencies-dev-tfstate`, prefix `dev`.
**prod**: separate bucket/prefix (wired when the prod environment lands — see below).

> The state bucket must exist **before** `tofu init`. Create it once, out of band:
> ```sh
> gcloud storage buckets create gs://job-tendencies-dev-tfstate \
>   --project=job-tendencies-dev --location=europe-west9 \
>   --uniform-bucket-level-access --public-access-prevention
> gcloud storage buckets update gs://job-tendencies-dev-tfstate --versioning
> ```

## Security Tier

**Current tier**: Tier 0 (set 2026-06-26)
**Basis**: pre-launch, single user (MAU ~1, MRR ~$0).

Controls implemented: encryption at rest/in transit (default-on), least-privilege per-binary
service accounts (no admin/editor on any runtime SA), secrets in Secret Manager (values never
in tf/tfvars/state), no public DB (no public IP) or bucket (UBLA + public-access-prevention),
authenticated Cloud Run (no `allUsers` invoker) + OIDC Pub/Sub push, Cloud SQL backups on.

### Deferred controls

| Control | Tier | Revisit when |
|---|---|---|
| Cloud Armor / WAF on the API edge | 1 | first paying users / public SPA |
| Private-IP-only Cloud SQL + Serverless VPC connector | 1 | MAU > ~1k |
| Automated backups with tested restore + retention | 1 | MAU > ~1k |
| IaC / container vulnerability scanning in CI | 1 | MAU > ~1k |
| SPA → API authentication (Identity Platform / IAP) | 1 | ~~API exposed to a browser~~ **Done (P4)** — see Identity Platform below. |

## Environments

| Environment | Purpose | Notes |
|---|---|---|
| dev | Development / testing (pure-cloud dev model) | `db-g1-small`, scale-to-zero, `force_destroy` on the raw bucket, no deletion protection. **Stood up now (Phase 1).** |
| prod | Production | **Not yet wired.** Phase 1 scope is dev only (per tech-breakdown-phase-1 + infrastructure.md "prod is a later pass"). Prod re-uses the same `modules/` with prod tfvars (REGIONAL Cloud SQL, deletion protection, no force_destroy) — add `environments/prod/` in a later phase. Tracked here so it is a known gap, not a silent omission. |

## Modules

| Module | Provisions |
|---|---|
| `database` | Cloud SQL Postgres 16 instance, `job_tendencies` db, one IAM DB user per worker SA. IAM auth, no public IP, backups on. |
| `blobstore` | Private GCS raw bucket (UBLA, public-access-prevention) + objectCreator/objectViewer bindings. |
| `secrets` | Secret Manager secret container (value added out of band) + secretAccessor bindings. |
| `pubsub` | One topic + OIDC push subscription + dead-letter topic/subscription. Instantiated per topic (scrape-tick, listing-extract). |
| `cloud-run-service` | Reusable Cloud Run v2 service + runtime SA + optional push invoker. Instantiated for api / scrape-worker / extract-worker. |
| `scheduler` | Cloud Scheduler cron → `scrape.tick`, `Europe/Paris`, created paused. |

Cross-cutting data-plane IAM (cloudsql.client, pubsub.publisher, storage, secretAccessor) is
granted in `environments/dev/main.tf`, the composition root — modules stay single-purpose.

## Identity Platform (backend-proxied auth, P4)

`environments/dev/auth.tf` enables Identity Platform (`google_identity_platform_config`,
email/password provider) and declares two out-of-band secrets read by the `api` service:

| Secret | Env var | Purpose |
|---|---|---|
| `idp-api-key-dev` | `IDP_API_KEY` | Identity Toolkit REST key for `signInWithPassword` (proxied by the API). |
| `session-cookie-key-dev` | `SESSION_COOKIE_KEY` | Hex-encoded 32-byte AES-256-GCM key encrypting the httpOnly session cookie. |

The SPA never calls Identity Platform — the `api` service (BFF) proxies all identity calls and
holds the session in an httpOnly cookie. Values live only in Secret Manager, never in tf/state.

## LLM Provider (ADR-006)

One provider serves every LLM task (adapter generation, listing extraction, identity
import) — no per-task routing, no fallback. Selected at startup by the `LLM_PROVIDER`
env var, wired from `local.llm_provider` in `environments/dev/main.tf`. Flip that one
local (`"claude"` or `"deepseek"`) to switch providers on both `api` and
`extract-worker`.

| Secret | Env var | Purpose |
|---|---|---|
| `claude-api-key-dev` | `ANTHROPIC_API_KEY` | Anthropic API key. Version required before apply regardless of provider. |
| `deepseek-api-key-dev` | `DEEPSEEK_API_KEY` | DeepSeek API key. Secret container always provisioned; a **version** is only required before apply when `local.llm_provider = "deepseek"` — the env var is only wired into `secret_env` in that case. |

`DEEPSEEK_MODEL_ID` is a plain env var (`local.deepseek_model_id`), not a secret.

## Prerequisites for `tofu apply`

1. State bucket created (above).
2. Claude API key added to the secret after first apply creates the container:
   ```sh
   gcloud secrets versions add claude-api-key-dev --project=job-tendencies-dev --data-file=- <<<"$ANTHROPIC_API_KEY"
   ```
   If `local.llm_provider = "deepseek"`, also add the DeepSeek key:
   ```sh
   gcloud secrets versions add deepseek-api-key-dev --project=job-tendencies-dev --data-file=- <<<"$DEEPSEEK_API_KEY"
   ```
3. **Identity Platform (P4)** — after the first apply creates the secret containers and
   enables Identity Platform, do the following out of band (the `api` revision will not boot
   until both secret versions exist):
   ```sh
   # a. Create a restricted API key for the Identity Toolkit REST API (Console:
   #    APIs & Services -> Credentials -> Create API key -> restrict to
   #    "Identity Toolkit API"), then store its value:
   gcloud secrets versions add idp-api-key-dev --project=job-tendencies-dev --data-file=- <<<"$IDP_API_KEY"

   # b. Generate + store the 32-byte session cookie key (hex):
   gcloud secrets versions add session-cookie-key-dev --project=job-tendencies-dev --data-file=- <<<"$(openssl rand -hex 32)"

   # c. Create the single application user (single-user scope; not in tf/state):
   gcloud identity-platform tenants  # n/a — no tenant; create the user via:
   #    Console -> Identity Platform -> Users -> Add user (email + password), OR the
   #    Admin SDK / REST signUp endpoint with the API key above.
   ```
