# Build the manager binary
FROM golang:1.19@sha256:c4a5b7270d4e632c84547475cb50a0f9c77daaf84d39f58e78bb32304a82175d as builder

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

FROM registry.access.redhat.com/ubi8/ubi-micro:latest@sha256:443db9a646aaf9374f95d266ba0c8656a52d70d0ffcc386a782cea28fa32e55d
WORKDIR /
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
