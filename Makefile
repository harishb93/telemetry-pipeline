# Telemetry Pipeline Makefile

# Configuration
REGISTRY ?= localhost:5000
TAG ?= latest
HELM_RELEASE ?= telemetry-pipeline
HELM_NAMESPACE ?= default

.PHONY: help build build-for-system-tests build-dashboard test coverage clean clean-docker openapi-gen docker-build docker-build-and-push docker-push docker-deploy helm-install helm-uninstall helm-status helm-quickstart helm-quickstart-down helm-quickstart-status helm-quickstart-logs helm-port-forward run-collector run-streamer run-api run-mq lint deps all deploy dev ci registry-start registry-stop registry-status system-tests system-tests-quick system-tests-performance docker-up docker-down docker-logs docker-status docker-health-check docker-setup docker-setup-build docker-setup-down sample-data

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
build: build-collector build-streamer build-api build-mq build-dashboard

# Build system-test targets
build-for-system-tests: build-collector build-streamer build-api build-mq

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

build-dashboard:
	@echo "Building React dashboard..."
	@if [ -d "dashboard" ]; then \
		cd dashboard && npm ci && npm run build; \
		echo "Dashboard built successfully!"; \
	else \
		echo "Dashboard directory not found, skipping..."; \
	fi

# Test targets
test:
	@echo "Running unit and integration tests with coverage..."
	@pkgs=$$(go list ./... | grep -v '/tests/'); \
	mode_written=false; \
	rm -f coverage.out; \
	for p in $$pkgs; do \
		echo "Testing $$p"; \
		GOTOOLCHAIN=local go test $$p -v -covermode=atomic -coverprofile=profile.out -tags="!system" || exit $$?; \
		if [ -f profile.out ]; then \
			if ! $$mode_written; then \
				cat profile.out > coverage.out; \
				mode_written=true; \
			else \
				grep -h -v "^mode:" profile.out >> coverage.out; \
			fi; \
			rm -f profile.out; \
		fi; \
	done
	@echo "Coverage profile saved to coverage.out"

coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out
	@echo "Coverage report generated: coverage.html"

# System test targets
system-tests: build-for-system-tests
	@echo "Running comprehensive system tests..."
	@echo "This will test end-to-end functionality with all components"
	@echo "Building required binaries first..."
	@make build-collector build-streamer build-api >/dev/null 2>&1
	@echo "Starting system test suite..."
	cd tests && go test -v -timeout=10m -tags=system \
		-run="TestSystem|TestFunctional|TestPerformance|TestIntegration" \
		./...
	@echo "System tests completed!"

system-tests-quick: build-for-system-tests
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
	-docker rmi dashboard:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/mq-service:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/telemetry-streamer:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/telemetry-collector:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/api-gateway:$(TAG) 2>/dev/null || true
	-docker rmi $(REGISTRY)/dashboard:$(TAG) 2>/dev/null || true
	@echo "Docker images cleaned!"

# Docker targets
docker-build:
	@echo "Building Docker images with tag: $(TAG)"
	@echo "Registry: $(REGISTRY)"
	@echo "Making entrypoint scripts executable..."
	@chmod +x deploy/docker/entrypoint-*.sh 2>/dev/null || true
	docker build -f deploy/docker/mq-service.Dockerfile -t mq-service:$(TAG) .
	docker build -f deploy/docker/telemetry-streamer.Dockerfile -t telemetry-streamer:$(TAG) .
	docker build -f deploy/docker/telemetry-collector.Dockerfile -t telemetry-collector:$(TAG) .
	docker build -f deploy/docker/api-gateway.Dockerfile -t api-gateway:$(TAG) .
	docker build -f deploy/docker/dashboard.Dockerfile -t dashboard:$(TAG) .
	@echo "Tagging images for registry $(REGISTRY)..."
	docker tag mq-service:$(TAG) $(REGISTRY)/mq-service:$(TAG)
	docker tag telemetry-streamer:$(TAG) $(REGISTRY)/telemetry-streamer:$(TAG)
	docker tag telemetry-collector:$(TAG) $(REGISTRY)/telemetry-collector:$(TAG)
	docker tag api-gateway:$(TAG) $(REGISTRY)/api-gateway:$(TAG)
	docker tag dashboard:$(TAG) $(REGISTRY)/dashboard:$(TAG)
	@echo "Docker images built successfully!"

