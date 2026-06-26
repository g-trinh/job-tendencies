locals {
  env = "dev"
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
  # No push invoker: the API is not a push target. SPA auth is deferred (Tier 1).

  depends_on = [google_project_service.apis]
}

module "scrape_worker" {
  source             = "../../modules/cloud-run-service"
  name               = "scrape-worker"
  env                = local.env
  region             = var.region
  project_id         = var.project_id
  max_instances      = 1 # pinned: in-process per-board rate limiter stays authoritative
  concurrency        = 1
  allow_push_invoker = true
  push_auth_sa_email = google_service_account.push_auth.email

  depends_on = [google_project_service.apis]
}

module "extract_worker" {
  source             = "../../modules/cloud-run-service"
  name               = "extract-worker"
  env                = local.env
  region             = var.region
  project_id         = var.project_id
  max_instances      = 5
  concurrency        = 4
  allow_push_invoker = true
  push_auth_sa_email = google_service_account.push_auth.email

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

# --- Data-plane IAM (composition root grants least-privilege to the SAs) ------
resource "google_project_iam_member" "cloudsql_client" {
  for_each = toset([
    module.api.sa_email,
    module.scrape_worker.sa_email,
    module.extract_worker.sa_email,
  ])

  project = var.project_id
  role    = "roles/cloudsql.client"
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

# api publishes scrape.tick on demand; scrape-worker publishes listing.extract.
resource "google_pubsub_topic_iam_member" "api_publishes_scrape_tick" {
  project = var.project_id
  topic   = module.scrape_tick.topic_name
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
