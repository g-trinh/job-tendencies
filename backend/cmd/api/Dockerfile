# Dockerfile for the Job Tendencies API binary.
#
# Build context must be the backend/ directory:
#   docker build -f cmd/api/Dockerfile -t api:local .
#
# Stage 1 — Build
# ─────────────────────────────────────────────────────────────
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Cache module downloads as a separate layer (invalidated only when go.{mod,sum} change).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /bin/api ./cmd/api

# Stage 2 — Runtime
# ─────────────────────────────────────────────────────────────
# distroless/static contains no shell, no package manager, no libc —
# just enough to run a statically compiled Go binary.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/api /bin/api

# Cloud Run sets PORT=8080; fall back to 8080 if unset.
EXPOSE 8080

ENTRYPOINT ["/bin/api"]
