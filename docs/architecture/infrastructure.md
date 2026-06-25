# Infrastructure (GCP / OpenTofu)

Provider **GCP** (user-confirmed). Region **europe-west9 (Paris)**. IaC: **OpenTofu**.
This document is the design; `.tf` files and `tofu apply` are a later pass. No
state-mutating command is run without explicit per-action user confirmation.

## 1. Defaults

- **Region**: `europe-west9` (Paris). Single-region, zonal resources where cheap.
- **State backend**: GCS bucket, **separate state per environment** (own bucket/prefix per
  `dev` and `prod`) — never shared state, never workspaces for prod-vs-dev.
- **Security tier**: **Tier 0** (pre-launch, single user). Higher-tier controls noted as
  deferred (§6).

## 2. Repo layout

```
infra/
├── README.md                 # provider decision, region, state backend, security tier
├── modules/
│   ├── database/             # Cloud SQL Postgres instance, db, IAM users
│   ├── pubsub/               # topics scrape.tick + listing.extract, subs, DLQs
│   ├── cloud-run-service/    # reusable: one Cloud Run service + SA + IAM
│   ├── scheduler/            # Cloud Scheduler cron -> Pub/Sub
│   ├── blobstore/            # GCS raw bucket
│   └── secrets/              # Secret Manager (Claude key) + accessor bindings
└── environments/
    ├── dev/                  # thin: module calls + dev tfvars, own GCS state
    └── prod/                 # thin: module calls + prod tfvars, separate GCS state
```

Modules: single responsibility, typed `variables.tf`, `outputs.tf`, pinned `versions.tf`,
`README.md`. Every module wired into **both** dev and prod (sizing differs via tfvars).

## 3. Module set

| Module | Provisions |
|---|---|
| `database` | Cloud SQL Postgres instance, database, IAM DB users (one per worker SA) |
| `pubsub` | `scrape.tick` + `listing.extract` topics, push subscriptions (OIDC), `*.dlq` dead-letter topics + subscriptions |
| `cloud-run-service` | Reusable: one Cloud Run service, its runtime service account, IAM invoker bindings, env/secret wiring. Instantiated 3× (api, scrape-worker, extract-worker) |
| `scheduler` | Cloud Scheduler job (cron from app config) targeting `scrape.tick` |
| `blobstore` | GCS bucket for raw HTML/JSON (uniform bucket-level access, no public) |
| `secrets` | Secret Manager secret(s) (Claude API key, DB connection), accessor IAM bindings |

## 4. Per-binary service accounts + least-privilege IAM

Each binary runs as its own SA; grants are the minimum each needs.

| Service account | Roles (least privilege) |
|---|---|
| `api-sa` | `roles/cloudsql.client` (DB), `roles/pubsub.publisher` on `scrape.tick`, `roles/secretmanager.secretAccessor` on the Claude secret (adapter gen) |
| `scrape-sa` | `roles/cloudsql.client`, `roles/pubsub.publisher` on `listing.extract`, `roles/storage.objectCreator` on the raw bucket. Pub/Sub push subscription invokes scrape-worker via a push-auth SA with `roles/run.invoker` on it |
| `extract-sa` | `roles/cloudsql.client`, `roles/storage.objectViewer` on the raw bucket, `roles/secretmanager.secretAccessor` on the Claude secret. Push subscription invokes extract-worker via a push-auth SA with `roles/run.invoker` |

No `roles/*` admin/editor on any runtime SA. Cloud Run services require authentication
(no `allUsers` invoker). Pub/Sub push uses OIDC tokens minted for the push-auth SA.

## 5. Key resource sketches (HCL — reviewable, not full boilerplate)

### Cloud SQL Postgres (module `database`)
```hcl
resource "google_sql_database_instance" "main" {
  name             = "jt-${var.env}-pg"
  database_version = "POSTGRES_16"
  region           = var.region            # europe-west9
  settings {
    tier              = var.db_tier         # dev: db-g1-small ; prod: per tfvars
    availability_type = var.env == "prod" ? "REGIONAL" : "ZONAL"
    ip_configuration {
      ipv4_enabled = false                  # no public IP
      # private_network = var.vpc_self_link # Tier 1: private IP + VPC connector
    }
    backup_configuration { enabled = true } # Tier 0 baseline
    database_flags { name = "cloudsql.iam_authentication" value = "on" }
  }
  deletion_protection = var.env == "prod"
}

resource "google_sql_database" "app" {
  name     = "job_tendencies"
  instance = google_sql_database_instance.main.name
}

# IAM DB user per worker SA (passwordless, IAM auth)
resource "google_sql_user" "extract" {
  name     = trimsuffix(var.extract_sa_email, ".gserviceaccount.com")
  instance = google_sql_database_instance.main.name
  type     = "CLOUD_IAM_SERVICE_ACCOUNT"
}
```

