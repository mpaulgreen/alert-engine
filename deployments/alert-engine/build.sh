#!/bin/bash
set -euo pipefail

# Alert Engine Container Build Script
# Builds and optionally pushes the container image for OpenShift deployment

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
IMAGE_NAME="${IMAGE_NAME:-alert-engine}"
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

Build and optionally push Alert Engine container image.

OPTIONS:
    -h, --help          Show this help message
    -p, --push          Push image to registry after build
    -t, --tag TAG       Image tag (default: latest)
    -r, --registry REG  Container registry (default: quay.io/alert-engine)
    -n, --name NAME     Image name (default: alert-engine)
    -v, --version VER   Build version to embed in binary
    --no-cache          Build without using cache
    --dry-run           Show commands without executing
    --test              Run tests before building
    --update-secret     Load .env file and update secret.yaml with SLACK_WEBHOOK_URL

ENVIRONMENT VARIABLES:
    IMAGE_NAME          Override default image name
    IMAGE_TAG           Override default image tag
    REGISTRY            Override default registry
    BUILD_VERSION       Version to embed in binary
    CONTAINER_TOOL      Container tool to use (podman or docker, auto-detected)

EXAMPLES:
    $0                                      # Build image locally
    $0 --push                               # Build and push to registry
    $0 --tag v1.0.0 --push                 # Build and push with specific tag
    $0 --registry myregistry.com/myorg     # Use different registry
    $0 --version 1.2.3 --test --push       # Run tests, build with version, and push
    $0 --update-secret                      # Update secret.yaml with .env values
    $0 --update-secret --push              # Update secret and build/push image

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

load_env_file() {
    local env_file="$SCRIPT_DIR/.env"
    
    if [[ -f "$env_file" ]]; then
        log_info "Loading environment variables from .env file..."
        set -a  # automatically export all variables
        source "$env_file"
        set +a  # stop automatically exporting
        log_success "Environment variables loaded from .env"
    else
        log_warning ".env file not found at $env_file"
        log_info "Create .env file with: SLACK_WEBHOOK_URL=your_webhook_url"
        log_info "Or copy from template: cp .env.template .env"
        return 1
    fi
}

update_secret_with_env() {
    local secret_file="$SCRIPT_DIR/secret.yaml"
    
    if [[ -z "${SLACK_WEBHOOK_URL:-}" ]]; then
        log_warning "SLACK_WEBHOOK_URL not set in environment. Skipping secret update."
        return 0
    fi
    
    if [[ ! -f "$secret_file" ]]; then
        log_error "Secret file not found: $secret_file"
        return 1
    fi
    
    log_info "Updating secret with SLACK_WEBHOOK_URL from environment..."
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: Update $secret_file with base64 encoded SLACK_WEBHOOK_URL"
        return 0
    fi
    
    # Base64 encode the webhook URL
    local encoded_url
    if command -v base64 >/dev/null 2>&1; then
        encoded_url=$(echo -n "$SLACK_WEBHOOK_URL" | base64)
    else
        log_error "base64 command not found. Cannot encode webhook URL."
        return 1
    fi
    
    # Update the secret file
    if command -v yq >/dev/null 2>&1; then
        # Use yq if available
        yq eval ".data.\"SLACK_WEBHOOK_URL\" = \"$encoded_url\"" -i "$secret_file"
        log_success "Updated secret with yq"
    elif command -v sed >/dev/null 2>&1; then
        # Fallback to sed
        sed -i.bak "s|SLACK_WEBHOOK_URL: \".*\"|SLACK_WEBHOOK_URL: \"$encoded_url\"|g" "$secret_file"
        log_success "Updated secret with sed"
    else
        log_warning "Neither yq nor sed available. Please manually update SLACK_WEBHOOK_URL in $secret_file"
        log_info "Base64 encoded URL: $encoded_url"
    fi
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check container tool
    local tool=$(detect_container_tool)
    log_info "Using container tool: $tool"
    
    # Check if we're in the right directory structure
    if [[ ! -f "$PROJECT_ROOT/go.mod" ]]; then
        log_error "go.mod not found in project root: $PROJECT_ROOT"
        log_error "Please run this script from the alert-engine project directory"
        exit 1
    fi
    
    # Check if required files exist
    local required_files=("$SCRIPT_DIR/Dockerfile")
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            log_error "Required file not found: $file"
            exit 1
        fi
    done
    
    # Check if Go is installed (for testing)
    if [[ "${RUN_TESTS:-false}" == "true" ]]; then
        if ! command -v go >/dev/null 2>&1; then
            log_error "Go not found. Please install Go or skip tests with --no-test"
            exit 1
        fi
    fi
    
    log_success "Prerequisites check passed"
    return 0
}

run_tests() {
    log_info "Running Go tests..."
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: go test ./..."
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Run unit tests
    log_info "Running unit tests..."
    go test -v ./internal/... ./pkg/... || {
        log_error "Unit tests failed"
        exit 1
    }
    
    # Run integration tests if they exist
    if find . -name "*integration_test.go" -type f | grep -q .; then
        log_info "Running integration tests..."
        go test -v -tags=integration ./... || {
            log_error "Integration tests failed"
            exit 1
        }
    fi
    
    log_success "All tests passed"
}

