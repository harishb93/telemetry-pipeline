#!/bin/sh
set -e

# Default values
PORT=${PORT:-"8081"}
DATA_DIR=${DATA_DIR:-"/data"}
COLLECTOR_PORT=${COLLECTOR_PORT:-"8080"}
LOG_LEVEL=${LOG_LEVEL:-"INFO"}
LOG_FORMAT=${LOG_FORMAT:-"text"}

# Build command line arguments
ARGS=""

if [ -n "$PORT" ]; then
    ARGS="$ARGS -port=$PORT"
fi

if [ -n "$DATA_DIR" ]; then
    ARGS="$ARGS -data-dir=$DATA_DIR"
fi

if [ -n "$COLLECTOR_PORT" ]; then
    ARGS="$ARGS -collector-port=$COLLECTOR_PORT"
fi

# Add any additional arguments passed to the container
ARGS="$ARGS $@"

# Set logging environment variables
export LOG_LEVEL="$LOG_LEVEL"
export LOG_FORMAT="$LOG_FORMAT"

# Log the command being executed
echo "Starting api-gateway with arguments: $ARGS"
echo "Log level: $LOG_LEVEL, format: $LOG_FORMAT"

# Execute the binary
exec /usr/local/bin/api-gateway $ARGS