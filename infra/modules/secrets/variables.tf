variable "env" {
  type        = string
  description = "Environment slug (dev | prod)."
}

variable "project_id" {
  type        = string
  description = "GCP project id."
}

variable "secret_id" {
  type        = string
  description = "Logical secret id (e.g. claude-api-key). Suffixed with env."
}

variable "accessor_members" {
  type        = list(string)
  description = "IAM members granted roles/secretmanager.secretAccessor on this secret."
  default     = []
}
