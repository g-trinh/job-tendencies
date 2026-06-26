variable "env" {
  type        = string
  description = "Environment slug (dev | prod)."
}

variable "project_id" {
  type        = string
  description = "GCP project id."
}

variable "name" {
  type        = string
  description = "Logical topic name (e.g. scrape-tick, listing-extract). Suffixed with env."
}

variable "push_endpoint" {
  type        = string
  description = "HTTPS endpoint the push subscription delivers to (worker URL + route)."
}

variable "push_auth_sa_email" {
  type        = string
  description = "Service account whose OIDC token authenticates push delivery."
}

variable "push_audience" {
  type        = string
  description = "OIDC audience for the push token (usually the worker base URL)."
}

variable "ack_deadline_seconds" {
  type        = number
  description = "Push ack deadline."
  default     = 60
}

variable "max_delivery_attempts" {
  type        = number
  description = "Deliveries before a message is dead-lettered."
  default     = 5
}

variable "minimum_backoff" {
  type        = string
  description = "Retry policy minimum backoff."
  default     = "10s"
}

variable "maximum_backoff" {
  type        = string
  description = "Retry policy maximum backoff."
  default     = "600s"
}
