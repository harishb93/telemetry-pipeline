#!/bin/sh
set -e

# Default values
CSV_FILE=${CSV_FILE:-"/app/data/telemetry.csv"}
BROKER_URL=${BROKER_URL:-"http://localhost:9090"}
TOPIC=${TOPIC:-"telemetry"}
RATE=${RATE:-"10.0"}
WORKERS=${WORKERS:-"2"}
LOG_LEVEL=${LOG_LEVEL:-"INFO"}
LOG_FORMAT=${LOG_FORMAT:-"text"}

# Build command line arguments
ARGS=""

if [ -n "$CSV_FILE" ]; then
    ARGS="$ARGS -csv-file=$CSV_FILE"
fi

if [ -n "$BROKER_URL" ]; then
    ARGS="$ARGS -broker-url=$BROKER_URL"
fi

if [ -n "$TOPIC" ]; then
    ARGS="$ARGS -topic=$TOPIC"
fi

if [ -n "$RATE" ]; then
    ARGS="$ARGS -rate=$RATE"
fi

if [ -n "$WORKERS" ]; then
    ARGS="$ARGS -workers=$WORKERS"
fi

# Add any additional arguments passed to the container
ARGS="$ARGS $@"

# Set logging environment variables
export LOG_LEVEL="$LOG_LEVEL"
export LOG_FORMAT="$LOG_FORMAT"

# Log the command being executed
echo "Starting telemetry-streamer with arguments: $ARGS"
echo "Log level: $LOG_LEVEL, format: $LOG_FORMAT"

# Execute the binary
exec /usr/local/bin/telemetry-streamer $ARGS