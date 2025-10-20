# Makefile Documentation

Complete guide to all available Makefile targets for building, testing, and deploying the GPU Telemetry Pipeline.

---

## Quick Reference

```bash
# Build and test
make build            # Build all components (including dashboard)
make build-dashboard  # Build React dashboard only
make test             # Run unit tests with coverage
make dev              # Build + test (development workflow)
make all              # Full build pipeline (clean + deps + build + test + coverage)
make ci               # CI pipeline (clean + deps + build + test + coverage)

# Docker workflows  
make docker-build     # Build Docker images (all 5 services)
make docker-build-and-push # Use build-and-push.sh script
make docker-push      # Push to registry
make docker-up        # Start services with Docker Compose (comprehensive)
make docker-down      # Stop Docker Compose services
make docker-logs      # Show Docker Compose logs
make docker-health-check # Check service health

# Docker setup scripts
make docker-setup     # Use setup.sh (existing images)
make docker-setup-build # Use setup.sh with build
make docker-setup-down # Use setup.sh teardown

# Kubernetes deployment
make helm-install     # Deploy using individual charts
make helm-uninstall   # Remove all Helm releases
make helm-status      # Check deployment status
make helm-quickstart  # Use quickstart.sh (full Kind setup)
make helm-quickstart-down # Use quickstart.sh cleanup
make helm-port-forward # Start port forwarding

# Local development
make run-collector    # Run collector locally
make run-streamer     # Run streamer locally  
make run-api          # Run API gateway locally
make run-mq           # Run MQ service locally

# System testing
make system-tests     # Comprehensive system tests
make system-tests-quick # Quick functional tests
make system-tests-performance # Performance tests

# Registry management
make registry-start   # Start local Docker registry
make registry-stop    # Stop local Docker registry
make registry-status  # Check registry status

# Cleanup
make clean            # Remove build artifacts
make clean-docker     # Remove Docker images
```

---

## Configuration

Set these variables to customize behavior:

```bash
# Build with custom registry
make docker-build REGISTRY=gcr.io/my-project TAG=v1.0.0

# Deploy to Kubernetes
make helm-install HELM_NAMESPACE=production TAG=v1.0.0
```

| Variable | Default | Description |
|----------|---------|-------------|
| `REGISTRY` | `localhost:5000` | Docker registry URL |
| `TAG` | `latest` | Docker image tag |
| `HELM_RELEASE` | `telemetry-pipeline` | Helm release name (legacy) |
| `HELM_NAMESPACE` | `default` | Kubernetes namespace (legacy) |
| `HELM_NAMESPACE_REAL` | `gpu-telemetry` | Actual namespace for individual charts |

---

## Primary Targets

### Build Targets

#### `make build`
Builds all Go binaries and React dashboard.

```bash
make build
# Output:
# bin/telemetry-streamer
# bin/telemetry-collector
# bin/api-gateway
# bin/mq-service
# dashboard/dist/ (React build)
```

**Individual build targets**:
```bash
make build-collector  # Build collector only
make build-streamer   # Build streamer only
make build-api        # Build API gateway only
make build-mq         # Build MQ service only
make build-dashboard  # Build React dashboard only (npm ci && npm run build)
```

#### `make test`
Runs all Go tests with atomic coverage reporting, excluding system tests.

```bash
make test
# Output: coverage.out
# Shows: tests run, passed, failed
# Excludes: system test suite (tagged with "system")
```

#### `make system-tests`
Runs comprehensive end-to-end system tests.

```bash
make system-tests
# Runs: Full system integration tests
# Duration: ~10 minutes
# Requires: All binaries built
```

#### `make system-tests-quick`
Runs quick functional system tests.

```bash
make system-tests-quick  
# Runs: Essential system tests only
# Duration: ~5 minutes
# Good for: Development iteration
```

#### `make system-tests-performance`
Runs performance benchmarking tests.

```bash
make system-tests-performance
# Runs: Load and performance tests
# Duration: ~15 minutes  
# Tests: Throughput, latency, resource usage
```

#### `make coverage`
Generates HTML coverage report.

```bash
make coverage
# Output: coverage.html (open in browser)
# Shows: coverage percentage per package
```

