#!/bin/bash

# Local E2E Environment Setup Script for Alert Engine
# This script sets up a complete local testing environment with Docker services

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose-local-e2e.yml"
CONFIG_FILE="${SCRIPT_DIR}/config_local_e2e.yaml"
LOG_FILE="${SCRIPT_DIR}/setup.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "${LOG_FILE}"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "${LOG_FILE}"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "${LOG_FILE}"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "${LOG_FILE}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check Podman
    if ! command -v podman &> /dev/null; then
        error "Podman is not installed or not in PATH"
        exit 1
    fi
    
    # Check Podman Compose
    if ! podman compose version &> /dev/null; then
        error "Podman compose is not available"
        exit 1
    fi
    
    # Check if Podman is working
    if ! podman info &> /dev/null; then
        error "Podman is not working properly"
        exit 1
    fi
    
    # Check Go
    if ! command -v go &> /dev/null; then
        error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check Python (for mock log forwarder)
    if ! command -v python3 &> /dev/null; then
        error "Python3 is not installed or not in PATH"
        exit 1
    fi
    
    success "All prerequisites are met"
}

# Clean up any existing resources
cleanup_existing() {
    log "Cleaning up any existing E2E resources..."
    
    # Stop containers if running and remove volumes to prevent Kafka cluster ID conflicts
    if podman compose -f "${COMPOSE_FILE}" ps -q | grep -q .; then
        warning "Stopping existing containers and cleaning volumes..."
        podman compose -f "${COMPOSE_FILE}" down --remove-orphans --volumes || true
    else
        # Even if no containers are running, clean up any orphaned volumes
        warning "Cleaning up any existing volumes..."
        podman compose -f "${COMPOSE_FILE}" down --volumes || true
    fi
    
    # Remove networks if they exist
    if podman network ls | grep -q "alert-engine-e2e"; then
        warning "Removing existing network..."
        podman network rm alert-engine-e2e || true
    fi
    
    success "Cleanup completed"
}

# Start Docker services
start_docker_services() {
    log "Starting Podman services..."
    
    # Start base services (Redis and Kafka)
    podman compose -f "${COMPOSE_FILE}" up -d zookeeper-e2e redis-e2e
    
    log "Waiting for Zookeeper and Redis to be healthy..."
    sleep 10
    
    # Start Kafka
    podman compose -f "${COMPOSE_FILE}" up -d kafka-e2e
    
    log "Waiting for Kafka container to start..."
    
    # Wait for Kafka container to be running (up to 30 seconds)
    local max_attempts=6
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if podman compose -f "${COMPOSE_FILE}" ps --format json kafka-e2e | grep -q '"State":"running"'; then
            success "Kafka container is running"
            break
        fi
        
        log "Attempt $attempt/$max_attempts: Waiting for Kafka container..."
        sleep 5
        ((attempt++))
    done
    
    if [ $attempt -gt $max_attempts ]; then
        error "Kafka container failed to start within expected time"
        exit 1
    fi
    
    # Give Kafka a few extra seconds to initialize internally
    log "Allowing Kafka time to initialize..."
    sleep 10
    
    success "All Podman services are running"
}

# Create Kafka topics
create_kafka_topics() {
    log "Creating Kafka topics..."
    
    # Create application-logs topic (used by alert engine)
    podman compose -f "${COMPOSE_FILE}" exec -T kafka-e2e kafka-topics --create \
        --bootstrap-server localhost:9094 \
        --topic application-logs \
        --partitions 3 \
        --replication-factor 1 \
        --if-not-exists
    
    # Verify topics
    log "Verifying topics..."
    podman compose -f "${COMPOSE_FILE}" exec -T kafka-e2e kafka-topics --list --bootstrap-server localhost:9094
    
    success "Kafka topics created successfully"
}

# Setup Python environment for mock log forwarder
setup_python_environment() {
    log "Setting up Python environment for mock log forwarder..."
    
    # Check if virtual environment exists
    if [ ! -d "${SCRIPT_DIR}/venv" ]; then
        log "Creating Python virtual environment..."
        python3 -m venv "${SCRIPT_DIR}/venv"
    fi
    
    # Activate virtual environment and install dependencies
    source "${SCRIPT_DIR}/venv/bin/activate"
    
    # Install kafka-python if not already installed
    if ! pip show kafka-python &> /dev/null; then
        log "Installing kafka-python..."
        pip install kafka-python
    fi
    
    success "Python environment is ready"
}

