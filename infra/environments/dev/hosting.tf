# --- Firebase Hosting (SPA) --------------------------------------------------
# OpenTofu owns Firebase enablement on the GCP project and the Hosting site.
# Site content + the /api -> Cloud Run rewrite live in frontend/firebase.json,
# shipped by `npm run deploy` (firebase deploy --only hosting).
# ponytail: inlined, not a module — one site, dev-only. Promote to modules/ when
# prod needs the same wiring.

resource "google_firebase_project" "default" {
  provider   = google-beta
  project    = var.project_id
  depends_on = [google_project_service.apis]
}

resource "google_firebase_hosting_site" "spa" {
  provider   = google-beta
  project    = var.project_id
  site_id    = var.project_id # -> https://job-tendencies-dev.web.app
  depends_on = [google_firebase_project.default]
}
