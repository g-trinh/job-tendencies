variable "env" {
  type        = string
  description = "Environment slug (dev | prod)."
}

variable "region" {
  type        = string
  description = "GCP location for the bucket."
}

variable "project_id" {
  type        = string
  description = "GCP project id (bucket names are global; project scopes ownership)."
}

variable "object_creator_members" {
  type        = list(string)
  description = "IAM members granted roles/storage.objectCreator (e.g. scrape-worker SA)."
  default     = []
}

variable "object_viewer_members" {
  type        = list(string)
  description = "IAM members granted roles/storage.objectViewer (e.g. extract-worker SA)."
  default     = []
}

variable "force_destroy" {
  type        = bool
  description = "Allow tofu to delete a non-empty bucket. Keep false in prod."
  default     = false
}
