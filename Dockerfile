# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY version/ version/
COPY vendor/ vendor/
# Build
ARG secret_injector_version=dev
RUN CGO_ENABLED=0 \
    GO111MODULE=on \
    go build \
    -ldflags "-X \"github.com/1Password/kubernetes-secret-injector/version.Version=$secret_injector_version\"" \
    -mod vendor \
    -a -o injector ./cmd

# Use distroless as minimal base image to package the secret-injector binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/injector .

# install the prestop script
COPY ./prestop.sh .

USER 65532:65532

ENTRYPOINT ["/injector"]