### Docker Targets

#### `make docker-build`
Builds Docker images using multi-stage builds.

```bash
# Build with default tag
make docker-build
# Creates: telemetry-streamer:latest, telemetry-collector:latest, etc.

# Build with custom tag and registry
make docker-build TAG=v1.0.0 REGISTRY=gcr.io/my-project
# Creates: gcr.io/my-project/telemetry-streamer:v1.0.0, etc.
```

**Images created**:
- `mq-service:TAG`
- `telemetry-streamer:TAG`
- `telemetry-collector:TAG`
- `api-gateway:TAG`
- `dashboard:TAG` (React frontend)

**Registry tagging**: All images are automatically tagged for the specified registry.

**Entrypoint scripts**: Automatically makes all entrypoint scripts executable before building.

#### `make docker-push`
Pushes Docker images to registry.

```bash
# Push to default registry
make docker-push
# Pushes to localhost:5000

# Push to custom registry
make docker-push REGISTRY=gcr.io/my-project TAG=v1.0.0
# Pushes to gcr.io/my-project
```

**Pushes all 5 images** including the new dashboard image.

#### `make docker-build-and-push`
Uses the `deploy/docker/build-and-push.sh` script for building and pushing.

```bash
make docker-build-and-push TAG=v1.0.0
# Runs the build-and-push.sh script with specified tag
# Includes registry startup, health checks, and verification
```

### Helm Targets

#### `make helm-install`
Deploys the pipeline to Kubernetes using individual Helm charts (like quickstart.sh).

```bash
# Deploy with default settings
make helm-install
# Deploys to gpu-telemetry namespace using individual charts

# Deploy with custom tag and registry
make helm-install TAG=v1.0.0 REGISTRY=gcr.io/my-project
```

**Installation order**:
1. `shared-resources` (creates namespace)
2. `mq-service` (waits for ready)
3. `telemetry-collector` (waits for ready)
4. `api-gateway` (waits for available)
5. `dashboard` (waits for available)
6. `telemetry-streamer` (DaemonSet)

**Creates**:
- Kubernetes namespace `gpu-telemetry`
- Individual Helm releases for each component
- StatefulSets (mq-service, collector)
- Deployments (api-gateway, dashboard)
- DaemonSet (telemetry-streamer)
- Services and ConfigMaps

#### `make helm-uninstall`
Removes all individual Helm releases.

```bash
make helm-uninstall
# Removes all 5 individual releases:
# - telemetry-streamer
# - dashboard  
# - api-gateway
# - telemetry-collector
# - mq-service
# - shared-resources
```

#### `make helm-status`
Shows deployment status and pod information.

```bash
make helm-status
# Shows:
# - All Helm releases across namespaces
# - Pod status in gpu-telemetry namespace
# - Service status in gpu-telemetry namespace
```

### Build & Quality Targets

#### `make deps`
Downloads Go dependencies.

```bash
make deps
# Equivalent to: go mod tidy && go mod download
```

#### `make lint`
Runs code linter (golangci-lint or go vet).

```bash
make lint
# Checks code style
# Detects common issues
# Reports problems
```

#### `make openapi-gen`
Generates OpenAPI/Swagger documentation using swag tool.

```bash
make openapi-gen
# Installs swag if not present
# Creates:
# - api/swagger.json
# - api/swagger.yaml
# - api/docs.go (embedded)
# Also references: api/openapi.yaml (manual spec)
```

#### `make docs` 
Alias for `openapi-gen` for convenience.

```bash
make docs
# Same as make openapi-gen
```

---

## Workflow Targets

### `make all`
Complete build pipeline (recommended for final builds).

```bash
make all
# Runs: clean → deps → build → test → coverage → openapi-gen
```

### `make dev`
Development workflow (fast build + test).

```bash
make dev
# Runs: build → test
# Quick iteration for development
```

### `make ci`
CI pipeline simulation.

```bash
make ci
# Runs: clean → deps → build → test → coverage
# For continuous integration systems
```

### `make deploy`
Complete deployment pipeline.

```bash
make deploy
# Runs: clean → deps → build → test → docker-build → docker-push → helm-install
# Full production deployment
```