build_image() {
    local tool="${CONTAINER_TOOL:-$(detect_container_tool)}"
    local cache_flag=""
    local build_args=""
    
    if [[ "${NO_CACHE:-false}" == "true" ]]; then
        cache_flag="--no-cache"
    fi
    
    if [[ -n "${BUILD_VERSION:-}" ]]; then
        build_args="--build-arg BUILD_VERSION=${BUILD_VERSION}"
    fi
    
    log_info "Building container image..."
    log_info "Image: $FULL_IMAGE"
    log_info "Context: $PROJECT_ROOT"
    log_info "Dockerfile: $SCRIPT_DIR/Dockerfile"
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: $tool build --platform linux/amd64 $cache_flag $build_args -f $SCRIPT_DIR/Dockerfile -t $FULL_IMAGE $PROJECT_ROOT"
        return 0
    fi
    
    # Build the image for x86_64 architecture (OpenShift standard)
    # Use project root as context but specify the Dockerfile location
    $tool build --platform linux/amd64 $cache_flag $build_args \
        -f "$SCRIPT_DIR/Dockerfile" \
        -t "$FULL_IMAGE" \
        "$PROJECT_ROOT"
    
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
        sed -i.bak "s|image: quay.io/your-registry/alert-engine:.*|image: $FULL_IMAGE|g" "$deployment_file"
        sed -i.bak "s|image: quay.io/alert-engine/alert-engine:.*|image: $FULL_IMAGE|g" "$deployment_file"
        log_success "Updated deployment manifest with sed"
    else
        log_warning "Neither yq nor sed available. Please manually update the image reference in $deployment_file"
    fi
}

update_kustomization() {
    local kustomization_file="$SCRIPT_DIR/kustomization.yaml"
    
    if [[ ! -f "$kustomization_file" ]]; then
        log_warning "Kustomization file not found: $kustomization_file"
        return 0
    fi
    
    log_info "Updating kustomization with new image reference..."
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: Update $kustomization_file with image $FULL_IMAGE"
        return 0
    fi
    
    # Update image reference in kustomization.yaml
    if command -v yq >/dev/null 2>&1; then
        yq eval ".images[0].newTag = \"$IMAGE_TAG\"" -i "$kustomization_file"
        yq eval ".images[0].name = \"$REGISTRY/$IMAGE_NAME\"" -i "$kustomization_file"
        log_success "Updated kustomization with yq"
    elif command -v sed >/dev/null 2>&1; then
        sed -i.bak "s|newTag: .*|newTag: $IMAGE_TAG|g" "$kustomization_file"
        log_success "Updated kustomization with sed"
    else
        log_warning "Neither yq nor sed available. Please manually update the image reference in $kustomization_file"
    fi
}

show_next_steps() {
    log_info ""
    log_success "Build completed successfully!"
    log_info "Image: $FULL_IMAGE"
    log_info ""
    log_info "Next steps:"
    log_info "1. Ensure .env file has SLACK_WEBHOOK_URL (or update secret.yaml manually)"
    log_info "2. Apply the deployment:"
    log_info "   oc apply -k ."
    log_info "3. Check deployment status:"
    log_info "   oc get all -n alert-engine"
    log_info "4. Monitor logs:"
    log_info "   oc logs -n alert-engine deployment/alert-engine -f"
    log_info "5. Test health endpoint:"
    log_info "   oc port-forward -n alert-engine svc/alert-engine 8080:8080"
    log_info "   curl http://localhost:8080/health"
}

main() {
    local push_image_flag=false
    local run_tests_flag=false
    local update_secret_flag=false
    
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
            -v|--version)
                BUILD_VERSION="$2"
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
            --test)
                run_tests_flag=true
                shift
                ;;
            --update-secret)
                update_secret_flag=true
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
    log_info "Alert Engine Container Build Script"
    log_info "===================================="
    log_info "Project Root: $PROJECT_ROOT"
    log_info "Build Context: $SCRIPT_DIR"
    
    # Load .env file if update-secret flag is set or if .env exists
    if [[ "$update_secret_flag" == "true" ]] || [[ -f "$SCRIPT_DIR/.env" ]]; then
        load_env_file
    fi
    
    # Update secret if flag is set
    if [[ "$update_secret_flag" == "true" ]]; then
        update_secret_with_env
    fi
    
    check_prerequisites
    
    if [[ "$run_tests_flag" == "true" ]]; then
        RUN_TESTS=true run_tests
    fi
    
    build_image
    
    if [[ "$push_image_flag" == "true" ]]; then
        push_image
        update_deployment_manifest
        update_kustomization
        show_next_steps
    else
        log_info ""
        log_success "Build completed successfully!"
        log_info "To push the image, run: $0 --push"
        log_info "To run tests before building, add: --test"
        log_info "To update secrets from .env, add: --update-secret"
    fi
}

# Execute main function
main "$@" 