# Build the manager binary
FROM golang:1.26.10@sha256:672c944cfd07599b5f1bb179869270cdc94c6254e41db4d431259693cb286b82 AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY version/ version/

# Download dependencies
RUN go mod download

# Build
ARG secret_injector_version=dev
RUN CGO_ENABLED=0 \
    GO111MODULE=on \
    go build \
    -ldflags "-X \"github.com/1Password/kubernetes-secrets-injector/version.Version=$secret_injector_version\"" \
    -a -o injector ./cmd

# Use distroless as minimal base image to package the secrets-injector binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:d62067879de890a1def9da0e7766b7d20810a5e5555cef8506b752437d7a3e88
WORKDIR /
COPY --from=builder /workspace/injector .

# install the prestop script
COPY ./prestop.sh .

USER 65532:65532

ENTRYPOINT ["/injector"]
