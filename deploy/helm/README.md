# Telemetry Pipeline Helm Chart

This Helm chart deploys a complete GPU telemetry pipeline on Kubernetes, consisting of three main components:

- **Telemetry Streamer**: DaemonSet that streams GPU telemetry data from CSV files
- **Telemetry Collector**: Deployment that collects and processes telemetry data via message queue
- **API Gateway**: Deployment that provides REST API access to telemetry data

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+
- Persistent Volume provisioner support in the underlying infrastructure (for data persistence)

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Streamer       │    │   Collector      │    │   API Gateway   │
│  (DaemonSet)    │───▶│  (Deployment)    │───▶│  (Deployment)   │
│                 │    │                  │    │                 │
│ • CSV Data      │    │ • Message Queue  │    │ • REST API      │
│ • Rate Control  │    │ • Persistence    │    │ • OpenAPI Spec  │
│ • Multi-Node    │    │ • Health Checks  │    │ • Ingress       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Installation

### Quick Start

```bash
# Add the repository (if available)
helm repo add telemetry-pipeline https://charts.example.com/telemetry-pipeline
helm repo update

# Install with default values
helm install my-telemetry-pipeline telemetry-pipeline/telemetry-pipeline

# Or install from local directory
helm install my-telemetry-pipeline ./deploy/helm/telemetry-pipeline
```

### Custom Installation

```bash
# Install with custom values
helm install my-telemetry-pipeline ./deploy/helm/telemetry-pipeline \
  --set apiGateway.ingress.enabled=true \
  --set apiGateway.ingress.hosts[0].host=telemetry-api.example.com \
  --set collector.persistence.size=50Gi \
  --set streamer.workers=4
```

### Production Installation

```bash
# Create a production values file
cat > production-values.yaml <<EOF
# Production configuration
collector:
  replicaCount: 3
  persistence:
    enabled: true
    size: 100Gi
    storageClass: fast-ssd
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 1000m
      memory: 2Gi
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 70

apiGateway:
  replicaCount: 3
  ingress:
    enabled: true
    className: nginx
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-prod
      nginx.ingress.kubernetes.io/rate-limit: "100"
    hosts:
      - host: telemetry-api.production.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: telemetry-api-tls
        hosts:
          - telemetry-api.production.com
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10

mq:
  persistence:
    enabled: true
    size: 20Gi
    storageClass: fast-ssd

monitoring:
  serviceMonitor:
    enabled: true
    labels:
      release: prometheus
EOF

helm install telemetry-production ./deploy/helm/telemetry-pipeline \
  -f production-values.yaml
```

## Configuration

### Component Configuration

#### Telemetry Streamer (DaemonSet)

The streamer runs as a DaemonSet to collect telemetry data from each node:

```yaml
streamer:
  enabled: true
  workers: 2                    # Workers per node
  rate: 10.0                   # Messages per second per worker
  
  # CSV data configuration
  csvData: |                   # Inline CSV data
    gpu_id,utilization,temperature,memory_used
    gpu-001,85.5,72.3,4096
    # ... more data
  
  # Alternative: Use ConfigMap or PVC for large CSV files
  persistence:
    enabled: true              # Enable persistent storage
    size: 5Gi                 # Storage size for CSV files
  
  # Resource limits
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi
```

**CSV Data Options:**

1. **Inline CSV** (default): Small datasets defined directly in values.yaml
2. **ConfigMap**: For medium datasets, create a ConfigMap:
   ```bash
   kubectl create configmap telemetry-csv --from-file=telemetry.csv
   ```
3. **Persistent Volume**: For large datasets, enable persistence and mount CSV files

#### Telemetry Collector (Deployment)

The collector processes telemetry data and provides message queue functionality:

```yaml
collector:
  enabled: true
  replicaCount: 1
  workers: 4                   # Processing workers
  maxEntriesPerGPU: 10000     # Max entries per GPU in storage
  checkpointEnabled: true      # Enable checkpointing
  
  # Persistence for collected data
  persistence:
    enabled: true
    size: 10Gi
    storageClass: "fast-ssd"
  
  # Message Queue persistence
mq:
  persistence:
    enabled: true
    size: 5Gi
    dir: "/var/lib/mq"
  
  # Autoscaling
  autoscaling:
    enabled: true
    minReplicas: 1
    maxReplicas: 3
    targetCPUUtilizationPercentage: 80
```

#### API Gateway (Deployment)

The API gateway provides REST API access to telemetry data:

```yaml
apiGateway:
  enabled: true
  replicaCount: 2
  port: 8081
  
  # CORS configuration
  cors:
    enabled: true
    allowedOrigins: ["*"]
    allowedMethods: ["GET", "POST", "OPTIONS"]
  
  # Ingress configuration
  ingress:
    enabled: true
    className: "nginx"
    annotations:
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
    hosts:
      - host: telemetry-api.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: telemetry-api-tls
        hosts:
          - telemetry-api.example.com
```

