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
