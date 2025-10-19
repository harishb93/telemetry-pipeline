# GPU Telemetry Pipeline - Quick Start# GPU Telemetry Pipeline - Helm Charts# GPU Telemetry Pipeline - Helm Deployment Guide# Telemetry Pipeline Helm Chart



This directory contains the complete Helm charts and quick start script for deploying the GPU Telemetry Pipeline.



## Quick StartThis directory contains Helm charts for deploying the complete GPU Telemetry Pipeline to Kubernetes.



### Prerequisites



Ensure you have the following tools installed:## Chart StructureThis guide provides comprehensive instructions for deploying the GPU Telemetry Pipeline using Helm charts.This Helm chart deploys a complete GPU telemetry pipeline on Kubernetes, consisting of three main components:

- Docker

- Kind (Kubernetes in Docker)

- kubectl

- Helm```

- curl

deploy/helm/

### One-Command Setup

├── telemetry-pipeline/          # Umbrella chart for complete stack## Overview- **Telemetry Streamer**: DaemonSet that streams GPU telemetry data from CSV files

```bash

cd deploy/helm│   ├── Chart.yaml

./quickstart.sh

```│   ├── values.yaml- **Telemetry Collector**: Deployment that collects and processes telemetry data via message queue



This will:│   └── templates/

1. ✅ Create a Kind cluster with local registry

2. ✅ Build and push all Docker images│       └── NOTES.txtThe GPU Telemetry Pipeline consists of the following components:- **API Gateway**: Deployment that provides REST API access to telemetry data

3. ✅ Deploy all components with Helm

4. ✅ Set up port forwarding└── charts/                      # Individual component charts

5. ✅ Show access URLs

    ├── shared-resources/        # Namespace, ConfigMaps, PVCs

### Access URLs

    ├── mq-service/             # Message Queue (StatefulSet)

After successful deployment:

- **Dashboard**: http://localhost:8080    ├── telemetry-collector/    # Data Collector (StatefulSet)1. **Shared Resources** - Namespace, ConfigMaps, and PVCs## Prerequisites

- **API Gateway**: http://localhost:8081

- **Health Check**: http://localhost:8081/health    ├── telemetry-streamer/     # Data Streamer (DaemonSet)



## Commands    ├── api-gateway/            # API Gateway (Deployment)2. **MQ Service** - Message queue (StatefulSet with persistent storage)



### Full Setup    └── dashboard/              # Web Dashboard (Deployment)

```bash

./quickstart.sh up                    # Full setup (default)```3. **Telemetry Collector** - Data collector (StatefulSet with persistent storage)- Kubernetes 1.16+

./quickstart.sh up -t v1.0.0          # Setup with specific image tag

./quickstart.sh --skip-build          # Skip building, use existing images

./quickstart.sh --skip-cluster        # Use existing cluster

```## Quick Start4. **Telemetry Streamer** - CSV data streamer (DaemonSet)- Helm 3.0+



### Status and Monitoring

```bash

./quickstart.sh status                # Show deployment status### Deploy Complete Stack5. **API Gateway** - REST API service (Deployment with autoscaling)- Persistent Volume provisioner support in the underlying infrastructure (for data persistence)

./quickstart.sh logs                  # Show component logs

./quickstart.sh port-forward          # Restart port forwarding

```

```bash6. **Dashboard** - React frontend (Deployment with Ingress)

### Cleanup

```bash# Navigate to helm directory

./quickstart.sh down                  # Cleanup everything

```cd deploy/helm## Architecture



## Manual Deployment Order



If you prefer manual deployment, use this order:# Install the complete pipeline## Architecture



```bashhelm install gpu-telemetry telemetry-pipeline/

# 1. Create cluster and registry (if needed)

kind create cluster --config=kind-config.yaml```



# 2. Build and push images# Or with custom values

../docker/build-and-push.sh

helm install gpu-telemetry telemetry-pipeline/ \```┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐

# 3. Deploy in order

helm install shared-resources charts/shared-resources/  --set global.namespace=my-telemetry \

helm install mq-service charts/mq-service/ --namespace gpu-telemetry

helm install telemetry-collector charts/telemetry-collector/ --namespace gpu-telemetry  --set dashboard.ingress.enabled=true \┌─────────────────────────────────────────────────────────────────┐│  Streamer       │    │   Collector      │    │   API Gateway   │

helm install api-gateway charts/api-gateway/ --namespace gpu-telemetry

helm install dashboard charts/dashboard/ --namespace gpu-telemetry  --set dashboard.ingress.hosts[0].host=telemetry.mydomain.com

helm install telemetry-streamer charts/telemetry-streamer/ --namespace gpu-telemetry

```│                          Kubernetes Cluster                     ││  (DaemonSet)    │───▶│  (Deployment)    │───▶│  (Deployment)   │

