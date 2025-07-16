#!/bin/bash
set -e

# Script to run Kafka integration tests with different execution modes
# Usage: ./run_kafka_integration_tests.sh [mode] [options]
# Modes: parallel, sequential, race-safe

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
MODE="sequential"
VERBOSE=false
RACE_DETECTION=false
CLEANUP=true
DOCKER_CLEANUP=false

usage() {
    echo -e "${BLUE}Usage: $0 [OPTIONS]${NC}"
    echo
    echo -e "${YELLOW}Execution Modes:${NC}"
    echo "  -m, --mode MODE       Test execution mode: parallel, sequential, race-safe (default: sequential)"
    echo
    echo -e "${YELLOW}Options:${NC}"
    echo "  -v, --verbose         Enable verbose output"
    echo "  -r, --race            Enable race detection (implies sequential mode)"
    echo "  -c, --cleanup         Clean up containers after tests (default: true)"
    echo "  -d, --docker-cleanup  Clean up all test containers before starting"
    echo "  -h, --help           Show this help message"
    echo
    echo -e "${YELLOW}Examples:${NC}"
    echo "  $0                              # Run in sequential mode (safe)"
    echo "  $0 -m parallel                  # Run in parallel mode (faster, may conflict)"
    echo "  $0 -m race-safe -r              # Run with race detection (slower, thorough)"
    echo "  $0 -v -d                        # Verbose mode with docker cleanup"
    echo
    echo -e "${YELLOW}Modes explained:${NC}"
    echo "  sequential:  One test at a time, minimal resource conflicts"
    echo "  parallel:    Multiple tests simultaneously, faster but may conflict"
    echo "  race-safe:   Sequential with optimizations for race detection"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -m|--mode)
            MODE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -r|--race)
            RACE_DETECTION=true
            shift
            ;;
        -c|--cleanup)
            CLEANUP=true
            shift
            ;;
        -d|--docker-cleanup)
            DOCKER_CLEANUP=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            exit 1
            ;;
    esac
done

# Validate mode
case $MODE in
    parallel|sequential|race-safe)
        ;;
    *)
        echo -e "${RED}Invalid mode: $MODE${NC}"
        echo "Valid modes: parallel, sequential, race-safe"
        exit 1
        ;;
esac

# If race detection is enabled, force sequential mode
if [ "$RACE_DETECTION" = true ]; then
    if [ "$MODE" = "parallel" ]; then
        echo -e "${YELLOW}Warning: Race detection enabled, switching to race-safe mode${NC}"
        MODE="race-safe"
    fi
fi

echo -e "${BLUE}Kafka Integration Test Runner${NC}"
echo -e "Mode: ${GREEN}$MODE${NC}"
echo -e "Race Detection: ${GREEN}$RACE_DETECTION${NC}"
echo -e "Verbose: ${GREEN}$VERBOSE${NC}"
echo ""

# Change to project directory
cd "$PROJECT_ROOT"

# Clean up Docker containers if requested
if [ "$DOCKER_CLEANUP" = true ]; then
    echo -e "${YELLOW}Cleaning up existing test containers...${NC}"
    
    # Detect container runtime (docker or podman)
    CONTAINER_CMD=""
    if command -v docker >/dev/null 2>&1; then
        CONTAINER_CMD="docker"
    elif command -v podman >/dev/null 2>&1; then
        CONTAINER_CMD="podman"
    fi
    
    if [ -n "$CONTAINER_CMD" ]; then
        echo -e "${BLUE}Using $CONTAINER_CMD for cleanup${NC}"
        
        # Remove any existing Kafka test containers
        $CONTAINER_CMD ps -a --filter "ancestor=confluentinc/cp-kafka:7.4.0" --format "{{.ID}}" | xargs -r $CONTAINER_CMD rm -f 2>/dev/null || true
        
        # Clean up any testcontainers networks
        $CONTAINER_CMD network ls --filter "name=testcontainers" --format "{{.ID}}" | xargs -r $CONTAINER_CMD network rm 2>/dev/null || true
        
        echo -e "${GREEN}Container cleanup completed${NC}"
    else
        echo -e "${YELLOW}Warning: Neither docker nor podman command found, skipping container cleanup${NC}"
        echo -e "${YELLOW}Testcontainers will handle cleanup automatically${NC}"
    fi
    echo ""
fi

