terraform {
  required_version = ">= 1.6.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region

  # Identity Platform / Firebase APIs require a quota (billing) project header.
  # Without this, requests are billed to the caller's default project and fail
  # with SERVICE_DISABLED / "requires a quota project".
  billing_project       = var.project_id
  user_project_override = true
}

# Firebase resources (google_firebase_*) live only in the beta provider.
provider "google-beta" {
  project = var.project_id
  region  = var.region

  billing_project       = var.project_id
  user_project_override = true
}
