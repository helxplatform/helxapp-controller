# Build the ldap_plugin binary
FROM golang:1.25-bookworm AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o ldap_plugin .

# Use distroless as minimal base image
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /app/ldap_plugin .
USER 65532:65532

ENTRYPOINT ["/ldap_plugin"]
