# Module: secrets

Declares a Secret Manager secret container and grants `secretAccessor` to the SAs that
read it. **The secret value is never in tf/tfvars/state** — add it out of band:

```sh
gcloud secrets versions add claude-api-key-dev --project=<project> --data-file=- <<<"$KEY"
```

## Inputs

| Name | Type | Description |
|---|---|---|
| `env` | string | Environment slug. |
| `project_id` | string | GCP project id. |
| `secret_id` | string | Logical id (e.g. `claude-api-key`), suffixed with env. |
| `accessor_members` | list(string) | Members granted `roles/secretmanager.secretAccessor`. |

## Outputs

| Name | Description |
|---|---|
| `secret_id` | Full secret id (with env suffix). |
| `secret_name` | Fully-qualified resource name. |
