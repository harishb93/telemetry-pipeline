# Docker Deployment for Telemetry Pipeline

This directory contains Docker configurations for building and deploying the telemetry pipeline components.

## Quick Start

Get the entire telemetry pipeline running in 3 commands:

```bash
# 1. Build all images locally
docker compose up -d --build

# 2. Wait a few seconds for services to start, then check health
curl http://localhost:8081/health

# 3. Query the API
curl http://localhost:8081/api/v1/gpus

# Stop everything
docker compose down
```

That's it! All three components (Streamer, Collector, API Gateway) will be running with proper networking and data persistence.

For more control or to use the registry, see [Building Images](#building-images) below.

## Components

### 1. Telemetry Streamer
- **Image**: `telemetry-streamer`
- **Purpose**: Streams GPU telemetry data from CSV files
- **Ports**: No external ports (communicates via message queue)
- **Configuration**: Environment variables for CSV file path, rate, workers

### 2. Telemetry Collector  
- **Image**: `telemetry-collector`
- **Purpose**: Collects and processes telemetry data via message queue
- **Ports**: 
  - `8080`: Health endpoint
  - `9000`: Message broker
  - `9091`: Metrics endpoint
- **Configuration**: Environment variables for workers, data directory, persistence

### 3. API Gateway
- **Image**: `api-gateway`
- **Purpose**: Provides REST API access to telemetry data
- **Ports**:
  - `8081`: HTTP API
  - `9092`: Metrics endpoint
- **Configuration**: Environment variables for port, data directory

## Building Images

### Quick Build (All Components)

```bash
# Build all images and push to local registry
./deploy/docker/build-and-push.sh

# Build with specific tag
./deploy/docker/build-and-push.sh -t v1.0.0

# Build with custom registry port
./deploy/docker/build-and-push.sh -p 5001
```

### Manual Build (Individual Components)

```bash
# Build telemetry-streamer
docker build -f deploy/docker/telemetry-streamer.Dockerfile -t telemetry-streamer:latest .

# Build telemetry-collector
docker build -f deploy/docker/telemetry-collector.Dockerfile -t telemetry-collector:latest .

# Build api-gateway
docker build -f deploy/docker/api-gateway.Dockerfile -t api-gateway:latest .
```

## Running Containers

### Using Docker Compose (Recommended)

A pre-configured `docker-compose.yml` is included in this directory with all services configured with proper networking, volumes, and health checks.

Start the entire stack with:

```bash
# Build and start all services
docker compose up -d --build

# View logs
docker compose logs -f

# Stop the stack
docker compose down
```

Or use the provided setup script:

```bash
# Build and start all services
./deploy/docker/setup.sh

# Start services in background
./deploy/docker/setup.sh -d

# View logs
./deploy/docker/setup.sh -l
```

The `docker-compose.yml` includes:
- **Build contexts**: Images are automatically built from Dockerfiles
- **Networking**: Custom bridge network for service-to-service communication
- **Volumes**: Persistent data storage for the collector
- **Health checks**: Automatic service monitoring
- **Environment variables**: Configured for optimal performance
- **Dependencies**: Proper startup order with `depends_on`

### Using Docker Run (Individual Containers)

#### 1. Start Telemetry Collector

```bash
docker run -d \
  --name telemetry-collector \
  -p 8080:8080 \
  -p 9000:9000 \
  -p 9091:9091 \
  -e WORKERS=4 \
  -e DATA_DIR=/data \
  -e HEALTH_PORT=8080 \
  -e BROKER_PORT=9000 \
  -e CHECKPOINT_ENABLED=true \
  -v telemetry-data:/data \
  localhost:5000/telemetry-collector:latest
```

#### 2. Start Telemetry Streamer

```bash
# Create sample CSV file
mkdir -p /tmp/telemetry-data
cat > /tmp/telemetry-data/telemetry.csv << EOF
gpu_id,utilization,temperature,memory_used
gpu-001,85.5,72.3,4096
gpu-002,90.2,75.1,8192
gpu-003,45.0,65.0,2048
EOF

docker run -d \
  --name telemetry-streamer \
  -e CSV_FILE=/data/telemetry.csv \
  -e BROKER_PORT=9000 \
  -e RATE=10.0 \
  -e WORKERS=2 \
  -v /tmp/telemetry-data/telemetry.csv:/data/telemetry.csv:ro \
  --link telemetry-collector \
  localhost:5000/telemetry-streamer:latest
```

#### 3. Start API Gateway

```bash
docker run -d \
  --name api-gateway \
  -p 8081:8081 \
  -p 9092:9092 \
  -e PORT=8081 \
  -e DATA_DIR=/data \
  -e COLLECTOR_PORT=8080 \
  -v telemetry-data:/data:ro \
  --link telemetry-collector \
  localhost:5000/api-gateway:latest
```

## Environment Variables

### Telemetry Streamer

| Variable | Description | Default |
|----------|-------------|---------|
| `CSV_FILE` | Path to CSV data file | `/data/telemetry.csv` |
| `BROKER_PORT` | Message broker port | `9000` |
| `RATE` | Messages per second per worker | `10.0` |
| `WORKERS` | Number of workers | `2` |

### Telemetry Collector

| Variable | Description | Default |
|----------|-------------|---------|
| `WORKERS` | Number of processing workers | `4` |
| `DATA_DIR` | Data storage directory | `/data` |
| `MAX_ENTRIES` | Max entries per GPU | `10000` |
| `HEALTH_PORT` | Health check port | `8080` |
| `BROKER_PORT` | Message broker port | `9000` |
| `CHECKPOINT_ENABLED` | Enable checkpointing | `true` |

### API Gateway

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | API server port | `8081` |
| `DATA_DIR` | Data directory path | `/data` |
| `COLLECTOR_PORT` | Collector health port | `8080` |

## Health Checks

### Check Component Health

```bash
# Collector health
curl http://localhost:8080/health

# API Gateway health  
curl http://localhost:8081/health

# API Gateway endpoints
curl http://localhost:8081/api/v1/gpus
curl http://localhost:8081/api/v1/gpus/gpu-001/telemetry
```

### Check Registry Contents

```bash
# List all images in local registry
curl http://localhost:5000/v2/_catalog

# List tags for specific image
curl http://localhost:5000/v2/telemetry-collector/tags/list
```

## Development Workflow

### 1. Make Code Changes

```bash
# Edit source code
vim internal/api/handlers.go
```

### 2. Rebuild and Test

```bash
# Rebuild all images
./deploy/docker/build-and-push.sh -t dev

# Update running containers
docker compose down
docker compose up -d --build
```

### 3. Debug Containers

```bash
# View logs
docker logs telemetry-collector -f
docker logs api-gateway -f
docker compose logs -f

# Execute shell in container (alpine provides /bin/sh)
docker exec -it telemetry-collector /bin/sh

# Inspect container
docker inspect telemetry-collector

# Check container processes
docker top telemetry-collector

# View resource usage
docker stats telemetry-collector
```

## Production Considerations

### 1. Multi-stage Build Optimization

The Dockerfiles use multi-stage builds for:
- **Security**: Minimal attack surface using Alpine Linux with non-root user
- **Size**: Smaller image size (~150-200MB runtime with Alpine vs 500MB+ with full OS)
- **Performance**: Faster image pulls and container startup

Base images:
- **Build stage**: `golang:1.25-alpine` - Includes Go toolchain and compilers
- **Runtime stage**: `alpine:3.18` - Minimal Linux with shell support for configuration scripts

### 2. Security Features

- **Non-root user**: All containers run as `nonroot:nonroot` (uid:gid 1000:1000)
- **Read-only entrypoint**: Configuration scripts run before application startup
- **Alpine base**: Minimal attack surface, security-focused Linux distribution
- **CA certificates**: Included for HTTPS connections
- **No package manager**: Reduces attack surface after container builds
- **CA certificates**: Included for HTTPS connections

### 3. Resource Limits

Add resource limits in production docker-compose.yml:

```yaml
services:
  telemetry-collector:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 1G
        reservations:
          cpus: '0.25'
          memory: 512M
```

Or use Make targets:

```bash
# Deploy with resource limits (see Makefile for details)
make docker-deploy
```

### 4. Monitoring and Logging

```yaml
services:
  telemetry-collector:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## Troubleshooting

### Common Issues and Solutions

#### 1. **`docker compose up -d` fails immediately**

**Symptom**: Services start but containers crash or won't stay running

**Solutions**:
```bash
# Check container logs
docker compose logs -f

# Verify images built successfully
docker images | grep telemetry

# Check if services exited
docker compose ps

# Re-run with build
docker compose up -d --build

# Check for permission errors in entrypoint scripts
ls -la deploy/docker/entrypoint-*.sh
chmod +x deploy/docker/entrypoint-*.sh
```

#### 2. **`port is already allocated` error**

**Symptom**: Error when starting docker-compose: `bind: address already in use`

**Solutions**:
```bash
# Find and stop existing containers
docker ps | grep telemetry
docker stop <container-id>

# Or change ports in docker-compose.override.yml
cp docker-compose.yml docker-compose.override.yml
# Edit docker-compose.override.yml to use different ports (e.g., 8082:8081)

# Check which process is using the port
lsof -i :8081
```

#### 3. **Registry not accessible**

**Symptom**: `error pulling image configuration` or `failed to pull image from registry`

**Solutions**:
```bash
# Start the registry
docker run -d -p 5000:5000 --name kind-registry \
  -v /tmp/registry:/var/lib/registry \
  registry:2

# Verify registry is running
curl http://localhost:5000/v2/

# Build and push images to registry
./deploy/docker/build-and-push.sh

# Check what's in registry
curl http://localhost:5000/v2/_catalog | jq .
```

#### 4. **Build failures with Alpine base image**

**Symptom**: Build errors related to missing packages or tools

**Solutions**:
```bash
# Clean build cache
docker builder prune

# Build with full output for debugging
docker compose build --no-cache telemetry-collector

# Check Dockerfile syntax
docker run --rm -i hadolint/hadolint < deploy/docker/telemetry-collector.Dockerfile

# For Alpine-specific issues, ensure apk commands are in RUN statements
# (Multi-line RUN prevents layer explosion)
```

#### 5. **Shell script permission errors in container**

**Symptom**: `Permission denied` when entrypoint.sh runs

**Solutions**:
```bash
# Ensure entrypoint scripts have executable permissions on host
chmod +x deploy/docker/entrypoint-*.sh

# Verify they're copied correctly in Dockerfile
docker exec -it telemetry-collector ls -la /entrypoint.sh

# Test entrypoint directly
docker exec -it telemetry-collector /bin/sh /entrypoint.sh
```

#### 6. **Network connectivity between services**

**Symptom**: Services can't communicate (e.g., `telemetry-collector` can't connect to broker)

**Solutions**:
```bash
# Check Docker network
docker network ls
docker network inspect telemetry-pipeline_telemetry

# Verify service DNS resolution
docker exec telemetry-collector nslookup telemetry-collector
docker exec telemetry-streamer nslookup telemetry-collector

# Test connectivity with curl inside container
docker exec -it telemetry-streamer wget -O- http://telemetry-collector:8080/health

# Check depends_on order
docker compose ps  # Should start collector first
```

#### 7. **Data persistence issues**

**Symptom**: Data lost after `docker compose down`

**Solutions**:
```bash
# Verify volumes are defined
docker volume ls | grep telemetry

# Backup volume data
docker run -v telemetry-pipeline_collector-data:/data \
  -v $(pwd):/backup alpine tar czf /backup/data.tar.gz -C / data

# Check volume permissions
docker exec -it telemetry-collector ls -la /data

# Mount volume with correct permissions
# Ensure docker-compose.yml has: volumes: - collector-data:/data
```

### Debug Workflows

#### View Complete Logs
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f telemetry-collector

# Last 50 lines
docker compose logs --tail=50 telemetry-collector

# With timestamps
docker compose logs -f --timestamps telemetry-collector
```

#### Shell Access for Debugging
```bash
# Open shell in running container
docker exec -it telemetry-collector /bin/sh

# Inside the shell, useful commands:
ps aux                          # See running processes
netstat -tlnp                   # Check listening ports
ls -la /                        # Verify file structure
cat /entrypoint.sh              # View entrypoint script
env | sort                      # View all environment variables
```

#### Inspect Container Details
```bash
# Full container information
docker inspect telemetry-collector

# Just network settings
docker inspect -f '{{json .NetworkSettings}}' telemetry-collector | jq .

# Environment variables set in container
docker inspect -f '{{json .Config.Env}}' telemetry-collector | jq .

# Entry point/command
docker inspect -f '{{json .Config.Cmd}}' telemetry-collector | jq .
```

#### Monitor Resource Usage
```bash
# Real-time stats for all containers
docker stats

# Specific container
docker stats telemetry-collector

# Store stats over time
docker stats --no-stream > docker-stats.log
```

### Performance Tuning

#### 1. **Build Performance**
```bash
# Use BuildKit for faster builds (up to 10x faster)
export DOCKER_BUILDKIT=1
docker compose build

# Or in docker-compose.yml
version: '3.8'
services:
  telemetry-collector:
    build:
      context: ../..
      dockerfile: deploy/docker/telemetry-collector.Dockerfile
      # BuildKit options for faster builds
```

#### 2. **Runtime Performance**
```bash
# Check container resource limits
docker stats --no-stream

# Monitor metrics endpoints
curl http://localhost:9091/metrics  # collector
curl http://localhost:9092/metrics  # api-gateway

# Analyze slow operations
docker exec telemetry-collector /bin/sh
time curl http://localhost:8080/health
```

#### 3. **Image Size Optimization**
```bash
# Check current image sizes
docker images | grep telemetry

# Analyze layers
docker history telemetry-collector:latest

# Remove build cache to reduce stored images
docker builder prune

# View detailed image content
docker run --rm telemetry-collector:latest sh -c "du -sh /"
```

### Integration with Make

Use the Makefile for simplified management:

```bash
# View all Docker-related targets
make help | grep -i docker

# Build all images
make docker-build

# Run Docker Compose stack
make docker-deploy

# Stop the stack
make docker-down

# View logs
docker compose logs -f

# Clean up
docker compose down -v
```

## Integration with Kubernetes

The built images can be used directly in Kubernetes:

```bash
# Update Helm values to use local registry
helm install telemetry ./deploy/helm/telemetry-pipeline \
  --set streamer.image.registry=localhost:5000 \
  --set collector.image.registry=localhost:5000 \
  --set apiGateway.image.registry=localhost:5000 \
  --set global.imageTag=latest
```

For Kind clusters, the local registry can be connected:

```bash
# Connect registry to Kind cluster
docker network connect kind kind-registry
```