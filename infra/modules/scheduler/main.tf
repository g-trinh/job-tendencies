# Cloud Scheduler cron -> Pub/Sub scrape.tick. Created paused until Phase 7.
resource "google_cloud_scheduler_job" "scrape" {
  project   = var.project_id
  name      = "scrape-schedule-${var.env}"
  region    = var.region
  schedule  = var.cron
  time_zone = var.time_zone
  paused    = var.paused

  pubsub_target {
    topic_name = var.topic_id
    data       = base64encode(jsonencode({ trigger = "scheduled" }))
  }
}
