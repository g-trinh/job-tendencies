# Dockerfile for the Job Tendencies api binary.
#
# Copies a HOST-built static linux/amd64 binary — no in-image compile.
# Build the binary first:  make build-linux   (from backend/)
# Build context = backend/:  docker build -f cmd/api/Dockerfile -t apiocal .
FROM gcr.io/distroless/static-debian12:nonroot
COPY bin/api /bin/api
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
