# Deployment Guide

Complete guide for deploying the GPU Telemetry Pipeline to Docker and Kubernetes.

---

## Deployment Overview

The pipeline can be deployed in multiple environments:

| Environment | Best For |
|-------------|----------|
| **Docker Compose** | Development, local testing |
| **Docker Swarm** | Small production deployments |
| **Kubernetes (Kind)** | Learning, local development |
| **Kubernetes (Production)** | Enterprise, multi-node clusters |
| **Managed Kubernetes** | AWS EKS, GCP GKE, Azure AKS |

---

## Part 1: Docker Deployment

### Architecture

```
┌──────────────────────────────────────────┐
│         Docker Compose Network           │
├──────────────────────────────────────────┤
│                                          │
│  ┌──────────────┐  ┌──────────────┐    │
│  │   Streamer   │─→│  MQ Service  │    │
│  │  Container   │  │  Container   │    │
│  └──────────────┘  └──────┬───────┘    │
│                           │            │
│  ┌──────────────┐         │            │
│  │  Collector   │←────────┘            │
│  │  Container   │                      │
│  └──────────────┘                      │
│       │                                 │
│  ┌────▼──────────┐                    │
│  │  API Gateway  │                    │
│  │  Container    │                    │
│  └──────────────┘                      │
│       │                                 │
│  ┌────▼──────────┐                    │
│  │   Dashboard   │                    │
│  │   Container   │                    │
│  └──────────────┘                      │
│                                        │
│  Volumes: data, config                │
│  Network: bridge (overlay)             │
│  Ports: 8080, 8081, 9090              │
└──────────────────────────────────────────┘
```

### Prerequisites

- Docker 20.10+
- Docker Compose 1.29+
- 4GB free disk space
- Ports 8080, 8081, 9090 available

### Files Overview

```
deploy/docker/
├── docker-compose.yml          # Service definitions
├── setup.sh                    # Automated setup script
├── telemetry-streamer.Dockerfile
├── telemetry-collector.Dockerfile
├── api-gateway.Dockerfile
├── mq-service.Dockerfile
├── dashboard.Dockerfile
├── entrypoint-collector.sh
├── entrypoint-api-gateway.sh
├── entrypoint-mq.sh
├── entrypoint-streamer.sh
├── entrypoint-api-gateway.sh
├── build-and-push.sh
├── setup.sh
├── nginx.conf
└── sample-data/
    └── telemetry.csv          # Sample GPU telemetry data
```

### Step-by-Step Docker Setup

#### 1. Start Docker Registry

```bash
# Start local registry for storing images
docker run -d -p 5000:5000 --name registry registry:2

# Verify registry is running
curl http://localhost:5000/v2/
```

#### 2. Build Images

```bash
cd deploy/docker

# Build all images
docker compose build

# View built images
docker images | grep telemetry
```

Each image:
- Uses multi-stage builds for smaller size
- Optimized for production deployments
- Includes health check endpoints

#### 3. Push to Local Registry (Optional)

```bash
# Tag images
docker tag telemetry-streamer:latest localhost:5000/telemetry-streamer:latest
docker tag telemetry-collector:latest localhost:5000/telemetry-collector:latest

# Push to registry
docker push localhost:5000/telemetry-streamer:latest
docker push localhost:5000/telemetry-collector:latest
```

#### 4. Start Services

```bash
# Start all services
docker compose up -d

# Check status
docker compose ps

# Expected output:
# NAME                    STATUS
# telemetry-streamer     Up 2 seconds
# mq-service            Up 2 seconds
# telemetry-collector   Up 2 seconds
# api-gateway           Up 2 seconds
# dashboard             Up 2 seconds
```

#### 5. Verify Deployment

```bash
# Check API health
curl http://localhost:8081/health | jq .

# Get GPU list
curl http://localhost:8081/api/v1/gpus | jq .

# View telemetry
curl http://localhost:8081/api/v1/gpus/gpu_0/telemetry | jq .

# Access dashboard
open http://localhost:8080
```

### Docker Networking

```bash
# Create custom network (if needed)
docker network create kind kind-registry
```

### Docker Monitoring

```bash
# View service logs
docker compose logs -f api-gateway

# Show resource usage
docker stats

# Inspect container
docker inspect telemetry-collector

# Connect to container shell
docker exec -it telemetry-collector sh
```

### Docker Cleanup

```bash
# Stop services
docker compose down

# Remove volumes
docker compose down -v

# Remove images
docker rmi telemetry-streamer telemetry-collector

# Clean up registry
docker stop registry && docker rm kind-registry
docker volume rm kind-registry
```

---

## Part 2: Kubernetes Deployment

### Prerequisites

