# Makefile Documentation

Complete guide to all available Makefile targets for building, testing, and deploying the GPU Telemetry Pipeline.

---

## Quick Reference

```bash
# Build and test
make dev              # Build + test (development workflow)
make all              # Full build pipeline
make ci               # CI pipeline (build + test + coverage)

# Docker deployment
make docker-build     # Build Docker images
make docker-push      # Push to registry
make docker-deploy    # Docker Compose deployment

# Kubernetes deployment
make helm-install     # Deploy to Kubernetes
make helm-status      # Check deployment status

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
| `HELM_RELEASE` | `telemetry-pipeline` | Helm release name |
| `HELM_NAMESPACE` | `telemetry-pipeline` | Kubernetes namespace |

---

## Primary Targets

### Build Targets

#### `make build`
Builds all three Go binaries.

```bash
make build
# Output:
# bin/telemetry-streamer
# bin/telemetry-collector
# bin/api-gateway
```

**Equivalent to**:
```bash
go build -o bin/telemetry-streamer ./cmd/telemetry-streamer
go build -o bin/telemetry-collector ./cmd/telemetry-collector
go build -o bin/api-gateway ./cmd/api-gateway
```

#### `make test`
Runs all Go tests with coverage reporting.

```bash
make test
# Output: coverage.out
# Shows: tests run, passed, failed
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
- `telemetry-streamer:TAG`
- `telemetry-collector:TAG`
- `api-gateway:TAG`
- `mq-service:TAG`
- `dashboard:TAG`

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

**Builds first**, then pushes.

### Helm Targets

#### `make helm-install`
Deploys the pipeline to Kubernetes using Helm.

```bash
# Deploy with default settings
make helm-install
# Deploys to telemetry-pipeline namespace

# Deploy to custom namespace
make helm-install HELM_NAMESPACE=production HELM_RELEASE=prod-telemetry
# Deploys to production namespace with release name prod-telemetry
```

**Creates**:
- Kubernetes namespace
- Deployments (streamer, collector, api-gateway, mq, dashboard)
- Services (internal cluster IPs)
- PersistentVolumeClaims
- ConfigMaps

#### `make helm-uninstall`
Removes Helm deployment.

```bash
make helm-uninstall
# Removes release and resources

# Uninstall specific release
make helm-uninstall HELM_RELEASE=prod-telemetry HELM_NAMESPACE=production
```

#### `make helm-status`
Shows deployment status and pod information.

```bash
make helm-status
# Shows:
# - Release status
# - Pod status
# - Service endpoints
# - Resource usage
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
Generates OpenAPI/Swagger documentation.

```bash
make openapi-gen
# Creates:
# - api/docs.go
# - api/swagger.json
# - api/swagger.yaml
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

### `make docker-deploy`
Docker Compose deployment.

```bash
make docker-deploy
# Builds images and starts with Docker Compose
# Good for quick testing without Kubernetes
```

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
# Lists all images in registry
# Shows registry health
```

---

## Running Services Locally

### `make run-collector`
Runs telemetry collector locally.

```bash
make run-collector
# Starts: ./bin/telemetry-collector
# With default configuration
# Good for testing collector in isolation
```

### `make run-streamer`
Runs telemetry streamer locally.

```bash
make run-streamer
# Starts: ./bin/telemetry-streamer
# With default configuration
# Streams sample data to local MQ
```

### `make run-api`
Runs API gateway locally.

```bash
make run-api
# Starts: ./bin/api-gateway
# Provides REST API on port 8081
# Aggregates health from services
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

### `make clean-all`
Full cleanup (binaries, Docker, volumes).

```bash
make clean-all
# Removes binaries, images, and Docker volumes
# Use with caution - removes data
```

---

## Example Workflows

### Development Cycle

```bash
# Setup
make deps
make build

# Development iteration
vim internal/api/handlers.go
make dev              # build + test

# Check coverage
make coverage

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
# Start registry
make registry-start

# Build images
make docker-build

# Deploy with Docker Compose
make docker-deploy

# Test API
curl http://localhost:8081/health

# Cleanup
cd deploy/docker && docker-compose down
make registry-stop
```

### Kubernetes Development

```bash
# Start local registry
make registry-start

# Build and push images
make docker-build REGISTRY=localhost:5000 TAG=dev
make docker-push REGISTRY=localhost:5000 TAG=dev

# Deploy to Kubernetes
make helm-install HELM_NAMESPACE=dev TAG=dev

# Check status
make helm-status

# View logs
kubectl logs -n dev -l app=collector -f

# Iterate
# ... make code changes ...

# Redeploy
make docker-build TAG=dev
make docker-push TAG=dev
# Kubernetes will pull new image on next pod restart

# Cleanup
make helm-uninstall HELM_NAMESPACE=dev
make registry-stop
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

### Build Failures

```bash
# Full clean rebuild
make clean
make deps
make build

# Check specific package
go build ./cmd/telemetry-collector
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

# Image build failed
docker build -t test -f deploy/docker/telemetry-collector.Dockerfile .

# Check image
docker image inspect telemetry-collector:latest
```

### Kubernetes Issues

```bash
# Check pod status
kubectl get pods -n telemetry-pipeline

# View pod logs
kubectl logs -n telemetry-pipeline -l app=collector

# Debug pod
kubectl describe pod -n telemetry-pipeline <pod-name>

# Test connectivity
kubectl exec -it <pod-name> -n telemetry-pipeline -- curl http://api-gateway:8081/health
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