### `make docker-up`
Comprehensive Docker Compose deployment with health checks and sample data.

```bash
make docker-up
# 1. Builds all Docker images (including dashboard)
# 2. Creates and clears data directories 
# 3. Generates DCGM-format sample data
# 4. Starts all services with docker-compose
# 5. Waits 15 seconds for startup
# 6. Performs health checks on all services
# 7. Shows comprehensive endpoint information
```

**Created directories**:
- `deploy/docker/data/` (cleared on each run)
- `deploy/docker/mq-data/` (cleared on each run)  
- `deploy/docker/sample-data/` (with telemetry.csv)

**Service endpoints displayed**:
- 🌐 Dashboard: http://localhost:5173
- 🏥 MQ Health: http://localhost:9090/health
- 📊 MQ Stats: http://localhost:9090/stats  
- 🏥 Collector Health: http://localhost:8080/health
- 🌐 API Gateway: http://localhost:8081
- 🏥 Gateway Health: http://localhost:8081/health
- 📚 API Documentation: http://localhost:8081/swagger/

### `make docker-down`
Stops Docker Compose services.

```bash
make docker-down
# Stops all Docker Compose services
# Removes containers but keeps volumes
```

### `make docker-logs`
Shows Docker Compose logs.

```bash
make docker-logs
# Follows logs from all services
# Use Ctrl+C to exit
```

### `make docker-status`
Shows Docker Compose service status.

```bash
make docker-status
# Shows running containers and their status
```

### `make docker-health-check`
Checks health of all Docker services.

```bash
make docker-health-check
# Checks health endpoints:
# ✅ MQ Service is healthy
# ✅ Telemetry Collector is healthy  
# ✅ API Gateway is healthy
# ✅ Dashboard is healthy
# ⚠️  Some services are not healthy (if any fail)
```

## Script Integration

### `make docker-setup`
Uses the `deploy/docker/setup.sh` script directly.

```bash
make docker-setup
# Equivalent to: cd deploy/docker && ./setup.sh
# Uses existing images, comprehensive health checks
```

### `make docker-setup-build`
Uses the setup script with build flag.

```bash
make docker-setup-build TAG=v1.0.0
# Equivalent to: cd deploy/docker && ./setup.sh -b -t v1.0.0
# Builds images first, then starts services
```

### `make docker-setup-down`
Uses the setup script to teardown.

```bash
make docker-setup-down
# Equivalent to: cd deploy/docker && ./setup.sh -d
# Stops and removes all containers
```

## Helm Quickstart Integration

### `make helm-quickstart`
Uses the `deploy/helm/quickstart.sh` script for complete Kind cluster setup.

```bash
make helm-quickstart TAG=v1.0.0
# Equivalent to: cd deploy/helm && ./quickstart.sh up -t v1.0.0
# 1. Checks prerequisites (docker, kind, kubectl, helm, curl)
# 2. Starts local registry (kind-registry)
# 3. Creates Kind cluster with registry integration
# 4. Builds and pushes all Docker images
# 5. Deploys all Helm charts in correct order
# 6. Waits for all components to be ready
# 7. Sets up port forwarding
# 8. Shows access URLs and helpful commands
```

### `make helm-quickstart-down`
Complete cleanup using quickstart script.

```bash
make helm-quickstart-down
# Equivalent to: cd deploy/helm && ./quickstart.sh down
# 1. Kills port forwarding
# 2. Uninstalls all Helm releases
# 3. Deletes Kind cluster
# 4. Stops and removes local registry
```

### `make helm-quickstart-status`
Shows quickstart deployment status.

```bash
make helm-quickstart-status
# Equivalent to: cd deploy/helm && ./quickstart.sh status
# Shows cluster, registry, namespace, pods, and services
```

### `make helm-quickstart-logs`
Shows logs from all quickstart components.

```bash
make helm-quickstart-logs  
# Equivalent to: cd deploy/helm && ./quickstart.sh logs
# Shows logs from mq-service, collector, api-gateway, dashboard, streamer
```

### `make helm-port-forward`
Sets up port forwarding for Kubernetes services.

