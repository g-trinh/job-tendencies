# Dev state lives in its own GCS bucket/prefix — never shared with prod.
# The state bucket must exist before `tofu init` (created out of band — see infra/README.md).
terraform {
  backend "gcs" {
    bucket = "job-tendencies-dev-tfstate"
    prefix = "dev"
  }
}
