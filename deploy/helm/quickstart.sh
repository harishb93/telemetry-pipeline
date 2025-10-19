#!/bin/bash
set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
REGISTRY_NAME="kind-registry"
REGISTRY_PORT="5000"
REGISTRY_URL="localhost:${REGISTRY_PORT}"
CLUSTER_NAME="kind"
NAMESPACE="gpu-telemetry"
IMAGE_TAG="${IMAGE_TAG:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
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

log_header() {
    echo -e "\n${CYAN}${BOLD}=== $1 ===${NC}\n"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS] [COMMAND]"
    echo ""
    echo "GPU Telemetry Pipeline Quick Start Script"
    echo ""
    echo "COMMANDS:"
    echo "  up          Create cluster, build images, and deploy pipeline (default)"
    echo "  down        Destroy cluster and cleanup"
    echo "  status      Show status of cluster and deployments"
    echo "  logs        Show logs from all components"
    echo "  port-forward Start port forwarding for dashboard and API gateway"
    echo ""
    echo "OPTIONS:"
    echo "  -t, --tag TAG        Image tag (default: latest)"
    echo "  -c, --cluster NAME   Cluster name (default: kind)"
    echo "  -n, --namespace NS   Kubernetes namespace (default: gpu-telemetry)"
    echo "  --skip-build         Skip building images"
    echo "  --skip-push          Skip pushing images to registry"
    echo "  --skip-cluster       Skip cluster creation (use existing)"
    echo "  --debug              Enable debug output"
    echo "  -h, --help           Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  IMAGE_TAG            Image tag to use (overridden by -t/--tag)"
    echo "  SKIP_BUILD           Skip image building if set to 'true'"
    echo "  SKIP_PUSH            Skip image pushing if set to 'true'"
    echo "  SKIP_CLUSTER         Skip cluster creation if set to 'true'"
    echo ""
    echo "Examples:"
    echo "  $0                   # Full setup with defaults"
    echo "  $0 up -t v1.0.0      # Setup with specific image tag"
    echo "  $0 --skip-build      # Skip building, use existing images"
    echo "  $0 --skip-push       # Build but don't push images"
    echo "  $0 down              # Cleanup everything"
    echo "  $0 status            # Show current status"
}

# Function to check prerequisites
check_prerequisites() {
    log_header "Checking Prerequisites"
    
    local missing_deps=()
    
    # Check for required tools
    for tool in docker kind kubectl helm curl; do
        if ! command -v $tool >/dev/null 2>&1; then
            missing_deps+=($tool)
        else
            log_success "✓ $tool found"
        fi
    done
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_info "Please install the missing tools and try again"
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running or not accessible"
        exit 1
    fi
    
    log_success "All prerequisites satisfied"
}

# Function to check if registry is running
is_registry_running() {
    docker ps --filter "name=${REGISTRY_NAME}" --filter "status=running" --format "{{.Names}}" | grep -q "^${REGISTRY_NAME}$"
}

# Function to start local registry
start_registry() {
    log_header "Setting Up Local Registry"
    
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
            -p "127.0.0.1:${REGISTRY_PORT}:5000" \
            --network bridge \
            registry:2
    fi
    
    # Wait for registry to be ready
    log_info "Waiting for registry to be ready..."
    for i in {1..30}; do
        if curl -f http://${REGISTRY_URL}/v2/ >/dev/null 2>&1; then
            log_success "Registry is ready at http://${REGISTRY_URL}"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "Registry failed to start after 30 seconds"
            exit 1
        fi
        sleep 1
    done
}

# Function to create Kind cluster
create_cluster() {
    log_header "Creating Kind Cluster"
    
    # Check if cluster already exists
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_info "Cluster '${CLUSTER_NAME}' already exists"
        return 0
    fi
    
    log_info "Creating Kind cluster '${CLUSTER_NAME}' with registry support..."
    
    # Create cluster with registry config
    cat <<EOF | kind create cluster --name="${CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry]
    config_path = "/etc/containerd/certs.d"
EOF
    
    # Configure registry for all nodes
    log_info "Configuring registry access for cluster nodes..."
    REGISTRY_DIR="/etc/containerd/certs.d/localhost:${REGISTRY_PORT}"
    for node in $(kind get nodes --name="${CLUSTER_NAME}"); do
        docker exec "${node}" mkdir -p "${REGISTRY_DIR}"
        cat <<EOF | docker exec -i "${node}" cp /dev/stdin "${REGISTRY_DIR}/hosts.toml"
[host."http://${REGISTRY_NAME}:5000"]
EOF
    done
    
    # Connect registry to cluster network
    log_info "Connecting registry to cluster network..."
    if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REGISTRY_NAME}")" = 'null' ]; then
        docker network connect "kind" "${REGISTRY_NAME}"
    fi
    
    # Document the local registry
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REGISTRY_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
    
    log_success "Kind cluster '${CLUSTER_NAME}' created and configured"
}

