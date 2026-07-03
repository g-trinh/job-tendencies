variable "project_id" {
  type        = string
  description = "GCP project id for the prod environment."
}

variable "region" {
  type        = string
  description = "Primary GCP region."
  default     = "europe-west9"
}

variable "db_tier" {
  type        = string
  description = "Cloud SQL machine tier for prod. Must be a dedicated-core tier — REGIONAL HA is unsupported on shared-core (db-g1-small/db-f1-micro)."
  default     = "db-custom-1-3840"
}

variable "scheduler_region" {
  type        = string
  description = "Cloud Scheduler region (europe-west9 is unsupported for Scheduler; use europe-west1)."
  default     = "europe-west1"
}

variable "global_cron" {
  type        = string
  description = "Global scrape.tick cron (board-manager global schedule). Europe/Paris."
  default     = "0 3 * * *"
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