```bash
make helm-port-forward
# 1. Kills existing kubectl port-forward processes
# 2. Starts API Gateway port forward (localhost:8081)
# 3. Starts Dashboard port forward (localhost:8080)
# 4. Shows access URLs
# 5. Runs in background
```

**Access URLs after port forwarding**:
- 🌐 Dashboard: http://localhost:8080
- 🌐 API Gateway: http://localhost:8081  
- 🏥 API Health: http://localhost:8081/health

---

## Registry & Local Development

### `make registry-start`
Starts local Docker registry.

```bash
make registry-start
# Starts registry on localhost:5000
# Used for storing images locally
```

**Container**:
```
docker run -d -p 5000:5000 --name registry registry:2
```

### `make registry-stop`
Stops local Docker registry.

```bash
make registry-stop
# Stops and removes registry container
```

### `make registry-status`
Shows registry status and contents.

```bash
make registry-status
# ✅ Registry is running
# 📋 Registry contents: (lists available images)
# ❌ Registry is not running (if stopped)
```

## Help System

### `make help`
Shows all available targets with descriptions.

```bash
make help
# Shows: Complete list of all targets
# Includes: Build, test, Docker, Kubernetes targets
# Format: target - description
```

---

## Running Services Locally

### `make run-collector`
Runs telemetry collector locally.

```bash
make run-collector
# Starts: ./bin/telemetry-collector
# Args: --workers=4 --data-dir=./data --health-port=8080 --mq-url=http://localhost:9090
# Subscribes to MQ service for telemetry data
```

### `make run-streamer`
Runs telemetry streamer locally.

```bash
make run-streamer
# Starts: ./bin/telemetry-streamer
# Args: --csv-file=data/sample_gpu_data.csv --workers=2 --rate=5 --broker-url=http://localhost:9090
# Publishes sample data to MQ service
```

### `make run-api`
Runs API gateway locally.

```bash
make run-api
# Starts: ./bin/api-gateway
# Args: --port=8081 --data-dir=./data
# Provides REST API on port 8081
```

### `make run-mq`
Runs MQ service locally.

```bash
make run-mq
# Starts: ./bin/mq-service
# Args: --port=9090 --persistence --persistence-dir=./mq-data
# HTTP port 9090, gRPC port 9091
```

### `make sample-data`
Creates sample telemetry data for testing.

```bash
make sample-data
# Creates: data/sample_gpu_data.csv
# Contains: Sample GPU metrics for 4 GPUs
# Good for: Local testing without real GPU data
```

---

## Cleanup Targets

### `make clean`
Removes all build artifacts.

```bash
make clean
# Removes:
# - bin/ (compiled binaries)
# - coverage.out
# - coverage.html
# - api/docs.go
# - api/swagger.*
```

### `make clean-docker`
Removes Docker images.

```bash
make clean-docker
# Removes images for current TAG

# Remove specific tag
make clean-docker TAG=v1.0.0
```

## New Testing Features

### `make test-integration`
Runs integration tests specifically.

```bash
make test-integration
# Runs: Integration test suite
# Target: API Gateway integration tests
# Good for: Testing service interactions
```

---

## Example Workflows

### Development Cycle

```bash
# Initial setup
make deps
make sample-data
make build

# Development iteration
vim internal/api/handlers.go
make dev              # build + test

# Test specific component
make run-mq &         # Start MQ service
make run-collector &  # Start collector
make run-api &        # Start API gateway

# Check coverage
make coverage

# Run system tests
make system-tests-quick

# Ready to commit
make lint
```

### Release Build

```bash
# Prepare release
export TAG=v1.2.0
export REGISTRY=gcr.io/my-project

# Full build and test
make all

# Build and push containers
make docker-build TAG=$TAG REGISTRY=$REGISTRY
make docker-push TAG=$TAG REGISTRY=$REGISTRY

# Deploy to staging
make helm-install HELM_NAMESPACE=staging TAG=$TAG

# Deploy to production (after testing)
make helm-install HELM_NAMESPACE=production TAG=$TAG
```

### Local Testing with Docker