# Set up environment variables for testcontainers
# Detect container runtime and configure accordingly
if command -v docker >/dev/null 2>&1; then
    export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE="/var/run/docker.sock"
    export DOCKER_HOST="unix:///var/run/docker.sock"
    echo -e "${BLUE}Detected Docker - using Docker socket${NC}"
elif command -v podman >/dev/null 2>&1; then
    export TESTCONTAINERS_RYUK_DISABLED=true
    export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE="unix:///run/podman/podman.sock"
    export DOCKER_HOST="unix:///run/podman/podman.sock"
    echo -e "${BLUE}Detected Podman - using Podman socket with testcontainers configuration${NC}"
else
    echo -e "${YELLOW}Warning: Neither docker nor podman detected, using default configuration${NC}"
    export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE="/var/run/docker.sock"
    export DOCKER_HOST="unix:///var/run/docker.sock"
fi

# Configure based on mode
case $MODE in
    parallel)
        PARALLEL_FLAG=""
        ADDITIONAL_FLAGS=""
        echo -e "${YELLOW}Running in parallel mode - tests may conflict with each other${NC}"
        ;;
    sequential)
        PARALLEL_FLAG="-p 1"
        ADDITIONAL_FLAGS=""
        echo -e "${GREEN}Running in sequential mode - safe but slower${NC}"
        ;;
    race-safe)
        PARALLEL_FLAG="-p 1"
        ADDITIONAL_FLAGS=""
        export DISABLE_RYUK="true"  # Better cleanup control
        export TESTCONTAINERS_RYUK_DISABLED="true"
        echo -e "${GREEN}Running in race-safe mode - optimized for race detection${NC}"
        ;;
esac

# Add race detection flag if enabled
if [ "$RACE_DETECTION" = true ]; then
    ADDITIONAL_FLAGS="$ADDITIONAL_FLAGS -race"
    echo -e "${YELLOW}Race detection enabled - tests will run slower but detect race conditions${NC}"
fi

# Add verbose flag if enabled
if [ "$VERBOSE" = true ]; then
    ADDITIONAL_FLAGS="$ADDITIONAL_FLAGS -v"
fi

echo ""
echo -e "${BLUE}Starting integration tests...${NC}"
echo ""

# Build the go test command
TEST_CMD="go test -tags=integration ./internal/kafka/ $PARALLEL_FLAG $ADDITIONAL_FLAGS"

# Show the command being run
echo -e "${YELLOW}Running: $TEST_CMD${NC}"
echo ""

# Run the tests and capture the exit code
set +e
$TEST_CMD
TEST_EXIT_CODE=$?
set -e

# Report results
echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ All integration tests passed!${NC}"
else
    echo -e "${RED}❌ Some integration tests failed (exit code: $TEST_EXIT_CODE)${NC}"
    
    # Provide suggestions based on the failure
    if [ "$MODE" = "parallel" ]; then
        echo ""
        echo -e "${YELLOW}💡 Suggestion: Try running in sequential mode to avoid container conflicts:${NC}"
        echo "   $0 -m sequential"
    fi
    
    if [ "$RACE_DETECTION" = false ]; then
        echo ""
        echo -e "${YELLOW}💡 Suggestion: Try running with race detection to identify race conditions:${NC}"
        echo "   $0 -m race-safe -r"
    fi
fi

# Clean up if requested
if [ "$CLEANUP" = true ]; then
    echo ""
    echo -e "${YELLOW}Cleaning up test containers...${NC}"
    
    # Give containers a moment to shut down gracefully
    sleep 2
    
    # Detect container runtime (docker or podman)
    CONTAINER_CMD=""
    if command -v docker >/dev/null 2>&1; then
        CONTAINER_CMD="docker"
    elif command -v podman >/dev/null 2>&1; then
        CONTAINER_CMD="podman"
    fi
    
    if [ -n "$CONTAINER_CMD" ]; then
        echo -e "${BLUE}Using $CONTAINER_CMD for cleanup${NC}"
        # Remove any remaining Kafka test containers
        $CONTAINER_CMD ps -a --filter "ancestor=confluentinc/cp-kafka:7.4.0" --format "{{.ID}}" | xargs -r $CONTAINER_CMD rm -f 2>/dev/null || true
        echo -e "${GREEN}Container cleanup completed${NC}"
    else
        echo -e "${YELLOW}Warning: Neither docker nor podman command found, skipping manual cleanup${NC}"
        echo -e "${YELLOW}Testcontainers should have handled cleanup automatically${NC}"
    fi
    
    echo -e "${GREEN}Cleanup completed${NC}"
fi

exit $TEST_EXIT_CODE 