# 4. Port forwarding

kubectl port-forward -n gpu-telemetry svc/api-gateway 8081:8081 &

kubectl port-forward service/dashboard 8080:80 -n gpu-telemetry &

```### Deploy Individual Components├─────────────────────────────────────────────────────────────────┤│                 │    │                  │    │                 │



## Components



### Individual Charts```bash│  Default Namespace                 │  gpu-telemetry Namespace    ││ • CSV Data      │    │ • Message Queue  │    │ • REST API      │

- `charts/shared-resources/` - Namespace and common resources

- `charts/mq-service/` - Redis message queue (StatefulSet)# Deploy only shared resources (namespace, PVCs, ConfigMaps)

- `charts/telemetry-collector/` - Data collection service (StatefulSet)

- `charts/api-gateway/` - REST API gateway (Deployment)helm install shared-resources charts/shared-resources/│  ┌─────────────────┐              │  ┌─────────────────────────┐ ││ • Rate Control  │    │ • Persistence    │    │ • OpenAPI Spec  │

- `charts/dashboard/` - React web dashboard (Deployment)

- `charts/telemetry-streamer/` - Data streaming service (Deployment)



### Service Dependencies# Deploy message queue│  │   Dashboard     │◄─────────────┼──┤      API Gateway       │ ││ • Multi-Node    │    │ • Health Checks  │    │ • Ingress       │

```

Dashboard → API Gateway → Telemetry Collector → MQ Servicehelm install mq-service charts/mq-service/ \

                     ↗                      ↗

         Telemetry Streamer ──────────────────  --dependency-update│  │  (Deployment)   │              │  │     (Deployment)       │ │└─────────────────┘    └──────────────────┘    └─────────────────┘

```



## Configuration

# Deploy collector│  │   + Ingress     │              │  │    + Autoscaling       │ │```

### Environment Variables

- `IMAGE_TAG` - Docker image tag (default: latest)helm install telemetry-collector charts/telemetry-collector/ \

- `SKIP_BUILD` - Skip image building (default: false)

- `SKIP_CLUSTER` - Skip cluster creation (default: false)  --dependency-update│  └─────────────────┘              │  └─────────────────────────┘ │



### Command Line Options

- `-t, --tag TAG` - Image tag

- `-c, --cluster NAME` - Cluster name (default: kind)# Deploy streamer│           │                       │              │               │## Installation

- `-n, --namespace NS` - Kubernetes namespace (default: gpu-telemetry)

- `--skip-build` - Skip building imageshelm install telemetry-streamer charts/telemetry-streamer/ \

- `--skip-cluster` - Skip cluster creation

- `--debug` - Enable debug output  --dependency-update│           └───────────────────────┼──────────────┘               │



## Troubleshooting



### Check Status# Deploy API gateway│                                   │                              │### Quick Start

```bash

./quickstart.sh statushelm install api-gateway charts/api-gateway/

kubectl get pods -n gpu-telemetry

kubectl get services -n gpu-telemetry│                                   │  ┌─────────────────────────┐ │

```

# Deploy dashboard

### View Logs

```bashhelm install dashboard charts/dashboard/│                                   │  │  Telemetry Collector    │ │```bash

./quickstart.sh logs

kubectl logs -f -n gpu-telemetry <pod-name>```

```

│                                   │  │    (StatefulSet)        │ │# Add the repository (if available)

### Registry Issues

```bash## Production Deployment

# Check registry

curl http://localhost:5000/v2/_catalog│                                   │  │      + PVC              │ │helm repo add telemetry-pipeline https://charts.example.com/telemetry-pipeline



# Check registry connectivity from cluster### Prerequisites

kubectl run debug --rm -i --tty --image=curlimages/curl -- /bin/sh

curl http://kind-registry:5000/v2/_catalog│                                   │  └─────────────────────────┘ │helm repo update

```

1. **Kubernetes cluster** (v1.20+)

### Port Forwarding Issues

```bash2. **Helm** (v3.0+)│                                   │              │               │

# Kill existing port forwards

pkill -f "kubectl port-forward"3. **Storage class** for persistent volumes



# Restart port forwarding4. **Ingress controller** (if using ingress)│                                   │  ┌─────────────────────────┐ │# Install with default values

./quickstart.sh port-forward

```



### Clean Slate### Production Example│                                   │  │     MQ Service          │ │helm install my-telemetry-pipeline telemetry-pipeline/telemetry-pipeline

