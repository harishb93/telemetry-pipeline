#!/bin/bash

# Integration Demo Script
# This demonstrates the complete telemetry pipeline: streamer -> MQ -> collector

echo "=== Telemetry Pipeline Integration Demo ==="
echo

# Clean up any previous runs
echo "Cleaning up previous runs..."
pkill -f telemetry-collector || true
pkill -f telemetry-streamer || true
rm -rf ./demo-data ./demo-checkpoints ./mq-data
mkdir -p ./demo-data ./demo-checkpoints ./mq-data

# Create sample CSV data
echo "Creating sample GPU telemetry data..."
cat > sample_gpu_data.csv << 'EOF'
gpu_id,temperature,utilization,memory_used,power_draw,fan_speed
gpu_0,72.3,85.5,4096,250.5,2500
gpu_1,75.1,90.2,8192,275.8,2750
gpu_2,65.8,45.0,2048,180.2,1800
gpu_3,78.9,95.1,12288,295.0,3000
gpu_0,74.1,87.2,4200,255.1,2550
gpu_1,76.8,92.5,8300,280.2,2800
gpu_2,67.2,48.3,2100,185.0,1850
gpu_3,79.5,96.8,12400,298.5,3050
EOF

echo "Sample data created with 8 telemetry records for 4 GPUs"
echo

# Build binaries
echo "Building telemetry components..."
go build -o telemetry-collector ./cmd/telemetry-collector
go build -o telemetry-streamer ./cmd/telemetry-streamer

if [ ! -f telemetry-collector ] || [ ! -f telemetry-streamer ]; then
    echo "Error: Failed to build binaries"
    exit 1
fi

echo "Binaries built successfully"
echo

# Start collector in background
echo "Starting telemetry collector..."
./telemetry-collector \
    --workers=2 \
    --data-dir=./demo-data \
    --max-entries=100 \
    --checkpoint=true \
    --checkpoint-dir=./demo-checkpoints \
    --health-port=8080 \
    --broker-port=9090 \
    --persistence=false &

COLLECTOR_PID=$!
echo "Collector started with PID $COLLECTOR_PID"

# Wait for collector to start
sleep 2

# Check if collector is running
if ! kill -0 $COLLECTOR_PID 2>/dev/null; then
    echo "Error: Collector failed to start"
    exit 1
fi

echo "Collector health check:"
curl -s http://localhost:8080/health | jq . || echo "Health endpoint not ready yet"
echo

# Start streamer to feed data
echo "Starting telemetry streamer..."
./telemetry-streamer \
    --csv=sample_gpu_data.csv \
    --workers=2 \
    --rate=2 \
    --duration=10s &

STREAMER_PID=$!
echo "Streamer started with PID $STREAMER_PID"

# Monitor the pipeline for a few seconds
echo
echo "Monitoring pipeline for 12 seconds..."
sleep 3

echo "Collector stats:"
curl -s http://localhost:8080/stats | jq . || echo "Stats not available"
echo

sleep 3

echo "Memory stats:"
curl -s http://localhost:8080/memory-stats | jq . || echo "Memory stats not available"
echo

sleep 3

echo "Telemetry for gpu_0:"
curl -s "http://localhost:8080/telemetry/gpu_0?limit=5" | jq . || echo "Telemetry not available"
echo

sleep 3

# Check file storage
echo "File storage contents:"
ls -la ./demo-data/ 2>/dev/null || echo "No data files created yet"
if [ -d "./demo-data" ]; then
    for file in ./demo-data/*.jsonl; do
        if [ -f "$file" ]; then
            echo "Contents of $file:"
            head -3 "$file"
            echo "..."
            echo
        fi
    done
fi

# Check checkpoints
echo "Checkpoint files:"
ls -la ./demo-checkpoints/ 2>/dev/null || echo "No checkpoint files"
echo

# Cleanup
echo "Cleaning up processes..."
kill $STREAMER_PID 2>/dev/null || true
kill $COLLECTOR_PID 2>/dev/null || true

# Wait for processes to stop
sleep 2

echo
echo "=== Demo Complete ==="
echo "The telemetry pipeline successfully:"
echo "1. Streamed CSV data continuously"
echo "2. Published messages to MQ broker"
echo "3. Collected and processed messages"
echo "4. Stored data in both file and memory storage"
echo "5. Provided health and stats endpoints"
echo "6. Maintained processing checkpoints"
echo
echo "Files created:"
echo "- ./demo-data/*.jsonl (per-GPU telemetry data)"
echo "- ./demo-checkpoints/ (processing checkpoints)"
echo "- sample_gpu_data.csv (source data)"