# Docker push target moved to end of file

# Helm targets (using individual charts like quickstart.sh)
HELM_NAMESPACE_REAL ?= gpu-telemetry

helm-install:
	@echo "Installing telemetry-pipeline components with Helm..."
	@echo "Registry: $(REGISTRY)"
	@echo "Tag: $(TAG)"
	@echo "Namespace: $(HELM_NAMESPACE_REAL)"
	@echo ""
	@echo "Installing shared-resources..."
	@cd deploy/helm && helm install shared-resources charts/shared-resources/ || true
	@sleep 2
	@echo "Installing mq-service..."
	@cd deploy/helm && helm install mq-service charts/mq-service/ --namespace $(HELM_NAMESPACE_REAL) \
		--set image.registry=$(REGISTRY) --set image.tag=$(TAG) || true
	@echo "Waiting for mq-service to be ready..."
	@kubectl wait --for=condition=ready --timeout=300s pod -l app.kubernetes.io/name=mq-service -n $(HELM_NAMESPACE_REAL) || true
	@echo "Installing telemetry-collector..."
	@cd deploy/helm && helm install telemetry-collector charts/telemetry-collector/ --namespace $(HELM_NAMESPACE_REAL) \
		--set image.registry=$(REGISTRY) --set image.tag=$(TAG) || true
	@kubectl wait --for=condition=ready --timeout=300s pod -l app.kubernetes.io/name=telemetry-collector -n $(HELM_NAMESPACE_REAL) || true
	@echo "Installing api-gateway..."
	@cd deploy/helm && helm install api-gateway charts/api-gateway/ --namespace $(HELM_NAMESPACE_REAL) \
		--set image.registry=$(REGISTRY) --set image.tag=$(TAG) || true
	@kubectl wait --for=condition=available --timeout=300s deployment/api-gateway -n $(HELM_NAMESPACE_REAL) || true
	@echo "Installing dashboard..."
	@cd deploy/helm && helm install dashboard charts/dashboard/ --namespace $(HELM_NAMESPACE_REAL) \
		--set image.registry=$(REGISTRY) --set image.tag=$(TAG) || true
	@kubectl wait --for=condition=available --timeout=300s deployment/dashboard -n $(HELM_NAMESPACE_REAL) || true
	@echo "Installing telemetry-streamer..."
	@cd deploy/helm && helm install telemetry-streamer charts/telemetry-streamer/ --namespace $(HELM_NAMESPACE_REAL) \
		--set image.registry=$(REGISTRY) --set image.tag=$(TAG) || true
	@echo ""
	@echo "✅ Helm installation completed!"
	@echo ""
	@echo "Check status with:"
	@echo "  kubectl get pods -n $(HELM_NAMESPACE_REAL)"
	@echo "  make helm-status"

helm-uninstall:
	@echo "Uninstalling Helm releases..."
	@echo "Uninstalling telemetry-streamer..."
	@-helm uninstall telemetry-streamer -n $(HELM_NAMESPACE_REAL) 2>/dev/null || true
	@echo "Uninstalling dashboard..."
	@-helm uninstall dashboard -n $(HELM_NAMESPACE_REAL) 2>/dev/null || true
	@echo "Uninstalling api-gateway..."
	@-helm uninstall api-gateway -n $(HELM_NAMESPACE_REAL) 2>/dev/null || true
	@echo "Uninstalling telemetry-collector..."
	@-helm uninstall telemetry-collector -n $(HELM_NAMESPACE_REAL) 2>/dev/null || true
	@echo "Uninstalling mq-service..."
	@-helm uninstall mq-service -n $(HELM_NAMESPACE_REAL) 2>/dev/null || true
	@echo "Uninstalling shared-resources..."
	@-helm uninstall shared-resources 2>/dev/null || true
	@echo "✅ All Helm releases uninstalled!"

