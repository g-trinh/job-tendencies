output "api_service_uri" {
  value       = module.api.service_uri
  description = "API Cloud Run URI."
}

output "scrape_worker_uri" {
  value       = module.scrape_worker.service_uri
  description = "scrape-worker Cloud Run URI."
}

output "extract_worker_uri" {
  value       = module.extract_worker.service_uri
  description = "extract-worker Cloud Run URI."
}

output "db_connection_name" {
  value       = module.database.connection_name
  description = "Cloud SQL connection name for the connector."
}

output "raw_bucket" {
  value       = module.blobstore.bucket_name
  description = "Raw payload bucket name."
}

output "scrape_tick_topic" {
  value       = module.scrape_tick.topic_name
  description = "scrape.tick topic name."
}

output "listing_extract_topic" {
  value       = module.listing_extract.topic_name
  description = "listing.extract topic name."
}

output "hosting_site_id" {
  value       = google_firebase_hosting_site.spa.site_id
  description = "Firebase Hosting site id (deploy target for frontend/.firebaserc)."
}

output "hosting_default_url" {
  value       = google_firebase_hosting_site.spa.default_url
  description = "Default Firebase Hosting URL for the SPA."
}
