# Build the manager binary
FROM golang:1.20@sha256:098d628490c97d4419ed44a23d893f37b764f3bea06e0827183e8af4120e19be as builder

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
