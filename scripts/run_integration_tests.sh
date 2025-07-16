#!/bin/bash

# Integration Testing Script for Alert Engine
# This script runs integration tests using Docker/Podman containers

set -e

echo "=========================================="
echo "Alert Engine - Integration Test Suite"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.test.yml"
CONTAINER_ENGINE="docker"
PROJECT_NAME="alert-engine-test"
SKIP_HEALTH_CHECK="${SKIP_HEALTH_CHECK:-false}"

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to detect container engine
detect_container_engine() {
    # Check if Podman is available and running
    if command -v podman &> /dev/null && podman ps &> /dev/null; then
        CONTAINER_ENGINE="podman"
        print_status $BLUE "Using Podman for container orchestration"
        
        # Check if podman machine is running
        if ! podman machine list --format "{{.Running}}" | grep -q "true"; then
            print_status $YELLOW "Starting Podman machine..."
            podman machine start podman-machine-default
        fi
        
        # Set environment variables for testcontainers to work with Podman
        export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
        export DOCKER_HOST=unix:///var/run/docker.sock
        
    elif command -v docker-compose &> /dev/null || command -v docker &> /dev/null; then
        CONTAINER_ENGINE="docker"
        print_status $BLUE "Using Docker for container orchestration"
    else
        print_status $RED "❌ Neither Docker nor Podman found. Please install one of them."
        exit 1
    fi
}

# Function to start test containers
start_containers() {
    print_status $YELLOW "Starting test containers..."
    
    # Use docker-compose for both Docker and Podman (Podman Desktop provides compatibility)
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d
    
    print_status $GREEN "✅ Test containers started"
    
    # Wait for services to be healthy
    print_status $YELLOW "Waiting for services to be ready..."
    print_status $YELLOW "  Waiting 60 seconds for initial container startup..."
    sleep 60
    
    # Check container health status first
    print_status $YELLOW "Checking container health status..."
    if [ "$CONTAINER_ENGINE" = "podman" ]; then
        podman ps --filter "name=${PROJECT_NAME}" --format "table {{.Names}}\t{{.Status}}"
    else
        docker ps --filter "name=${PROJECT_NAME}" --format "table {{.Names}}\t{{.Status}}"
    fi
    
    # Check service health (can be skipped with SKIP_HEALTH_CHECK=true)
    if [ "$SKIP_HEALTH_CHECK" = "true" ]; then
        print_status $YELLOW "⚠️  Skipping health check (SKIP_HEALTH_CHECK=true)"
        print_status $YELLOW "  Proceeding directly to tests..."
    else
        if ! check_service_health; then
            print_status $RED "❌ Health check failed!"
            print_status $YELLOW "💡 You can bypass health checks by setting: SKIP_HEALTH_CHECK=true"
            print_status $YELLOW "💡 Or run tests directly with: go test -tags=integration -v ./internal/api"
            return 1
        fi
    fi
}

# Function to stop test containers
stop_containers() {
    print_status $YELLOW "Stopping test containers..."
    
    # Use docker-compose for both Docker and Podman (Podman Desktop provides compatibility)
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down -v
    
    print_status $GREEN "✅ Test containers stopped"
}

