output "secret_id" {
  value       = google_secret_manager_secret.this.secret_id
  description = "Full secret id (with env suffix)."
}

output "secret_name" {
  value       = google_secret_manager_secret.this.name
  description = "Fully-qualified secret resource name."
}
