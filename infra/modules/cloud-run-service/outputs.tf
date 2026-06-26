output "sa_email" {
  value       = google_service_account.svc.email
  description = "Runtime service account email."
}

output "service_name" {
  value       = google_cloud_run_v2_service.svc.name
  description = "Cloud Run service name."
}

output "service_uri" {
  value       = google_cloud_run_v2_service.svc.uri
  description = "Public/internal URI of the service."
}
