output "bucket_name" {
  value       = google_storage_bucket.raw.name
  description = "Raw payload bucket name."
}

output "bucket_url" {
  value       = google_storage_bucket.raw.url
  description = "gs:// URL of the raw bucket."
}