# Function to check service health
check_service_health() {
    print_status $YELLOW "Checking service health..."
    
    # Determine container exec command
    local CONTAINER_EXEC_CMD="docker"
    if [ "$CONTAINER_ENGINE" = "podman" ]; then
        CONTAINER_EXEC_CMD="podman"
    fi
    
    # Enhanced Kafka health check with retries
    print_status $YELLOW "  Checking Kafka health (with retries)..."
    local kafka_ready=false
    local max_attempts=12  # 2 minutes total (12 * 10s)
    local attempt=1
    
    while [ $attempt -le $max_attempts ] && [ "$kafka_ready" = false ]; do
        print_status $YELLOW "    Attempt $attempt/$max_attempts: Testing Kafka connectivity..."
        
        # Method 1: Check if Kafka is accepting connections
        if timeout 10 bash -c "</dev/tcp/localhost/9093" &> /dev/null; then
            print_status $YELLOW "    TCP connection successful, testing Kafka API..."
            
            # Method 2: Try to list topics using containerized kafka-topics command
            if timeout 15 $CONTAINER_EXEC_CMD exec "${PROJECT_NAME}-kafka-test-1" \
                kafka-topics --bootstrap-server localhost:29092 --list &> /dev/null; then
                print_status $GREEN "  ✅ Kafka is healthy and API is responding"
                kafka_ready=true
                break
            else
                print_status $YELLOW "    Kafka API not ready yet..."
            fi
        else
            print_status $YELLOW "    Kafka TCP connection not ready..."
        fi
        
        if [ $attempt -lt $max_attempts ]; then
            print_status $YELLOW "    Waiting 10 seconds before retry..."
            sleep 10
        fi
        attempt=$((attempt + 1))
    done
    
    if [ "$kafka_ready" = false ]; then
        print_status $RED "  ❌ Kafka is not responding after $max_attempts attempts"
        print_status $YELLOW "  📋 Showing Kafka logs for debugging:"
        $CONTAINER_EXEC_CMD logs "${PROJECT_NAME}-kafka-test-1" --tail 50
        return 1
    fi
    
    # Enhanced Redis health check
    print_status $YELLOW "  Checking Redis health..."
    if timeout 10 redis-cli -h localhost -p 6380 -a testpass ping &> /dev/null; then
        print_status $GREEN "  ✅ Redis is healthy"
    else
        # Fallback to TCP check
        if timeout 5 bash -c "</dev/tcp/localhost/6380" &> /dev/null; then
            print_status $GREEN "  ✅ Redis is responding (TCP check)"
        else
            print_status $RED "  ❌ Redis is not responding"
            print_status $YELLOW "  📋 Showing Redis logs for debugging:"
            $CONTAINER_EXEC_CMD logs "${PROJECT_NAME}-redis-test-1" --tail 20
            return 1
        fi
    fi
    
    print_status $GREEN "🎉 All services are healthy!"
    return 0
}

# Function to run integration tests
run_integration_tests() {
    print_status $YELLOW "Running integration tests..."
    
    # Set environment variables for tests
    export KAFKA_BROKERS="localhost:9093"
    export REDIS_ADDR="localhost:6380"
    export REDIS_PASSWORD="testpass"
    
    # Run integration tests with build tag and proper timeouts
    if go test -tags=integration -v ./internal/kafka -timeout=5m; then
        print_status $GREEN "✅ Kafka integration tests PASSED"
    else
        print_status $RED "❌ Kafka integration tests FAILED"
        return 1
    fi
    
    # Run storage integration tests using testcontainers (no external Redis needed)
    if go test -tags=integration -v ./internal/storage -timeout=5m; then
        print_status $GREEN "✅ Storage integration tests PASSED"
    else
        print_status $RED "❌ Storage integration tests FAILED"
        return 1
    fi
    
    # Run notifications integration tests using mock HTTP server (no external dependencies)
    if go test -tags=integration -v ./internal/notifications -timeout=3m; then
        print_status $GREEN "✅ Notifications integration tests PASSED"
    else
        print_status $RED "❌ Notifications integration tests FAILED"
        return 1
    fi
    
    # Run API integration tests using HTTP server (no external dependencies)
    if go test -tags=integration -v ./internal/api -timeout=5m; then
        print_status $GREEN "✅ API integration tests PASSED"
    else
        print_status $RED "❌ API integration tests FAILED"
        return 1
    fi
}

# Function to run tests using container
run_tests_in_container() {
    print_status $YELLOW "Running tests inside container..."
    
    # Use docker-compose for both Docker and Podman
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" run --rm go-test \
        sh -c "
            # Install dependencies
            go mod download
            
            # Set environment variables to indicate container mode
            export CONTAINER_MODE=true
            export USE_EXISTING_SERVICES=true
            
            # Run notifications integration tests (using mock HTTP server)
            if [ -d './internal/notifications' ]; then
                echo 'Running Notifications integration tests...'
                go test -tags=integration -v ./internal/notifications -timeout=3m
            fi
            
            # Run API integration tests (using HTTP server)
            if [ -d './internal/api' ]; then
                echo 'Running API integration tests...'
                go test -tags=integration -v ./internal/api -timeout=5m
            fi
            
            # Note: Kafka and Storage tests are skipped in container mode
            # because they require testcontainers which need Docker-in-Docker
            echo ''
            echo '📝 Note: Kafka and Storage integration tests are skipped in container mode'
            echo '   They require testcontainers which need Docker-in-Docker capability'
            echo '   Use \"./scripts/run_integration_tests.sh\" (without container) to run all tests'
        "
}

