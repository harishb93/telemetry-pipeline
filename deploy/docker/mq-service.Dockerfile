# Multi-stage build for mq-service
FROM golang:1.24-alpine AS builder

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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mq-service ./cmd/mq-service

# Final stage - use alpine for small size with shell support
FROM alpine:3.18

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy the binary from builder
COPY --from=builder /app/mq-service /usr/local/bin/mq-service

# Copy entrypoint script
COPY deploy/docker/entrypoint-mq.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Expose port
EXPOSE 9090

# Create non-root user
RUN addgroup -g 1000 nonroot && adduser -u 1000 -G nonroot -s /sbin/nologin -D nonroot

# Create data directory with proper permissions
RUN mkdir -p /var/lib/mq && chown -R nonroot:nonroot /var/lib/mq

# Switch to non-root user
USER nonroot:nonroot

# Set working directory
WORKDIR /app

# Default command
ENTRYPOINT ["/entrypoint.sh"]
CMD ["mq-service"]