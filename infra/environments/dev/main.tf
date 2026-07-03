locals {
  env = "dev"

  # Container image base. Defaults to the project's Artifact Registry repo; the
  # binaries are pushed there by `make image-*` (see backend/Makefile).
  image_base = var.image_registry != "" ? var.image_registry : "${var.region}-docker.pkg.dev/${var.project_id}/job-tendencies"

  # Deterministic resource names, reconstructed instead of read from module
  # outputs: database/blobstore/secrets already depend on the service SAs, so
  # feeding their outputs back as service env_vars would form a dependency cycle.
  # These names are fixed by each module's naming scheme.
  cloud_sql_instance = "${var.project_id}:${var.region}:jt-${local.env}-pg"
  db_name            = "job_tendencies"
  raw_bucket         = "${var.project_id}-raw-${local.env}"
  scrape_topic       = "scrape-tick-${local.env}"
  extract_topic      = "listing-extract-${local.env}"

  # LLM provider switch (ADR-006): one provider serves every LLM task (adapter
  # generation, listing extraction, identity import). Flip this to "deepseek" to
  # switch providers everywhere; requires the deepseek-api-key-dev secret version
  # to exist before apply (see infra/README.md).
  llm_provider      = "deepseek"
  deepseek_model_id = "deepseek-v4-flash"

  # Frontend origin allowed to call the API cross-origin (CORS ALLOWED_ORIGINS).
  # Prod is same-origin via the Firebase Hosting /api rewrite, so CORS only
  # matters for local dev hitting the deployed API and direct-to-Cloud-Run calls.
  frontend_origin = "https://${google_firebase_hosting_site.spa.site_id}.web.app"
  allowed_origins = join(",", [local.frontend_origin, "http://localhost:5173"])

  # DATABASE_URL is required by config.Load() but only goose migrations use it;
  # the runtime connects via the Cloud SQL Go connector (CLOUD_SQL_INSTANCE +
  # DB_IAM_USER). Non-empty placeholder so the required-var check passes.
  database_url = "postgres://iam@/${local.db_name}?host=/cloudsql/${local.cloud_sql_instance}"

  # Shared DB env. DB_IAM_USER is self-injected by the cloud-run-service module.
  svc_db_env = {
    GCP_PROJECT_ID     = var.project_id
    CLOUD_SQL_INSTANCE = local.cloud_sql_instance
    DB_NAME            = local.db_name
    DATABASE_URL       = local.database_url
  }
}

# --- Project APIs ------------------------------------------------------------
resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    "sqladmin.googleapis.com",
    "pubsub.googleapis.com",
    "cloudscheduler.googleapis.com",
    "secretmanager.googleapis.com",
    "storage.googleapis.com",
    "iam.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "firebase.googleapis.com",
    "firebasehosting.googleapis.com",
    "identitytoolkit.googleapis.com",
  ])

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

# --- Push-auth SA: mints OIDC tokens for Pub/Sub push, invokes the workers ----
resource "google_service_account" "push_auth" {
  project      = var.project_id
  account_id   = "pubsub-push-${local.env}"
  display_name = "Pub/Sub push OIDC invoker (${local.env})"
}

# Pub/Sub service agent must be able to mint OIDC tokens as the push-auth SA.
data "google_project" "this" {
  project_id = var.project_id
}