# Function to run performance tests
run_performance_tests() {
    print_status $YELLOW "Running performance tests..."
    
    export KAFKA_BROKERS="localhost:9093"
    export REDIS_ADDR="localhost:6380"
    export REDIS_PASSWORD="testpass"
    
    # Set up environment variables for Podman/Docker testcontainers
    export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
    export DOCKER_HOST=unix:///var/run/docker.sock
    export TESTCONTAINERS_RYUK_DISABLED=true
    
    # Run Kafka performance tests
    if go test -tags=integration -bench=. -benchmem ./internal/kafka -timeout=10m; then
        print_status $GREEN "✅ Kafka performance tests completed"
    else
        print_status $YELLOW "⚠️  Kafka performance tests not found or skipped"
    fi
    
    # Run Storage performance tests
    if go test -tags=integration -v ./internal/storage -run TestRedisStore_Integration_Performance -timeout=10m; then
        print_status $GREEN "✅ Storage performance tests completed"
    else
        print_status $YELLOW "⚠️  Storage performance tests not found or skipped"
    fi
    
    # Run Notifications performance tests
    if go test -tags=integration -bench=. -benchmem ./internal/notifications -timeout=5m; then
        print_status $GREEN "✅ Notifications performance tests completed"
    else
        print_status $YELLOW "⚠️  Notifications performance tests not found or skipped"
    fi
    
    # Run API performance tests
    if go test -tags=integration -bench=. -benchmem ./internal/api -timeout=5m; then
        print_status $GREEN "✅ API performance tests completed"
    else
        print_status $YELLOW "⚠️  API performance tests not found or skipped"
    fi
}

# Function to cleanup on exit
cleanup() {
    print_status $YELLOW "Cleaning up..."
    stop_containers
}

# Trap to ensure cleanup on script exit
trap cleanup EXIT

# Function to show service logs
show_logs() {
    local service=$1
    print_status $BLUE "Showing logs for $service..."
    
    # Use docker-compose for both Docker and Podman
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs "$service"
}

# Main execution
main() {
    local test_mode="$1"
    
    # Set working directory to project root
    cd "$(dirname "$0")/.."
    
    # Detect container engine
    detect_container_engine
    
    # Check if compose file exists
    if [ ! -f "$COMPOSE_FILE" ]; then
        print_status $RED "❌ Docker Compose file '$COMPOSE_FILE' not found"
        exit 1
    fi
    
    # Start containers
    start_containers
    
    case "$test_mode" in
        "container")
            run_tests_in_container
            ;;
        "performance")
            run_integration_tests
            run_performance_tests
            ;;
        "logs")
            show_logs "kafka-test"
            show_logs "redis-test"
            ;;
        *)
            run_integration_tests
            ;;
    esac
    
    print_status $GREEN "🎉 Integration tests completed successfully!"
}

# Show usage information
show_usage() {
    echo "Usage: $0 [MODE]"
    echo ""
    echo "Modes:"
    echo "  (none)        Run integration tests on host"
    echo "  container     Run tests inside container (Notifications & API only)"
    echo "  performance   Run integration and performance tests"
    echo "  logs          Show container logs"
    echo "  --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run integration tests"
    echo "  $0 container          # Run tests in container (limited scope)"
    echo "  $0 performance        # Run performance tests"
    echo "  $0 logs               # Show service logs"
    echo ""
    echo "Environment Variables:"
    echo "  COMPOSE_FILE         Docker compose file (default: scripts/docker-compose.test.yml)"
    echo "  PROJECT_NAME         Container project name (default: alert-engine-test)"
    echo "  SKIP_HEALTH_CHECK    Skip service health checks (default: false)"
    echo ""
    echo "Troubleshooting:"
    echo "  If Kafka health check fails:"
    echo "    SKIP_HEALTH_CHECK=true $0"
    echo "  Or run tests directly with proper timeouts:"
    echo "    go test -tags=integration -v ./internal/api -timeout=5m"
    echo "    go test -tags=integration -v ./internal/notifications -timeout=3m"
    echo "    go test -tags=integration -v ./internal/storage -timeout=5m"
    echo "    go test -tags=integration -v ./internal/kafka -timeout=5m"
    echo ""
}

# Handle command line arguments
case "$1" in
    --help|-h)
        show_usage
        exit 0
        ;;
    container|performance|logs)
        main "$1"
        ;;
    "")
        main
        ;;
    *)
        echo "Unknown mode: $1"
        show_usage
        exit 1
        ;;
esac 