```bash
# Option 1: Comprehensive setup with health checks
make docker-up
# - Builds all images (including dashboard)
# - Creates sample DCGM data
# - Starts services and waits for health
# - Shows all endpoints

# Option 2: Use setup script (existing images)
make docker-setup

# Option 3: Use setup script with build
make docker-setup-build TAG=v1.0.0

# Test services
curl http://localhost:5173/         # Dashboard  
curl http://localhost:8081/health   # API Gateway
curl http://localhost:9090/health   # MQ Service  
curl http://localhost:8080/health   # Collector

# Check all service health at once
make docker-health-check

# View logs
make docker-logs

# Check status
make docker-status

# Cleanup options
make docker-down           # Stop compose services
make docker-setup-down     # Use setup script teardown 
make registry-stop         # Stop registry if started
```

### Kubernetes Development

```bash
# Option 1: Full quickstart (recommended for new development)
make helm-quickstart TAG=dev
# - Sets up Kind cluster with local registry
# - Builds and pushes all images  
# - Deploys all components
# - Sets up port forwarding
# Access: http://localhost:8080 (dashboard), http://localhost:8081 (API)

# Option 2: Manual setup
make registry-start
make docker-build-and-push TAG=dev
make helm-install TAG=dev

# Check status
make helm-quickstart-status  # Comprehensive status
make helm-status            # Basic Helm status

# View logs  
make helm-quickstart-logs   # All component logs
kubectl logs -n gpu-telemetry -l app.kubernetes.io/name=collector -f

# Port forwarding
make helm-port-forward      # Start port forwarding
pkill -f "kubectl port-forward"  # Stop port forwarding

# Iterate
# ... make code changes ...

# Rebuild and push
make docker-build-and-push TAG=dev-v2
# Update deployment with new tag
make helm-uninstall && make helm-install TAG=dev-v2

# Cleanup options
make helm-quickstart-down   # Complete cleanup (Kind cluster + registry)
make helm-uninstall         # Just remove Helm releases
```

### CI/CD Integration

```bash
# GitHub Actions example
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.24'
      - run: make ci              # Tests and coverage
      - run: make docker-build    # Build images
      - run: make docker-push     # Push to registry
      - run: make helm-install    # Deploy to staging
```

---

## Makefile Variables & Debugging

### View All Targets

```bash
make help
# Lists all available targets with descriptions
```

### Dry Run

```bash
# Show commands without executing
make -n docker-build

# Show what would happen
make -n deploy
```

### Verbose Output

```bash
# Show detailed execution
make -d docker-build

# Very detailed
make --debug docker-build
```

### Check Variable Values

```bash
# Show all variables
make -p | grep REGISTRY

# Show specific variable
echo $(REGISTRY)
```

---

## Troubleshooting

### Test Failures

```bash
# Full clean rebuild
make clean
make deps
make build

# Run different test suites
make test                     # Unit tests only
make test-integration        # Integration tests
make system-tests-quick      # Quick system tests  
make system-tests           # Full system tests

# Check specific package
go test ./internal/mq -v

# Debug system tests
cd tests && go test -v -timeout=10m -tags=system -run="TestSystem" ./...
```

### Test Failures

```bash
# Run single package tests
go test ./internal/mq -v

# Run specific test
go test -run TestPublish ./internal/mq -v

# With timeout
go test -timeout 30s ./...
```

### Docker Issues

```bash
# Registry not running
make registry-start
make registry-status

# Docker Compose issues  
make docker-status       # Check service status
make docker-health-check # Check service health endpoints
make docker-logs         # View logs
make docker-down         # Clean stop
make docker-up           # Restart with full setup

# Try different Docker approaches
make docker-setup        # Use setup.sh script
make docker-setup-build  # Use setup.sh with build
make docker-build-and-push # Use build-and-push.sh script

# Image build failed (check dashboard build)
make build-dashboard     # Test React build separately
docker build -t test -f deploy/docker/dashboard.Dockerfile .

# Check images
docker images | grep telemetry
docker image inspect dashboard:latest  # Check dashboard image

# Clean Docker state
make clean-docker
make docker-setup-down   # Full teardown with script
```

### Kubernetes Issues

