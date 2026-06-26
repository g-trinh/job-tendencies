output "topic_id" {
  value       = google_pubsub_topic.main.id
  description = "Fully-qualified topic id (projects/.../topics/...)."
}

output "topic_name" {
  value       = google_pubsub_topic.main.name
  description = "Short topic name."
}

output "dlq_topic_id" {
  value       = google_pubsub_topic.dlq.id
  description = "Dead-letter topic id."
}

output "subscription_name" {
  value       = google_pubsub_subscription.push.name
  description = "Push subscription name."
}
