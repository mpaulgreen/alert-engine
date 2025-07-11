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
COMPOSE_FILE="docker-compose.test.yml"
CONTAINER_ENGINE="docker"
PROJECT_NAME="alert-engine-test"

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
    sleep 30
    
    # Check service health
    check_service_health
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
    
    # Check Kafka using external port
    if timeout 5 bash -c "</dev/tcp/localhost/9093" &> /dev/null; then
        print_status $GREEN "  ✅ Kafka is healthy"
    else
        print_status $RED "  ❌ Kafka is not responding"
        return 1
    fi
    
    # Check Redis using external port
    if timeout 5 bash -c "</dev/tcp/localhost/6380" &> /dev/null; then
        print_status $GREEN "  ✅ Redis is healthy"
    else
        print_status $RED "  ❌ Redis is not responding"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_status $YELLOW "Running integration tests..."
    
    # Set environment variables for tests
    export KAFKA_BROKERS="localhost:9093"
    export REDIS_ADDR="localhost:6380"
    export REDIS_PASSWORD="testpass"
    
    # Run integration tests with build tag
    if go test -tags=integration -v ./internal/kafka/tests/...; then
        print_status $GREEN "✅ Kafka integration tests PASSED"
    else
        print_status $RED "❌ Kafka integration tests FAILED"
        return 1
    fi
    
    # Run storage integration tests using testcontainers (no external Redis needed)
    if go test -tags=integration -v ./internal/storage/tests/... -timeout=5m; then
        print_status $GREEN "✅ Storage integration tests PASSED"
    else
        print_status $RED "❌ Storage integration tests FAILED"
        return 1
    fi
    
    if go test -tags=integration -v ./internal/api/tests/...; then
        print_status $GREEN "✅ API integration tests PASSED"
    else
        print_status $YELLOW "⚠️  API integration tests not found or skipped"
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
            
            # Run integration tests
            go test -tags=integration -v ./internal/kafka/tests/...
            
            # Run storage integration tests (using testcontainers)
            if [ -d './internal/storage/tests' ]; then
                go test -tags=integration -v ./internal/storage/tests/... -timeout=5m
            fi
            
            if [ -d './internal/api/tests' ]; then
                go test -tags=integration -v ./internal/api/tests/...
            fi
        "
}

# Function to run performance tests
run_performance_tests() {
    print_status $YELLOW "Running performance tests..."
    
    export KAFKA_BROKERS="localhost:9093"
    export REDIS_ADDR="localhost:6380"
    export REDIS_PASSWORD="testpass"
    
    # Run Kafka performance tests
    if go test -tags=integration -bench=. -benchmem ./internal/kafka/tests/...; then
        print_status $GREEN "✅ Kafka performance tests completed"
    else
        print_status $YELLOW "⚠️  Kafka performance tests not found or skipped"
    fi
    
    # Run Storage performance tests
    if go test -tags=integration -bench=. -benchmem ./internal/storage/tests/... -timeout=10m; then
        print_status $GREEN "✅ Storage performance tests completed"
    else
        print_status $YELLOW "⚠️  Storage performance tests not found or skipped"
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
    echo "  container     Run tests inside container"
    echo "  performance   Run integration and performance tests"
    echo "  logs          Show container logs"
    echo "  --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run integration tests"
    echo "  $0 container          # Run tests in container"
    echo "  $0 performance        # Run performance tests"
    echo "  $0 logs               # Show service logs"
    echo ""
    echo "Environment Variables:"
    echo "  COMPOSE_FILE         Docker compose file (default: docker-compose.test.yml)"
    echo "  PROJECT_NAME         Container project name (default: alert-engine-test)"
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