```bash
# Use quickstart for comprehensive status
make helm-quickstart-status

# Check pod status (gpu-telemetry namespace)
kubectl get pods -n gpu-telemetry
make helm-status

# View component logs
make helm-quickstart-logs        # All components
kubectl logs -n gpu-telemetry -l app.kubernetes.io/name=collector -f
kubectl logs -n gpu-telemetry -l app.kubernetes.io/name=dashboard -f

# Debug individual pods
kubectl describe pod -n gpu-telemetry <pod-name>

# Test connectivity
kubectl exec -it <pod-name> -n gpu-telemetry -- curl http://api-gateway:8081/health

# Port forwarding issues
pkill -f "kubectl port-forward"  # Kill existing
make helm-port-forward           # Restart

# Complete reset
make helm-quickstart-down        # Full cleanup
make helm-quickstart TAG=latest  # Fresh setup

# Registry issues in Kind
make registry-status
docker exec kind-registry cat /etc/containerd/certs.d/localhost:5000/hosts.toml
```

---

## Performance Tips

### Faster Builds

```bash
# Use make parallelism
make -j4 all

# Skip tests for faster iteration during development
make build

# Only test changed packages
go test ./internal/mq -v
```

### Smaller Docker Images

```bash
# Multi-stage builds already used
# Check image sizes
docker images | grep telemetry

# Reduce further with Alpine base images
```

### Kubernetes Deployment

```bash
# Increase timeout for slow clusters
make helm-install HELM_NAMESPACE=production --timeout=600s

# Check what will be deployed
helm template telemetry-pipeline ./deploy/helm/charts
```

---

## New Features & Updates

### Enhanced Testing
- **System Tests**: Comprehensive end-to-end testing with `make system-tests`
- **Performance Tests**: Load testing with `make system-tests-performance`  
- **Quick Tests**: Fast iteration with `make system-tests-quick`
- **Integration Tests**: Service interaction testing with `make test-integration`

### Local Development
- **Individual Services**: Run each service independently (`run-mq`, `run-collector`, etc.)
- **Sample Data**: Generate test data with `make sample-data`
- **Docker Compose**: Simplified with `docker-up`/`docker-down`/`docker-logs`

### Registry Management
- **Smart Registry**: Automatic creation and management with `registry-start`
- **Registry Status**: Check contents and health with `registry-status`
- **Container Reuse**: Reuses existing registry containers

### Build System
- **Four Services**: Now builds `mq-service` in addition to the original three
- **Individual Builds**: Build specific components with `build-collector`, etc.
- **Help System**: Complete target documentation with `make help`

## Advanced Usage

### Custom Build Tags

```bash
# Include git commit hash
make docker-build TAG=$(git rev-parse --short HEAD)

# Include date
make docker-build TAG=$(date +%Y%m%d-%H%M%S)

# Include version from file
VERSION=$(cat VERSION)
make docker-build TAG=$VERSION
```

### Multiple Registries

```bash
# Push to multiple registries
make docker-build
docker tag telemetry-collector:latest myregistry.com/telemetry-collector:latest
docker push myregistry.com/telemetry-collector:latest
```

### Helm Customization

```bash
# Deploy with custom values
make helm-install HELM_RELEASE=my-release \
  --set collector.replicas=3 \
  --set api-gateway.replicas=3

# See what will be deployed
helm template my-release ./deploy/helm/charts
```

---

## Reference

### Make Syntax

```makefile
# Variables
VAR ?= default_value          # Assign if not set
VAR = $(shell command)        # Execute command

# Targets
target: dependency
	command
	@echo "Done"               # @ silences output
	@command                   # Run silently

# Phony targets (not files)
.PHONY: target                # Prevents file conflicts
```

### Common Commands Used

```bash
go build              # Build Go binaries
go test               # Run tests
go mod tidy           # Clean dependencies
docker build          # Build container
docker push           # Push to registry
helm install          # Deploy to Kubernetes
kubectl get pods      # Check pod status
```

---

For more information:
- [Quickstart Guide](../quickstart/README.md) - Get started in minutes
- [Deployment Guide](../deployment/README.md) - Detailed deployment instructions
- [Components Reference](../components/README.md) - Service descriptions
