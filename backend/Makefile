# Job Tendencies — backend Makefile
#
# All targets run from the backend/ directory.
#
# ── Open Question #1 ──────────────────────────────────────────────────────────
# IMAGE_REGISTRY is intentionally empty. The Artifact Registry repo name and
# GCP project id are unresolved (see docs/plan/tech-breakdown-phase-0.md §Open
# Questions). Set these before running image-* or deploy-dev-* targets:
#
#   make image-api IMAGE_REGISTRY=europe-west9-docker.pkg.dev/MY_PROJECT/MY_REPO
#
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: build test lint migrate sqlc \
        image-api image-scrape-worker image-extract-worker \
        deploy-dev-api deploy-dev-scrape-worker deploy-dev-extract-worker \
        help

# ── Configuration ─────────────────────────────────────────────────────────────

# Artifact Registry image base path. Override via env or make flag.
# Example: europe-west9-docker.pkg.dev/my-project/job-tendencies
IMAGE_REGISTRY ?=
IMAGE_TAG      ?= latest

# GCP settings for Cloud Run deploys.
GCP_PROJECT ?= $(shell gcloud config get-value project 2>/dev/null)
REGION      ?= europe-west9

# sqlc version pinned for reproducibility.
SQLC_VERSION  := v1.27.0
# goose version pinned for reproducibility.
GOOSE_VERSION := v3.27.1

# ── Core targets ──────────────────────────────────────────────────────────────

## build: compile all three binaries into bin/
build:
	go build -o bin/ ./cmd/...

## test: run all unit tests
test:
	go test ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## migrate: apply pending SQL migrations (requires DATABASE_URL)
migrate:
	@if [ -z "$(DATABASE_URL)" ]; then \
	    echo "ERROR: DATABASE_URL is not set"; exit 1; \
	fi
	go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) \
	    -dir migrations postgres "$(DATABASE_URL)" up

## sqlc: regenerate database query code from migrations/ + internal/infra/queries/
sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

# ── Image targets (build locally; push blocked on Open Question #1) ──────────

## image-api: build the api Docker image locally
image-api:
	@if [ -z "$(IMAGE_REGISTRY)" ]; then \
	    echo "ERROR: IMAGE_REGISTRY is not set (Open Question #1 — see Makefile header)"; exit 1; \
	fi
	docker build \
	    --platform linux/amd64 \
	    -f cmd/api/Dockerfile \
	    -t $(IMAGE_REGISTRY)/api:$(IMAGE_TAG) \
	    .
	docker push $(IMAGE_REGISTRY)/api:$(IMAGE_TAG)

## image-scrape-worker: build the scrape-worker Docker image locally
image-scrape-worker:
	@if [ -z "$(IMAGE_REGISTRY)" ]; then \
	    echo "ERROR: IMAGE_REGISTRY is not set (Open Question #1 — see Makefile header)"; exit 1; \
	fi
	docker build \
	    --platform linux/amd64 \
	    -f cmd/scrape-worker/Dockerfile \
	    -t $(IMAGE_REGISTRY)/scrape-worker:$(IMAGE_TAG) \
	    .
	docker push $(IMAGE_REGISTRY)/scrape-worker:$(IMAGE_TAG)

## image-extract-worker: build the extract-worker Docker image locally
image-extract-worker:
	@if [ -z "$(IMAGE_REGISTRY)" ]; then \
	    echo "ERROR: IMAGE_REGISTRY is not set (Open Question #1 — see Makefile header)"; exit 1; \
	fi
	docker build \
	    --platform linux/amd64 \
	    -f cmd/extract-worker/Dockerfile \
	    -t $(IMAGE_REGISTRY)/extract-worker:$(IMAGE_TAG) \
	    .
	docker push $(IMAGE_REGISTRY)/extract-worker:$(IMAGE_TAG)

# ── Deploy targets (blocked on Open Question #1) ──────────────────────────────
# These targets deploy to the Cloud Run dev environment.
# They require IMAGE_REGISTRY, GCP_PROJECT, and the image to have been pushed.

## deploy-dev-api: deploy api to Cloud Run dev
deploy-dev-api:
	@if [ -z "$(IMAGE_REGISTRY)" ]; then \
	    echo "ERROR: IMAGE_REGISTRY is not set (Open Question #1 — see Makefile header)"; exit 1; \
	fi
	gcloud run deploy api \
	    --image $(IMAGE_REGISTRY)/api:$(IMAGE_TAG) \
	    --region $(REGION) \
	    --project $(GCP_PROJECT) \
	    --platform managed \
	    --no-allow-unauthenticated

## deploy-dev-scrape-worker: deploy scrape-worker to Cloud Run dev
deploy-dev-scrape-worker:
	@if [ -z "$(IMAGE_REGISTRY)" ]; then \
	    echo "ERROR: IMAGE_REGISTRY is not set (Open Question #1 — see Makefile header)"; exit 1; \
	fi
	gcloud run deploy scrape-worker \
	    --image $(IMAGE_REGISTRY)/scrape-worker:$(IMAGE_TAG) \
	    --region $(REGION) \
	    --project $(GCP_PROJECT) \
	    --platform managed \
	    --no-allow-unauthenticated \
	    --max-instances 1 \
	    --concurrency 1

## deploy-dev-extract-worker: deploy extract-worker to Cloud Run dev
deploy-dev-extract-worker:
	@if [ -z "$(IMAGE_REGISTRY)" ]; then \
	    echo "ERROR: IMAGE_REGISTRY is not set (Open Question #1 — see Makefile header)"; exit 1; \
	fi
	gcloud run deploy extract-worker \
	    --image $(IMAGE_REGISTRY)/extract-worker:$(IMAGE_TAG) \
	    --region $(REGION) \
	    --project $(GCP_PROJECT) \
	    --platform managed \
	    --no-allow-unauthenticated

# ── Help ──────────────────────────────────────────────────────────────────────

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | column -t -s ':'
