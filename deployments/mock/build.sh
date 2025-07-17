#!/bin/bash
set -euo pipefail

# MockLogGenerator Container Build Script
# Builds and optionally pushes the container image for OpenShift deployment

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_NAME="${IMAGE_NAME:-mock-log-generator}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-quay.io/alert-engine}"
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
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

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build and optionally push MockLogGenerator container image.

OPTIONS:
    -h, --help          Show this help message
    -p, --push          Push image to registry after build
    -t, --tag TAG       Image tag (default: latest)
    -r, --registry REG  Container registry (default: quay.io/alert-engine)
    -n, --name NAME     Image name (default: mock-log-generator)
    --no-cache          Build without using cache
    --dry-run           Show commands without executing

ENVIRONMENT VARIABLES:
    IMAGE_NAME          Override default image name
    IMAGE_TAG           Override default image tag
    REGISTRY            Override default registry
    CONTAINER_TOOL      Container tool to use (podman or docker, auto-detected)

EXAMPLES:
    $0                                      # Build image locally
    $0 --push                               # Build and push to registry
    $0 --tag v1.0.0 --push                 # Build and push with specific tag
    $0 --registry myregistry.com/myorg     # Use different registry

EOF
}

detect_container_tool() {
    if command -v podman >/dev/null 2>&1; then
        echo "podman"
    elif command -v docker >/dev/null 2>&1; then
        echo "docker"
    else
        log_error "Neither podman nor docker found. Please install one of them."
        exit 1
    fi
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check container tool
    local tool=$(detect_container_tool)
    log_info "Using container tool: $tool"
    
    # Check if required files exist
    local required_files=("Dockerfile" "requirements.txt" "mock_log_generator.py")
    for file in "${required_files[@]}"; do
        if [[ ! -f "$SCRIPT_DIR/$file" ]]; then
            log_error "Required file not found: $file"
            exit 1
        fi
    done
    
    log_success "Prerequisites check passed"
    return 0
}

build_image() {
    local tool="${CONTAINER_TOOL:-$(detect_container_tool)}"
    local cache_flag=""
    
    if [[ "${NO_CACHE:-false}" == "true" ]]; then
        cache_flag="--no-cache"
    fi
    
    log_info "Building container image..."
    log_info "Image: $FULL_IMAGE"
    log_info "Context: $SCRIPT_DIR"
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: $tool build --platform linux/amd64 $cache_flag -t $FULL_IMAGE $SCRIPT_DIR"
        return 0
    fi
    
    # Change to script directory for build context
    cd "$SCRIPT_DIR"
    
    # Build the image for x86_64 architecture (OpenShift standard)
    $tool build --platform linux/amd64 $cache_flag -t "$FULL_IMAGE" .
    
    if [[ $? -eq 0 ]]; then
        log_success "Image built successfully: $FULL_IMAGE"
    else
        log_error "Image build failed"
        exit 1
    fi
}

push_image() {
    local tool="${CONTAINER_TOOL:-$(detect_container_tool)}"
    
    log_info "Pushing image to registry..."
    log_info "Image: $FULL_IMAGE"
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: $tool push $FULL_IMAGE"
        return 0
    fi
    
    $tool push "$FULL_IMAGE"
    
    if [[ $? -eq 0 ]]; then
        log_success "Image pushed successfully: $FULL_IMAGE"
    else
        log_error "Image push failed"
        exit 1
    fi
}

update_deployment_manifest() {
    local deployment_file="$SCRIPT_DIR/deployment.yaml"
    
    if [[ ! -f "$deployment_file" ]]; then
        log_warning "Deployment file not found: $deployment_file"
        return 0
    fi
    
    log_info "Updating deployment manifest with new image reference..."
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: Update $deployment_file with image $FULL_IMAGE"
        return 0
    fi
    
    # Update image reference in deployment.yaml
    if command -v yq >/dev/null 2>&1; then
        # Use yq if available
        yq eval ".spec.template.spec.containers[0].image = \"$FULL_IMAGE\"" -i "$deployment_file"
        log_success "Updated deployment manifest with yq"
    elif command -v sed >/dev/null 2>&1; then
        # Fallback to sed
        sed -i.bak "s|image: quay.io/alert-engine/mock-log-generator:.*|image: $FULL_IMAGE|g" "$deployment_file"
        log_success "Updated deployment manifest with sed"
    else
        log_warning "Neither yq nor sed available. Please manually update the image reference in $deployment_file"
    fi
}

main() {
    local push_image_flag=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -p|--push)
                push_image_flag=true
                shift
                ;;
            -t|--tag)
                IMAGE_TAG="$2"
                FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
                shift 2
                ;;
            -r|--registry)
                REGISTRY="$2"
                FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
                shift 2
                ;;
            -n|--name)
                IMAGE_NAME="$2"
                FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
                shift 2
                ;;
            --no-cache)
                NO_CACHE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Main execution
    log_info "MockLogGenerator Container Build Script"
    log_info "========================================"
    
    check_prerequisites
    build_image
    
    if [[ "$push_image_flag" == "true" ]]; then
        push_image
        update_deployment_manifest
        
        log_info ""
        log_success "Build and push completed successfully!"
        log_info "Next steps:"
        log_info "1. Apply the updated deployment: oc apply -f deployment.yaml"
        log_info "2. Or use kustomize: oc apply -k ."
        log_info "3. Check pod status: oc get pods -n mock-logs -l app=mock-log-generator"
    else
        log_info ""
        log_success "Build completed successfully!"
        log_info "To push the image, run: $0 --push"
    fi
}

# Execute main function
main "$@" 