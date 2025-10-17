# Telemetry Pipeline Makefile

# Configuration
REGISTRY ?= localhost:5000
TAG ?= latest
HELM_RELEASE ?= telemetry-pipeline
HELM_NAMESPACE ?= default

.PHONY: help build test coverage clean clean-docker openapi-gen docker-build docker-push docker-deploy helm-install helm-uninstall helm-status run-collector run-streamer run-api lint deps all deploy dev ci registry-start registry-stop registry-status

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build all binaries"
	@echo "  test          - Run all tests with coverage"
	@echo "  coverage      - Generate coverage report"
	@echo "  docker-build  - Build Docker images (TAG=$(TAG))"
	@echo "  docker-push   - Push Docker images to registry ($(REGISTRY))"
	@echo "  helm-install  - Install to local cluster with Helm"
	@echo "  openapi-gen   - Generate OpenAPI specification"
	@echo "  clean         - Clean build artifacts"
	@echo "  run-collector - Run telemetry collector"
	@echo "  run-streamer  - Run telemetry streamer"
	@echo "  run-api       - Run API gateway"
	@echo "  lint          - Run linter"
	@echo "  deps          - Install dependencies"
	@echo ""
	@echo "Configuration:"
	@echo "  REGISTRY      - Docker registry (default: $(REGISTRY))"
	@echo "  TAG           - Docker image tag (default: $(TAG))"
	@echo "  HELM_RELEASE  - Helm release name (default: $(HELM_RELEASE))"
	@echo "  HELM_NAMESPACE- Helm namespace (default: $(HELM_NAMESPACE))"

# Build targets
build: build-collector build-streamer build-api

build-collector:
	@echo "Building telemetry collector..."
	go build -o bin/telemetry-collector ./cmd/telemetry-collector

build-streamer:
	@echo "Building telemetry streamer..."
	go build -o bin/telemetry-streamer ./cmd/telemetry-streamer

build-api:
	@echo "Building API gateway..."
	go build -o bin/api-gateway ./cmd/api-gateway

# Test targets
test:
	@echo "Running tests with coverage..."
	go test ./... -v -coverprofile=coverage.out
	@echo "Coverage profile saved to coverage.out"

coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out
	@echo "Coverage report generated: coverage.html"

# OpenAPI generation
openapi-gen:
	@echo "Generating OpenAPI specification..."
	@command -v swag >/dev/null 2>&1 || { echo "Installing swag..."; go install github.com/swaggo/swag/cmd/swag@latest; }
	~/go/bin/swag init -g cmd/api-gateway/main.go -o api --parseDependency --parseInternal
	@echo "OpenAPI spec generated in api/swagger.json and api/swagger.yaml"
	@echo "Manual spec available in api/openapi.yaml"

# Run targets
run-collector:
	@echo "Starting telemetry collector..."
	./bin/telemetry-collector --workers=4 --data-dir=./data --health-port=8080

run-streamer:
	@echo "Starting telemetry streamer..."
	./bin/telemetry-streamer --csv=sample_data.csv --workers=2 --rate=5

run-api:
	@echo "Starting API gateway..."
	./bin/api-gateway --port=8081 --data-dir=./data

# Development targets
dev-setup: deps build
	@echo "Development environment ready!"

deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, running basic checks..."; \
		go vet ./...; \
		go fmt ./...; \
	fi

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf api/docs.go api/swagger.json api/swagger.yaml
	@echo "Clean completed!"

clean-docker:
	@echo "Cleaning Docker images..."
	-docker rmi telemetry-streamer:$(TAG) 2>/dev/null || true
	-docker rmi telemetry-collector:$(TAG) 2>/dev/null || true
	-docker rmi api-gateway:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/telemetry-streamer:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/telemetry-collector:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/api-gateway:$(TAG) 2>/dev/null || true
	@echo "Docker images cleaned!"

# Docker targets
docker-build:
	@echo "Building Docker images with tag: $(TAG)"
	@echo "Registry: $(REGISTRY)"
	docker build -f deploy/docker/telemetry-streamer.Dockerfile -t telemetry-streamer:$(TAG) .
	docker build -f deploy/docker/telemetry-collector.Dockerfile -t telemetry-collector:$(TAG) .
	docker build -f deploy/docker/api-gateway.Dockerfile -t api-gateway:$(TAG) .
	@echo "Tagging images for registry $(REGISTRY)..."
	docker tag telemetry-streamer:$(TAG) $(REGISTRY)/telemetry-streamer:$(TAG)
	docker tag telemetry-collector:$(TAG) $(REGISTRY)/telemetry-collector:$(TAG)
	docker tag api-gateway:$(TAG) $(REGISTRY)/api-gateway:$(TAG)
	@echo "Docker images built successfully!"

docker-push: docker-build
	@echo "Pushing Docker images to registry: $(REGISTRY)"
	docker push $(REGISTRY)/telemetry-streamer:$(TAG)
	docker push $(REGISTRY)/telemetry-collector:$(TAG)
	docker push $(REGISTRY)/api-gateway:$(TAG)
	@echo "Docker images pushed successfully!"

