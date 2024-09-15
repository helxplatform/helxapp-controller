# Use the official Golang image to create a build artifact.
#FROM golang:1.20 as builder
FROM golang:1.20-bullseye as builder

# Copy local code to the container image.
WORKDIR /app
COPY . .

# Build the binary.
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -v -o ldap_plugin

# Use a Docker multi-stage build to create a lean production image.
#FROM alpine:latest
#RUN apk --no-cache add ca-certificates

#WORKDIR /root/
#COPY --from=builder /app/ldap_plugin .
#COPY ldap_plugin .

# Run the web service on container startup.
CMD ["/app/ldap_plugin"]
