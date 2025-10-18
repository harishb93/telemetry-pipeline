#!/bin/sh
set -e

# Default values
GRPC_PORT=${GRPC_PORT:-"9091"}
HTTP_PORT=${HTTP_PORT:-"9090"}
PERSISTENCE_ENABLED=${PERSISTENCE_ENABLED:-"true"}
PERSISTENCE_DIR=${PERSISTENCE_DIR:-"/var/lib/mq"}
MAX_RETRIES=${MAX_RETRIES:-"3"}
ACK_TIMEOUT=${ACK_TIMEOUT:-"30s"}
LOG_LEVEL=${LOG_LEVEL:-"INFO"}
LOG_FORMAT=${LOG_FORMAT:-"text"}

# Build command line arguments
ARGS=""

if [ -n "$GRPC_PORT" ]; then
    ARGS="$ARGS -grpc-port=$GRPC_PORT"
fi

if [ -n "$HTTP_PORT" ]; then
    ARGS="$ARGS -http-port=$HTTP_PORT"
fi

if [ "$PERSISTENCE_ENABLED" = "true" ]; then
    ARGS="$ARGS -persistence"
    if [ -n "$PERSISTENCE_DIR" ]; then
        ARGS="$ARGS -persistence-dir=$PERSISTENCE_DIR"
    fi
fi



if [ -n "$MAX_RETRIES" ]; then
    ARGS="$ARGS -max-retries=$MAX_RETRIES"
fi

if [ -n "$ACK_TIMEOUT" ]; then
    ARGS="$ARGS -ack-timeout=$ACK_TIMEOUT"
fi

# Add any additional arguments passed to the container (if not the default CMD)
if [ "$#" -gt 0 ] && [ "$1" != "mq-service" ]; then
    ARGS="$ARGS $@"
fi

# Set logging environment variables
export LOG_LEVEL="$LOG_LEVEL"
export LOG_FORMAT="$LOG_FORMAT"

# Log the command being executed
echo "Starting mq-service with arguments: $ARGS"
echo "Log level: $LOG_LEVEL, format: $LOG_FORMAT"

# Execute the binary
exec /usr/local/bin/mq-service $ARGS