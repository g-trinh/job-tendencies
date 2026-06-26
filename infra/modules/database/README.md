# Module: database

Cloud SQL Postgres 16 instance, the `job_tendencies` database, and one passwordless
IAM DB user per worker service account. IAM authentication on, no public IP, backups on.

## Inputs

| Name | Type | Description |
|---|---|---|
| `env` | string | Environment slug (`dev` \| `prod`). |
| `region` | string | GCP region. |
| `db_tier` | string | Machine tier (e.g. `db-g1-small`). |
| `iam_user_emails` | list(string) | SA emails to create IAM DB users for. |
| `deletion_protection` | bool | Block instance deletion (true in prod). |

## Outputs

| Name | Description |
|---|---|
| `instance_name` | Cloud SQL instance name. |
| `connection_name` | `project:region:instance` for the Cloud SQL connector. |
| `database_name` | Application database name. |
