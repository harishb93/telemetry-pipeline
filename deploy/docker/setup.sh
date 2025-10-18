#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

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

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Setup and run telemetry pipeline with Docker"
    echo ""
    echo "OPTIONS:"
    echo "  -b, --build          Build images before starting"
    echo "  -t, --tag TAG        Image tag to use (default: latest)"
    echo "  -d, --down           Stop and remove containers"
    echo "  -l, --logs           Show logs after starting"
    echo "  -h, --help           Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                   # Start with existing images"
    echo "  $0 -b                # Build images and start"
    echo "  $0 -d                # Stop all containers"
    echo "  $0 -b -t v1.0.0      # Build with specific tag and start"
}

# Check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running or not accessible"
        exit 1
    fi
}

# Build images
build_images() {
    local tag=$1
    log_info "Building images with tag: $tag"
    
    if [ -f "./build-and-push.sh" ]; then
        ./build-and-push.sh -t "$tag"
    else
        log_error "build-and-push.sh not found in current directory"
        exit 1
    fi
}

# Start services
start_services() {
    log_info "Starting telemetry pipeline services..."
    
    # Create necessary directories with proper permissions
    log_info "Creating data directories..."
    mkdir -p data mq-data sample-data
    
    # Ensure proper ownership for Docker volumes (UID:GID 1000:1000)
    if [ "$(id -u)" = "0" ]; then
        # Running as root, set ownership
        chown -R 1000:1000 data mq-data sample-data
    elif [ "$(id -u)" != "1000" ]; then
        # Not running as the expected user, warn but continue
        log_warning "Not running as UID 1000 - you may encounter permission issues"
        log_warning "Consider running: sudo chown -R 1000:1000 data mq-data sample-data"
    fi
    
    # Check if sample data exists
    if [ ! -f "./sample-data/telemetry.csv" ]; then
        log_warning "Sample data not found, creating default telemetry.csv"
        cat > sample-data/telemetry.csv << 'EOF'
# DCGM format CSV with hostname field
# Fields: timestamp,gpu_id,utilization,temperature,memory_used,power_draw,fan_speed,hostname
2024-01-01T12:00:00Z,GPU-f2b8d424-ed80-cddd-67d0-00bf52c03704,85.5,72.3,4096,250.5,2500,mtv5-dgx1-hgpu-031
2024-01-01T12:00:01Z,GPU-a1c4d567-12ab-3456-78ef-90123456789a,90.2,75.1,8192,275.2,2600,mtv5-dgx1-hgpu-022
2024-01-01T12:00:02Z,GPU-b2d5e678-23bc-4567-89fg-01234567890b,45.0,65.0,2048,180.1,2200,mtv5-dgx1-hgpu-010
2024-01-01T12:00:03Z,GPU-c3e6f789-34cd-5678-90gh-12345678901c,78.3,69.5,6144,225.8,2400,mtv5-dgx1-hgpu-031
2024-01-01T12:00:04Z,GPU-d4f7g890-45de-6789-01hi-23456789012d,92.1,77.8,7168,285.4,2700,mtv5-dgx1-hgpu-012
EOF
        # Set proper permissions on the created file
        chmod 644 sample-data/telemetry.csv
        if [ "$(id -u)" = "0" ]; then
            chown 1000:1000 sample-data/telemetry.csv
        fi
        log_success "Created sample telemetry data with DCGM format"
    fi
    
    # Start services
    log_info "Starting containers..."
    docker compose up -d
    
    log_success "Services started successfully!"
    
    # Wait for services to be ready
    log_info "Waiting for services to be ready..."
    sleep 15
    
    # Check service health
    check_services
}

# Stop services
stop_services() {
    log_info "Stopping telemetry pipeline services..."
    docker compose down -v
    log_success "Services stopped successfully!"
}

# Check service health
check_services() {
    local failed=0
    
    log_info "Checking service health..."
    
    # Check collector health
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        log_success "✓ Telemetry Collector is healthy"
    else
        log_error "✗ Telemetry Collector is not responding"
        failed=1
    fi
    
    # Check API gateway health
    if curl -f http://localhost:8081/health >/dev/null 2>&1; then
        log_success "✓ API Gateway is healthy"
    else
        log_error "✗ API Gateway is not responding"
        failed=1
    fi
    
    if [ $failed -eq 0 ]; then
        log_success "All services are healthy!"
        show_endpoints
    else
        log_warning "Some services are not healthy. Check logs with: docker compose logs"
    fi
}

# Show service endpoints
show_endpoints() {
    echo ""
    log_info "Service endpoints:"
    echo "  🏥 Collector Health:  http://localhost:8080/health"
    echo "  📊 Collector Metrics: http://localhost:9091/metrics"
    echo "  🌐 API Gateway:       http://localhost:8081"
    echo "  🏥 Gateway Health:    http://localhost:8081/health"
    echo "  📚 API Documentation: http://localhost:8081/swagger/"
    echo "  📈 Gateway Metrics:   http://localhost:9092/metrics"
    echo ""
    log_info "API endpoints:"
    echo "  📝 List GPUs:         curl http://localhost:8081/api/v1/gpus"
    echo "  📊 GPU Telemetry:     curl http://localhost:8081/api/v1/gpus/gpu-001/telemetry"
    echo "  📝 List Hosts:        curl http://localhost:8081/api/v1/hosts"
    echo "  📊 GPU On Hosts:      curl http://localhost:8081/api/v1/hosts/host-A/gpus"
    echo ""
    log_info "Management commands:"
    echo "  📋 View logs:         docker compose logs -f"
    echo "  🛑 Stop services:     $0 -d"
    echo "  🔄 Restart:           docker compose restart"
}

# Show logs
show_logs() {
    log_info "Showing service logs (Press Ctrl+C to exit)..."
    docker compose logs -f
}

# Main function
main() {
    local build_images_flag=false
    local show_logs_flag=false
    local stop_flag=false
    local image_tag="latest"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--build)
                build_images_flag=true
                shift
                ;;
            -t|--tag)
                image_tag="$2"
                shift 2
                ;;
            -d|--down)
                stop_flag=true
                shift
                ;;
            -l|--logs)
                show_logs_flag=true
                shift
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
    
    # Check Docker
    check_docker
    
    # Stop services if requested
    if [ "$stop_flag" = "true" ]; then
        stop_services
        exit 0
    fi
    
    # Build images if requested
    if [ "$build_images_flag" = "true" ]; then
        build_images "$image_tag"
    fi
    
    # Start services
    start_services
    
    # Show logs if requested
    if [ "$show_logs_flag" = "true" ]; then
        show_logs
    fi
}

# Change to script directory
cd "$(dirname "${BASH_SOURCE[0]}")"

# Run main function
main "$@"