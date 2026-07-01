# --- Identity Platform (backend-proxied auth, P4) ----------------------------
# Enables Identity Platform (Firebase Auth's GCP tier) with the email/password
# provider. The SPA NEVER talks to Identity Platform directly — the API (BFF)
# proxies signInWithPassword and holds the session in an httpOnly cookie.
# ponytail: inlined, not a module — one config per project, dev-only. Promote to
# modules/ when prod needs the same wiring.

resource "google_identity_platform_config" "default" {
  project = var.project_id

  sign_in {
    allow_duplicate_emails = false

    email {
      enabled           = true
      password_required = true
    }
  }

  depends_on = [google_project_service.apis]
}

# API key + session cookie key secrets. Values are added out of band (never in
# tf/tfvars/state — see infra/README.md), same pattern as claude-api-key.
# Accessor is the api runtime SA only.
module "idp_api_key_secret" {
  source           = "../../modules/secrets"
  env              = local.env
  project_id       = var.project_id
  secret_id        = "idp-api-key"
  accessor_members = ["serviceAccount:${module.api.sa_email}"]

  depends_on = [google_project_service.apis]
}

module "session_cookie_key_secret" {
  source           = "../../modules/secrets"
  env              = local.env
  project_id       = var.project_id
  secret_id        = "session-cookie-key"
  accessor_members = ["serviceAccount:${module.api.sa_email}"]

  depends_on = [google_project_service.apis]
}
