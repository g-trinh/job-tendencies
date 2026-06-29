# Dockerfile for the Job Tendencies scrape-worker binary.
#
# Copies a HOST-built static linux/amd64 binary — no in-image compile.
# Build the binary first:  make build-linux   (from backend/)
# Build context = backend/:  docker build -f cmd/scrape-worker/Dockerfile -t scrape-workerocal .
FROM gcr.io/distroless/static-debian12:nonroot
COPY bin/scrape-worker /bin/scrape-worker
EXPOSE 8080
ENTRYPOINT ["/bin/scrape-worker"]
