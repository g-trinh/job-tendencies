output "job_name" {
  value       = google_cloud_scheduler_job.scrape.name
  description = "Scheduler job name."
}
