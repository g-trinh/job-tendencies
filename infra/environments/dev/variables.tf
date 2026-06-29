variable "project_id" {
  type        = string
  description = "GCP project id for the dev environment."
}

variable "region" {
  type        = string
  description = "Primary GCP region."
  default     = "europe-west9"
}

variable "db_tier" {
  type        = string
  description = "Cloud SQL machine tier for dev."
  default     = "db-g1-small"
}

variable "scheduler_region" {
  type        = string
  description = "Cloud Scheduler region (europe-west9 is unsupported for Scheduler; use europe-west1)."
  default     = "europe-west1"
}

variable "image_registry" {
  type        = string
  description = "Artifact Registry base path for service images. Empty -> <region>-docker.pkg.dev/<project>/job-tendencies."
  default     = ""
}

variable "image_tag" {
  type        = string
  description = "Container image tag deployed to Cloud Run."
  default     = "latest"
}
