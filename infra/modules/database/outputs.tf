output "instance_name" {
  value       = google_sql_database_instance.main.name
  description = "Cloud SQL instance name."
}

output "connection_name" {
  value       = google_sql_database_instance.main.connection_name
  description = "Instance connection name (project:region:instance) for the Cloud SQL connector."
}

output "database_name" {
  value       = google_sql_database.app.name
  description = "Application database name."
}
