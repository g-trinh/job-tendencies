# One topic + its OIDC push subscription + a dead-letter topic/subscription.
# Instantiated once per logical topic (scrape-tick, listing-extract).

data "google_project" "this" {
  project_id = var.project_id
}

locals {
  # Built-in Pub/Sub service agent — needs publish on the DLQ and subscribe on the
  # source subscription for dead-lettering to work.
  pubsub_sa = "serviceAccount:service-${data.google_project.this.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

resource "google_pubsub_topic" "main" {
  project = var.project_id
  name    = "${var.name}-${var.env}"
}

resource "google_pubsub_topic" "dlq" {
  project = var.project_id
  name    = "${var.name}-dlq-${var.env}"
}

resource "google_pubsub_subscription" "push" {
  project = var.project_id
  name    = "${var.name}-push-${var.env}"
  topic   = google_pubsub_topic.main.id

  push_config {
    push_endpoint = var.push_endpoint
    oidc_token {
      service_account_email = var.push_auth_sa_email
      audience              = var.push_audience
    }
  }

  ack_deadline_seconds = var.ack_deadline_seconds

  retry_policy {
    minimum_backoff = var.minimum_backoff
    maximum_backoff = var.maximum_backoff
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dlq.id
    max_delivery_attempts = var.max_delivery_attempts
  }
}

# Plain pull subscription on the DLQ so dead-lettered messages are retained for inspection.
resource "google_pubsub_subscription" "dlq" {
  project = var.project_id
  name    = "${var.name}-dlq-sub-${var.env}"
  topic   = google_pubsub_topic.dlq.id

  message_retention_duration = "604800s" # 7 days
}

# Dead-lettering IAM: Pub/Sub agent publishes to the DLQ and acks the source subscription.
resource "google_pubsub_topic_iam_member" "dlq_publisher" {
  project = var.project_id
  topic   = google_pubsub_topic.dlq.name
  role    = "roles/pubsub.publisher"
  member  = local.pubsub_sa
}

resource "google_pubsub_subscription_iam_member" "source_subscriber" {
  project      = var.project_id
  subscription = google_pubsub_subscription.push.name
  role         = "roles/pubsub.subscriber"
  member       = local.pubsub_sa
}
