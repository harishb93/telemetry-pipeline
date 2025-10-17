# Docker Deployment for Telemetry Pipeline

This directory contains Docker configurations for building and deploying the telemetry pipeline components.

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

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  telemetry-collector:
    image: localhost:5000/telemetry-collector:latest
    ports:
      - "8080:8080"
      - "9000:9000"
      - "9091:9091"
    environment:
      - WORKERS=4
      - DATA_DIR=/data
      - HEALTH_PORT=8080
      - BROKER_PORT=9000
      - CHECKPOINT_ENABLED=true
    volumes:
      - collector-data:/data
    networks:
      - telemetry

  telemetry-streamer:
    image: localhost:5000/telemetry-streamer:latest
    environment:
      - CSV_FILE=/data/telemetry.csv
      - BROKER_PORT=9000
      - RATE=10.0
      - WORKERS=2
    volumes:
      - ./sample-data/telemetry.csv:/data/telemetry.csv:ro
    depends_on:
      - telemetry-collector
    networks:
      - telemetry

  api-gateway:
    image: localhost:5000/api-gateway:latest
    ports:
      - "8081:8081"
      - "9092:9092"
    environment:
      - PORT=8081
      - DATA_DIR=/data
      - COLLECTOR_PORT=8080
    volumes:
      - collector-data:/data:ro
    depends_on:
      - telemetry-collector
    networks:
      - telemetry

volumes:
  collector-data:

networks:
  telemetry:
    driver: bridge
```

Run the stack:

```bash
docker-compose up -d
```

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
docker-compose down
docker-compose up -d
```

### 3. Debug Containers

```bash
# View logs
docker logs telemetry-collector -f
docker logs api-gateway -f

# Execute shell in container (if using alpine base)
docker exec -it telemetry-collector /bin/sh

# Inspect container
docker inspect telemetry-collector
```

## Production Considerations

### 1. Multi-stage Build Optimization

The Dockerfiles use multi-stage builds with distroless base images for:
- **Security**: Minimal attack surface with no shell or package manager
- **Size**: Smaller image size (~10-20MB vs 100MB+ with full OS)
- **Performance**: Faster image pulls and container startup

### 2. Security Features

- **Non-root user**: All containers run as `nonroot:nonroot`
- **Read-only filesystem**: Containers use read-only root filesystem
- **Distroless base**: No shell or unnecessary binaries
- **CA certificates**: Included for HTTPS connections

### 3. Resource Limits

Add resource limits in production:

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

### Common Issues

1. **Registry not accessible**
   ```bash
   # Check registry status
   docker ps | grep registry
   curl http://localhost:5000/v2/
   ```

2. **Build failures**
   ```bash
   # Check Docker daemon
   docker info
   
   # Clean build cache
   docker builder prune
   ```

3. **Container startup issues**
   ```bash
   # Check logs
   docker logs <container-name>
   
   # Check entrypoint script permissions
   ls -la deploy/docker/entrypoint-*.sh
   ```

4. **Network connectivity**
   ```bash
   # Check container networking
   docker network ls
   docker inspect <container-name>
   ```

### Performance Tuning

1. **Build performance**
   ```bash
   # Use buildkit for faster builds
   export DOCKER_BUILDKIT=1
   docker build ...
   ```

2. **Runtime performance**
   ```bash
   # Monitor container resources
   docker stats
   
   # Check container metrics
   curl http://localhost:9091/metrics  # collector
   curl http://localhost:9092/metrics  # api-gateway
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