helm-status:
	@echo "Checking Helm release status..."
	@echo ""
	@echo "📋 Helm Releases:"
	@helm list -A
	@echo ""
	@echo "📋 Pod Status in $(HELM_NAMESPACE_REAL):"
	@kubectl get pods -n $(HELM_NAMESPACE_REAL) -o wide 2>/dev/null || echo "Namespace $(HELM_NAMESPACE_REAL) not found"
	@echo ""
	@echo "📋 Service Status in $(HELM_NAMESPACE_REAL):"
	@kubectl get services -n $(HELM_NAMESPACE_REAL) 2>/dev/null || echo "Namespace $(HELM_NAMESPACE_REAL) not found"

# Helm quickstart (matches deploy/helm/quickstart.sh functionality)
helm-quickstart:
	@echo "Running Helm quickstart..."
	@chmod +x deploy/helm/quickstart.sh
	cd deploy/helm && ./quickstart.sh up -t $(TAG)
	@echo "Helm quickstart completed!"

helm-quickstart-down:
	@echo "Running Helm quickstart cleanup..."
	@chmod +x deploy/helm/quickstart.sh
	cd deploy/helm && ./quickstart.sh down
	@echo "Helm quickstart cleanup completed!"

helm-quickstart-status:
	@echo "Checking Helm quickstart status..."
	@chmod +x deploy/helm/quickstart.sh
	cd deploy/helm && ./quickstart.sh status

helm-quickstart-logs:
	@echo "Showing Helm quickstart logs..."
	@chmod +x deploy/helm/quickstart.sh
	cd deploy/helm && ./quickstart.sh logs

