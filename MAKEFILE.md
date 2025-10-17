# Telemetry Pipeline - Makefile Documentation

This document describes all available Makefile targets for the telemetry pipeline project.

## Quick Start

```bash
# Build everything
make all

# Complete deployment to Kubernetes
make deploy

# Development workflow
make dev
```

## Configuration Variables

Set these environment variables or pass them to make:

```bash
make docker-build TAG=v1.0.0 REGISTRY=my-registry.com:5000
```

| Variable | Description | Default |
|----------|-------------|---------|
| `REGISTRY` | Docker registry URL | `localhost:5000` |
| `TAG` | Docker image tag | `latest` |
| `HELM_RELEASE` | Helm release name | `telemetry-pipeline` |
| `HELM_NAMESPACE` | Kubernetes namespace | `default` |

## Primary Targets

### Build Targets

#### `make build`
Builds all three binaries:
- `bin/telemetry-streamer`
- `bin/telemetry-collector` 
- `bin/api-gateway`

```bash
make build
```

#### `make test`
Runs all tests with coverage profile generation:
```bash
make test
# Generates coverage.out
```

#### `make coverage`
Generates HTML coverage report and displays coverage summary:
```bash
make coverage
# Generates coverage.html and shows coverage percentages
```

### Docker Targets

#### `make docker-build`
Builds Docker images using multi-stage Dockerfiles:

```bash
# Build with default tag (latest)
make docker-build

# Build with custom tag
make docker-build TAG=v1.0.0

# Build for different registry
make docker-build TAG=v1.0.0 REGISTRY=gcr.io/my-project
```

Creates images:
- `telemetry-streamer:TAG`
- `telemetry-collector:TAG`
- `api-gateway:TAG`
- `REGISTRY/telemetry-streamer:TAG`
- `REGISTRY/telemetry-collector:TAG`
- `REGISTRY/api-gateway:TAG`

#### `make docker-push`
Builds and pushes Docker images to registry:

```bash
# Push to default registry (localhost:5000)
make docker-push

# Push to custom registry
make docker-push REGISTRY=gcr.io/my-project TAG=v1.0.0
```

### Helm Targets

#### `make helm-install`
Installs the telemetry pipeline to Kubernetes using Helm:

```bash
# Install with default settings
make helm-install

# Install with custom configuration
make helm-install HELM_RELEASE=my-telemetry HELM_NAMESPACE=monitoring TAG=v1.0.0
```

Equivalent to:
```bash
helm upgrade --install telemetry-pipeline ./deploy/helm/telemetry-pipeline \
  --namespace default \
  --create-namespace \
  --set streamer.image.registry=localhost:5000 \
  --set streamer.image.tag=latest \
  --set collector.image.registry=localhost:5000 \
  --set collector.image.tag=latest \
  --set apiGateway.image.registry=localhost:5000 \
  --set apiGateway.image.tag=latest \
  --wait --timeout=300s
```

#### `make helm-uninstall`
Removes the Helm release:

```bash
make helm-uninstall HELM_RELEASE=my-telemetry HELM_NAMESPACE=monitoring
```

#### `make helm-status`
Shows Helm release status and pod information:

```bash
make helm-status
```

### OpenAPI Target

#### `make openapi-gen`
Regenerates OpenAPI specification from code comments:

```bash
make openapi-gen
```

Generates:
- `api/docs.go` - Go documentation package
- `api/swagger.json` - OpenAPI JSON spec
- `api/swagger.yaml` - OpenAPI YAML spec

Uses the `swag` tool to parse Swagger comments in the codebase.

## Workflow Targets

### `make all`
Complete build pipeline:
```bash
make all
# Equivalent to: clean deps build test coverage openapi-gen
```

### `make deploy` 
Complete deployment pipeline:
```bash
make deploy
# Equivalent to: clean deps build test docker-build docker-push helm-install
```

### `make dev`
Development workflow:
```bash
make dev
# Equivalent to: build test
```

### `make ci`
CI pipeline simulation:
```bash
make ci
# Equivalent to: clean deps build test coverage
```

### `make docker-deploy`
Quick Docker Compose deployment:
```bash
make docker-deploy
# Builds images and starts services with Docker Compose
```

## Utility Targets

### Registry Management

#### `make registry-start`
Starts local Docker registry on port 5000:
```bash
make registry-start
```

#### `make registry-stop`
Stops local Docker registry:
```bash
make registry-stop
```

#### `make registry-status`
Shows registry status and contents:
```bash
make registry-status
```

### Cleanup Targets

#### `make clean`
Removes build artifacts:
```bash
make clean
# Removes: bin/, coverage.out, coverage.html, api/docs.go, api/swagger.*
```

#### `make clean-docker`
Removes Docker images:
```bash
make clean-docker TAG=v1.0.0
# Removes local and registry-tagged images
```

### Development Targets

#### `make deps`
Installs Go dependencies:
```bash
make deps
# Runs: go mod tidy && go mod download
```

#### `make lint`
Runs linter (golangci-lint if available, otherwise go vet + go fmt):
```bash
make lint
```

#### `make run-collector`
Runs telemetry collector locally:
```bash
make run-collector
```

#### `make run-streamer`
Runs telemetry streamer locally:
```bash
make run-streamer
```

#### `make run-api`
Runs API gateway locally:
```bash
make run-api
```

## Example Workflows

### Development Workflow

```bash
# Initial setup
make deps
make build

# Development cycle
vim internal/api/handlers.go
make dev  # build + test

# Generate API docs
make openapi-gen
```

### Docker Development

```bash
# Start local registry
make registry-start

# Build and deploy with Docker
make docker-deploy

# Check services
curl http://localhost:8081/health
curl http://localhost:8081/api/v1/gpus

# Clean up
cd deploy/docker && ./setup.sh -d
```

### Kubernetes Development

```bash
# Build and push images
make docker-build docker-push

# Deploy to Kubernetes
make helm-install

# Check deployment
kubectl get pods -l app.kubernetes.io/instance=telemetry-pipeline
kubectl port-forward svc/telemetry-pipeline-api-gateway 8081:80

# Access API
curl http://localhost:8081/health

# Clean up
make helm-uninstall
```

### Production Release

```bash
# Prepare release
export TAG=v1.2.0
export REGISTRY=gcr.io/my-project

# Full deployment pipeline
make deploy

# Verify deployment
make helm-status
kubectl get pods -n production
```

### CI/CD Integration

```bash
# In CI pipeline
make ci

# In CD pipeline (after tests pass)
make docker-build docker-push
make helm-install HELM_NAMESPACE=staging

# Production deployment
make helm-install HELM_NAMESPACE=production TAG=v1.2.0
```

## Troubleshooting

### Common Issues

1. **Registry not available**
   ```bash
   make registry-start
   make registry-status
   ```

2. **Helm installation fails**
   ```bash
   # Check cluster connection
   kubectl cluster-info
   
   # Check Helm status
   helm list -A
   
   # Debug installation
   make helm-status
   ```

3. **Build failures**
   ```bash
   # Clean and rebuild
   make clean
   make deps
   make build
   ```

4. **Test failures**
   ```bash
   # Run specific package tests
   go test ./internal/api -v
   
   # Check coverage
   make coverage
   ```

### Debugging Commands

```bash
# Show all make targets
make help

# Dry run (show commands without executing)
make -n docker-build

# Verbose output
make -d docker-build

# Check variable values
make -p | grep REGISTRY
```