# Quickstart Guide

Get the GPU Telemetry Pipeline up and running in minutes. Choose your preferred deployment method below.

---

## Prerequisites

Before you start, ensure you have the following installed:

### For Docker Setup
- **Docker**: [Install Docker](https://docs.docker.com/get-docker/) (version 20.10+)
- **Docker Compose**: Usually included with Docker Desktop

### For Kubernetes (Kind) Setup
- **Docker**: [Install Docker](https://docs.docker.com/get-docker/)
- **Kind**: [Install Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) (version 0.20+)
- **kubectl**: [Install kubectl](https://kubernetes.io/docs/tasks/tools/) (version 1.27+)
- **Helm**: [Install Helm](https://helm.sh/docs/intro/install/) (version 3.12+)

### Optional Tools
- **jq**: For JSON formatting in demos (`brew install jq` or `apt-get install jq`)
- **curl**: For API testing (usually pre-installed)

---

## Quick Start Option 1: Docker (5 minutes)

### Simplest Start

```bash
# Clone the repository
git clone https://github.com/harishb93/telemetry-pipeline.git
cd telemetry-pipeline

# Run the setup script
cd deploy/docker
./setup.sh

# Access the dashboard
open http://localhost:8080
# or: curl http://localhost:8081/health
```

The setup script will:
- Build all Docker images
- Start a local Docker registry
- Deploy services with Docker Compose
- Verify all services are running

### What Gets Deployed

```
http://localhost:8080  → React Dashboard
http://localhost:8081  → API Gateway
http://localhost:9090  → Message Queue (Admin)
```

### Manual Docker Setup

If you prefer more control:

```bash
cd deploy/docker

# 1. Start local Docker registry
docker run -d -p 5000:5000 --name registry registry:2

# 2. Build images
docker-compose build

# 3. Start services
docker-compose up -d

# 4. Check service status
docker-compose ps

# 5. View logs
docker-compose logs -f api-gateway

# 6. Test the API
curl http://localhost:8081/health

# 7. Stop services
docker-compose down
```

### Environment Variables

Edit `.env` in `deploy/docker/` to customize:

```env
# Image registry
REGISTRY=localhost:5000
TAG=latest

# Port mappings
DASHBOARD_PORT=8080
API_GATEWAY_PORT=8081
MQ_PORT=9090

# Performance tuning
STREAMER_WORKERS=4
COLLECTOR_WORKERS=4
```

---

## Quick Start Option 2: Kubernetes with Kind (10 minutes)

### One-Command Deployment

```bash
# From the repository root
cd deploy/helm

# Run the quickstart script
./quickstart.sh

# Access the dashboard (in a new terminal)
kubectl port-forward -n telemetry-pipeline svc/dashboard 8080:80

# Open browser
open http://localhost:8080
```

The quickstart script will:
- Create a Kind cluster with local registry
- Build and push all Docker images
- Deploy Helm charts to the cluster
- Verify all pods are running

### Step-by-Step Kubernetes Setup

```bash
# 1. Create a Kind cluster with local registry
kind create cluster --name telemetry --config kind-cluster-config.yaml

# 2. Start local Docker registry
docker run -d -p 5000:5000 --name registry registry:2

# 3. Build and push images
docker build -t localhost:5000/telemetry-streamer:latest \
  -f deploy/docker/telemetry-streamer.Dockerfile .
docker push localhost:5000/telemetry-streamer:latest

# Repeat for other images:
# - telemetry-collector
# - api-gateway
# - mq-service
# - dashboard

# 4. Create namespace
kubectl create namespace telemetry-pipeline

# 5. Deploy via Helm
helm upgrade --install telemetry-pipeline ./deploy/helm/charts \
  --namespace telemetry-pipeline \
  --create-namespace \
  --set images.registry=localhost:5000 \
  --set images.tag=latest

# 6. Wait for pods to be ready
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/instance=telemetry-pipeline \
  -n telemetry-pipeline --timeout=300s

# 7. Port-forward to access services
kubectl port-forward -n telemetry-pipeline svc/dashboard 8080:80
kubectl port-forward -n telemetry-pipeline svc/api-gateway 8081:8081

# 8. Access services
open http://localhost:8080  # Dashboard
curl http://localhost:8081/health  # API
```

### Verify Kubernetes Deployment

```bash
# Check pod status
kubectl get pods -n telemetry-pipeline

# Check services
kubectl get svc -n telemetry-pipeline

# View logs
kubectl logs -n telemetry-pipeline deployment/telemetry-collector -f

# Check resource usage
kubectl top pods -n telemetry-pipeline

# Describe a pod (for debugging)
kubectl describe pod -n telemetry-pipeline <pod-name>
```

### Clean Up

```bash
# Remove Helm deployment
helm uninstall telemetry-pipeline -n telemetry-pipeline

# Delete namespace
kubectl delete namespace telemetry-pipeline

# Delete Kind cluster
kind delete cluster --name telemetry

# Stop local registry
docker stop registry && docker rm registry
```

---

## Testing the Pipeline

### 1. Check Service Health

```bash
# API Gateway health
curl http://localhost:8081/health | jq .

# Response includes collector status:
# {
#   "status": "healthy",
#   "collector": {
#     "status": "healthy"
#   }
# }
```

### 2. Get Available GPUs

```bash
curl http://localhost:8081/api/v1/gpus | jq .
```

### 3. Get Telemetry Data

```bash
# List latest telemetry for gpu_0
curl http://localhost:8081/api/v1/gpus/gpu_0/telemetry | jq .

# Get specific number of entries
curl "http://localhost:8081/api/v1/gpus/gpu_0/telemetry?limit=5" | jq .

# Get entries in reverse order
curl "http://localhost:8081/api/v1/gpus/gpu_0/telemetry?reverse=true" | jq .
```

### 4. Check Queue Statistics

```bash
# MQ queue stats
curl http://localhost:9090/stats | jq .

# Response shows message count and throughput
```

### 5. Dashboard

Open your browser:
- **Docker**: http://localhost:8080
- **Kubernetes**: http://localhost:8080 (after port-forward)

You'll see:
- List of available GPUs
- Real-time telemetry charts
- Service health status
- Message queue statistics

---

## Configuration

### Docker Compose Configuration

Edit `deploy/docker/docker-compose.yml`:

```yaml
services:
  telemetry-streamer:
    environment:
      WORKERS: 4           # Concurrent data streamers
      RATE: 5              # Messages per second
      DURATION: 0          # 0 = infinite, or duration like "1h"
  
  telemetry-collector:
    environment:
      WORKERS: 4           # Concurrent message processors
      MAX_ENTRIES: 1000    # Max cache entries per GPU
      CHECKPOINT: "true"   # Enable recovery checkpoints

  api-gateway:
    environment:
      PORT: 8081          # API port
      HEALTH_PORT: 8081   # Health check port
```

### Kubernetes Configuration

Edit `deploy/helm/charts/*/values.yaml`:

```yaml
# Streamer workers
streamer:
  replicas: 2
  resources:
    requests:
      cpu: 100m
      memory: 128Mi

# Collector workers
collector:
  replicas: 2
  resources:
    requests:
      cpu: 200m
      memory: 256Mi

# API Gateway replicas
apiGateway:
  replicas: 2
```

---

## Troubleshooting

### Docker Issues

**Services won't start**:
```bash
# Check Docker daemon
docker ps

# View compose logs
docker-compose logs api-gateway

# Rebuild everything
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

**Port already in use**:
```bash
# Find process using port 8081
lsof -i :8081

# Kill the process
kill -9 <PID>
```

### Kubernetes Issues

**Pods stuck in pending**:
```bash
# Check node resources
kubectl describe nodes

# Check pod events
kubectl describe pod -n telemetry-pipeline <pod-name>

# Check logs for startup errors
kubectl logs -n telemetry-pipeline <pod-name>
```

**API Gateway unreachable**:
```bash
# Verify pod is running
kubectl get pod -n telemetry-pipeline -l app=api-gateway

# Check port-forward is active
kubectl port-forward -n telemetry-pipeline svc/api-gateway 8081:8081

# Test from another terminal
curl http://localhost:8081/health
```

**Dashboard shows "Error" for services**:
```bash
# Check API Gateway logs
kubectl logs -n telemetry-pipeline deployment/api-gateway

# Verify network connectivity between pods
kubectl exec -it deployment/dashboard -n telemetry-pipeline -- \
  curl http://api-gateway:8081/health
```

---

## Next Steps

After getting the pipeline running:

1. **Explore the Data**: Check what telemetry is being collected
   ```bash
   curl http://localhost:8081/api/v1/gpus | jq .
   ```

2. **Monitor Performance**: Watch real-time metrics in the dashboard
   - Open http://localhost:8080
   - Select a GPU to view its telemetry

3. **Read Architecture**: Understand the system design
   - See [System Architecture](../architecture/README.md)

4. **Deploy to Production**: Configure for your environment
   - See [Deployment Guide](../deployment/README.md)

5. **Understand Components**: Learn what each service does
   - See [Components Guide](../components/README.md)

---

## Common Tasks

### Stream Custom CSV Data

1. Place your CSV file in `deploy/docker/sample-data/`
2. Update Docker environment or Kubernetes values to point to your CSV
3. Restart the streamer service

### Scale the Pipeline

**Docker**:
```bash
# Scale specific service in docker-compose
docker-compose up -d --scale telemetry-collector=3
```

**Kubernetes**:
```bash
# Scale deployment
kubectl scale deployment telemetry-collector \
  -n telemetry-pipeline --replicas=3
```

### Export Telemetry Data

```bash
# Export to CSV
curl "http://localhost:8081/api/v1/gpus/gpu_0/telemetry?limit=1000" | \
  jq -r '.[] | [.timestamp, .metrics.temperature, .metrics.utilization] | @csv' > data.csv

# Export to JSON
curl "http://localhost:8081/api/v1/gpus/gpu_0/telemetry?limit=1000" > data.json
```

---

## Getting Help

- **Logs**: Check service logs for error messages
- **Health Endpoint**: `curl http://localhost:8081/health` to verify services
- **API Documentation**: Available at `/docs` (if Swagger UI enabled)
- **Dashboard**: Visual indication of service health and metrics

---

For more details, see:
- [Full Architecture Documentation](../architecture/README.md)
- [Deployment Guide](../deployment/README.md)
- [Components Reference](../components/README.md)