# Build alert engine
build_alert_engine() {
    log "Building alert engine..."
    
    cd "${PROJECT_ROOT}/alert-engine"
    
    # Build the Go application
    if ! go build -o bin/alert-engine ./cmd/server; then
        error "Failed to build alert engine"
        exit 1
    fi
    
    success "Alert engine built successfully"
}

# Test connectivity
test_connectivity() {
    log "Testing service connectivity..."
    
    # Test Redis
    if podman compose -f "${COMPOSE_FILE}" exec -T redis-e2e redis-cli -a e2epass ping | grep -q PONG; then
        success "Redis connectivity: OK"
    else
        error "Redis connectivity: FAILED"
        exit 1
    fi
    
    # Test Kafka (check if container is running)
    if podman compose -f "${COMPOSE_FILE}" ps --format json kafka-e2e | grep -q '"State":"running"'; then
        success "Kafka container status: OK"
    else
        error "Kafka container status: FAILED"
        exit 1
    fi
}

# Start debug services (optional)
start_debug_services() {
    if [ "${ENABLE_DEBUG_UI:-false}" = "true" ]; then
        log "Starting debug UI services..."
        podman compose -f "${COMPOSE_FILE}" --profile debug up -d
        
        log "Debug UIs available at:"
        log "  - Kafka UI: http://localhost:8081"
        log "  - Redis Commander: http://localhost:8082"
    fi
}

# Display status and next steps
display_status() {
    success "Local E2E environment setup completed!"
    
    echo ""
    echo "=== Environment Status ==="
    echo "Podman Services:"
    podman compose -f "${COMPOSE_FILE}" ps
    
    echo ""
    echo "=== Service Endpoints ==="
    echo "Kafka Broker: localhost:9094"
    echo "Redis: localhost:6379 (password: e2epass)"
    echo "Alert Engine Config: ${CONFIG_FILE}"
    
    echo ""
    echo "=== Slack Configuration ==="
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]] && [[ "$SLACK_WEBHOOK_URL" != "https://hooks.slack.com/services/YOUR_WORKSPACE/YOUR_CHANNEL/YOUR_WEBHOOK_TOKEN" ]]; then
        echo "✅ Slack webhook URL configured and exported"
        echo "   Channel: $(grep -o '#[^"]*' ${CONFIG_FILE} | head -1 || echo '#test-mp-channel')"
        echo "   Webhook: ${SLACK_WEBHOOK_URL:0:50}..."
        echo "   Environment: SLACK_WEBHOOK_URL exported for Go process inheritance"
    else
        echo "⚠️  Slack webhook URL not configured (using placeholder)"
        echo "   To enable Slack notifications:"
        echo "   1. Edit ${SCRIPT_DIR}/.env"
        echo "   2. Replace SLACK_WEBHOOK_URL with your actual webhook URL"
        echo "   3. Get webhook URL from: https://api.slack.com/apps"
        echo "   4. Important: Use 'source .env && export SLACK_WEBHOOK_URL' when starting alert engine"
    fi
    
    echo ""
    echo "=== Next Steps ==="
    echo "1. Start the alert engine with proper environment variables:"
    echo "   # Method 1: Using exported environment (RECOMMENDED)"
    echo "   cd ${PROJECT_ROOT} && source ${SCRIPT_DIR}/.env && export SLACK_WEBHOOK_URL && CONFIG_PATH=${CONFIG_FILE} go run cmd/server/main.go"
    echo ""
    echo "   # Method 2: Using binary (if built)"
    echo "   cd ${PROJECT_ROOT} && source ${SCRIPT_DIR}/.env && export SLACK_WEBHOOK_URL && CONFIG_PATH=${CONFIG_FILE} ./bin/alert-engine"
    echo ""
    echo "2. Start the mock log forwarder:"
    echo "   cd ${SCRIPT_DIR} && source .env && source venv/bin/activate && python3 mock_log_forwarder.py --mode test"
    echo ""
    echo "3. Run E2E tests:"
    echo "   cd ${PROJECT_ROOT}/local_e2e/tests && ./run_e2e_tests.sh"
    echo ""
    echo "4. To stop the environment:"
    echo "   cd ${SCRIPT_DIR} && ./teardown_local_e2e.sh"
    
    echo ""
    echo "=== Helper Scripts ==="
    echo "Quick start alert engine:"
    echo "   ${SCRIPT_DIR}/start_alert_engine.sh"
    echo ""
    echo "Test Slack webhook:"
    echo "   ${SCRIPT_DIR}/test_slack.sh"
    
    if [ "${ENABLE_DEBUG_UI:-false}" = "true" ]; then
        echo ""
        echo "=== Debug UIs ==="
        echo "Kafka UI: http://localhost:8081"
        echo "Redis Commander: http://localhost:8082"
    fi
}

