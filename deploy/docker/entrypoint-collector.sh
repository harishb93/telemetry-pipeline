#!/bin/sh
set -e

# Default values
WORKERS=${WORKERS:-"1"}
DATA_DIR=${DATA_DIR:-"/data"}
MAX_ENTRIES=${MAX_ENTRIES:-"10000"}
HEALTH_PORT=${HEALTH_PORT:-"8080"}
MQ_URL=${MQ_URL:-"http://localhost:9090"}
TOPIC=${TOPIC:-"telemetry"}
CHECKPOINT_ENABLED=${CHECKPOINT_ENABLED:-"true"}
LOG_LEVEL=${LOG_LEVEL:-"INFO"}
LOG_FORMAT=${LOG_FORMAT:-"text"}

# Build command line arguments
ARGS=""

if [ -n "$WORKERS" ]; then
    ARGS="$ARGS -workers=$WORKERS"
fi

if [ -n "$DATA_DIR" ]; then
    ARGS="$ARGS -data-dir=$DATA_DIR"
fi

if [ -n "$MAX_ENTRIES" ]; then
    ARGS="$ARGS -max-entries=$MAX_ENTRIES"
fi

if [ -n "$HEALTH_PORT" ]; then
    ARGS="$ARGS -health-port=$HEALTH_PORT"
fi

if [ -n "$MQ_URL" ]; then
    ARGS="$ARGS -mq-url=$MQ_URL"
fi

if [ -n "$TOPIC" ]; then
    ARGS="$ARGS -mq-topic=$TOPIC"
fi

if [ "$CHECKPOINT_ENABLED" = "true" ]; then
    ARGS="$ARGS -checkpoint"
fi

# Add any additional arguments passed to the container
ARGS="$ARGS $@"

# Set logging environment variables
export LOG_LEVEL="$LOG_LEVEL"
export LOG_FORMAT="$LOG_FORMAT"

# Log the command being executed
echo "Starting telemetry-collector with arguments: $ARGS"
echo "Log level: $LOG_LEVEL, format: $LOG_FORMAT"

# Execute the binary
exec /usr/local/bin/telemetry-collector $ARGS