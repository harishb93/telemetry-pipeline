# Multi-stage build for telemetry-collector
FROM golang:1.21-alpine AS builder

# Install git for go mod download
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o telemetry-collector ./cmd/telemetry-collector

# Final stage - use distroless for security and minimal size
FROM gcr.io/distroless/static-debian11:nonroot

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/telemetry-collector /usr/local/bin/telemetry-collector

# Copy entrypoint script
COPY deploy/docker/entrypoint-collector.sh /entrypoint.sh

# Expose ports
EXPOSE 8080 9000 9091

# Create non-root user directories
USER nonroot:nonroot

# Create data directory with proper permissions
WORKDIR /app

# Default command
ENTRYPOINT ["/entrypoint.sh"]
CMD ["telemetry-collector"]