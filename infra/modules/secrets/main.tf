# Secret Manager secret. The secret VALUE is never in tf/tfvars/state — it is added
# out of band with `gcloud secrets versions add`. This resource only declares the
# container and who may read it.
resource "google_secret_manager_secret" "this" {
  project   = var.project_id
  secret_id = "${var.secret_id}-${var.env}"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_iam_member" "accessors" {
  for_each = toset(var.accessor_members)

  project   = var.project_id
  secret_id = google_secret_manager_secret.this.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = each.value
}
