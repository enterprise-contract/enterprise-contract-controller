# Build the manager binary
FROM docker.io/library/golang:1.21@sha256:2ff79bcdaff74368a9fdcb06f6599e54a71caf520fd2357a55feddd504bcaffb as builder

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
