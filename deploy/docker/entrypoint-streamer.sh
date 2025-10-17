#!/bin/sh
set -e

# Default values
CSV_FILE=${CSV_FILE:-"/data/telemetry.csv"}
BROKER_PORT=${BROKER_PORT:-"9000"}
RATE=${RATE:-"10.0"}
WORKERS=${WORKERS:-"2"}

# Build command line arguments
ARGS=""

if [ -n "$CSV_FILE" ]; then
    ARGS="$ARGS -csv-file=$CSV_FILE"
fi

if [ -n "$BROKER_PORT" ]; then
    # Construct broker URL from collector service and port
    BROKER_URL="http://telemetry-collector:$BROKER_PORT"
    ARGS="$ARGS -broker-url=$BROKER_URL"
fi

if [ -n "$RATE" ]; then
    ARGS="$ARGS -rate=$RATE"
fi

if [ -n "$WORKERS" ]; then
    ARGS="$ARGS -workers=$WORKERS"
fi

# Add any additional arguments passed to the container
ARGS="$ARGS $@"

# Log the command being executed
echo "Starting telemetry-streamer with arguments: $ARGS"

# Execute the binary
exec /usr/local/bin/telemetry-streamer $ARGS