# Function to make entrypoint scripts executable
make_scripts_executable() {
    log_info "Making entrypoint scripts executable..."
    if [ -d "${PROJECT_ROOT}/deploy/docker" ]; then
        chmod +x "${PROJECT_ROOT}/deploy/docker/entrypoint-"*.sh 2>/dev/null || true
    fi
}

# Function to build and push a component
build_and_push_component() {
    local component=$1
    local dockerfile=$2
    
    # Build phase
    if [ "${SKIP_BUILD}" != "true" ]; then
        log_info "Building ${component}..."
        
        # Build the image
        docker build \
            -f "${PROJECT_ROOT}/deploy/docker/${dockerfile}" \
            -t "${component}:${IMAGE_TAG}" \
            "${PROJECT_ROOT}" || {
            log_error "Failed to build ${component}"
            return 1
        }
        
        # Tag for local registry
        docker tag "${component}:${IMAGE_TAG}" "${REGISTRY_URL}/${component}:${IMAGE_TAG}"
        
        log_success "Successfully built ${component}"
    else
        log_info "Skipping build for ${component}"
    fi
    
    # Push phase
    if [ "${SKIP_PUSH}" != "true" ]; then
        log_info "Pushing ${component} to ${REGISTRY_URL}..."
        docker push "${REGISTRY_URL}/${component}:${IMAGE_TAG}" || {
            log_error "Failed to push ${component}"
            return 1
        }
        
        log_success "Successfully pushed ${component}"
    else
        log_info "Skipping push for ${component}"
    fi
}

# Function to build and push all images
build_and_push_images() {
    log_header "Building and Pushing Docker Images"
    
    # Make entrypoint scripts executable
    make_scripts_executable
    
    # Build and push each component
    local components=(
        "mq-service:mq-service.Dockerfile"
        "telemetry-streamer:telemetry-streamer.Dockerfile"
        "telemetry-collector:telemetry-collector.Dockerfile"
        "api-gateway:api-gateway.Dockerfile"
        "dashboard:dashboard.Dockerfile"
    )
    
    for component_info in "${components[@]}"; do
        IFS=':' read -r component dockerfile <<< "$component_info"
        build_and_push_component "$component" "$dockerfile"
    done
    
    # Verify all images are in registry (only if not skipping push)
    if [ "${SKIP_PUSH}" != "true" ]; then
        log_info "Verifying images in registry..."
        for component_info in "${components[@]}"; do
            IFS=':' read -r component dockerfile <<< "$component_info"
            if curl -f http://${REGISTRY_URL}/v2/${component}/tags/list >/dev/null 2>&1; then
                log_success "✓ ${component} found in registry"
            else
                log_error "✗ ${component} not found in registry"
                return 1
            fi
        done
    fi
    
    log_success "All images processed successfully!"
}

# Function to wait for deployment to be ready
wait_for_deployment() {
    local deployment=$1
    local namespace=$2
    local timeout=${3:-300}
    
    log_info "Waiting for deployment '${deployment}' to be ready..."
    
    if kubectl wait --for=condition=available --timeout=${timeout}s deployment/${deployment} -n ${namespace} >/dev/null 2>&1; then
        log_success "✓ Deployment '${deployment}' is ready"
        return 0
    else
        log_error "✗ Deployment '${deployment}' failed to become ready within ${timeout}s"
        return 1
    fi
}

# Function to wait for statefulset to be ready
wait_for_statefulset() {
    local statefulset=$1
    local namespace=$2
    local timeout=${3:-300}
    
    log_info "Waiting for statefulset '${statefulset}' to be ready..."
    
    if kubectl wait --for=condition=ready --timeout=${timeout}s pod -l app.kubernetes.io/name=${statefulset} -n ${namespace} >/dev/null 2>&1; then
        log_success "✓ StatefulSet '${statefulset}' is ready"
        return 0
    else
        log_error "✗ StatefulSet '${statefulset}' failed to become ready within ${timeout}s"
        return 1
    fi
}

