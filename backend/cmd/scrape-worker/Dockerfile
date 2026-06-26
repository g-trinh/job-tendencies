# Dockerfile for the Job Tendencies scrape-worker binary.
#
# Build context must be the backend/ directory:
#   docker build -f cmd/scrape-worker/Dockerfile -t scrape-worker:local .
#
# Cloud Run settings (set at deploy time, not here):
#   --max-instances=1 --concurrency=1
#
# Stage 1 — Build
# ─────────────────────────────────────────────────────────────
FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /bin/scrape-worker ./cmd/scrape-worker

# Stage 2 — Runtime
# ─────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/scrape-worker /bin/scrape-worker

EXPOSE 8080

ENTRYPOINT ["/bin/scrape-worker"]
