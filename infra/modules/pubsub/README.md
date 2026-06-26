# Module: pubsub

One topic + its OIDC push subscription + a dead-letter topic and retention subscription.
Instantiate once per logical topic (`scrape-tick`, `listing-extract`). Dead-letter IAM for
the Pub/Sub service agent is wired automatically.

## Inputs

| Name | Type | Description |
|---|---|---|
| `env` | string | Environment slug. |
| `project_id` | string | GCP project id. |
| `name` | string | Logical topic name, suffixed with env. |
| `push_endpoint` | string | HTTPS worker endpoint for push delivery. |
| `push_auth_sa_email` | string | SA whose OIDC token authenticates push. |
| `push_audience` | string | OIDC audience (worker base URL). |
| `ack_deadline_seconds` | number | Push ack deadline (default 60). |
| `max_delivery_attempts` | number | Deliveries before dead-lettering (default 5). |
| `minimum_backoff` / `maximum_backoff` | string | Retry backoff bounds. |

## Outputs

| Name | Description |
|---|---|
| `topic_id` | Fully-qualified topic id. |
| `topic_name` | Short topic name. |
| `dlq_topic_id` | Dead-letter topic id. |
| `subscription_name` | Push subscription name. |
