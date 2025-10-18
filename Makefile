# Telemetry Pipeline Makefile

# Configuration
REGISTRY ?= localhost:5000
TAG ?= latest
HELM_RELEASE ?= telemetry-pipeline
HELM_NAMESPACE ?= default

.PHONY: help build test coverage clean clean-docker openapi-gen docker-build docker-push docker-deploy helm-install helm-uninstall helm-status run-collector run-streamer run-api lint deps all deploy dev ci registry-start registry-stop registry-status system-tests system-tests-quick system-tests-performance

# Default target (this will be replaced by the comprehensive help target later)
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
build: build-collector build-streamer build-api build-mq

build-collector:
	@echo "Building telemetry collector..."
	go build -o bin/telemetry-collector ./cmd/telemetry-collector

build-streamer:
	@echo "Building telemetry streamer..."
	go build -o bin/telemetry-streamer ./cmd/telemetry-streamer

build-api:
	@echo "Building API gateway..."
	go build -o bin/api-gateway ./cmd/api-gateway

build-mq:
	@echo "Building MQ service..."
	go build -o bin/mq-service ./cmd/mq-service

# Test targets
test:
	@echo "Running unit and integration tests with coverage..."
	@set -e; \
	pkgs=$$(go list ./... | grep -v '/tests/'); \
	for p in $$pkgs; do \
		GOTOOLCHAIN=local go test $$p -v -coverprofile=coverage_$$(echo $$p | tr '/' '-').out -tags="!system" || exit $$?; \
	done; \
	cat coverage_*.out > coverage.out
	@echo "Coverage profile saved to coverage.out"

coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out
	@echo "Coverage report generated: coverage.html"

# System test targets
system-tests: build
	@echo "Running comprehensive system tests..."
	@echo "This will test end-to-end functionality with all components"
	@echo "Building required binaries first..."
	@make build-collector build-streamer build-api >/dev/null 2>&1
	@echo "Starting system test suite..."
	cd tests && go test -v -timeout=10m -tags=system \
		-run="TestSystem|TestFunctional|TestPerformance|TestIntegration" \
		./...
	@echo "System tests completed!"

system-tests-quick: build
	@echo "Running quick system tests (functional only)..."
	@make build-collector build-streamer build-api >/dev/null 2>&1
	cd tests && go test -v -timeout=5m -tags=system \
		-run="TestSystemEndToEnd|TestSystemIntegration" \
		./...
	@echo "Quick system tests completed!"

system-tests-performance: build
	@echo "Running performance system tests..."
	@make build-collector build-streamer build-api >/dev/null 2>&1
	cd tests && go test -v -timeout=15m -tags=system \
		-run="TestSystemPerformance" \
		./...
	@echo "Performance tests completed!"

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
	./bin/telemetry-collector --workers=4 --data-dir=./data --health-port=8080 --mq-url=http://localhost:9090

run-streamer:
	@echo "Starting telemetry streamer..."
	./bin/telemetry-streamer --csv-file=data/sample_gpu_data.csv --workers=2 --rate=5 --broker-url=http://localhost:9090

run-api:
	@echo "Starting API gateway..."
	./bin/api-gateway --port=8081 --data-dir=./data

run-mq:
	@echo "Starting MQ service..."
	./bin/mq-service --port=9090 --persistence --persistence-dir=./mq-data

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
		echo "golangci-lint not found, installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh; \
		export PATH=$$PATH:$(CURDIR)/bin && golangci-lint run; \
	fi

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf api/docs.go api/swagger.json api/swagger.yaml
	@echo "Clean completed!"

clean-docker:
	@echo "Cleaning Docker images..."
	-docker rmi mq-service:$(TAG) 2>/dev/null || true
	-docker rmi telemetry-streamer:$(TAG) 2>/dev/null || true
	-docker rmi telemetry-collector:$(TAG) 2>/dev/null || true
	-docker rmi api-gateway:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/mq-service:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/telemetry-streamer:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/telemetry-collector:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/api-gateway:$(TAG) 2>/dev/null || true
	@echo "Docker images cleaned!"

