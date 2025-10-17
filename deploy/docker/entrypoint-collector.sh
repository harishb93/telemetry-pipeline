#!/bin/sh
set -e

# Default values
WORKERS=${WORKERS:-"4"}
DATA_DIR=${DATA_DIR:-"/data"}
MAX_ENTRIES=${MAX_ENTRIES:-"10000"}
HEALTH_PORT=${HEALTH_PORT:-"8080"}
BROKER_PORT=${BROKER_PORT:-"9000"}
CHECKPOINT_ENABLED=${CHECKPOINT_ENABLED:-"true"}

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

if [ -n "$BROKER_PORT" ]; then
    ARGS="$ARGS -broker-port=$BROKER_PORT"
fi

if [ "$CHECKPOINT_ENABLED" = "true" ]; then
    ARGS="$ARGS -checkpoint"
fi

# Add any additional arguments passed to the container
ARGS="$ARGS $@"

# Log the command being executed
echo "Starting telemetry-collector with arguments: $ARGS"

# Execute the binary
exec /usr/local/bin/telemetry-collector $ARGS