# Create .env file with default configuration
create_env_file() {
    local env_file="${SCRIPT_DIR}/.env"
    
    if [[ ! -f "$env_file" ]]; then
        log "Creating .env file with default configuration"
        cat > "$env_file" << 'EOF'
# E2E Testing Environment Variables
# 
# SLACK CONFIGURATION
# Replace this with your actual Slack webhook URL for testing
# You can get this by creating a Slack app and enabling incoming webhooks
# Format: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR_WORKSPACE/YOUR_CHANNEL/YOUR_WEBHOOK_TOKEN

# Note: The above is a placeholder URL. For actual testing, you need to:
# 1. Go to https://api.slack.com/apps
# 2. Create a new app or use existing one
# 3. Enable "Incoming Webhooks" feature
# 4. Create a webhook for your desired channel
# 5. Replace the URL above with your actual webhook URL
#
# If you don't have a Slack workspace for testing, the alert engine will
# still work but notifications won't be sent (check logs for webhook errors)
EOF
        success "Created .env file at ${env_file}"
        warning "Please edit .env file with your actual Slack webhook URL for notifications to work"
    else
        log ".env file already exists"
    fi
}

# Main execution
main() {
    # Clear log file before any logging
    > "${LOG_FILE}"
    
    # Create .env file if it doesn't exist
    create_env_file
    
    # Load and export environment variables
    if [[ -f "${SCRIPT_DIR}/.env" ]]; then
        log "Loading environment variables from .env file"
        source "${SCRIPT_DIR}/.env"
        
        # Explicitly export critical environment variables for Go process inheritance
        if [[ -n "$SLACK_WEBHOOK_URL" ]]; then
            export SLACK_WEBHOOK_URL
            log "SLACK_WEBHOOK_URL exported for Go process inheritance"
        fi
        
        if [[ -n "$REDIS_ADDRESS" ]]; then
            export REDIS_ADDRESS
        fi
        
        if [[ -n "$KAFKA_BROKERS" ]]; then
            export KAFKA_BROKERS
        fi
        
        if [[ -n "$CONFIG_PATH" ]]; then
            export CONFIG_PATH
        fi
        
        # Validate Slack configuration
        if [[ -n "$SLACK_WEBHOOK_URL" ]] && [[ "$SLACK_WEBHOOK_URL" != "https://hooks.slack.com/services/YOUR_WORKSPACE/YOUR_CHANNEL/YOUR_WEBHOOK_TOKEN" ]]; then
            log "Slack webhook URL loaded and exported successfully"
        else
            warning "SLACK_WEBHOOK_URL not configured or using placeholder URL"
            warning "Slack notifications will not work until you provide a real webhook URL"
        fi
    else
        warning "No .env file found in ${SCRIPT_DIR}"
    fi
    
    log "Starting Local E2E Environment Setup for Alert Engine"
    log "Script directory: ${SCRIPT_DIR}"
    log "Project root: ${PROJECT_ROOT}"
    
    check_prerequisites
    cleanup_existing
    start_docker_services
    create_kafka_topics
    setup_python_environment
    build_alert_engine
    test_connectivity
    start_debug_services
    
    # Ensure helper scripts are executable
    chmod +x "${SCRIPT_DIR}/start_alert_engine.sh" 2>/dev/null || true
    chmod +x "${SCRIPT_DIR}/test_slack.sh" 2>/dev/null || true
    
    display_status
    
    success "Setup completed successfully!"
}

# Handle interrupts
trap 'error "Setup interrupted by user"; exit 1' INT TERM

# Run main function
main "$@" 