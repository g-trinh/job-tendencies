# Dockerfile for the Job Tendencies extract-worker binary.
#
# Build context must be the backend/ directory:
#   docker build -f cmd/extract-worker/Dockerfile -t extract-worker:local .
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
    -o /bin/extract-worker ./cmd/extract-worker

# Stage 2 — Runtime
# ─────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/extract-worker /bin/extract-worker

EXPOSE 8080

ENTRYPOINT ["/bin/extract-worker"]
