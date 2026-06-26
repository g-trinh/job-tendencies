# Module: scheduler

Cloud Scheduler cron job publishing to the `scrape.tick` topic. Time zone `Europe/Paris`.
Created **paused** by default — Phase 7 unpauses it.

## Inputs

| Name | Type | Description |
|---|---|---|
| `env` | string | Environment slug. |
| `region` | string | Scheduler region. |
| `project_id` | string | GCP project id. |
| `topic_id` | string | Target Pub/Sub topic id. |
| `cron` | string | Cron expression (default hourly). |
| `time_zone` | string | IANA TZ (default `Europe/Paris`). |
| `paused` | bool | Create paused (default true). |

## Outputs

| Name | Description |
|---|---|
| `job_name` | Scheduler job name. |