```bash

./quickstart.sh down    # Full cleanup

./quickstart.sh up      # Fresh start

``````bash│                                   │  │    (StatefulSet)        │ │



## Development# Deploy with production settings



### Rebuilding Single Componenthelm install gpu-telemetry telemetry-pipeline/ \│                                   │  │      + PVC              │ │# Or install from local directory

```bash

# Build specific component  --set global.namespace=gpu-telemetry-prod \

docker build -f deploy/docker/dashboard.Dockerfile -t localhost:5000/dashboard:latest .

docker push localhost:5000/dashboard:latest  --set global.storageClass=fast-ssd \│                                   │  └─────────────────────────┘ │helm install my-telemetry-pipeline ./deploy/helm/telemetry-pipeline



# Restart deployment  --set dashboard.ingress.enabled=true \

kubectl rollout restart deployment/dashboard -n gpu-telemetry

```  --set dashboard.ingress.hosts[0].host=telemetry.company.com \│                                   │              ▲               │```



### Updating Charts  --namespace gpu-telemetry-prod \

```bash

# Update specific chart  --create-namespace│  ┌─────────────────────────────────┼──────────────┘               │

helm upgrade dashboard charts/dashboard/ --namespace gpu-telemetry

```

# Update all charts

helm upgrade shared-resources charts/shared-resources/│  │          Every Node             │                              │### Custom Installation

helm upgrade mq-service charts/mq-service/ --namespace gpu-telemetry

# ... etc## Component Overview

```

│  │  ┌─────────────────────────┐    │  ┌─────────────────────────┐ │

## Architecture

### 1. Shared Resources

```

┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐- **Type**: Job (creates resources)│  │  │  Telemetry Streamer     │────┼──┤     ConfigMap           │ │```bash

│   Dashboard     │    │   API Gateway   │    │ Telemetry       │

│   (React UI)    │───▶│   (REST API)    │───▶│ Collector       │- **Purpose**: Sets up namespace, ConfigMaps, and PersistentVolumeClaims

│   Port: 80      │    │   Port: 8081    │    │ (Data Ingestion)│

└─────────────────┘    └─────────────────┘    │   Port: 8080    │- **Dependencies**: None│  │  │    (DaemonSet)          │    │  │   (telemetry.csv)       │ │# Install with custom values

                                              └─────────────────┘

                                                       │

                                                       ▼

┌─────────────────┐                          ┌─────────────────┐### 2. Message Queue Service│  │  │  + CSV ConfigMap        │    │  │                         │ │helm install my-telemetry-pipeline ./deploy/helm/telemetry-pipeline \

│ Telemetry       │                          │   MQ Service    │

│ Streamer        │◀─────────────────────────│   (Redis)       │- **Type**: StatefulSet

│ (Stream Proc.)  │                          │   Port: 9090    │

│ Port: 8082      │                          └─────────────────┘- **Purpose**: Provides message queuing for telemetry data│  │  └─────────────────────────┘    │  └─────────────────────────┘ │  --set apiGateway.ingress.enabled=true \

└─────────────────┘

```- **Dependencies**: Shared Resources



## Support- **Persistence**: Yes (5Gi default)│  └─────────────────────────────────┼──────────────────────────────┤  --set apiGateway.ingress.hosts[0].host=telemetry-api.example.com \



For issues or questions:

1. Check the logs: `./quickstart.sh logs`

2. Check status: `./quickstart.sh status`### 3. Telemetry Collector└─────────────────────────────────────────────────────────────────┘  --set collector.persistence.size=50Gi \

3. Try cleanup and restart: `./quickstart.sh down && ./quickstart.sh up`
- **Type**: StatefulSet

- **Purpose**: Collects and processes telemetry data```  --set streamer.workers=4

- **Dependencies**: Message Queue Service

- **Persistence**: Yes (10Gi default)```



### 4. Telemetry Streamer## Prerequisites

- **Type**: DaemonSet

- **Purpose**: Streams GPU telemetry data from nodes### Production Installation

- **Dependencies**: Shared Resources (for ConfigMap)

- **Hostname Filtering**: Supports HOSTNAME_LIST environment variable- Kubernetes cluster (v1.19+)



### 5. API Gateway- Helm 3.0+```bash

- **Type**: Deployment with autoscaling

- **Purpose**: Provides REST API access to telemetry data- Ingress controller (nginx recommended)# Create a production values file

- **Dependencies**: Telemetry Collector

- **Replicas**: 2 default, autoscales 2-10- Storage class for persistent volumescat > production-values.yaml <<EOF



### 6. Dashboard# Production configuration

