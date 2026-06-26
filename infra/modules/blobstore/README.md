# Module: blobstore

Private GCS bucket for raw HTML/JSON payloads. Uniform bucket-level access, public
access prevention enforced. Grants objectCreator / objectViewer to the SAs that need them.

## Inputs

| Name | Type | Description |
|---|---|---|
| `env` | string | Environment slug. |
| `region` | string | Bucket location. |
| `project_id` | string | GCP project id. |
| `object_creator_members` | list(string) | Members granted `roles/storage.objectCreator`. |
| `object_viewer_members` | list(string) | Members granted `roles/storage.objectViewer`. |
| `force_destroy` | bool | Allow deleting a non-empty bucket (false in prod). |

## Outputs

| Name | Description |
|---|---|
| `bucket_name` | Raw payload bucket name. |
| `bucket_url` | `gs://` URL. |
