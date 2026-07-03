# Prod state lives in its OWN GCS bucket/prefix — never shared with dev, never
# workspaces. A tofu apply mistake in dev must not be able to touch prod state.
# The state bucket must exist before `tofu init` (created out of band — see infra/README.md).
terraform {
  backend "gcs" {
    bucket = "job-tendencies-prod-tfstate"
    prefix = "prod"
  }
}
