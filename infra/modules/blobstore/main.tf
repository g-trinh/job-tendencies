# Private GCS bucket for raw HTML/JSON. Uniform bucket-level access, no public access.
resource "google_storage_bucket" "raw" {
  name                        = "${var.project_id}-raw-${var.env}"
  location                    = var.region
  project                     = var.project_id
  force_destroy               = var.force_destroy
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = false
  }
}

resource "google_storage_bucket_iam_member" "creators" {
  for_each = toset(var.object_creator_members)

  bucket = google_storage_bucket.raw.name
  role   = "roles/storage.objectCreator"
  member = each.value
}

resource "google_storage_bucket_iam_member" "viewers" {
  for_each = toset(var.object_viewer_members)

  bucket = google_storage_bucket.raw.name
  role   = "roles/storage.objectViewer"
  member = each.value
}