- Kubernetes 1.27+
- kubectl CLI
- Helm 3.12+
- Docker registry access (local or remote)

### Architecture

```
Kubernetes Cluster (telemetry-pipeline namespace)

┌─────────────────────────────────────────────────┐
│  Telemetry Streamer (DaemonSet)                 │
│  • Runs on every node                           │
│  • Independent telemetry collection             │
│  └─ Pods: streamer-xxxx (n replicas per node)   │
└──────────────┬──────────────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────────────┐
│  MQ Service (Deployment)                         │
│  • In-memory message broker                      │
│  └─ Pods: mq-service-xxxx (2+ replicas)         │
└──────────────┬───────────────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────────────┐
│  Telemetry Collector (Deployment)               │
│  • Consumes messages                             │
│  • Dual persistence (file + memory)              │
│  └─ Pods: collector-xxxx (2+ replicas)          │
└──────────────┬───────────────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────────────┐
│  API Gateway (Deployment)                       │
│  • REST API for data access                      │
│  • Health aggregation                            │
│  └─ Pods: api-gateway-xxxx (2+ replicas)        │
└──────────────┬───────────────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────────────┐
│  Dashboard (Deployment)                          │
│  • React frontend                                │
│  • Real-time monitoring                          │
│  └─ Pods: dashboard-xxxx (2+ replicas)          │
└──────────────────────────────────────────────────┘

Services:
├─ telemetry-streamer (DaemonSet Service)
├─ mq-service (ClusterIP)
├─ telemetry-collector (ClusterIP)
├─ api-gateway (ClusterIP)
└─ dashboard (ClusterIP)

Storage:
├─ PersistentVolumeClaim: collector-data
└─ PersistentVolumeClaim: mq-persistence (optional)
```

### Helm Chart Structure

```
deploy/helm
├── quickstart.sh                 # Ready-reckoner script
└── charts/                       # Subchart dependencies
    ├── streamer/
    ├── collector/
    ├── api-gateway/
    ├── mq-service/
    └── dashboard/
```

### Kubernetes Deployment Steps

#### 1. Create Cluster

```bash
# Using Kind (for local development)
kind create cluster --name telemetry

# Check cluster
kubectl cluster-info
kubectl get nodes
```

#### 2. Create Registry Secret (for private registries)

```bash
# Skip if using local registry

kubectl create namespace gpu-telemetry

kubectl create secret docker-registry regcred \
  --docker-server=<registry-url> \
  --docker-username=<username> \
  --docker-password=<password> \
  -n telemetry-pipeline
```

#### 3. Deploy via Helm

```bash
# Navigate to helm directory
cd deploy/helm

# Create namespace
kubectl create namespace gpu-telemetry

# Install Helm charts
./quickstart.sh

# Verify installation
helm list -A
```

#### 4. Verify Deployment

```bash
# Check pods
kubectl get pods -n gpu-telemetry

# Check services
kubectl get svc -n gpu-telemetry

# Check persistent volumes
kubectl get pvc -n gpu-telemetry

# View pod details
kubectl describe pod -n gpu-telemetry <pod-name>
```

#### 5. Access Services

```bash
# Port-forward API Gateway
kubectl port-forward -n gpu-telemetry svc/api-gateway 8081:80

# Port-forward Dashboard
kubectl port-forward -n gpu-telemetry svc/dashboard 8080:80

# Test API (in another terminal)
curl http://localhost:8081/health

# Open dashboard
open http://localhost:8080
```

### Kubernetes Monitoring

```bash
# View resource usage
kubectl top pods -n gpu-telemetry
kubectl top nodes

# Stream logs
kubectl logs -f deployment/api-gateway -n gpu-telemetry

# Describe pod for events
kubectl describe pod -n gpu-telemetry <pod-name>

# Get pod YAML
kubectl get pod -n gpu-telemetry <pod-name> -o yaml

# Execute command in pod
kubectl exec -it <pod-name> -n gpu-telemetry -- /bin/sh
```

### Kubernetes Scaling

```bash
# Manual scaling
kubectl scale deployment api-gateway --replicas=3 -n gpu-telemetry

# Check scaling events
kubectl describe deployment api-gateway -n gpu-telemetry
```

### Kubernetes Cleanup

```bash
# Delete Helm release, namesace, cluster(Kind) and registry
./quickstart.sh down
```

---

## Troubleshooting Deployment

### Docker Issues

```bash
# Container won't start
docker logs telemetry-collector-0 -n gpu-telemetry

# Port conflict
lsof -i :8081

# Network issues
docker network inspect telemetry-pipeline
```
---

## Next Steps

- Review [System Architecture](../architecture/README.md) for design details
- Check [Components Guide](../components/README.md) for service details
- Explore [Makefile Documentation](../makefile.md) for build automation
