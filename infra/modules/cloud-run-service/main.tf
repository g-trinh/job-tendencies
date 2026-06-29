# Reusable Cloud Run service: one runtime SA + the service + an optional push invoker.
# Instantiated per binary (api, scrape-worker, extract-worker). No allUsers invoker.

resource "google_service_account" "svc" {
  project      = var.project_id
  account_id   = "${var.name}-sa"
  display_name = "${var.name} (${var.env}) runtime SA"
}

locals {
  # The service's own SA is also its Cloud SQL IAM DB user (email minus the
  # .gserviceaccount.com suffix). Self-injected here so callers don't form a
  # module-output -> module-input cycle (database/blobstore/secrets already
  # depend on this SA). Caller env_vars override on key collision.
  injected_env = {
    DB_IAM_USER = trimsuffix(google_service_account.svc.email, ".gserviceaccount.com")
  }
}

resource "google_cloud_run_v2_service" "svc" {
  project = var.project_id
  # No env suffix: dev and prod are separate projects, so the project already scopes the env.
  name                = var.name
  location            = var.region
  ingress             = var.ingress
  deletion_protection = var.env == "prod"

  template {
    service_account = google_service_account.svc.email

    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

    max_instance_request_concurrency = var.concurrency

    containers {
      image = var.image

      resources {
        limits = {
          cpu    = var.cpu
          memory = var.memory
        }
      }

      dynamic "env" {
        for_each = merge(local.injected_env, var.env_vars)
        content {
          name  = env.key
          value = env.value
        }
      }

      dynamic "env" {
        for_each = var.secret_env
        content {
          name = env.key
          value_source {
            secret_key_ref {
              secret  = env.value.secret
              version = env.value.version
            }
          }
        }
      }
    }
  }
}

# Authenticated push only: the push-auth SA may invoke worker services. Never allUsers.
resource "google_cloud_run_v2_service_iam_member" "push_invoker" {
  count = var.allow_push_invoker ? 1 : 0

  project  = var.project_id
  name     = google_cloud_run_v2_service.svc.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "serviceAccount:${var.push_auth_sa_email}"
}
