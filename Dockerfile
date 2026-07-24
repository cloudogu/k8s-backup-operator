# Build the manager binary
FROM golang:1.26.0 AS builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the Go source (relies on .dockerignore to filter)
COPY . .

# Copy .git files as the build process builds the current commit id into the binary via ldflags.
# We removed this entry as changes in the repository makes all cached layers invalid leading to rebuilding all layers.
# TODO resolve COMMIT_ID
#COPY .git .git

# Build
# the GOARCH has no default value to allow the binary to be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -mod=mod -a -o k8s-backup-operator ./cmd/

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
LABEL maintainer="hello@cloudogu.com" \
      NAME="k8s-backup-operator" \
      VERSION="3.3.2"

WORKDIR /
COPY --from=builder /workspace/k8s-backup-operator .

# the linter has a problem with the valid colon-syntax
# dockerfile_lint - ignore
USER 65532:65532

ENTRYPOINT ["/k8s-backup-operator"]
