#!/bin/bash
set -euo pipefail

# MockLogGenerator OpenShift Deployment Script
# Automates the complete deployment process for MockLogGenerator

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-mock-logs}"
IMAGE_NAME="${IMAGE_NAME:-mock-log-generator}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-quay.io/alert-engine}"
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
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

log_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

log_check() {
    echo -e "${CYAN}[CHECK]${NC} $1"
}

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS] [COMMAND]

Deploy MockLogGenerator to OpenShift for Alert Engine testing.

COMMANDS:
    deploy      Deploy MockLogGenerator (default)
    undeploy    Remove MockLogGenerator deployment
    status      Check deployment status
    logs        Show application logs
    restart     Restart the deployment
    build       Build container image only
    validate    Validate prerequisites and configuration

OPTIONS:
    -h, --help              Show this help message
    -n, --namespace NS      Kubernetes namespace (default: mock-logs)
    -t, --tag TAG           Container image tag (default: latest)
    -r, --registry REG      Container registry (default: quay.io/alert-engine)
    --image IMAGE           Full image reference (overrides registry/name/tag)
    --skip-build            Skip container image build
    --skip-push             Skip container image push
    --skip-validation       Skip prerequisite validation
    --dry-run               Show commands without executing
    --wait                  Wait for deployment to be ready
    --follow-logs           Follow logs after deployment

ENVIRONMENT VARIABLES:
    NAMESPACE               Override default namespace
    REGISTRY                Override default registry
    IMAGE_TAG               Override default image tag
    KUBECONFIG              Kubernetes config file

EXAMPLES:
    $0                                  # Deploy with defaults
    $0 deploy --wait --follow-logs      # Deploy and monitor
    $0 -n test-environment deploy       # Deploy to different namespace
    $0 --registry myregistry.com/org    # Use custom registry
    $0 undeploy                         # Remove deployment
    $0 status                           # Check current status
    $0 validate                         # Check prerequisites only

EOF
}

check_prerequisites() {
    log_step "Checking prerequisites..."
    
    # Check if oc command is available
    if ! command -v oc >/dev/null 2>&1; then
        log_error "OpenShift CLI (oc) not found. Please install it."
        exit 1
    fi
    
    # Check if we're logged into OpenShift
    if ! oc whoami >/dev/null 2>&1; then
        log_error "Not logged into OpenShift. Please run 'oc login' first."
        exit 1
    fi
    
    # Check cluster connectivity
    if ! oc cluster-info >/dev/null 2>&1; then
        log_error "Cannot connect to OpenShift cluster."
        exit 1
    fi
    
    log_success "OpenShift connectivity verified"
    
    # Check if required files exist
    local required_files=("configmap.yaml" "serviceaccount.yaml" "deployment.yaml" "networkpolicy.yaml")
    for file in "${required_files[@]}"; do
        if [[ ! -f "$SCRIPT_DIR/$file" ]]; then
            log_error "Required manifest file not found: $file"
            exit 1
        fi
    done
    
    log_success "Required manifest files found"
    
    # Check if Alert Engine infrastructure is deployed
    log_check "Verifying Alert Engine infrastructure..."
    
    # Check OpenShift Logging Operator
    if ! oc get clusterlogforwarder -n openshift-logging >/dev/null 2>&1; then
        log_warning "OpenShift Logging not configured - logs may not be collected"
        log_warning "Make sure ClusterLogForwarder is set up to collect logs"
    else
        log_success "OpenShift Logging found"
    fi
    
    # Check Vector pods (log collectors)
    if ! oc get pods -n openshift-logging -l component=vector >/dev/null 2>&1; then
        log_warning "Vector log collector pods not found"
        log_warning "Logs may not be collected from MockLogGenerator"
    else
        log_success "Vector log collectors found"
    fi
    
    log_success "Prerequisites check completed"
}

create_namespace() {
    log_step "Ensuring namespace exists..."
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: oc create namespace $NAMESPACE --dry-run=client -o yaml | oc apply -f -"
        return 0
    fi
    
    # Create namespace if it doesn't exist
    if ! oc get namespace "$NAMESPACE" >/dev/null 2>&1; then
        log_info "Creating namespace: $NAMESPACE"
        oc create namespace "$NAMESPACE"
        log_success "Namespace created: $NAMESPACE"
    else
        log_info "Namespace already exists: $NAMESPACE"
    fi
}