# Docker targets
docker-build:
	@echo "Building Docker images with tag: $(TAG)"
	@echo "Registry: $(REGISTRY)"
	docker build -f deploy/docker/mq-service.Dockerfile -t mq-service:$(TAG) .
	docker build -f deploy/docker/telemetry-streamer.Dockerfile -t telemetry-streamer:$(TAG) .
	docker build -f deploy/docker/telemetry-collector.Dockerfile -t telemetry-collector:$(TAG) .
	docker build -f deploy/docker/api-gateway.Dockerfile -t api-gateway:$(TAG) .
	@echo "Tagging images for registry $(REGISTRY)..."
	docker tag mq-service:$(TAG) $(REGISTRY)/mq-service:$(TAG)
	docker tag telemetry-streamer:$(TAG) $(REGISTRY)/telemetry-streamer:$(TAG)
	docker tag telemetry-collector:$(TAG) $(REGISTRY)/telemetry-collector:$(TAG)
	docker tag api-gateway:$(TAG) $(REGISTRY)/api-gateway:$(TAG)
	@echo "Docker images built successfully!"

# Docker push target moved to end of file

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

# Docker Compose targets
docker-up: docker-build
	@echo "Starting services with Docker Compose..."
	@mkdir -p deploy/docker/data deploy/docker/mq-data
	@chmod 755 deploy/docker/data deploy/docker/mq-data
	cd deploy/docker && docker-compose up -d
	@echo "Services started! Check http://localhost:8081"
	@echo "API Gateway: http://localhost:8081"
	@echo "MQ Service: http://localhost:9090"
	@echo "Collector Health: http://localhost:8080/health"

docker-down:
	@echo "Stopping services with Docker Compose..."
	cd deploy/docker && docker-compose down
	@echo "Services stopped!"

docker-logs:
	@echo "Showing Docker Compose logs..."
	cd deploy/docker && docker-compose logs -f

docker-status:
	@echo "Checking Docker Compose status..."
	cd deploy/docker && docker-compose ps

# Quick docker deployment (backward compatibility)
docker-deploy: docker-up

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

# Generate API documentation
docs:
	@echo "Generating API documentation..."
	@which swag > /dev/null || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	swag init -g cmd/api-gateway/main.go -o api/
	@echo "API documentation generated in api/ directory"

# Docker push target for releasing images
docker-push:
	@echo "Pushing Docker images to registry: $(REGISTRY)"
	docker push $(REGISTRY)/mq-service:$(TAG)
	docker push $(REGISTRY)/telemetry-streamer:$(TAG)
	docker push $(REGISTRY)/telemetry-collector:$(TAG)
	docker push $(REGISTRY)/api-gateway:$(TAG)
	@echo "Docker images pushed successfully!"

# Integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v ./cmd/api-gateway/... -tags=integration
	@echo "Integration tests completed!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build             - Build all components (collector, streamer, api-gateway, mq-service)"
	@echo "  test              - Run unit tests"
	@echo "  test-integration  - Run integration tests"
	@echo "  coverage          - Generate test coverage report"
	@echo "  lint              - Run linter"
	@echo "  clean             - Clean build artifacts"
	@echo "  clean-docker      - Clean Docker images"
	@echo "  docker-build      - Build Docker images for all services"
	@echo "  docker-push       - Push Docker images to registry"
	@echo "  docker-up         - Start all services with Docker Compose"
	@echo "  docker-down       - Stop Docker Compose services"
	@echo "  docker-logs       - Show Docker Compose logs"
	@echo "  docker-status     - Check Docker Compose status"
	@echo "  docs              - Generate API documentation"
	@echo "  deps              - Install dependencies"
	@echo "  registry-start    - Start local Docker registry"
	@echo "  registry-stop     - Stop local Docker registry"
	@echo "  registry-status   - Check registry status"
	@echo "  sample-data       - Create sample test data"
	@echo "  help              - Show this help message"

.PHONY: build test test-integration coverage lint clean clean-docker docker-build docker-push docs deps deploy undeploy registry-up registry-down registry-status sample-data help