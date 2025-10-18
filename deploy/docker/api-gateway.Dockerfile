# Multi-stage build for api-gateway
FROM golang:1.25-alpine AS builder

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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./cmd/api-gateway

# Final stage - use alpine for small size with shell support
FROM alpine:3.18

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy the binary from builder
COPY --from=builder /app/api-gateway /usr/local/bin/api-gateway

# Copy entrypoint script
COPY deploy/docker/entrypoint-api-gateway.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Expose ports
EXPOSE 8081 9092

# Create non-root user
RUN addgroup -g 1000 nonroot && adduser -u 1000 -G nonroot -s /sbin/nologin -D nonroot

# Create data directory with proper permissions (read-only for API gateway)
RUN mkdir -p /data && chown -R nonroot:nonroot /data

# Switch to non-root user
USER nonroot:nonroot

# Set working directory
WORKDIR /app

# Default command
ENTRYPOINT ["/entrypoint.sh"]
CMD ["api-gateway"]