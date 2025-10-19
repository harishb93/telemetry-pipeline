#!/bin/bash
set -e

# Configuration
REGISTRY_NAME="kind-registry"
REGISTRY_PORT="5000"
REGISTRY_URL="localhost:${REGISTRY_PORT}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if registry is running
is_registry_running() {
    docker ps --filter "name=${REGISTRY_NAME}" --filter "status=running" --format "{{.Names}}" | grep -q "^${REGISTRY_NAME}$"
}

# Function to start local registry
start_registry() {
    if is_registry_running; then
        log_info "Registry ${REGISTRY_NAME} is already running"
        return 0
    fi
    
    log_info "Starting local Docker registry ${REGISTRY_NAME} on port ${REGISTRY_PORT}..."
    
    # Check if registry container exists but is stopped
    if docker ps -a --filter "name=${REGISTRY_NAME}" --format "{{.Names}}" | grep -q "^${REGISTRY_NAME}$"; then
        log_info "Registry container exists but is stopped. Starting it..."
        docker start ${REGISTRY_NAME}
    else
        # Create new registry container
        docker run -d \
            --restart=always \
            --name ${REGISTRY_NAME} \
            -p ${REGISTRY_PORT}:5000 \
            registry:2
    fi
    
    # Wait for registry to be ready
    log_info "Waiting for registry to be ready..."
    for i in {1..30}; do
        if curl -f http://${REGISTRY_URL}/v2/ >/dev/null 2>&1; then
            log_success "Registry is ready!"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "Registry failed to start after 30 seconds"
            exit 1
        fi
        sleep 1
    done
}

# Function to build and push a component
build_and_push_component() {
    local component=$1
    local dockerfile=$2
    
    log_info "Building ${component}..."
    
    # Build the image
    docker build \
        -f "${PROJECT_ROOT}/deploy/docker/${dockerfile}" \
        -t "${component}:${IMAGE_TAG}" \
        "${PROJECT_ROOT}"
    
    # Tag for local registry
    docker tag "${component}:${IMAGE_TAG}" "${REGISTRY_URL}/${component}:${IMAGE_TAG}"
    
    # Push to local registry
    log_info "Pushing ${component} to ${REGISTRY_URL}..."
    docker push "${REGISTRY_URL}/${component}:${IMAGE_TAG}"
    
    log_success "Successfully built and pushed ${component}"
}

# Function to verify images in registry
verify_images() {
    log_info "Verifying images in registry..."
    
    local components=("mq-service" "telemetry-streamer" "telemetry-collector" "api-gateway" "dashboard")
    
    for component in "${components[@]}"; do
        if curl -f http://${REGISTRY_URL}/v2/${component}/tags/list >/dev/null 2>&1; then
            log_success "✓ ${component} found in registry"
        else
            log_error "✗ ${component} not found in registry"
            return 1
        fi
    done
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Build and push telemetry pipeline Docker images to local registry"
    echo ""
    echo "OPTIONS:"
    echo "  -t, --tag TAG        Image tag (default: latest)"
    echo "  -p, --port PORT      Registry port (default: 5000)"
    echo "  -n, --name NAME      Registry container name (default: kind-registry)"
    echo "  -h, --help           Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  IMAGE_TAG            Image tag to use (overridden by -t/--tag)"
    echo ""
    echo "Examples:"
    echo "  $0                   # Build with default settings"
    echo "  $0 -t v1.0.0         # Build with specific tag"
    echo "  $0 -p 5001           # Use different registry port"
}

# Function to make entrypoint scripts executable
make_scripts_executable() {
    log_info "Making entrypoint scripts executable..."
    chmod +x "${PROJECT_ROOT}/deploy/docker/entrypoint-"*.sh
}

# Main function
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--tag)
                IMAGE_TAG="$2"
                shift 2
                ;;
            -p|--port)
                REGISTRY_PORT="$2"
                REGISTRY_URL="localhost:${REGISTRY_PORT}"
                shift 2
                ;;
            -n|--name)
                REGISTRY_NAME="$2"
                shift 2
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    log_info "Starting build and push process..."
    log_info "Registry: ${REGISTRY_URL}"
    log_info "Image tag: ${IMAGE_TAG}"
    log_info "Project root: ${PROJECT_ROOT}"
    
    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running or not accessible"
        exit 1
    fi
    
    # Make entrypoint scripts executable
    make_scripts_executable
    
    # Start registry if not running
    start_registry
    
    # Build and push each component
    build_and_push_component "mq-service" "mq-service.Dockerfile"
    build_and_push_component "telemetry-streamer" "telemetry-streamer.Dockerfile"
    build_and_push_component "telemetry-collector" "telemetry-collector.Dockerfile"
    build_and_push_component "api-gateway" "api-gateway.Dockerfile"
    build_and_push_component "dashboard" "dashboard.Dockerfile"
    
    # Verify all images are in registry
    verify_images
    
    log_success "All images built and pushed successfully!"
    log_info "Registry URL: http://${REGISTRY_URL}"
    log_info "To list all images: curl http://${REGISTRY_URL}/v2/_catalog"
    
    # Show image information
    echo ""
    log_info "Built images:"
    echo "  - ${REGISTRY_URL}/mq-service:${IMAGE_TAG}"
    echo "  - ${REGISTRY_URL}/telemetry-streamer:${IMAGE_TAG}"
    echo "  - ${REGISTRY_URL}/telemetry-collector:${IMAGE_TAG}"
    echo "  - ${REGISTRY_URL}/api-gateway:${IMAGE_TAG}"
    echo "  - ${REGISTRY_URL}/dashboard:${IMAGE_TAG}"
    
    # Show example usage
    echo ""
    log_info "Example usage in Kubernetes:"
    echo "  kubectl run mq-service --image=${REGISTRY_URL}/mq-service:${IMAGE_TAG}"
    echo "  kubectl run telemetry-streamer --image=${REGISTRY_URL}/telemetry-streamer:${IMAGE_TAG}"
    echo "  kubectl run telemetry-collector --image=${REGISTRY_URL}/telemetry-collector:${IMAGE_TAG}"
    echo "  kubectl run api-gateway --image=${REGISTRY_URL}/api-gateway:${IMAGE_TAG}"
    echo "  kubectl run dashboard --image=${REGISTRY_URL}/dashboard:${IMAGE_TAG}"
    
    echo ""
    log_info "For Kind cluster integration:"
    echo "  kind create cluster --config=kind-config.yaml"
    echo "  kubectl apply -f deploy/k8s/"
}

# Run main function
main "$@"