resource "google_service_account_iam_member" "pubsub_token_creator" {
  service_account_id = google_service_account.push_auth.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${data.google_project.this.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

# --- Cloud Run services (one SA each) ----------------------------------------
module "api" {
  source     = "../../modules/cloud-run-service"
  name       = "api"
  env        = local.env
  region     = var.region
  project_id = var.project_id
  ingress    = "INGRESS_TRAFFIC_ALL"
  image      = "${local.image_base}/api:${var.image_tag}"
  # Public invoker: the SPA reaches the API unauthenticated via the Firebase
  # Hosting /api rewrite; the app's session-cookie guard is the real access
  # control (P4). No push invoker: the API is not a push target.
  allow_public_invoker = true
  env_vars = merge(local.svc_db_env, {
    PUBSUB_SCRAPE_TOPIC_ID  = local.scrape_topic
    PUBSUB_EXTRACT_TOPIC_ID = local.extract_topic # P5-4: reextract re-publishes listing.extract
    ALLOWED_ORIGINS         = local.allowed_origins
    LLM_PROVIDER            = local.llm_provider # ADR-006
    DEEPSEEK_MODEL_ID       = local.deepseek_model_id
  })
  # Backend-proxied auth (P4). The API signs in against Identity Platform with
  # IDP_API_KEY and encrypts the session cookie with SESSION_COOKIE_KEY. Both
  # secret VERSIONS must exist before apply (see auth.tf / infra/README.md),
  # else the revision fails to start.
  #
  # DEEPSEEK_API_KEY secret VERSION is only required before apply when
  # local.llm_provider = "deepseek" (ADR-006); the api binary calls DeepSeek for
  # adapter generation on that path. Only wired into secret_env in that case, so
  # switching back to "claude" does not require the deepseek secret to exist.
  secret_env = merge(
    {
      IDP_API_KEY        = { secret = "idp-api-key-${local.env}" }
      SESSION_COOKIE_KEY = { secret = "session-cookie-key-${local.env}" }
    },
    local.llm_provider == "deepseek" ? { DEEPSEEK_API_KEY = { secret = "deepseek-api-key-${local.env}" } } : {}
  )

  depends_on = [google_project_service.apis]
}

module "scrape_worker" {
  source             = "../../modules/cloud-run-service"
  name               = "scrape-worker"
  env                = local.env
  region             = var.region
  project_id         = var.project_id
  image              = "${local.image_base}/scrape-worker:${var.image_tag}"
  max_instances      = 1 # pinned: in-process per-board rate limiter stays authoritative
  concurrency        = 1
  allow_push_invoker = true
  push_auth_sa_email = google_service_account.push_auth.email
  env_vars = merge(local.svc_db_env, {
    GCS_RAW_BUCKET          = local.raw_bucket
    PUBSUB_EXTRACT_TOPIC_ID = local.extract_topic
    PUBSUB_PUSH_SA          = google_service_account.push_auth.email
  })

  depends_on = [google_project_service.apis]
}

module "extract_worker" {
  source             = "../../modules/cloud-run-service"
  name               = "extract-worker"
  env                = local.env
  region             = var.region
  project_id         = var.project_id
  image              = "${local.image_base}/extract-worker:${var.image_tag}"
  max_instances      = 5
  concurrency        = 4
  allow_push_invoker = true
  push_auth_sa_email = google_service_account.push_auth.email
  env_vars = merge(local.svc_db_env, {
    GCS_RAW_BUCKET    = local.raw_bucket
    PUBSUB_PUSH_SA    = google_service_account.push_auth.email
    LLM_PROVIDER      = local.llm_provider # ADR-006
    DEEPSEEK_MODEL_ID = local.deepseek_model_id
  })
  # LLM provider key from Secret Manager. Requires the matching secret VERSION to
  # exist before apply (claude-api-key-dev always; deepseek-api-key-dev only when
  # local.llm_provider = "deepseek" — ADR-006), else the revision fails to start.
  secret_env = merge(
    { ANTHROPIC_API_KEY = { secret = "claude-api-key-${local.env}" } },
    local.llm_provider == "deepseek" ? { DEEPSEEK_API_KEY = { secret = "deepseek-api-key-${local.env}" } } : {}
  )

  depends_on = [google_project_service.apis]
}

# --- Datastore ----------------------------------------------------------------
module "database" {
  source       = "../../modules/database"
  env          = local.env
  region       = var.region
  db_tier      = var.db_tier
  ipv4_enabled = true # dev: connector + IAM auth only; private IP is Tier 1
  iam_user_emails = [
    module.api.sa_email,
    module.scrape_worker.sa_email,
    module.extract_worker.sa_email,
  ]
  deletion_protection = false

  depends_on = [google_project_service.apis]
}

module "blobstore" {
  source                 = "../../modules/blobstore"
  env                    = local.env
  region                 = var.region
  project_id             = var.project_id
  object_creator_members = ["serviceAccount:${module.scrape_worker.sa_email}"]
  object_viewer_members  = ["serviceAccount:${module.extract_worker.sa_email}"]
  force_destroy          = true # dev only

  depends_on = [google_project_service.apis]
}

module "claude_secret" {
  source     = "../../modules/secrets"
  env        = local.env
  project_id = var.project_id
  secret_id  = "claude-api-key"
  accessor_members = [
    "serviceAccount:${module.api.sa_email}",            # adapter generation
    "serviceAccount:${module.extract_worker.sa_email}", # extraction
  ]

  depends_on = [google_project_service.apis]
}

# ADR-006: second LLM provider. Secret container always provisioned; a secret
# VERSION is only required before apply when local.llm_provider = "deepseek"
# (see infra/README.md).
module "deepseek_secret" {
  source     = "../../modules/secrets"
  env        = local.env
  project_id = var.project_id
  secret_id  = "deepseek-api-key"
  accessor_members = [
    "serviceAccount:${module.api.sa_email}",            # adapter generation
    "serviceAccount:${module.extract_worker.sa_email}", # extraction
  ]

  depends_on = [google_project_service.apis]
}

# --- Data-plane IAM (composition root grants least-privilege to the SAs) ------
locals {
  db_sa_emails = toset([
    module.api.sa_email,
    module.scrape_worker.sa_email,
    module.extract_worker.sa_email,
  ])
}

resource "google_project_iam_member" "cloudsql_client" {
  for_each = local.db_sa_emails

  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${each.value}"
}

# cloudsql.instances.login — required for Cloud SQL IAM database authentication.
# cloudsql.client only grants connect; without instanceUser, IAM login fails 28000.
resource "google_project_iam_member" "cloudsql_instance_user" {
  for_each = local.db_sa_emails

  project = var.project_id
  role    = "roles/cloudsql.instanceUser"
  member  = "serviceAccount:${each.value}"
}

# --- Messaging ----------------------------------------------------------------
module "scrape_tick" {
  source             = "../../modules/pubsub"
  env                = local.env
  project_id         = var.project_id
  name               = "scrape-tick"
  push_endpoint      = "${module.scrape_worker.service_uri}/push/scrape-tick"
  push_auth_sa_email = google_service_account.push_auth.email
  push_audience      = module.scrape_worker.service_uri

  depends_on = [google_project_service.apis]
}

module "listing_extract" {
  source             = "../../modules/pubsub"
  env                = local.env
  project_id         = var.project_id
  name               = "listing-extract"
  push_endpoint      = "${module.extract_worker.service_uri}/push/listing-extract"
  push_auth_sa_email = google_service_account.push_auth.email
  push_audience      = module.extract_worker.service_uri

  depends_on = [google_project_service.apis]
}

# api publishes scrape.tick on demand, and listing.extract for reextract (P5-4);
# scrape-worker also publishes listing.extract.
resource "google_pubsub_topic_iam_member" "api_publishes_scrape_tick" {
  project = var.project_id
  topic   = module.scrape_tick.topic_name
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${module.api.sa_email}"
}

resource "google_pubsub_topic_iam_member" "api_publishes_listing_extract" {
  project = var.project_id
  topic   = module.listing_extract.topic_name
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${module.api.sa_email}"
}

resource "google_pubsub_topic_iam_member" "scrape_publishes_listing_extract" {
  project = var.project_id
  topic   = module.listing_extract.topic_name
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${module.scrape_worker.sa_email}"
}

# --- Scheduler (paused until Phase 7) ----------------------------------------
module "scheduler" {
  source     = "../../modules/scheduler"
  env        = local.env
  region     = var.scheduler_region
  project_id = var.project_id
  topic_id   = module.scrape_tick.topic_id
  paused     = true

  depends_on = [google_project_service.apis]
}
