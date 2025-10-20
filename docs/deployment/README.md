# Deployment Guide

Complete guide for deploying the GPU Telemetry Pipeline to Docker and Kubernetes.

---

## Deployment Overview

The pipeline can be deployed in multiple environments:

| Environment | Best For | Complexity |
|-------------|----------|-----------|
| **Docker Compose** | Development, local testing | ⭐ (Simple) |
| **Docker Swarm** | Small production deployments | ⭐⭐ (Medium) |
| **Kubernetes (Kind)** | Learning, local development | ⭐⭐ (Medium) |
| **Kubernetes (Production)** | Enterprise, multi-node clusters | ⭐⭐⭐ (Complex) |
| **Managed Kubernetes** | AWS EKS, GCP GKE, Azure AKS | ⭐⭐⭐ (Complex) |

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
├── .env                        # Configuration variables
├── setup.sh                    # Automated setup script
├── Dockerfiles/                # Container definitions
│   ├── telemetry-streamer.Dockerfile
│   ├── telemetry-collector.Dockerfile
│   ├── api-gateway.Dockerfile
│   ├── mq-service.Dockerfile
│   └── dashboard.Dockerfile
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
docker-compose build

# View built images
docker images | grep telemetry
```

Each image:
- Uses multi-stage builds for smaller size
- Optimized for production deployments
- Includes health check endpoints

#### 3. Push to Registry (Optional)

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
docker-compose up -d

# Check status
docker-compose ps

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

### Docker Configuration

Edit `deploy/docker/.env`:

```env
# Service names (don't change these)
COMPOSE_PROJECT_NAME=telemetry-pipeline

# Image configuration
REGISTRY=localhost:5000
IMAGE_TAG=latest

# Port mappings
DASHBOARD_PORT=8080
API_GATEWAY_PORT=8081
MQ_PORT=9090

# Streamer configuration
STREAMER_WORKERS=4                    # Concurrent workers
STREAMER_RATE=5                       # Messages per second
STREAMER_DURATION=0                   # 0 = infinite

# Collector configuration
COLLECTOR_WORKERS=4                   # Processing workers
COLLECTOR_MAX_ENTRIES=1000            # Cache size per GPU
COLLECTOR_CHECKPOINT=true             # Enable recovery

# MQ configuration
MQ_PERSISTENCE=false                  # Disk persistence
MQ_PERSISTENCE_PATH=/data/mq          # Persistence directory

# Logging
LOG_LEVEL=info                        # debug, info, warn, error
```

### Docker Persistence

#### Data Volume

```bash
# Create named volume for persistent data
docker volume create telemetry-data

# Mount in docker-compose.yml
volumes:
  telemetry_data:
    external: true
```

#### Checkpoint Recovery

```bash
# Collector checkpoints are stored at:
/data/checkpoints/collector.checkpoint

# MQ persistence (if enabled):
/data/mq/messages.db
```

### Docker Networking

```bash
# Create custom network (if needed)
docker network create telemetry-net

# Connect containers
docker network connect telemetry-net telemetry-streamer
```

### Docker Monitoring

```bash
# View service logs
docker-compose logs -f api-gateway

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
docker-compose down

# Remove volumes
docker-compose down -v

# Remove images
docker rmi telemetry-streamer telemetry-collector

# Clean up registry
docker stop registry && docker rm registry
docker volume rm registry
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
deploy/helm/charts/
├── Chart.yaml                    # Chart metadata
├── values.yaml                   # Default values
├── templates/                    # Kubernetes templates
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── deployment.yaml
│   ├── statefulset.yaml
│   ├── daemonset.yaml
│   ├── service.yaml
│   ├── pvc.yaml
│   └── hpa.yaml (autoscaling)
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

kubectl create namespace telemetry-pipeline

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
kubectl create namespace telemetry-pipeline

# Install Helm chart
helm install telemetry-pipeline ./charts \
  --namespace telemetry-pipeline \
  --create-namespace \
  --values custom-values.yaml \
  --wait \
  --timeout 5m

# Verify installation
helm list -n telemetry-pipeline
```

#### 4. Verify Deployment

```bash
# Check pods
kubectl get pods -n telemetry-pipeline

# Check services
kubectl get svc -n telemetry-pipeline

# Check persistent volumes
kubectl get pvc -n telemetry-pipeline

# View pod details
kubectl describe pod -n telemetry-pipeline <pod-name>
```

#### 5. Access Services

```bash
# Port-forward API Gateway
kubectl port-forward -n telemetry-pipeline svc/api-gateway 8081:80

# Port-forward Dashboard
kubectl port-forward -n telemetry-pipeline svc/dashboard 8080:80

# Test API (in another terminal)
curl http://localhost:8081/health

# Open dashboard
open http://localhost:8080
```

### Helm Configuration

Create `custom-values.yaml`:

```yaml
# Namespace
namespace: telemetry-pipeline

# Image configuration
image:
  registry: localhost:5000
  tag: latest
  pullPolicy: IfNotPresent