### Resource Configuration

Configure resource requests and limits for optimal performance:

```yaml
# Streamer resources (per DaemonSet pod)
streamer:
  resources:
    requests:
      cpu: 100m      # Low CPU for data streaming
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi

# Collector resources (data processing intensive)
collector:
  resources:
    requests:
      cpu: 250m      # Higher CPU for data processing
      memory: 512Mi
    limits:
      cpu: 500m
      memory: 1Gi

# API Gateway resources (web server)
apiGateway:
  resources:
    requests:
      cpu: 150m      # Moderate CPU for API serving
      memory: 256Mi
    limits:
      cpu: 300m
      memory: 512Mi
```

### Persistence Configuration

Configure storage for different components:

```yaml
# Collector data persistence
collector:
  persistence:
    enabled: true
    size: 50Gi                    # Size based on data retention needs
    storageClass: "fast-ssd"      # Use fast storage for better performance
    accessModes:
      - ReadWriteOnce

# Message Queue persistence
mq:
  persistence:
    enabled: true
    size: 10Gi                    # Size based on message volume
    storageClass: "standard"      # Standard storage is sufficient
    dir: "/var/lib/mq"

# Streamer persistence (optional, for large CSV files)
streamer:
  persistence:
    enabled: false                # Usually not needed for CSV streaming
    size: 5Gi
```

### Security Configuration

Configure security contexts and network policies:

```yaml
# Pod security contexts
collector:
  podSecurityContext:
    fsGroup: 2000
    runAsNonRoot: true
  securityContext:
    runAsUser: 1000
    readOnlyRootFilesystem: true
    capabilities:
      drop:
        - ALL

# Network policies (optional)
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
      - podSelector:
          matchLabels:
            app.kubernetes.io/name: telemetry-pipeline
      ports:
      - protocol: TCP
        port: 8080
```

## Usage Examples

### 1. Installing Telemetry Streamer Only

```bash
helm install telemetry-streamer ./deploy/helm/telemetry-pipeline \
  --set collector.enabled=false \
  --set apiGateway.enabled=false \
  --set streamer.workers=4 \
  --set streamer.rate=20.0
```

### 2. Installing Collector with High Availability

```bash
helm install telemetry-collector ./deploy/helm/telemetry-pipeline \
  --set streamer.enabled=false \
  --set apiGateway.enabled=false \
  --set collector.replicaCount=3 \
  --set collector.autoscaling.enabled=true \
  --set collector.persistence.size=100Gi
```

### 3. Installing API Gateway with Ingress

```bash
helm install telemetry-api ./deploy/helm/telemetry-pipeline \
  --set streamer.enabled=false \
  --set collector.enabled=false \
  --set apiGateway.ingress.enabled=true \
  --set apiGateway.ingress.hosts[0].host=api.telemetry.local
```

### 4. Custom CSV Data Configuration

```bash
# Create a ConfigMap with your CSV data
kubectl create configmap my-telemetry-csv --from-file=my-data.csv

# Install with custom ConfigMap
helm install my-telemetry ./deploy/helm/telemetry-pipeline \
  --set streamer.csvData="" \  # Disable inline CSV
  --set-file streamer.csvData=my-data.csv
```

### 5. Development Setup

```bash
# Minimal resources for development
helm install telemetry-dev ./deploy/helm/telemetry-pipeline \
  --set collector.persistence.enabled=false \
  --set mq.persistence.enabled=false \
  --set apiGateway.replicaCount=1 \
  --set collector.replicaCount=1 \
  --set-string collector.resources.requests.cpu=50m \
  --set-string collector.resources.requests.memory=128Mi
```

## Monitoring

### Prometheus Integration

Enable Prometheus monitoring:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    interval: 30s
    labels:
      release: prometheus  # Match your Prometheus operator label selector
```

### Grafana Dashboard

Deploy with Grafana dashboard:

```yaml
monitoring:
  grafana:
    enabled: true
    dashboardsConfigMap: "telemetry-dashboards"
```

## Troubleshooting

### Common Issues

1. **Streamer pods not starting**
   ```bash
   # Check CSV data format
   kubectl logs -l app.kubernetes.io/component=streamer
   
   # Verify ConfigMap
   kubectl get configmap -l app.kubernetes.io/name=telemetry-pipeline
   ```

2. **Collector persistence issues**
   ```bash
   # Check PVC status
   kubectl get pvc -l app.kubernetes.io/name=telemetry-pipeline
   
   # Check storage class
   kubectl get storageclass
   ```

3. **API Gateway ingress not working**
   ```bash
   # Check ingress status
   kubectl get ingress -l app.kubernetes.io/component=api-gateway
   
   # Verify ingress controller
   kubectl get pods -n ingress-nginx
   ```

### Debug Commands

```bash
# View all telemetry pipeline resources
kubectl get all -l app.kubernetes.io/name=telemetry-pipeline

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