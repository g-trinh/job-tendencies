# Module: cloud-run-service

Reusable Cloud Run v2 service: its own least-privilege runtime service account, the
service itself (env + Secret Manager wiring), and an optional `run.invoker` binding for
the push-auth SA. No `allUsers` invoker, ever. Instantiated 3× (api, scrape-worker,
extract-worker). Data-plane IAM (cloudsql.client, pubsub.publisher, storage, secretAccessor)
is granted by the environment that composes the modules, not here.

## Inputs

| Name | Type | Description |
|---|---|---|
| `name` | string | Service base name; also the SA id (`<name>-sa`). |
| `env` | string | Environment slug. |
| `region` | string | Cloud Run region. |
| `project_id` | string | GCP project id. |
| `image` | string | Container image (defaults to Cloud Run hello). |
| `cpu` / `memory` | string | Resource limits. |
| `min_instances` / `max_instances` | number | Scaling bounds (scrape-worker max = 1). |
| `concurrency` | number | Per-instance concurrency (scrape-worker = 1). |
| `ingress` | string | Ingress setting (workers internal-only, api all). |
| `env_vars` | map(string) | Plain env vars. |
| `secret_env` | map(object) | Secret-sourced env vars. |
| `allow_push_invoker` | bool | Grant run.invoker to the push-auth SA. |
| `push_auth_sa_email` | string | Push-auth SA email. |

## Outputs

| Name | Description |
|---|---|
| `sa_email` | Runtime SA email. |
| `service_name` | Cloud Run service name. |
| `service_uri` | Service URI. |
