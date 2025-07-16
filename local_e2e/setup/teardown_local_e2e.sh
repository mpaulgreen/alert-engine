#!/bin/bash

# Local E2E Environment Teardown Script for Alert Engine
# This script cleans up the complete local testing environment

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose-local-e2e.yml"
LOG_FILE="${SCRIPT_DIR}/teardown.log"

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

# Stop Podman containers
stop_containers() {
    log "Stopping Podman containers..."
    
    if podman compose -f "${COMPOSE_FILE}" ps -q | grep -q .; then
        # Stop all services including debug profile
        podman compose -f "${COMPOSE_FILE}" --profile debug down --remove-orphans
        success "Containers stopped"
    else
        warning "No running containers found"
    fi
}

# Remove Podman volumes
remove_volumes() {
    local remove_volumes=${1:-false}
    
    if [ "$remove_volumes" = "true" ]; then
        log "Removing Podman volumes..."
        
        # Remove volumes
        podman volume rm alert-engine-kafka-e2e-data 2>/dev/null || warning "Kafka volume not found"
        podman volume rm alert-engine-redis-e2e-data 2>/dev/null || warning "Redis volume not found"
        
        success "Volumes removed"
    else
        log "Keeping Podman volumes (use --remove-volumes to delete)"
    fi
}

# Remove Podman network
remove_network() {
    log "Removing Podman network..."
    
    if podman network ls | grep -q "alert-engine-e2e"; then
        podman network rm alert-engine-e2e
        success "Network removed"
    else
        warning "Network not found"
    fi
}

# Clean up Python environment
cleanup_python() {
    local remove_venv=${1:-false}
    
    if [ "$remove_venv" = "true" ] && [ -d "${SCRIPT_DIR}/venv" ]; then
        log "Removing Python virtual environment..."
        rm -rf "${SCRIPT_DIR}/venv"
        success "Python virtual environment removed"
    else
        log "Keeping Python virtual environment (use --remove-venv to delete)"
    fi
}

# Kill any running processes
kill_processes() {
    log "Checking for running alert engine processes..."
    
    # Find and kill alert engine processes
    local alert_pids=$(pgrep -f "alert-engine" || true)
    if [ -n "$alert_pids" ]; then
        warning "Found running alert engine processes: $alert_pids"
        echo "$alert_pids" | xargs kill -TERM || true
        sleep 2
        
        # Force kill if still running
        local remaining_pids=$(pgrep -f "alert-engine" || true)
        if [ -n "$remaining_pids" ]; then
            warning "Force killing remaining processes: $remaining_pids"
            echo "$remaining_pids" | xargs kill -KILL || true
        fi
        success "Alert engine processes terminated"
    else
        log "No alert engine processes found"
    fi
    
    # Find and kill mock log forwarder processes
    local forwarder_pids=$(pgrep -f "mock_log_forwarder" || true)
    if [ -n "$forwarder_pids" ]; then
        warning "Found running mock log forwarder processes: $forwarder_pids"
        echo "$forwarder_pids" | xargs kill -TERM || true
        sleep 2
        
        # Force kill if still running
        local remaining_pids=$(pgrep -f "mock_log_forwarder" || true)
        if [ -n "$remaining_pids" ]; then
            warning "Force killing remaining processes: $remaining_pids"
            echo "$remaining_pids" | xargs kill -KILL || true
        fi
        success "Mock log forwarder processes terminated"
    else
        log "No mock log forwarder processes found"
    fi
}

# Clean up log files
cleanup_logs() {
    local remove_logs=${1:-false}
    
    if [ "$remove_logs" = "true" ]; then
        log "Cleaning up log files..."
        
        # Remove setup/teardown logs
        rm -f "${SCRIPT_DIR}/setup.log"
        rm -f "${SCRIPT_DIR}/teardown.log"
        
        # Remove any alert engine logs
        find "${SCRIPT_DIR}" -name "*.log" -type f -delete 2>/dev/null || true
        
        success "Log files cleaned up"
    else
        log "Keeping log files (use --remove-logs to delete)"
    fi
}

# Display cleanup status
display_status() {
    success "Local E2E environment teardown completed!"
    
    echo ""
    echo "=== Cleanup Status ==="
    
    # Check for remaining containers
    local remaining_containers=$(podman ps -q --filter "name=alert-engine-" || true)
    if [ -n "$remaining_containers" ]; then
        warning "Some containers are still running:"
        podman ps --filter "name=alert-engine-"
    else
        success "No alert engine containers running"
    fi
    
    # Check for remaining volumes
    local remaining_volumes=$(podman volume ls -q --filter "name=alert-engine-" || true)
    if [ -n "$remaining_volumes" ]; then
        log "Remaining volumes:"
        podman volume ls --filter "name=alert-engine-"
    else
        success "No alert engine volumes remaining"
    fi
    
    # Check for remaining networks
    if podman network ls | grep -q "alert-engine-e2e"; then
        warning "Network still exists: alert-engine-e2e"
    else
        success "Network cleaned up"
    fi
    
    echo ""
    echo "=== Next Steps ==="
    echo "To completely remove all traces:"
    echo "  $0 --remove-all"
    echo ""
    echo "To start fresh:"
    echo "  ./setup_local_e2e.sh"
}

# Show help
show_help() {
    echo "Local E2E Environment Teardown Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --remove-volumes    Remove Podman volumes (persistent data)"
    echo "  --remove-venv       Remove Python virtual environment"
    echo "  --remove-logs       Remove log files"
    echo "  --remove-all        Remove everything (volumes, venv, logs)"
    echo "  --kill-processes    Kill running alert engine and forwarder processes"
    echo "  --help              Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                  # Basic cleanup (keep volumes and venv)"
    echo "  $0 --remove-all     # Complete cleanup"
    echo "  $0 --kill-processes # Stop all related processes"
}

# Parse command line arguments
parse_args() {
    REMOVE_VOLUMES=false
    REMOVE_VENV=false
    REMOVE_LOGS=false
    KILL_PROCESSES=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --remove-volumes)
                REMOVE_VOLUMES=true
                shift
                ;;
            --remove-venv)
                REMOVE_VENV=true
                shift
                ;;
            --remove-logs)
                REMOVE_LOGS=true
                shift
                ;;
            --remove-all)
                REMOVE_VOLUMES=true
                REMOVE_VENV=true
                REMOVE_LOGS=true
                KILL_PROCESSES=true
                shift
                ;;
            --kill-processes)
                KILL_PROCESSES=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Main execution
main() {
    # Clear log file before any logging
    > "${LOG_FILE}"
    
    log "Starting Local E2E Environment Teardown for Alert Engine"
    log "Script directory: ${SCRIPT_DIR}"
    
    if [ "$KILL_PROCESSES" = "true" ]; then
        kill_processes
    fi
    
    stop_containers
    remove_network
    remove_volumes "$REMOVE_VOLUMES"
    cleanup_python "$REMOVE_VENV"
    
    if [ "$REMOVE_LOGS" = "true" ]; then
        cleanup_logs "$REMOVE_LOGS"
    fi
    
    display_status
    
    success "Teardown completed successfully!"
}

# Handle interrupts
trap 'error "Teardown interrupted by user"; exit 1' INT TERM

# Parse arguments and run main function
parse_args "$@"
main 