# Function to wait for daemonset to be ready
wait_for_daemonset() {
    local daemonset=$1
    local namespace=$2
    local timeout=${3:-300}
    
    log_info "Waiting for daemonset '${daemonset}' to be ready..."
    
    # Wait for DaemonSet to have desired number of pods ready
    local end_time=$((SECONDS + timeout))
    while [ $SECONDS -lt $end_time ]; do
        local desired=$(kubectl get daemonset ${daemonset} -n ${namespace} -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
        local ready=$(kubectl get daemonset ${daemonset} -n ${namespace} -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
        
        if [ "${desired}" -gt 0 ] && [ "${ready}" -eq "${desired}" ]; then
            log_success "✓ DaemonSet '${daemonset}' is ready (${ready}/${desired} pods)"
            return 0
        fi
        
        log_info "DaemonSet '${daemonset}' not ready yet (${ready}/${desired} pods)..."
        sleep 5
    done
    
    log_error "✗ DaemonSet '${daemonset}' failed to become ready within ${timeout}s"
    return 1
}

# Function to deploy with Helm
deploy_pipeline() {
    log_header "Deploying Telemetry Pipeline with Helm"
    
    cd "${SCRIPT_DIR}"
    
    # Install in the correct order with dependency checks
    log_info "Installing shared-resources..."
    helm install shared-resources charts/shared-resources/ || {
        log_error "Failed to install shared-resources"
        return 1
    }
    
    # Wait a moment for namespace creation
    sleep 2
    
    log_info "Installing mq-service..."
    helm install mq-service charts/mq-service/ --namespace ${NAMESPACE} || {
        log_error "Failed to install mq-service"
        return 1
    }
    wait_for_statefulset "mq-service" "${NAMESPACE}"
    
    log_info "Installing telemetry-collector..."
    helm install telemetry-collector charts/telemetry-collector/ --namespace ${NAMESPACE} || {
        log_error "Failed to install telemetry-collector"
        return 1
    }
    wait_for_statefulset "telemetry-collector" "${NAMESPACE}"
    
    log_info "Installing api-gateway..."
    helm install api-gateway charts/api-gateway/ --namespace ${NAMESPACE} || {
        log_error "Failed to install api-gateway"
        return 1
    }
    wait_for_deployment "api-gateway" "${NAMESPACE}"
    
    log_info "Installing dashboard..."
    helm install dashboard charts/dashboard/ --namespace ${NAMESPACE} || {
        log_error "Failed to install dashboard"
        return 1
    }
    wait_for_deployment "dashboard" "${NAMESPACE}"
    
    log_info "Installing telemetry-streamer..."
    helm install telemetry-streamer charts/telemetry-streamer/ --namespace ${NAMESPACE} || {
        log_error "Failed to install telemetry-streamer"
        return 1
    }
    wait_for_daemonset "telemetry-streamer" "${NAMESPACE}"
    
    log_success "All components deployed successfully!"
}

# Function to show deployment status
show_status() {
    log_header "Deployment Status"
    
    # Check cluster
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_success "✓ Kind cluster '${CLUSTER_NAME}' is running"
    else
        log_error "✗ Kind cluster '${CLUSTER_NAME}' not found"
        return 1
    fi
    
    # Check registry
    if is_registry_running; then
        log_success "✓ Registry is running at http://${REGISTRY_URL}"
    else
        log_error "✗ Registry is not running"
    fi
    
    # Check namespace
    if kubectl get namespace ${NAMESPACE} >/dev/null 2>&1; then
        log_success "✓ Namespace '${NAMESPACE}' exists"
    else
        log_error "✗ Namespace '${NAMESPACE}' not found"
        return 1
    fi
    
    # Show pod status
    echo -e "\n${BOLD}Pod Status:${NC}"
    kubectl get pods -n ${NAMESPACE} -o wide
    
    # Show service status
    echo -e "\n${BOLD}Service Status:${NC}"
    kubectl get services -n ${NAMESPACE}
    
    # Show Helm releases
    echo -e "\n${BOLD}Helm Releases:${NC}"
    helm list -A
}

# Function to start port forwarding
start_port_forwarding() {
    log_header "Starting Port Forwarding"
    
    # Kill existing port forwards
    pkill -f "kubectl port-forward" 2>/dev/null || true
    sleep 2
    
    # Start API Gateway port forward
    log_info "Starting port forward for API Gateway (localhost:8081)..."
    kubectl port-forward -n ${NAMESPACE} svc/api-gateway 8081:8081 >/dev/null 2>&1 &
    
    # Start Dashboard port forward
    log_info "Starting port forward for Dashboard (localhost:8080)..."
    kubectl port-forward service/dashboard 8080:80 -n ${NAMESPACE} >/dev/null 2>&1 &
    
    # Wait a moment for port forwards to establish
    sleep 3
    
    log_success "Port forwarding started:"
    log_info "  Dashboard: http://localhost:8080"
    log_info "  API Gateway: http://localhost:8081"
    log_info "  API Health: http://localhost:8081/health"
    
    log_warning "Port forwarding runs in background. Use 'pkill -f \"kubectl port-forward\"' to stop."
}

# Function to show logs
show_logs() {
    log_header "Component Logs"
    
    local components=("mq-service" "telemetry-collector" "api-gateway" "dashboard" "telemetry-streamer")
    
    for component in "${components[@]}"; do
        echo -e "\n${BOLD}=== ${component} logs ===${NC}"
        kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/name=${component} --tail=20 --prefix=true
    done
}

# Function to cleanup everything
cleanup() {
    log_header "Cleaning Up"
    
    # Kill port forwards
    pkill -f "kubectl port-forward" 2>/dev/null || true
    
    # Delete Helm releases
    log_info "Uninstalling Helm releases..."
    helm uninstall telemetry-streamer -n ${NAMESPACE} 2>/dev/null || true
    helm uninstall dashboard -n ${NAMESPACE} 2>/dev/null || true
    helm uninstall api-gateway -n ${NAMESPACE} 2>/dev/null || true
    helm uninstall telemetry-collector -n ${NAMESPACE} 2>/dev/null || true
    helm uninstall mq-service -n ${NAMESPACE} 2>/dev/null || true
    helm uninstall shared-resources 2>/dev/null || true
    
    # Delete Kind cluster
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_info "Deleting Kind cluster '${CLUSTER_NAME}'..."
        kind delete cluster --name="${CLUSTER_NAME}"
    fi
    
    # Stop and remove registry
    if docker ps -a --filter "name=${REGISTRY_NAME}" --format "{{.Names}}" | grep -q "^${REGISTRY_NAME}$"; then
        log_info "Stopping and removing registry..."
        docker stop ${REGISTRY_NAME} >/dev/null 2>&1 || true
        docker rm ${REGISTRY_NAME} >/dev/null 2>&1 || true
    fi
    
    log_success "Cleanup completed"
}

# Main function to set up everything
setup_pipeline() {
    log_header "GPU Telemetry Pipeline Quick Start"
    log_info "Project: ${PROJECT_ROOT}"
    log_info "Registry: ${REGISTRY_URL}"
    log_info "Cluster: ${CLUSTER_NAME}"
    log_info "Namespace: ${NAMESPACE}"
    log_info "Image Tag: ${IMAGE_TAG}"
    log_info "Skip Build: ${SKIP_BUILD}"
    log_info "Skip Push: ${SKIP_PUSH}"
    log_info "Skip Cluster: ${SKIP_CLUSTER}"
    
    # Check prerequisites
    check_prerequisites
    
    # Start registry
    start_registry
    
    # Create cluster (if not skipped)
    if [ "${SKIP_CLUSTER}" != "true" ]; then
        create_cluster
    else
        log_warning "Skipping cluster creation"
    fi
    
    # Build and push images (if not skipped)
    if [ "${SKIP_BUILD}" != "true" ] || [ "${SKIP_PUSH}" != "true" ]; then
        build_and_push_images
    else
        log_warning "Skipping both image building and pushing"
    fi
    
    # Deploy pipeline
    deploy_pipeline
    
    # Show status
    show_status
    
    # Start port forwarding
    start_port_forwarding
    
    log_header "Setup Complete!"
    log_success "GPU Telemetry Pipeline is now running!"
    log_info ""
    log_info "Access URLs:"
    log_info "  Dashboard: http://localhost:8080"
    log_info "  API Gateway: http://localhost:8081"
    log_info "  API Health Check: http://localhost:8081/health"
    log_info ""
    log_info "Useful commands:"
    log_info "  $0 status          # Show current status"
    log_info "  $0 logs            # Show component logs"  
    log_info "  $0 port-forward    # Restart port forwarding"
    log_info "  $0 down            # Cleanup everything"
    log_info ""
    log_info "Kubernetes commands:"
    log_info "  kubectl get pods -n ${NAMESPACE}     # Show pods"
    log_info "  kubectl logs -f -n ${NAMESPACE} <pod-name>  # Follow logs"
}

# Parse command line arguments
COMMAND="up"
SKIP_BUILD="${SKIP_BUILD:-false}"
SKIP_PUSH="${SKIP_PUSH:-false}"
SKIP_CLUSTER="${SKIP_CLUSTER:-false}"
DEBUG="${DEBUG:-false}"

while [[ $# -gt 0 ]]; do
    case $1 in
        up|down|status|logs|port-forward)
            COMMAND="$1"
            shift
            ;;
        -t|--tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        -c|--cluster)
            CLUSTER_NAME="$2"
            shift 2
            ;;
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD="true"
            shift
            ;;
        --skip-push)
            SKIP_PUSH="true"
            shift
            ;;
        --skip-cluster)
            SKIP_CLUSTER="true"
            shift
            ;;
        --debug)
            DEBUG="true"
            set -x
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

# Execute command
case $COMMAND in
    up)
        setup_pipeline
        ;;
    down)
        cleanup
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    port-forward)
        start_port_forwarding
        ;;
    *)
        log_error "Unknown command: $COMMAND"
        show_usage
        exit 1
        ;;
esac