### Pub/Sub topic + push subscription + DLQ (module `pubsub`)
```hcl
resource "google_pubsub_topic" "scrape_tick" { name = "scrape-tick-${var.env}" }
resource "google_pubsub_topic" "scrape_tick_dlq" { name = "scrape-tick-dlq-${var.env}" }

resource "google_pubsub_subscription" "scrape_tick_push" {
  name  = "scrape-tick-push-${var.env}"
  topic = google_pubsub_topic.scrape_tick.id

  push_config {
    push_endpoint = "${var.scrape_worker_url}/push/scrape-tick"
    oidc_token {                                   # authenticated push
      service_account_email = var.push_auth_sa_email
      audience              = var.scrape_worker_url
    }
  }
  ack_deadline_seconds       = 60
  retry_policy { minimum_backoff = "10s" maximum_backoff = "600s" }
  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.scrape_tick_dlq.id
    max_delivery_attempts = 5
  }
}
```
(`listing.extract` topic/sub/DLQ are an identical instantiation with its own endpoint.)

### Cloud Run service (module `cloud-run-service`, instantiated per binary)
```hcl
resource "google_service_account" "svc" {
  account_id = "${var.name}-sa"                    # api-sa | scrape-sa | extract-sa
}

resource "google_cloud_run_v2_service" "svc" {
  name     = "${var.name}-${var.env}"
  location = var.region                            # europe-west9
  ingress  = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  template {
    service_account = google_service_account.svc.email
    scaling { min_instance_count = 0 max_instance_count = var.max_instances } # scrape-worker: 1
    containers {
      image = var.image
      resources { limits = { cpu = var.cpu, memory = var.memory } }
      dynamic "env" {
        for_each = var.secret_env                  # e.g. ANTHROPIC_API_KEY from Secret Manager
        content {
          name = env.key
          value_source { secret_key_ref { secret = env.value.secret version = "latest" } }
        }
      }
    }
    max_instance_request_concurrency = var.concurrency  # scrape-worker: 1
  }
}

# No allUsers invoker. Only the push-auth SA may invoke worker services.
resource "google_cloud_run_v2_service_iam_member" "invoker" {
  count    = var.allow_push_invoker ? 1 : 0
  name     = google_cloud_run_v2_service.svc.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "serviceAccount:${var.push_auth_sa_email}"
}
```

### Cloud Scheduler job (module `scheduler`)
```hcl
resource "google_cloud_scheduler_job" "scrape" {
  name      = "scrape-schedule-${var.env}"
  region    = var.region                           # europe-west9
  schedule  = var.cron                             # the one global schedule
  time_zone = "Europe/Paris"
  pubsub_target {
    topic_name = var.scrape_tick_topic_id
    data       = base64encode(jsonencode({ trigger = "scheduled" }))
  }
}
```

## 6. Security tiers (Tier 0 now, Tier 1 deferred)

**Tier 0 (applied now):** encryption at rest/in transit (default-on, verified),
least-privilege per-binary service accounts, secrets in Secret Manager (never in
`.tf`/`.tfvars`/state), no public DB or bucket, authenticated Cloud Run + OIDC Pub/Sub
push, project audit logging, backups enabled on Cloud SQL.

**Tier 1 (deferred — record in `infra/README.md`, build when MAU/paying users appear):**
Cloud Armor/WAF on the public API edge, private-IP-only Cloud SQL + Serverless VPC
connector (network segmentation), automated backups with tested restore + retention,
IaC/container vulnerability scanning in CI. These map to the single-user → multi-user
boundary in [deployment.md §5](deployment.md).

## 7. Workflow when implementing

`tofu fmt -recursive` + `tofu validate` (both envs) → `tofu plan` (dev first, reviewed with
the user) → commit → **stop**. `tofu apply` only on explicit per-action user confirmation.
