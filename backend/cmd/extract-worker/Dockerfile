# Dockerfile for the Job Tendencies extract-worker binary.
#
# Copies a HOST-built static linux/amd64 binary — no in-image compile.
# Build the binary first:  make build-linux   (from backend/)
# Build context = backend/:  docker build -f cmd/extract-worker/Dockerfile -t extract-workerocal .
FROM gcr.io/distroless/static-debian12:nonroot
COPY bin/extract-worker /bin/extract-worker
EXPOSE 8080
ENTRYPOINT ["/bin/extract-worker"]