build_and_push_image() {
    if [[ "${SKIP_BUILD:-false}" == "true" ]]; then
        log_info "Skipping container image build"
        return 0
    fi
    
    log_step "Building container image..."
    
    # Check if build script exists
    if [[ ! -f "$SCRIPT_DIR/build.sh" ]]; then
        log_error "Build script not found: $SCRIPT_DIR/build.sh"
        exit 1
    fi
    
    local build_args=("--tag" "$IMAGE_TAG" "--registry" "$REGISTRY")
    
    if [[ "${SKIP_PUSH:-false}" != "true" ]]; then
        build_args+=("--push")
    fi
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        build_args+=("--dry-run")
    fi
    
    "$SCRIPT_DIR/build.sh" "${build_args[@]}"
}

deploy_manifests() {
    log_step "Deploying MockLogGenerator manifests..."
    
    cd "$SCRIPT_DIR"
    
    # Update image reference in deployment
    if [[ "${DRY_RUN:-false}" != "true" ]]; then
        if command -v yq >/dev/null 2>&1; then
            yq eval ".spec.template.spec.containers[0].image = \"$FULL_IMAGE\"" -i deployment.yaml
        elif command -v sed >/dev/null 2>&1; then
            sed -i.bak "s|image: quay.io/alert-engine/mock-log-generator:.*|image: $FULL_IMAGE|g" deployment.yaml
        fi
    fi
    
    # Apply manifests in order
    local manifests=("configmap.yaml" "serviceaccount.yaml" "networkpolicy.yaml" "deployment.yaml")
    
    for manifest in "${manifests[@]}"; do
        log_info "Applying $manifest..."
        
        if [[ "${DRY_RUN:-false}" == "true" ]]; then
            echo "DRY RUN: oc apply -f $manifest -n $NAMESPACE"
        else
            oc apply -f "$manifest" -n "$NAMESPACE"
        fi
    done
    
    log_success "Manifests applied successfully"
}

wait_for_deployment() {
    if [[ "${WAIT:-false}" != "true" ]]; then
        return 0
    fi
    
    log_step "Waiting for deployment to be ready..."
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: oc rollout status deployment/mock-log-generator -n $NAMESPACE --timeout=300s"
        return 0
    fi
    
    # Wait for deployment to be ready
    if oc rollout status deployment/mock-log-generator -n "$NAMESPACE" --timeout=300s; then
        log_success "Deployment is ready"
    else
        log_error "Deployment failed to become ready within timeout"
        show_troubleshooting_info
        exit 1
    fi
}

show_status() {
    log_step "Checking deployment status..."
    
    echo ""
    echo "📊 Deployment Status:"
    oc get deployment mock-log-generator -n "$NAMESPACE" -o wide 2>/dev/null || log_warning "Deployment not found"
    
    echo ""
    echo "🚀 Pod Status:"
    oc get pods -n "$NAMESPACE" -l app=mock-log-generator -o wide 2>/dev/null || log_warning "No pods found"
    
    echo ""
    echo "📋 ConfigMap:"
    oc get configmap mock-log-generator-config -n "$NAMESPACE" 2>/dev/null || log_warning "ConfigMap not found"
    
    echo ""
    echo "🔐 ServiceAccount:"
    oc get serviceaccount mock-log-generator -n "$NAMESPACE" 2>/dev/null || log_warning "ServiceAccount not found"
    
    echo ""
    echo "🌐 NetworkPolicy:"
    oc get networkpolicy mock-log-generator-netpol -n "$NAMESPACE" 2>/dev/null || log_warning "NetworkPolicy not found"
}

show_logs() {
    log_step "Showing application logs..."
    
    local follow_flag=""
    if [[ "${FOLLOW_LOGS:-false}" == "true" ]]; then
        follow_flag="-f"
    fi
    
    if oc get pods -n "$NAMESPACE" -l app=mock-log-generator >/dev/null 2>&1; then
        oc logs -n "$NAMESPACE" -l app=mock-log-generator $follow_flag --tail=50
    else
        log_warning "No pods found for MockLogGenerator"
    fi
}