helm-port-forward:
	@echo "Starting port forwarding for Kubernetes services..."
	@echo "Killing existing port forwards..."
	@-pkill -f "kubectl port-forward" 2>/dev/null || true
	@sleep 2
	@echo "Starting API Gateway port forward (localhost:8081)..."
	@kubectl port-forward -n $(HELM_NAMESPACE_REAL) svc/api-gateway 8081:8081 >/dev/null 2>&1 &
	@echo "Starting Dashboard port forward (localhost:8080)..."
	@kubectl port-forward -n $(HELM_NAMESPACE_REAL) svc/dashboard 8080:80 >/dev/null 2>&1 &
	@sleep 3
	@echo "✅ Port forwarding started:"
	@echo "  🌐 Dashboard: http://localhost:8080"
	@echo "  🌐 API Gateway: http://localhost:8081"
	@echo "  🏥 API Health: http://localhost:8081/health"
	@echo ""
	@echo "⚠️  Port forwarding runs in background. Use 'pkill -f \"kubectl port-forward\"' to stop."

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
	@echo "Creating data directories..."
	@mkdir -p deploy/docker/data deploy/docker/mq-data deploy/docker/sample-data
	@chmod 755 deploy/docker/data deploy/docker/mq-data deploy/docker/sample-data
	@echo "Clearing data directories..."
	@rm -rf deploy/docker/data/* deploy/docker/mq-data/* 2>/dev/null || true
	@echo "Creating sample telemetry data..."
	@if [ ! -f "deploy/docker/sample-data/telemetry.csv" ]; then \
		echo "# DCGM format CSV with hostname field" > deploy/docker/sample-data/telemetry.csv; \
		echo "# Fields: timestamp,gpu_id,utilization,temperature,memory_used,power_draw,fan_speed,hostname" >> deploy/docker/sample-data/telemetry.csv; \
		echo "2024-01-01T12:00:00Z,GPU-f2b8d424-ed80-cddd-67d0-00bf52c03704,85.5,72.3,4096,250.5,2500,mtv5-dgx1-hgpu-031" >> deploy/docker/sample-data/telemetry.csv; \
		echo "2024-01-01T12:00:01Z,GPU-a1c4d567-12ab-3456-78ef-90123456789a,90.2,75.1,8192,275.2,2600,mtv5-dgx1-hgpu-022" >> deploy/docker/sample-data/telemetry.csv; \
		echo "2024-01-01T12:00:02Z,GPU-b2d5e678-23bc-4567-89fg-01234567890b,45.0,65.0,2048,180.1,2200,mtv5-dgx1-hgpu-010" >> deploy/docker/sample-data/telemetry.csv; \
		echo "2024-01-01T12:00:03Z,GPU-c3e6f789-34cd-5678-90gh-12345678901c,78.3,69.5,6144,225.8,2400,mtv5-dgx1-hgpu-031" >> deploy/docker/sample-data/telemetry.csv; \
		echo "2024-01-01T12:00:04Z,GPU-d4f7g890-45de-6789-01hi-23456789012d,92.1,77.8,7168,285.4,2700,mtv5-dgx1-hgpu-012" >> deploy/docker/sample-data/telemetry.csv; \
		chmod 644 deploy/docker/sample-data/telemetry.csv; \
		echo "Created sample telemetry data with DCGM format"; \
	fi
	cd deploy/docker && docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 15
	@echo "Checking service health..."
	@$(MAKE) docker-health-check
	@echo ""
	@echo "🎉 Services started successfully!"
	@echo ""
	@echo "Service endpoints:"
	@echo "  🌐 Dashboard:          http://localhost:5173"
	@echo "  🏥 MQ Health:         http://localhost:9090/health"
	@echo "  📊 MQ Stats:          http://localhost:9090/stats"
	@echo "  🏥 Collector Health:  http://localhost:8080/health"
	@echo "  🌐 API Gateway:       http://localhost:8081"
	@echo "  🏥 Gateway Health:    http://localhost:8081/health"
	@echo "  📚 API Documentation: http://localhost:8081/swagger/"
	@echo ""
	@echo "Management commands:"
	@echo "  📋 View logs:         make docker-logs"
	@echo "  🛑 Stop services:     make docker-down"
	@echo "  🔄 Restart services:  cd deploy/docker && docker-compose restart"

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

# Health check for Docker services
docker-health-check:
	@failed=0; \
	if curl -f http://localhost:9090/health >/dev/null 2>&1; then \
		echo "✅ MQ Service is healthy"; \
	else \
		echo "❌ MQ Service is not responding"; \
		failed=1; \
	fi; \
	if curl -f http://localhost:8080/health >/dev/null 2>&1; then \
		echo "✅ Telemetry Collector is healthy"; \
	else \
		echo "❌ Telemetry Collector is not responding"; \
		failed=1; \
	fi; \
	if curl -f http://localhost:8081/health >/dev/null 2>&1; then \
		echo "✅ API Gateway is healthy"; \
	else \
		echo "❌ API Gateway is not responding"; \
		failed=1; \
	fi; \
	if curl -f http://localhost:5173/ >/dev/null 2>&1; then \
		echo "✅ Dashboard is healthy"; \
	else \
		echo "❌ Dashboard is not responding"; \
		failed=1; \
	fi; \
	if [ $$failed -eq 0 ]; then \
		echo "✅ All services are healthy!"; \
	else \
		echo "⚠️  Some services are not healthy. Check logs with: make docker-logs"; \
	fi

# Docker setup using script (matches deploy/docker/setup.sh)
docker-setup:
	@echo "Running Docker setup script..."
	@chmod +x deploy/docker/setup.sh
	cd deploy/docker && ./setup.sh
	@echo "Docker setup completed using script!"

docker-setup-build:
	@echo "Running Docker setup with build..."
	@chmod +x deploy/docker/setup.sh
	cd deploy/docker && ./setup.sh -b -t $(TAG)
	@echo "Docker setup with build completed!"

docker-setup-down:
	@echo "Running Docker teardown..."
	@chmod +x deploy/docker/setup.sh
	cd deploy/docker && ./setup.sh -d
	@echo "Docker teardown completed!"

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
	docker push $(REGISTRY)/dashboard:$(TAG)
	@echo "Docker images pushed successfully!"

# Build and push using script (matches deploy/docker/build-and-push.sh)
docker-build-and-push:
	@echo "Running build-and-push script..."
	@chmod +x deploy/docker/build-and-push.sh
	cd deploy/docker && ./build-and-push.sh -t $(TAG)
	@echo "Build and push completed using script!"

# Integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v ./cmd/api-gateway/... -tags=integration
	@echo "Integration tests completed!"

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "🏗️  Build Targets:"
	@echo "  build             		- Build all components (collector, streamer, api-gateway, mq-service, dashboard)"
	@echo "  build-for-system-tests - Build components for system tests (collector, streamer, api-gateway, mq-service)"
	@echo "  build-dashboard   		- Build React dashboard only"
	@echo "  deps              		- Install Go and Node.js dependencies"
	@echo ""
	@echo "🧪 Test Targets:"
	@echo "  test              		- Run unit tests with coverage"
	@echo "  test-integration  		- Run integration tests"
	@echo "  system-tests      		- Run comprehensive system tests"
	@echo "  system-tests-quick 	- Run quick system tests"
	@echo "  system-tests-performance - Run performance tests"
	@echo "  coverage          - Generate test coverage report"
	@echo ""
	@echo "🐳 Docker Targets:"
	@echo "  docker-build      - Build Docker images for all services (includes dashboard)"
	@echo "  docker-build-and-push - Use build-and-push.sh script"
	@echo "  docker-push       - Push Docker images to registry"
	@echo "  docker-up         - Start all services with Docker Compose (comprehensive setup)"
	@echo "  docker-down       - Stop Docker Compose services"
	@echo "  docker-logs       - Show Docker Compose logs"
	@echo "  docker-status     - Check Docker Compose status"
	@echo "  docker-health-check - Check health of all Docker services"
	@echo "  docker-setup      - Use setup.sh script (existing images)"
	@echo "  docker-setup-build - Use setup.sh script with build"
	@echo "  docker-setup-down - Use setup.sh script to teardown"
	@echo ""
	@echo "☸️  Kubernetes/Helm Targets:"
	@echo "  helm-install      - Deploy to Kubernetes using individual charts"
	@echo "  helm-uninstall    - Remove all Helm releases"
	@echo "  helm-status       - Show Helm and pod status"
	@echo "  helm-quickstart   - Use quickstart.sh script (full Kind setup)"
	@echo "  helm-quickstart-down - Use quickstart.sh script to cleanup"
	@echo "  helm-quickstart-status - Check quickstart deployment status"
	@echo "  helm-quickstart-logs - Show quickstart deployment logs"
	@echo "  helm-port-forward - Start port forwarding for services"
	@echo ""
	@echo "🏃 Local Development:"
	@echo "  run-collector     - Run collector locally"
	@echo "  run-streamer      - Run streamer locally"
	@echo "  run-api           - Run API gateway locally"
	@echo "  run-mq            - Run MQ service locally"
	@echo "  sample-data       - Create sample test data"
	@echo ""
	@echo "🧹 Cleanup:"
	@echo "  clean             - Clean build artifacts"
	@echo "  clean-docker      - Clean Docker images"
	@echo ""
	@echo "🗂️  Registry Management:"
	@echo "  registry-start    - Start local Docker registry"
	@echo "  registry-stop     - Stop local Docker registry" 
	@echo "  registry-status   - Check registry status"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  docs              - Generate API documentation"
	@echo "  openapi-gen       - Generate OpenAPI specs"
	@echo ""
	@echo "🚀 Workflows:"
	@echo "  all               - Full build pipeline (clean + deps + build + test + coverage)"
	@echo "  dev               - Development workflow (build + test)"
	@echo "  ci                - CI pipeline (clean + deps + build + test + coverage)"
	@echo "  deploy            - Complete deployment (build + test + docker + helm)"
	@echo "  lint              - Run code linter"
	@echo "  help              - Show this help message"

.PHONY: build build-for-system-tests build-dashboard test test-integration coverage lint clean clean-docker docker-build docker-build-and-push docker-push docker-up docker-down docker-logs docker-status docker-health-check docker-setup docker-setup-build docker-setup-down docs deps deploy helm-install helm-uninstall helm-status helm-quickstart helm-quickstart-down helm-quickstart-status helm-quickstart-logs helm-port-forward registry-start registry-stop registry-status sample-data help run-collector run-streamer run-api run-mq system-tests system-tests-quick system-tests-performance all dev ci openapi-gen