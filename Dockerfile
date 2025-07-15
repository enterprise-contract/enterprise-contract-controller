# Build the manager binary
FROM docker.io/library/golang:1.23.6 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY api/go.mod api/go.mod
COPY api/go.sum api/go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

FROM registry.access.redhat.com/ubi8/ubi-micro:latest@sha256:c1b052e22f088693b2420bb62a47799ebf90bce8f63385e13e74dbdc3d60dec7
WORKDIR /
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