- **Type**: Deployment with autoscaling

- **Purpose**: Web-based dashboard for telemetry visualization## Quick Startcollector:

- **Dependencies**: API Gateway

- **Ingress**: Optional, enabled via values  replicaCount: 3

- **Namespace**: Deploys to default namespace by default

### 1. Deploy Complete Stack  persistence:

## Configuration

    enabled: true

### Global Settings

```bash    size: 100Gi

```yaml

global:# Navigate to the Helm directory    storageClass: fast-ssd

  namespace: gpu-telemetry

  imageRegistry: ""cd deploy/helm  resources:

  imagePullSecrets: []

  storageClass: ""    requests:

```

# Install the complete telemetry pipeline      cpu: 500m

### Enable/Disable Components

helm install gpu-telemetry telemetry-pipeline/      memory: 1Gi

```yaml

shared-resources:    limits:

  enabled: true

mq-service:# Or with custom values      cpu: 1000m

  enabled: true

telemetry-collector:helm install gpu-telemetry telemetry-pipeline/ -f custom-values.yaml      memory: 2Gi

  enabled: true

telemetry-streamer:```  autoscaling:

  enabled: true

api-gateway:    enabled: true

  enabled: true

dashboard:### 2. Verify Deployment    minReplicas: 2

  enabled: true

```    maxReplicas: 5



## Accessing Services```bash    targetCPUUtilizationPercentage: 70



### Development (Port Forward)# Check all resources



```bashkubectl get all -n gpu-telemetryapiGateway:

# Dashboard

kubectl port-forward svc/dashboard 3000:80kubectl get all -n default -l app=gpu-telemetry  replicaCount: 3



# API Gateway  ingress:

kubectl -n gpu-telemetry port-forward svc/api-gateway 8081:80

# Check persistent volumes    enabled: true

# Health check

kubectl -n gpu-telemetry port-forward svc/telemetry-collector 8080:8080kubectl get pvc -n gpu-telemetry    className: nginx

curl http://localhost:8080/health

```    annotations:



### Production (Ingress)# Check ingress      cert-manager.io/cluster-issuer: letsencrypt-prod



Configure ingress for external access:kubectl get ingress -n default      nginx.ingress.kubernetes.io/rate-limit: "100"



```yaml```    hosts:

dashboard:

  ingress:      - host: telemetry-api.production.com

    enabled: true

    className: "nginx"### 3. Access Dashboard        paths:

    hosts:

      - host: telemetry.company.com          - path: /

        paths:

          - path: /```bash            pathType: Prefix

            pathType: Prefix

```# If using ingress (recommended)    tls:



## Monitoring# Add to /etc/hosts: <ingress-ip> gpu-telemetry.local      - secretName: telemetry-api-tls



### Check Status# Then visit: http://gpu-telemetry.local        hosts:



```bash          - telemetry-api.production.com

# All components

kubectl get all -n gpu-telemetry# Or port forward  autoscaling:



# Persistent volumeskubectl port-forward -n default svc/dashboard 8080:5173    enabled: true

kubectl get pvc -n gpu-telemetry

# Visit: http://localhost:8080    minReplicas: 3

# DaemonSet status

kubectl get ds telemetry-streamer -n gpu-telemetry```    maxReplicas: 10

```



### View Logs

## Component-Specific Deploymentmq:

```bash

# Streamer (runs on all nodes)  persistence:

kubectl logs -n gpu-telemetry -l app=telemetry-streamer

### Deploy Individual Components    enabled: true

# Collector

kubectl logs -n gpu-telemetry -l app=telemetry-collector    size: 20Gi



# API GatewayYou can deploy components separately for testing or specific use cases:    storageClass: fast-ssd

kubectl logs -n gpu-telemetry -l app=api-gateway

```



## Troubleshooting```bashmonitoring:



### Common Issues# 1. Deploy shared resources first (required)  serviceMonitor:



1. **Persistent Volume Claims Pending**helm install shared-resources charts/shared-resources/    enabled: true

   - Check if storage class exists: `kubectl get storageclass`

   - Verify cluster has available storage    labels:



2. **Pods in CrashLoopBackOff**# 2. Deploy MQ service      release: prometheus

   - Check logs: `kubectl logs <pod-name> -n gpu-telemetry`

   - Verify dependencies are runninghelm install mq-service charts/mq-service/EOF



3. **DaemonSet Not Scheduling**

   - Check node selectors and tolerations

   - Verify nodes have required labels# 3. Deploy collector (depends on MQ)helm install telemetry-production ./deploy/helm/telemetry-pipeline \



