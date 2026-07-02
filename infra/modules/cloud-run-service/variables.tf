variable "name" {
  type        = string
  description = "Service base name (api | scrape-worker | extract-worker). Also drives the SA id."
}

variable "env" {
  type        = string
  description = "Environment slug (dev | prod)."
}

variable "region" {
  type        = string
  description = "Cloud Run region."
}

variable "project_id" {
  type        = string
  description = "GCP project id."
}

variable "image" {
  type        = string
  description = "Container image. Defaults to the Cloud Run hello image so dev can plan before real images exist."
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "cpu" {
  type        = string
  description = "CPU limit (e.g. \"1\")."
  default     = "1"
}

variable "memory" {
  type        = string
  description = "Memory limit (e.g. \"512Mi\")."
  default     = "512Mi"
}

variable "min_instances" {
  type        = number
  description = "Minimum instances (0 = scale to zero)."
  default     = 0
}

variable "max_instances" {
  type        = number
  description = "Maximum instances. scrape-worker must be 1."
  default     = 4
}

variable "concurrency" {
  type        = number
  description = "Max concurrent requests per instance. scrape-worker must be 1."
  default     = 80
}

variable "ingress" {
  type        = string
  description = "Cloud Run ingress. api: INGRESS_TRAFFIC_ALL; workers: INGRESS_TRAFFIC_INTERNAL_ONLY."
  default     = "INGRESS_TRAFFIC_ALL"
}

variable "env_vars" {
  type        = map(string)
  description = "Plain environment variables."
  default     = {}
}

variable "secret_env" {
  type = map(object({
    secret  = string
    version = optional(string, "latest")
  }))
  description = "Env vars sourced from Secret Manager: { ENV_NAME = { secret = id, version = \"latest\" } }."
  default     = {}
}

variable "allow_push_invoker" {
  type        = bool
  description = "Grant roles/run.invoker to the push-auth SA (worker services only)."
  default     = false
}

variable "allow_public_invoker" {
  type        = bool
  description = "Grant roles/run.invoker to allUsers (api only — the app enforces auth in-process)."
  default     = false
}

variable "push_auth_sa_email" {
  type        = string
  description = "Push-auth SA email granted run.invoker when allow_push_invoker is true."
  default     = ""
}
