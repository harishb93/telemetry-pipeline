# Telemetry Pipeline Makefile

.PHONY: help build test clean openapi-gen run-collector run-streamer run-api lint deps

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build all binaries"
	@echo "  test          - Run all tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  openapi-gen   - Generate OpenAPI specification"
	@echo "  run-collector - Run telemetry collector"
	@echo "  run-streamer  - Run telemetry streamer"
	@echo "  run-api       - Run API gateway"
	@echo "  lint          - Run linter"
	@echo "  deps          - Install dependencies"

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
	@echo "Running tests..."
	go test ./... -v

test-coverage:
	@echo "Running tests with coverage..."
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

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

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf api/docs api/swagger.json api/swagger.yaml

# Docker targets (preserve existing)
docker-build:
	docker build -t telemetry-streamer ./docker/telemetry-streamer
	docker build -t telemetry-collector ./docker/telemetry-collector
	docker build -t api-gateway ./docker/api-gateway

# Helm target (preserve existing)
helm-install:
	@echo "Helm install commands go here"

# Demo and integration testing
demo: build
	@echo "Running integration demo..."
	./demo.sh

# All-in-one targets
all: clean deps build test openapi-gen

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