4. **Ingress Not Working**helm install telemetry-collector charts/telemetry-collector/  -f production-values.yaml

   - Verify ingress controller is installed

   - Check ingress configuration: `kubectl get ingress````



### Validation Commands# 4. Deploy streamer (DaemonSet)



```bashhelm install telemetry-streamer charts/telemetry-streamer/## Configuration

# Lint charts

helm lint telemetry-pipeline/

helm lint charts/*/

# 5. Deploy API gateway (depends on collector)### Component Configuration

# Dry run

helm install test telemetry-pipeline/ --dry-runhelm install api-gateway charts/api-gateway/



# Template output#### Telemetry Streamer (DaemonSet)

helm template telemetry-pipeline/ > output.yaml

```# 6. Deploy dashboard (depends on API gateway)



## Upgradeshelm install dashboard charts/dashboard/The streamer runs as a DaemonSet to collect telemetry data from each node:



```bash```

# Upgrade complete stack

helm upgrade gpu-telemetry telemetry-pipeline/```yaml



# Upgrade with new values## Configuration Optionsstreamer:

helm upgrade gpu-telemetry telemetry-pipeline/ \

  --set telemetry-streamer.image.tag=v2.0.0  enabled: true

```

### Environment Variables  workers: 2                    # Workers per node

## Uninstalling

  rate: 10.0                   # Messages per second per worker

```bash

# Remove applicationThe telemetry streamer supports hostname filtering:  

helm uninstall gpu-telemetry

  # CSV data configuration

# Clean up namespace (if desired)

kubectl delete namespace gpu-telemetry```yaml  csvData: |                   # Inline CSV data

```

telemetry-streamer:    gpu_id,utilization,temperature,memory_used

## Support

  env:    gpu-001,85.5,72.3,4096

This Helm chart provides a production-ready deployment of the GPU Telemetry Pipeline with:

    - name: HOSTNAME_LIST    # ... more data

- ✅ High availability configurations

- ✅ Persistent storage for stateful components      value: "host1,host2,host3"  # Comma-separated list  

- ✅ Autoscaling for stateless components

- ✅ Proper dependency management    # Leave empty to process all hostnames  # Alternative: Use ConfigMap or PVC for large CSV files

- ✅ Security contexts and constraints

- ✅ Health checks and monitoring```  persistence:

- ✅ Ingress configuration for external access

- ✅ Namespace isolation    enabled: true              # Enable persistent storage

- ✅ ConfigMap-based configuration
### Resource Limits    size: 5Gi                 # Storage size for CSV files

  

Adjust resource limits based on your cluster capacity:  # Resource limits

  resources:

```yaml    requests:

# Example custom values      cpu: 100m

mq-service:      memory: 128Mi

  resources:    limits:

    limits:      cpu: 200m

      cpu: 1000m      memory: 256Mi

      memory: 1Gi```

    requests:

      cpu: 500m**CSV Data Options:**

      memory: 512Mi

1. **Inline CSV** (default): Small datasets defined directly in values.yaml

telemetry-collector:2. **ConfigMap**: For medium datasets, create a ConfigMap:

  resources:   ```bash

    limits:   kubectl create configmap telemetry-csv --from-file=telemetry.csv

      cpu: 2000m   ```

      memory: 2Gi3. **Persistent Volume**: For large datasets, enable persistence and mount CSV files

```

#### Telemetry Collector (Deployment)

### Storage Configuration

The collector processes telemetry data and provides message queue functionality:

Configure persistent storage:

```yaml

```yamlcollector:

shared-resources:  enabled: true

  persistentVolumes:  replicaCount: 1

    mqData:  workers: 4                   # Processing workers

      size: 20Gi  maxEntriesPerGPU: 10000     # Max entries per GPU in storage

      storageClass: "fast-ssd"  checkpointEnabled: true      # Enable checkpointing

    collectorData:  

      size: 50Gi  # Persistence for collected data

      storageClass: "fast-ssd"  persistence:

```    enabled: true

    size: 10Gi

### Ingress Configuration    storageClass: "fast-ssd"

  

Configure external access:  # Message Queue persistence

mq:

```yaml  persistence:

dashboard:    enabled: true

  ingress:    size: 5Gi

    enabled: true    dir: "/var/lib/mq"

    className: "nginx"  

    annotations:  # Autoscaling

      cert-manager.io/cluster-issuer: "letsencrypt-prod"  autoscaling:

    hosts:    enabled: true

      - host: gpu-telemetry.example.com    minReplicas: 1

        paths:    maxReplicas: 3

          - path: /    targetCPUUtilizationPercentage: 80

            pathType: Prefix```

    tls:

      - secretName: gpu-telemetry-tls#### API Gateway (Deployment)

        hosts:

          - gpu-telemetry.example.comThe API gateway provides REST API access to telemetry data:

```

```yaml

## Scaling and High AvailabilityapiGateway:

  enabled: true

### Enable Autoscaling  replicaCount: 2

  port: 8081

```yaml  

api-gateway:  # CORS configuration

  autoscaling:  cors:

    enabled: true    enabled: true

    minReplicas: 2    allowedOrigins: ["*"]

    maxReplicas: 10    allowedMethods: ["GET", "POST", "OPTIONS"]

    targetCPUUtilizationPercentage: 80  

  # Ingress configuration

dashboard:  ingress:

  autoscaling:    enabled: true

    enabled: true    className: "nginx"

    minReplicas: 2    annotations:

    maxReplicas: 5      cert-manager.io/cluster-issuer: "letsencrypt-prod"

    targetCPUUtilizationPercentage: 80    hosts:

```      - host: telemetry-api.example.com

        paths:

### Multi-Node Deployment          - path: /

            pathType: Prefix

The DaemonSet automatically deploys streamers to all nodes. Configure node selection:    tls:

      - secretName: telemetry-api-tls

```yaml        hosts:

telemetry-streamer:          - telemetry-api.example.com

  nodeSelector:```

    node-role.kubernetes.io/gpu: "true"

  tolerations:### Resource Configuration

    - key: nvidia.com/gpu

      operator: ExistsConfigure resource requests and limits for optimal performance:

      effect: NoSchedule

``````yaml

# Streamer resources (per DaemonSet pod)

## Monitoring and Troubleshootingstreamer:

  resources:

### Health Checks    requests:

      cpu: 100m      # Low CPU for data streaming

All components include health checks:      memory: 128Mi

    limits:

```bash      cpu: 200m

# Check component health via API      memory: 256Mi

kubectl port-forward -n gpu-telemetry svc/mq-service 9090:9090

curl http://localhost:9090/health# Collector resources (data processing intensive)

collector:

kubectl port-forward -n gpu-telemetry svc/telemetry-collector 8080:8080  resources:

curl http://localhost:8080/healthz    requests:

      cpu: 250m      # Higher CPU for data processing

kubectl port-forward -n gpu-telemetry svc/api-gateway 8081:8081      memory: 512Mi

curl http://localhost:8081/health    limits:

```      cpu: 500m

      memory: 1Gi

### View Logs

# API Gateway resources (web server)

```bashapiGateway:

# MQ Service logs  resources:

kubectl logs -n gpu-telemetry -l component=mq -f    requests:

      cpu: 150m      # Moderate CPU for API serving

# Collector logs      memory: 256Mi

kubectl logs -n gpu-telemetry -l component=collector -f    limits:

      cpu: 300m

# Streamer logs (DaemonSet - multiple pods)      memory: 512Mi

kubectl logs -n gpu-telemetry -l component=streamer -f```



# API Gateway logs### Persistence Configuration

kubectl logs -n gpu-telemetry -l component=api-gateway -f

Configure storage for different components:

# Dashboard logs

kubectl logs -n default -l component=dashboard -f```yaml

```# Collector data persistence

collector:

### Debug Common Issues  persistence:

    enabled: true

#### 1. Pods stuck in Pending    size: 50Gi                    # Size based on data retention needs

```bash    storageClass: "fast-ssd"      # Use fast storage for better performance

kubectl describe pod <pod-name> -n gpu-telemetry    accessModes:

# Check for resource constraints or node selector issues      - ReadWriteOnce

```

# Message Queue persistence

#### 2. PVC not bindingmq:

```bash  persistence:

kubectl get pvc -n gpu-telemetry    enabled: true

kubectl describe pvc <pvc-name> -n gpu-telemetry    size: 10Gi                    # Size based on message volume

# Check storage class and available storage    storageClass: "standard"      # Standard storage is sufficient

```    dir: "/var/lib/mq"



#### 3. Service dependencies# Streamer persistence (optional, for large CSV files)

```bashstreamer:

# Check service discovery  persistence:

kubectl get svc -n gpu-telemetry    enabled: false                # Usually not needed for CSV streaming

kubectl get endpoints -n gpu-telemetry    size: 5Gi

``````



## Upgrade and Maintenance### Security Configuration



### Upgrade DeploymentConfigure security contexts and network policies:



```bash```yaml

# Upgrade with new values# Pod security contexts

helm upgrade gpu-telemetry telemetry-pipeline/ -f new-values.yamlcollector:

  podSecurityContext:

# Upgrade individual components    fsGroup: 2000

helm upgrade telemetry-collector charts/telemetry-collector/    runAsNonRoot: true

```  securityContext:

    runAsUser: 1000

### Backup and Recovery    readOnlyRootFilesystem: true

    capabilities:

```bash      drop:

# Backup persistent data        - ALL

kubectl exec -n gpu-telemetry mq-service-0 -- tar czf - /var/lib/mq > mq-backup.tar.gz

kubectl exec -n gpu-telemetry telemetry-collector-0 -- tar czf - /app/data > collector-backup.tar.gz# Network policies (optional)

```networkPolicy:

  enabled: true

### Cleanup  policyTypes:

    - Ingress

```bash    - Egress

# Uninstall complete stack  ingress:

helm uninstall gpu-telemetry    - from:

      - podSelector:

# Clean up PVCs (if needed)          matchLabels:

kubectl delete pvc -n gpu-telemetry --all            app.kubernetes.io/name: telemetry-pipeline

      ports:

# Clean up namespace      - protocol: TCP

kubectl delete namespace gpu-telemetry        port: 8080

``````



## Custom CSV Data## Usage Examples



To use your own telemetry data, update the ConfigMap:### 1. Installing Telemetry Streamer Only



```yaml```bash

shared-resources:helm install telemetry-streamer ./deploy/helm/telemetry-pipeline \

  configMaps:  --set collector.enabled=false \

    telemetryData:  --set apiGateway.enabled=false \

      data: |  --set streamer.workers=4 \

        timestamp,metric_name,gpu_id,device,uuid,modelName,Hostname,container,pod,namespace,value,labels_raw  --set streamer.rate=20.0

        # Your CSV data here...```

```

### 2. Installing Collector with High Availability

Or create a separate ConfigMap and update the streamer configuration.

```bash

## Deployment Exampleshelm install telemetry-collector ./deploy/helm/telemetry-pipeline \

  --set streamer.enabled=false \

### Production Deployment with Custom Domain  --set apiGateway.enabled=false \

  --set collector.replicaCount=3 \

```bash  --set collector.autoscaling.enabled=true \

# Create custom values file  --set collector.persistence.size=100Gi

cat > production-values.yaml << EOF```

dashboard:

  ingress:### 3. Installing API Gateway with Ingress

    enabled: true

    className: "nginx"```bash

    annotations:helm install telemetry-api ./deploy/helm/telemetry-pipeline \

      cert-manager.io/cluster-issuer: "letsencrypt-prod"  --set streamer.enabled=false \

      nginx.ingress.kubernetes.io/ssl-redirect: "true"  --set collector.enabled=false \

    hosts:  --set apiGateway.ingress.enabled=true \

      - host: gpu-telemetry.company.com  --set apiGateway.ingress.hosts[0].host=api.telemetry.local

        paths:```

          - path: /

            pathType: Prefix### 4. Custom CSV Data Configuration

    tls:

      - secretName: gpu-telemetry-tls```bash

        hosts:# Create a ConfigMap with your CSV data

          - gpu-telemetry.company.comkubectl create configmap my-telemetry-csv --from-file=my-data.csv



shared-resources:# Install with custom ConfigMap

  persistentVolumes:helm install my-telemetry ./deploy/helm/telemetry-pipeline \

    mqData:  --set streamer.csvData="" \  # Disable inline CSV

      size: 50Gi  --set-file streamer.csvData=my-data.csv

      storageClass: "fast-ssd"```

    collectorData:

      size: 100Gi### 5. Development Setup

      storageClass: "fast-ssd"

```bash

api-gateway:# Minimal resources for development

  autoscaling:helm install telemetry-dev ./deploy/helm/telemetry-pipeline \

    enabled: true  --set collector.persistence.enabled=false \

    minReplicas: 3  --set mq.persistence.enabled=false \

    maxReplicas: 20  --set apiGateway.replicaCount=1 \

EOF  --set collector.replicaCount=1 \

  --set-string collector.resources.requests.cpu=50m \

# Deploy with production values  --set-string collector.resources.requests.memory=128Mi

helm install gpu-telemetry telemetry-pipeline/ -f production-values.yaml```

```

## Monitoring

### Development Deployment with Minimal Resources

### Prometheus Integration

```bash

# Create development values fileEnable Prometheus monitoring:

cat > dev-values.yaml << EOF

mq-service:```yaml

  resources:monitoring:

    limits:  serviceMonitor:

      cpu: 100m    enabled: true

      memory: 128Mi    interval: 30s

    requests:    labels:

      cpu: 50m      release: prometheus  # Match your Prometheus operator label selector

      memory: 64Mi```



telemetry-collector:### Grafana Dashboard

  resources:

    limits:Deploy with Grafana dashboard:

      cpu: 200m

      memory: 256Mi```yaml

    requests:monitoring:

      cpu: 100m  grafana:

      memory: 128Mi    enabled: true

    dashboardsConfigMap: "telemetry-dashboards"

api-gateway:```

  replicaCount: 1

  autoscaling:## Troubleshooting

    enabled: false

### Common Issues

dashboard:  

  replicaCount: 11. **Streamer pods not starting**

  autoscaling:   ```bash

    enabled: false   # Check CSV data format

   kubectl logs -l app.kubernetes.io/component=streamer

shared-resources:   

  persistentVolumes:   # Verify ConfigMap

    mqData:   kubectl get configmap -l app.kubernetes.io/name=telemetry-pipeline

      size: 1Gi   ```

    collectorData:

      size: 2Gi2. **Collector persistence issues**

EOF   ```bash

   # Check PVC status

# Deploy with development values   kubectl get pvc -l app.kubernetes.io/name=telemetry-pipeline

helm install gpu-telemetry-dev telemetry-pipeline/ -f dev-values.yaml   

```   # Check storage class

   kubectl get storageclass

## Support   ```



For issues and questions:3. **API Gateway ingress not working**

1. Check the logs of relevant components   ```bash

2. Verify service connectivity   # Check ingress status

3. Check resource utilization   kubectl get ingress -l app.kubernetes.io/component=api-gateway

4. Review Kubernetes events: `kubectl get events -n gpu-telemetry --sort-by=.metadata.creationTimestamp`   

   # Verify ingress controller

## Advanced Configuration   kubectl get pods -n ingress-nginx

   ```

See the individual chart `values.yaml` files for complete configuration options:

- `charts/shared-resources/values.yaml`### Debug Commands

- `charts/mq-service/values.yaml`

- `charts/telemetry-collector/values.yaml````bash

- `charts/telemetry-streamer/values.yaml`# View all telemetry pipeline resources

- `charts/api-gateway/values.yaml`kubectl get all -l app.kubernetes.io/name=telemetry-pipeline

- `charts/dashboard/values.yaml`
# Check component logs
kubectl logs -l app.kubernetes.io/component=streamer -f
kubectl logs -l app.kubernetes.io/component=collector -f
kubectl logs -l app.kubernetes.io/component=api-gateway -f

# Check configuration
kubectl get configmap -l app.kubernetes.io/name=telemetry-pipeline -o yaml

# Test API connectivity
kubectl port-forward svc/RELEASE-NAME-api-gateway 8081:80
curl http://localhost:8081/health
```

## Upgrading

### Upgrade Process

```bash
# Upgrade with new values
helm upgrade my-telemetry-pipeline ./deploy/helm/telemetry-pipeline \
  --reuse-values \
  --set collector.image.tag=v2.0.0

# Upgrade with values file
helm upgrade my-telemetry-pipeline ./deploy/helm/telemetry-pipeline \
  -f production-values.yaml
```

### Breaking Changes

- **v0.2.0**: Changed default persistence from `emptyDir` to `PVC`
- **v0.3.0**: Renamed service ports for consistency

## Uninstalling

```bash
# Uninstall the release
helm uninstall my-telemetry-pipeline

# Clean up PVCs (if needed)
kubectl delete pvc -l app.kubernetes.io/name=telemetry-pipeline
```

## Values Reference

| Parameter | Description | Default |
|-----------|-------------|---------|
| `streamer.enabled` | Enable telemetry streamer | `true` |
| `streamer.workers` | Number of workers per node | `2` |
| `streamer.rate` | Messages per second per worker | `10.0` |
| `streamer.csvData` | Inline CSV data | Sample data |
| `collector.enabled` | Enable telemetry collector | `true` |
| `collector.replicaCount` | Number of collector replicas | `1` |
| `collector.workers` | Number of processing workers | `4` |
| `collector.persistence.enabled` | Enable collector persistence | `true` |
| `collector.persistence.size` | Collector storage size | `10Gi` |
| `apiGateway.enabled` | Enable API gateway | `true` |
| `apiGateway.replicaCount` | Number of API gateway replicas | `2` |
| `apiGateway.ingress.enabled` | Enable ingress | `false` |
| `mq.persistence.enabled` | Enable MQ persistence | `true` |
| `mq.persistence.size` | MQ storage size | `5Gi` |

For a complete list of values, see `values.yaml`.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test the chart: `helm template ./deploy/helm/telemetry-pipeline`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.