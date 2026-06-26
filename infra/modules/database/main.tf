# Cloud SQL Postgres 16. IAM auth on, no public IP, backups on (Tier 0 baseline).
resource "google_sql_database_instance" "main" {
  name                = "jt-${var.env}-pg"
  database_version    = "POSTGRES_16"
  region              = var.region
  deletion_protection = var.deletion_protection

  settings {
    tier              = var.db_tier
    edition           = "ENTERPRISE" # shared-core tiers (db-g1-small) need ENTERPRISE, not ENTERPRISE_PLUS
    availability_type = var.env == "prod" ? "REGIONAL" : "ZONAL"

    ip_configuration {
      ipv4_enabled = var.ipv4_enabled
      ssl_mode     = "ENCRYPTED_ONLY"
      # ponytail: dev uses public IP with NO authorized_networks — reachable only via the
      # Cloud SQL connector + IAM auth (mTLS). Private IP (ipv4_enabled=false + private_network
      # + Serverless VPC connector) is the Tier 1 upgrade; an instance needs at least one of
      # public/private/PSC, so false alone is invalid without a VPC.
    }

    backup_configuration {
      enabled = true
    }

    database_flags {
      name  = "cloudsql.iam_authentication"
      value = "on"
    }
  }
}

resource "google_sql_database" "app" {
  name     = "job_tendencies"
  instance = google_sql_database_instance.main.name
}

# One passwordless IAM DB user per worker service account.
resource "google_sql_user" "iam" {
  for_each = toset(var.iam_user_emails)

  name     = trimsuffix(each.value, ".gserviceaccount.com")
  instance = google_sql_database_instance.main.name
  type     = "CLOUD_IAM_SERVICE_ACCOUNT"
}