# Helm targets
helm-install:
	@echo "Installing telemetry-pipeline with Helm..."
	@echo "Release: $(HELM_RELEASE)"
	@echo "Namespace: $(HELM_NAMESPACE)"
	@echo "Registry: $(REGISTRY)"
	@echo "Tag: $(TAG)"
	helm upgrade --install $(HELM_RELEASE) ./deploy/helm/telemetry-pipeline \
		--namespace $(HELM_NAMESPACE) \
		--create-namespace \
		--set streamer.image.registry=$(REGISTRY) \
		--set streamer.image.tag=$(TAG) \
		--set collector.image.registry=$(REGISTRY) \
		--set collector.image.tag=$(TAG) \
		--set apiGateway.image.registry=$(REGISTRY) \
		--set apiGateway.image.tag=$(TAG) \
		--wait \
		--timeout=300s
	@echo "Helm installation completed!"
	@echo ""
	@echo "Check status with:"
	@echo "  kubectl get pods -n $(HELM_NAMESPACE) -l app.kubernetes.io/instance=$(HELM_RELEASE)"
	@echo ""
	@echo "Access API Gateway:"
	@echo "  kubectl port-forward -n $(HELM_NAMESPACE) svc/$(HELM_RELEASE)-api-gateway 8081:80"

helm-uninstall:
	@echo "Uninstalling Helm release: $(HELM_RELEASE)"
	helm uninstall $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)
	@echo "Helm release uninstalled!"

helm-status:
	@echo "Checking Helm release status..."
	helm status $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)
	@echo ""
	kubectl get pods -n $(HELM_NAMESPACE) -l app.kubernetes.io/instance=$(HELM_RELEASE)

# Demo and integration testing
demo: build
	@echo "Running integration demo..."
	./demo.sh

# All-in-one targets
all: clean deps build test coverage openapi-gen

# Complete deployment pipeline
deploy: clean deps build test docker-build docker-push helm-install
	@echo "Complete deployment pipeline finished!"
	@echo "Application deployed to Kubernetes cluster"

# Development workflow
dev: build test
	@echo "Development build completed!"

# CI/CD pipeline simulation
ci: clean deps build test coverage
	@echo "CI pipeline completed successfully!"
	@echo "Coverage report: coverage.html"

# Quick docker deployment
docker-deploy: docker-build
	@echo "Starting services with Docker Compose..."
	cd deploy/docker && ./setup.sh -b
	@echo "Services started! Check http://localhost:8081"

# Registry management
registry-start:
	@echo "Starting local Docker registry..."
	@if ! docker ps --filter "name=kind-registry" --filter "status=running" --format "{{.Names}}" | grep -q "^kind-registry$$"; then \
		if docker ps -a --filter "name=kind-registry" --format "{{.Names}}" | grep -q "^kind-registry$$"; then \
			echo "Starting existing registry container..."; \
			docker start kind-registry; \
		else \
			echo "Creating new registry container..."; \
			docker run -d --restart=always --name kind-registry -p 5000:5000 registry:2; \
		fi; \
	else \
		echo "Registry is already running"; \
	fi
	@echo "Registry available at: http://localhost:5000"

registry-stop:
	@echo "Stopping local Docker registry..."
	-docker stop kind-registry 2>/dev/null || true
	@echo "Registry stopped"

registry-status:
	@echo "Checking registry status..."
	@if docker ps --filter "name=kind-registry" --filter "status=running" --format "{{.Names}}" | grep -q "^kind-registry$$"; then \
		echo "✅ Registry is running"; \
		echo "📋 Registry contents:"; \
		curl -s http://localhost:5000/v2/_catalog 2>/dev/null || echo "Could not fetch catalog"; \
	else \
		echo "❌ Registry is not running"; \
	fi

# Create sample data for testing
sample-data:
	@echo "Creating sample data..."
	@mkdir -p data
	@echo "gpu_id,temperature,utilization,memory_used,power_draw,fan_speed" > data/sample_gpu_data.csv
	@echo "gpu_0,72.3,85.5,4096,250.5,2500" >> data/sample_gpu_data.csv
	@echo "gpu_1,75.1,90.2,8192,275.8,2750" >> data/sample_gpu_data.csv
	@echo "gpu_2,65.8,45.0,2048,180.2,1800" >> data/sample_gpu_data.csv
	@echo "gpu_3,78.9,95.1,12288,295.0,3000" >> data/sample_gpu_data.csv
	@echo "gpu_0,74.1,87.2,4200,255.1,2550" >> data/sample_gpu_data.csv
	@echo "gpu_1,76.8,92.5,8300,280.2,2800" >> data/sample_gpu_data.csv
	@echo "gpu_2,67.2,48.3,2100,185.0,1850" >> data/sample_gpu_data.csv
	@echo "gpu_3,79.5,96.8,12400,298.5,3050" >> data/sample_gpu_data.csv
	@echo "Sample data created in data/sample_gpu_data.csv"