# Streamer configuration
streamer:
  replicas: 2
  workers: 4
  rate: 5
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi

# Collector configuration
collector:
  replicas: 2
  workers: 4
  maxEntries: 1000
  persistence:
    enabled: true
    storageClass: "fast-ssd"
    size: 10Gi
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 512Mi

# API Gateway configuration
apiGateway:
  replicas: 2
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi

# MQ Service configuration
mq:
  replicas: 2
  persistence:
    enabled: false
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi

# Dashboard configuration
dashboard:
  replicas: 2
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 200m
      memory: 128Mi

# Autoscaling
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilization: 70

# Service configuration
service:
  type: ClusterIP
  annotations: {}

# Ingress configuration
ingress:
  enabled: false
  className: nginx
  hosts:
    - host: telemetry.example.com
      paths:
        - path: /
          pathType: Prefix
```

### Kubernetes Monitoring

```bash
# View resource usage
kubectl top pods -n telemetry-pipeline
kubectl top nodes

# Stream logs
kubectl logs -f deployment/api-gateway -n telemetry-pipeline

# Describe pod for events
kubectl describe pod -n telemetry-pipeline <pod-name>

# Get pod YAML
kubectl get pod -n telemetry-pipeline <pod-name> -o yaml

# Execute command in pod
kubectl exec -it <pod-name> -n telemetry-pipeline -- /bin/sh
```

### Kubernetes Scaling

```bash
# Manual scaling
kubectl scale deployment api-gateway --replicas=3 -n telemetry-pipeline

# View HPA status
kubectl get hpa -n telemetry-pipeline

# Check scaling events
kubectl describe hpa -n telemetry-pipeline
```

### Kubernetes Upgrades

```bash
# Update Helm values
helm upgrade telemetry-pipeline ./charts \
  --namespace telemetry-pipeline \
  --values custom-values.yaml \
  --wait

# Rollback if needed
helm rollback telemetry-pipeline -n telemetry-pipeline

# Check history
helm history telemetry-pipeline -n telemetry-pipeline
```

### Kubernetes Cleanup

```bash
# Delete Helm release
helm uninstall telemetry-pipeline -n telemetry-pipeline

# Delete namespace
kubectl delete namespace telemetry-pipeline

# Delete cluster (Kind)
kind delete cluster --name telemetry

# Remove PersistentVolumes (if needed)
kubectl delete pv --all
```

---

## Part 3: Advanced Deployments

### Multi-Cluster Setup

```bash
# Deploy to multiple clusters
for cluster in cluster1 cluster2 cluster3; do
  kubectl config use-context $cluster
  helm install telemetry-pipeline ./charts \
    --namespace telemetry-pipeline \
    --create-namespace
done
```

### GitOps with ArgoCD

```bash
# Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Create ArgoCD Application
cat <<EOF | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: telemetry-pipeline
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/telemetry-pipeline.git
    targetRevision: HEAD
    path: deploy/helm/charts
  destination:
    server: https://kubernetes.default.svc
    namespace: telemetry-pipeline
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
EOF
```

### Production Hardening

1. **Resource Limits**: Set CPU/memory requests and limits
2. **Network Policies**: Restrict pod-to-pod communication
3. **RBAC**: Create service accounts with minimal permissions
4. **Secrets**: Use Kubernetes secrets for sensitive data
5. **Monitoring**: Deploy Prometheus + Grafana
6. **Logging**: Central logging with ELK or similar

```bash
# Example: Create RBAC
kubectl create serviceaccount telemetry-pipeline -n telemetry-pipeline
kubectl create role telemetry-reader \
  --verb=get,list,watch \
  --resource=pods \
  -n telemetry-pipeline
kubectl create rolebinding telemetry-reader \
  --role=telemetry-reader \
  --serviceaccount=telemetry-pipeline:telemetry-pipeline \
  -n telemetry-pipeline
```

---

## Troubleshooting Deployment

### Docker Issues

```bash
# Container won't start
docker logs telemetry-collector

# Port conflict
lsof -i :8081

# Network issues
docker network inspect telemetry-pipeline
```

### Kubernetes Issues

```bash
# Pod pending
kubectl describe pod <pod-name> -n telemetry-pipeline

# Image pull error
kubectl logs <pod-name> -n telemetry-pipeline

# Service unreachable
kubectl get svc -n telemetry-pipeline
kubectl port-forward svc/api-gateway 8081:80 -n telemetry-pipeline
```

### Performance Issues

```bash
# Check CPU/memory usage
kubectl top pods -n telemetry-pipeline
kubectl top nodes

# Increase resources
helm upgrade telemetry-pipeline ./charts \
  -n telemetry-pipeline \
  --set collector.replicas=3 \
  --set collector.workers=8
```

---

## Next Steps

- Review [System Architecture](../architecture/README.md) for design details
- Check [Components Guide](../components/README.md) for service details
- Explore [Makefile Documentation](../makefile.md) for build automation
