variable "env" {
  type        = string
  description = "Environment slug (dev | prod)."
}

variable "region" {
  type        = string
  description = "Cloud Scheduler region."
}

variable "project_id" {
  type        = string
  description = "GCP project id."
}

variable "topic_id" {
  type        = string
  description = "Pub/Sub topic id the job publishes to (scrape.tick)."
}

variable "cron" {
  type        = string
  description = "Cron schedule expression."
  default     = "0 * * * *"
}

variable "time_zone" {
  type        = string
  description = "IANA time zone."
  default     = "Europe/Paris"
}

variable "paused" {
  type        = bool
  description = "Create the job paused (true until Phase 7 turns scraping on)."
  default     = true
}