show_troubleshooting_info() {
    log_warning "Deployment issues detected. Here's some troubleshooting information:"
    
    echo ""
    echo "🔍 Pod Events:"
    oc describe pods -n "$NAMESPACE" -l app=mock-log-generator | grep -A 10 "Events:" || true
    
    echo ""
    echo "📝 Recent Pod Logs:"
    oc logs -n "$NAMESPACE" -l app=mock-log-generator --tail=20 || true
    
    echo ""
    echo "🚨 Pod Status:"
    oc get pods -n "$NAMESPACE" -l app=mock-log-generator -o yaml | grep -A 10 "conditions:" || true
}

undeploy() {
    log_step "Removing MockLogGenerator deployment..."
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: Remove all MockLogGenerator resources from $NAMESPACE"
        return 0
    fi
    
    # Delete resources in reverse order
    local manifests=("deployment.yaml" "networkpolicy.yaml" "serviceaccount.yaml" "configmap.yaml")
    
    for manifest in "${manifests[@]}"; do
        if [[ -f "$SCRIPT_DIR/$manifest" ]]; then
            log_info "Removing $manifest..."
            oc delete -f "$SCRIPT_DIR/$manifest" -n "$NAMESPACE" --ignore-not-found=true
        fi
    done
    
    # Also delete by label as backup
    oc delete all,configmap,serviceaccount,networkpolicy -n "$NAMESPACE" -l app=mock-log-generator --ignore-not-found=true
    
    log_success "MockLogGenerator removed successfully"
}

restart_deployment() {
    log_step "Restarting MockLogGenerator deployment..."
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        echo "DRY RUN: oc rollout restart deployment/mock-log-generator -n $NAMESPACE"
        return 0
    fi
    
    if oc get deployment mock-log-generator -n "$NAMESPACE" >/dev/null 2>&1; then
        oc rollout restart deployment/mock-log-generator -n "$NAMESPACE"
        log_success "Deployment restart initiated"
        
        if [[ "${WAIT:-false}" == "true" ]]; then
            wait_for_deployment
        fi
    else
        log_error "Deployment not found: mock-log-generator"
        exit 1
    fi
}

validate_only() {
    log_step "Validation mode - checking prerequisites only..."
    check_prerequisites
    log_success "Validation completed successfully"
    
    log_info "Configuration:"
    log_info "  Namespace: $NAMESPACE"
    log_info "  Image: $FULL_IMAGE"
    log_info "  Registry: $REGISTRY"
    
    echo ""
    log_info "To deploy, run: $0 deploy"
}

main() {
    local command="deploy"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
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
            --image)
                FULL_IMAGE="$2"
                shift 2
                ;;
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --skip-push)
                SKIP_PUSH=true
                shift
                ;;
            --skip-validation)
                SKIP_VALIDATION=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --wait)
                WAIT=true
                shift
                ;;
            --follow-logs)
                FOLLOW_LOGS=true
                shift
                ;;
            deploy|undeploy|status|logs|restart|build|validate)
                command="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Header
    echo ""
    log_info "MockLogGenerator OpenShift Deployment"
    log_info "====================================="
    log_info "Command: $command"
    log_info "Namespace: $NAMESPACE"
    log_info "Image: $FULL_IMAGE"
    echo ""
    
    # Execute command
    case $command in
        deploy)
            if [[ "${SKIP_VALIDATION:-false}" != "true" ]]; then
                check_prerequisites
            fi
            create_namespace
            build_and_push_image
            deploy_manifests
            wait_for_deployment
            show_status
            if [[ "${FOLLOW_LOGS:-false}" == "true" ]]; then
                show_logs
            fi
            ;;
        undeploy)
            undeploy
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs
            ;;
        restart)
            restart_deployment
            ;;
        build)
            build_and_push_image
            ;;
        validate)
            validate_only
            ;;
        *)
            log_error "Unknown command: $command"
            print_usage
            exit 1
            ;;
    esac
    
    echo ""
    log_success "Command '$command' completed successfully!"
}

# Execute main function
main "$@" 