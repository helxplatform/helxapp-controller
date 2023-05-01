# Use a lightweight base image with the necessary Golang version (e.g., golang:1.17-alpine)
FROM golang:1.20-alpine as builder

# Set the working directory
WORKDIR /workspace

# Copy the Go Modules manifests and install the dependencies
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the source code
COPY main.go main.go
COPY hack hack
COPY pkg pkg

# Build the controller binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o manager main.go

# Use a lightweight base image for the final image
FROM alpine:3.17

# Install any necessary system packages, such as CA certificates
RUN apk add --no-cache ca-certificates

# Set the working directory
WORKDIR /

# Copy the controller binary from the builder image
COPY --from=builder /workspace/manager /manager

# Define an entrypoint to start the controller
ENTRYPOINT ["/manager"]
