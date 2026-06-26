variable "env" {
  type        = string
  description = "Environment slug (dev | prod). Used in resource names and availability sizing."
}

variable "region" {
  type        = string
  description = "GCP region for the Cloud SQL instance."
}

variable "db_tier" {
  type        = string
  description = "Cloud SQL machine tier (e.g. db-g1-small for dev)."
}

variable "iam_user_emails" {
  type        = list(string)
  description = "Service-account emails that get a passwordless IAM DB user (one per worker SA)."
  default     = []
}

variable "ipv4_enabled" {
  type        = bool
  description = "Public IP. true for dev (connector + IAM auth only, no authorized networks). Tier 1 flips this to false with a private VPC."
  default     = false
}

variable "deletion_protection" {
  type        = bool
  description = "Block instance deletion. Should be true in prod."
  default     = false
}
