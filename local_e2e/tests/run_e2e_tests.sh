#!/bin/bash

# Comprehensive E2E Test Suite for Alert Engine
# Uses JSON configuration for readable test definitions
# Tests real alert functionality including Slack notifications

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
SETUP_DIR="$SCRIPT_DIR/../setup"

# Load environment variables
if [[ -f "$SETUP_DIR/.env" ]]; then
    source "$SETUP_DIR/.env"
    # Ensure critical variables are exported for subprocesses
    export SLACK_WEBHOOK_URL
    export REDIS_ADDRESS
    export REDIS_PASSWORD
    export REDIS_PORT
    export KAFKA_BROKERS
else
    # Set default environment variables if .env file is missing (excluding sensitive data)
    export REDIS_ADDRESS="127.0.0.1:6379"
    export REDIS_PASSWORD=""
    export REDIS_PORT="6379"
    export KAFKA_BROKERS="localhost:9094"
    # SLACK_WEBHOOK_URL must be provided via .env file for security
fi

# Test configuration
BASE_URL="http://localhost:8080"
TIMEOUT=30
CONFIG_FILE="$SCRIPT_DIR/comprehensive_e2e_test_config.json"
LOG_FILE="$SCRIPT_DIR/e2e_test_results.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test state  
stored_ids_file="/tmp/e2e_stored_ids.tmp"
log_forwarder_pid=""

# Helper functions for stored IDs
store_id() {
    local test_name="$1"
    local id="$2"
    echo "${test_name}=${id}" >> "$stored_ids_file"
}

get_stored_id() {
    local test_name="$1"
    if [[ -f "$stored_ids_file" ]]; then
        grep "^${test_name}=" "$stored_ids_file" | cut -d'=' -f2- | tail -1
    fi
}

# ============================================================================
# Utility Functions
# ============================================================================

log_and_display() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $message" >> "$LOG_FILE"
    echo "[$timestamp] $message" >&2  # Send to stderr to avoid contaminating stdout
}

display_colored() {
    local color="$1"
    local message="$2"
    echo -e "${color}${message}${NC}"
    log_and_display "$message"
}

success() { display_colored "$GREEN" "✅ $1"; }
error() { display_colored "$RED" "❌ $1"; }
info() { display_colored "$BLUE" "ℹ️  $1"; }
warning() { display_colored "$YELLOW" "⚠️  $1"; }

# ============================================================================
# API Functions
# ============================================================================

make_api_request() {
    local method="$1"
    local endpoint="$2"
    local body="$3"
    local expected_status="$4"
    
    local full_url="${BASE_URL}${endpoint}"
    
    log_and_display "API Request: $method $endpoint"
    if [[ -n "$body" && "$body" != "null" ]]; then
        log_and_display "Request body: $body"
    fi
    
    local response
    if [[ -n "$body" && "$body" != "null" ]]; then
        response=$(curl -X "$method" -s -w "%{http_code}" --max-time "$TIMEOUT" \
                   -H "Content-Type: application/json" \
                   -d "$body" \
                   "$full_url")
    else
        response=$(curl -X "$method" -s -w "%{http_code}" --max-time "$TIMEOUT" \
                   "$full_url")
    fi
    
    local http_code="${response: -3}"
    local response_body="${response%???}"
    
    log_and_display "Expected status: $expected_status, Got: $http_code"
    
    if [[ "$http_code" == "$expected_status" ]]; then
        log_and_display "Response body: $response_body"
        echo "$response_body"
        return 0
    else
        log_and_display "Error response: $response_body"
        return 1
    fi
}

# ============================================================================
# JSON Test Processing Functions  
# ============================================================================

substitute_variables() {
    local json_body="$1"
    
    # Check if SLACK_WEBHOOK_URL is set, reload if necessary
    if [[ -z "$SLACK_WEBHOOK_URL" ]]; then
        # Load environment variables silently to avoid contamination
        if [[ -f "$SETUP_DIR/.env" ]]; then
            source "$SETUP_DIR/.env"
            export SLACK_WEBHOOK_URL REDIS_ADDRESS REDIS_PASSWORD REDIS_PORT KAFKA_BROKERS
        fi
    fi
    
    if [[ -n "$SLACK_WEBHOOK_URL" ]]; then
        # Use python for safe substitution to handle special characters
        # All output is carefully controlled to avoid contamination
        python3 -c "
import sys
import os
import json

content = sys.stdin.read()
slack_url = os.environ.get('SLACK_WEBHOOK_URL', '')

if not slack_url:
    sys.exit(1)

# Replace the placeholder with the actual URL
result = content.replace('{SLACK_WEBHOOK_URL}', slack_url)

# Validate that the result is valid JSON
try:
    json.loads(result)
    print(result, end='')
except json.JSONDecodeError:
    sys.exit(1)
" <<< "$json_body" 2>/dev/null
    else
        echo "$json_body"
    fi
}

validate_response() {
    local response="$1"
    local expected_response="$2"
    local expected_contains="$3"
    local expected_type="$4"
    local validation="$5"
    
    # Check expected_response (exact match)
    if [[ -n "$expected_response" && "$expected_response" != "null" ]]; then
        local expected_json=$(echo "$expected_response" | jq -c '.')
        local actual_json=$(echo "$response" | jq -c '.')
        if [[ "$expected_json" == "$actual_json" ]]; then
            return 0
        else
            log_and_display "Expected response mismatch. Expected: $expected_json, Got: $actual_json"
            return 1
        fi
    fi
    
    # Check expected_response_contains (partial match) - FIXED VERSION
    if [[ -n "$expected_contains" && "$expected_contains" != "null" ]]; then
        # Simple validation: check if expected keys and values exist in response
        local validation_result=$(echo "$response" | jq --argjson expected "$expected_contains" '
            def contains_all($expected; $actual):
                $expected | to_entries | all(.key as $k | .value as $v | 
                    ($actual | has($k)) and 
                    (if ($v | type) == "object" then 
                        contains_all($v; $actual[$k]) 
                     else 
                        $actual[$k] == $v 
                     end)
                );
            contains_all($expected; .)
        ' 2>/dev/null)
        
        if [[ "$validation_result" != "true" ]]; then
            log_and_display "Response validation failed for expected_response_contains."
            log_and_display "Expected structure: $expected_contains"
            log_and_display "Actual response: $response"
            return 1
        fi
        log_and_display "Response validation passed for expected_response_contains"
    fi
    
    # Check expected type for arrays/objects
    if [[ -n "$expected_type" ]]; then
        local check_path="."
        # For wrapped responses, check the data field
        if echo "$response" | jq -e '.data' >/dev/null 2>&1; then
            check_path=".data"
        fi
        
        if [[ "$expected_type" == "array" ]]; then
            if ! echo "$response" | jq -e "${check_path} | type == \"array\"" >/dev/null 2>&1; then
                local actual_type=$(echo "$response" | jq -r "${check_path} | type // \"null\"")
                log_and_display "Expected array type but got: $actual_type"
                log_and_display "Response: $response"
                return 1
            fi
        fi
    fi
    
    # Check validation rules
    if [[ -n "$validation" ]]; then
        local check_path=".data"
        
        # Handle data_array_length validations - FIXED VERSION
        if [[ "$validation" =~ ^data_array_length[[:space:]]*\>\=[[:space:]]*([0-9]+)$ ]]; then
            local expected_length="${BASH_REMATCH[1]}"
            
            # Check if data field exists and is an array
            local data_type=$(echo "$response" | jq -r '.data | type // "null"')
            if [[ "$data_type" != "array" ]]; then
                log_and_display "Validation failed: data field should be an array, got: $data_type"
                log_and_display "Response: $response"
                return 1
            fi
            
            local actual_length=$(echo "$response" | jq -r '.data | length')
            if [[ "$actual_length" -ge "$expected_length" ]]; then
                log_and_display "Validation passed: data array length ($actual_length) >= $expected_length"
                return 0
            else
                log_and_display "Validation failed: data array length ($actual_length) should be >= $expected_length"
                log_and_display "Response: $response"
                return 1
            fi
        fi
        
        # Handle data_total_logs validations - NEW VALIDATION FOR LOG PROCESSING
        if [[ "$validation" =~ ^data_total_logs[[:space:]]*\>\=[[:space:]]*([0-9]+)$ ]]; then
            local expected_count="${BASH_REMATCH[1]}"
            
            local actual_count=$(echo "$response" | jq -r '.data.total_logs // 0')
            if [[ "$actual_count" -ge "$expected_count" ]]; then
                log_and_display "Validation passed: total_logs ($actual_count) >= $expected_count"
                return 0
            else
                log_and_display "Validation failed: total_logs ($actual_count) should be >= $expected_count"
                log_and_display "This indicates Kafka consumer is not processing logs!"
                log_and_display "Response: $response"
                return 1
            fi
        fi
        
        # Legacy validation patterns
        case "$validation" in
            "length >= 1")
                if ! echo "$response" | jq -e "${check_path} | length >= 1" >/dev/null 2>&1; then
                    log_and_display "Validation failed: length should be >= 1"
                    return 1
                fi
                ;;
            "length >= 2")
                if ! echo "$response" | jq -e "${check_path} | length >= 2" >/dev/null 2>&1; then
                    log_and_display "Validation failed: length should be >= 2"
                    return 1
                fi
                ;;
            "success_and_healthy")
                local success=$(echo "$response" | jq -r '.success // false')
                local status=$(echo "$response" | jq -r '.data.status // "unknown"')
                if [[ "$success" == "true" && "$status" == "healthy" ]]; then
                    log_and_display "Health check validation passed: success=$success, status=$status"
                    return 0
                else
                    log_and_display "Validation failed: response should have success=true and data.status=healthy"
                    log_and_display "Got: success=$success, status=$status"
                    log_and_display "Response: $response"
                    return 1
                fi
                ;;
        esac
    fi
    
    return 0
}

run_api_test() {
    local test_name="$1"
    local description="$2"
    local endpoint="$3"
    local method="$4"
    local body="$5"
    local expected_status="$6"
    local expected_response="$7"
    local expected_contains="$8"
    local expected_type="$9"
    local validation="${10}"
    local store_field="${11}"
    local wait_timeout="${12:-0}"
    
    info "Running test: $test_name - $description"
    
         # Substitute variables in body
     if [[ -n "$body" && "$body" != "null" ]]; then
         log_and_display "Processing JSON body for variable substitution..."
         body=$(substitute_variables "$body")
         log_and_display "JSON substitution completed"
     fi
    
    # Handle wait timeout for alert generation
    if [[ "$wait_timeout" -gt 0 ]]; then
        info "Waiting up to ${wait_timeout}s for alerts to be generated..."
        local attempts=0
        local max_attempts=$((wait_timeout / 10))
        
        while [[ $attempts -lt $max_attempts ]]; do
                         if response=$(make_api_request "$method" "$endpoint" "$body" "$expected_status" 2>/dev/null); then
                 if validate_response "$response" "$expected_response" "$expected_contains" "$expected_type" "$validation"; then
                     success "Test '$test_name' passed (attempt $((attempts + 1)))"
                     
                     # Store response field if requested - FIXED VERSION
                     if [[ -n "$store_field" && "$store_field" != "null" ]]; then
                         # Handle nested field paths like "data.id"
                         local id
                         if [[ "$store_field" == "data.id" ]]; then
                             id=$(echo "$response" | jq -r '.data.id // empty')
                         else
                             id=$(echo "$response" | jq -r ".${store_field} // empty")
                         fi
                         
                         if [[ -n "$id" && "$id" != "null" ]]; then
                             store_id "$test_name" "$id"
                             info "Stored ${store_field}: $id"
                         else
                             warning "Could not extract field ${store_field} from response"
                             log_and_display "Response for debugging: $response"
                         fi
                     fi
                    
                    return 0
                fi
            fi
            
            ((attempts++))
            sleep 10
        done
        
        error "Test '$test_name' failed after ${wait_timeout}s timeout"
        return 1
    fi
    
         # Regular API test
     if response=$(make_api_request "$method" "$endpoint" "$body" "$expected_status"); then
         if validate_response "$response" "$expected_response" "$expected_contains" "$expected_type" "$validation"; then
             success "Test '$test_name' passed"
             
             # Store response field if requested - FIXED VERSION
             if [[ -n "$store_field" && "$store_field" != "null" ]]; then
                 # Handle nested field paths like "data.id"
                 local id
                 if [[ "$store_field" == "data.id" ]]; then
                     id=$(echo "$response" | jq -r '.data.id // empty')
                 else
                     id=$(echo "$response" | jq -r ".${store_field} // empty")
                 fi
                 
                 if [[ -n "$id" && "$id" != "null" ]]; then
                     store_id "$test_name" "$id"
                     info "Stored ${store_field}: $id"
                 else
                     warning "Could not extract field ${store_field} from response"
                     log_and_display "Response for debugging: $response"
                 fi
             fi
            
            return 0
        else
            error "Test '$test_name' failed - response validation failed"
            return 1
        fi
    else
        error "Test '$test_name' failed - API request failed"
        return 1
    fi
}

# ============================================================================
# Action Functions
# ============================================================================

check_consumer_health() {
    info "Checking alert engine consumer health..."
    
    # Get current log stats
    local current_stats=$(make_api_request "GET" "/api/v1/system/logs/stats" "" "200" 2>/dev/null)
    if [[ $? -ne 0 ]]; then
        warning "Could not fetch log stats - alert engine might be down"
        return 1
    fi
    
    local last_updated=$(echo "$current_stats" | jq -r '.data.last_updated // empty')
    if [[ -n "$last_updated" ]]; then
        # Check if last update was more than 5 minutes ago
        local current_time=$(date -u +%s)
        local last_update_time=$(date -d "$last_updated" +%s 2>/dev/null || echo "0")
        local time_diff=$((current_time - last_update_time))
        
        if [[ $time_diff -gt 300 ]]; then
            warning "Consumer appears stale (last update: $last_updated). Logs may not be processing."
            info "This is expected if no recent logs have been sent."
        else
            info "Consumer health check passed (last update: $last_updated)"
        fi
    fi
    
    return 0
}

ensure_consumer_working() {
    info "Ensuring Kafka consumer is working properly..."
    
    # First check current state
    local initial_stats=$(make_api_request "GET" "/api/v1/system/logs/stats" "" "200" 2>/dev/null)
    local initial_count=$(echo "$initial_stats" | jq -r '.data.total_logs // 0')
    local initial_timestamp=$(echo "$initial_stats" | jq -r '.data.last_updated // ""')
    
    info "Initial log count: $initial_count, last updated: $initial_timestamp"
    
    # Send a few test logs and check if they're processed
    info "Sending test logs to verify consumer is working..."
    cd "$SETUP_DIR"
    if [[ -f "venv/bin/activate" ]]; then
        source venv/bin/activate
        timeout 10 python3 mock_log_forwarder.py --mode test >/dev/null 2>&1 &
        local forwarder_pid=$!
        sleep 10
        kill $forwarder_pid 2>/dev/null || true
    else
        warning "Python virtual environment not found, skipping consumer test"
        return 0
    fi
    
    # Wait a moment for processing
    sleep 5
    
    # Check if logs were processed
    local after_stats=$(make_api_request "GET" "/api/v1/system/logs/stats" "" "200" 2>/dev/null)
    local after_count=$(echo "$after_stats" | jq -r '.data.total_logs // 0')
    local after_timestamp=$(echo "$after_stats" | jq -r '.data.last_updated // ""')
    
    info "After test log count: $after_count, last updated: $after_timestamp"
    
    # Check if consumer processed new logs (only check log count, not timestamp)
    if [[ "$after_count" -gt "$initial_count" ]]; then
        success "Kafka consumer is working correctly"
        return 0
    else
        error "Kafka consumer is not processing logs!"
        info "Attempting to restart alert engine..."
        
        # Try to restart the alert engine
        cd "$PROJECT_ROOT"
        pkill -f "alert-engine" 2>/dev/null || true
        sleep 5
        ./start-local.sh >/dev/null 2>&1 &
        sleep 15
        
        # Verify it's working now
        if curl -s --max-time 5 "$BASE_URL/api/v1/health" >/dev/null; then
            success "Alert engine restarted successfully"
            return 0
        else
            error "Failed to restart alert engine"
            return 1
        fi
    fi
}

start_log_forwarder() {
    local duration="$1"
    
    info "Starting mock log forwarder for ${duration}s..."
    
    # Check if Python environment exists
    if [[ ! -f "$SETUP_DIR/venv/bin/activate" ]]; then
        error "Python virtual environment not found. Run setup first."
        return 1
    fi
    
    # Ensure Kafka consumer is working before starting
    if ! ensure_consumer_working; then
        error "Cannot start log forwarder - Kafka consumer is not working"
        return 1
    fi
    
    # Start log forwarder in background
    cd "$SETUP_DIR"
    source venv/bin/activate
    
    # Run mock log forwarder in test mode (completes automatically)
    python3 mock_log_forwarder.py --mode test >/dev/null 2>&1 &
    log_forwarder_pid=$!
    
    info "Mock log forwarder started (PID: $log_forwarder_pid)"
    
    # Wait for the process to complete (test mode runs for ~50 seconds)
    wait $log_forwarder_pid
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        info "Mock log forwarder completed successfully"
    else
        warning "Mock log forwarder exited with code: $exit_code"
    fi
    return 0
}

verify_slack_config() {
    info "Verifying Slack webhook configuration..."
    
    # Ensure SLACK_WEBHOOK_URL is loaded
    if [[ -z "$SLACK_WEBHOOK_URL" ]]; then
        log_and_display "SLACK_WEBHOOK_URL not set, attempting to load..."
        
        if [[ -f "$SETUP_DIR/.env" ]]; then
            source "$SETUP_DIR/.env"
            export SLACK_WEBHOOK_URL  # Ensure it's exported
        else
            log_and_display "ERROR: .env file not found - cannot load SLACK_WEBHOOK_URL"
        fi
    fi
    
    if [[ -z "$SLACK_WEBHOOK_URL" ]]; then
        error "SLACK_WEBHOOK_URL not configured after all loading attempts"
        return 1
    fi
    
    success "Slack webhook URL configured: ${SLACK_WEBHOOK_URL:0:50}..."
    info "Target channel: #test-mp-channel"
    info "Note: Check your Slack channel for actual notifications"
    
    return 0
}

# ============================================================================
# Main Test Execution
# ============================================================================

run_test_scenario() {
    local scenario="$1"
    
    local name=$(echo "$scenario" | jq -r '.name')
    local description=$(echo "$scenario" | jq -r '.description')
    local type=$(echo "$scenario" | jq -r '.type // "api"')
    
    case "$type" in
        "action")
            local action=$(echo "$scenario" | jq -r '.action')
            local duration=$(echo "$scenario" | jq -r '.duration // 30')
            
            case "$action" in
                "start_log_forwarder")
                    if start_log_forwarder "$duration"; then
                        success "Action '$name' completed"
                        return 0
                    else
                        error "Action '$name' failed"
                        return 1
                    fi
                    ;;
                "sleep")
                    info "Sleeping for ${duration}s..."
                    sleep "$duration"
                    success "Action '$name' completed"
                    return 0
                    ;;
                *)
                    error "Unknown action: $action"
                    return 1
                    ;;
            esac
            ;;
        "validation") 
            local action=$(echo "$scenario" | jq -r '.action')
            
            case "$action" in
                "verify_slack_config")
                    if verify_slack_config; then
                        success "Validation '$name' passed"
                        return 0
                    else
                        error "Validation '$name' failed"
                        return 1
                    fi
                    ;;
                *)
                    error "Unknown validation: $action"
                    return 1
                    ;;
            esac
            ;;
                 "api"|*)
             local endpoint=$(echo "$scenario" | jq -r '.endpoint')
             local method=$(echo "$scenario" | jq -r '.method')
             local body=$(echo "$scenario" | jq -c '.body // null')
             local expected_status=$(echo "$scenario" | jq -r '.expected_status')
             local expected_response=$(echo "$scenario" | jq -c '.expected_response // null')
             local expected_contains=$(echo "$scenario" | jq -c '.expected_response_contains // null')
             local expected_type=$(echo "$scenario" | jq -r '.expected_response_type // null')
             local validation=$(echo "$scenario" | jq -r '.validation // null')
             local store_field=$(echo "$scenario" | jq -r '.store_response_field // null')
             local wait_timeout=$(echo "$scenario" | jq -r '.wait_timeout // 0')
             
             run_api_test "$name" "$description" "$endpoint" "$method" "$body" "$expected_status" "$expected_response" "$expected_contains" "$expected_type" "$validation" "$store_field" "$wait_timeout"
             return $?
            ;;
    esac
}

cleanup_test_rules() {
     info "Cleaning up test rules..."
     
     if [[ -f "$stored_ids_file" ]]; then
         while IFS='=' read -r test_name rule_id; do
             if [[ "$test_name" == *"rule"* && -n "$rule_id" ]]; then
                 info "Deleting rule: $rule_id"
                 make_api_request "DELETE" "/api/v1/rules/$rule_id" "" "200" >/dev/null 2>&1 || true
             fi
         done < "$stored_ids_file"
         rm -f "$stored_ids_file"
     fi
     
     success "Test rules cleanup completed"
}

main() {
    echo "============================================================"
    echo " Comprehensive E2E Test Suite for Alert Engine"
    echo " Using JSON Configuration Format"
    echo "============================================================"
    
         # Initialize log file and stored IDs
     echo "Test started at $(date)" > "$LOG_FILE"
     > "$stored_ids_file"
    
    info "Configuration:"
    info "  Base URL: $BASE_URL"
    info "  Config file: $CONFIG_FILE"
    info "  Log file: $LOG_FILE"
    
    # Validate critical environment variables
    if [[ -z "$SLACK_WEBHOOK_URL" ]]; then
        error "SLACK_WEBHOOK_URL is not configured!"
        error "Please ensure your .env file contains: SLACK_WEBHOOK_URL=your_webhook_url"
        error "Location: $SETUP_DIR/.env"
        exit 1
    fi
    
    info "  Slack webhook: ${SLACK_WEBHOOK_URL:0:50}..."
    
    # Check if config file exists
    if [[ ! -f "$CONFIG_FILE" ]]; then
        error "Configuration file not found: $CONFIG_FILE"
        exit 1
    fi
    
    # Pre-flight check
    info "Performing pre-flight checks..."
    if ! curl -s --max-time 5 "$BASE_URL/api/v1/health" >/dev/null; then
        error "Alert engine is not responding at $BASE_URL"
        error "Please start the alert engine first"
        exit 1
    fi
    success "Alert engine is responding"
    
    # Load and run test scenarios
    local scenarios=$(jq -c '.test_scenarios[]' "$CONFIG_FILE")
    local total_tests=$(echo "$scenarios" | wc -l | tr -d ' ')
    local passed_tests=0
    local failed_tests=0
    
    info "Running $total_tests test scenarios..."
    echo ""
    
    while IFS= read -r scenario; do
        local test_name=$(echo "$scenario" | jq -r '.name')
        
        if run_test_scenario "$scenario"; then
            ((passed_tests++))
        else
            ((failed_tests++))
            
            # FAIL FAST: Critical infrastructure tests that should stop execution immediately
            case "$test_name" in
                "health_check")
                    error "💥 CRITICAL FAILURE: Alert engine is not responding!"
                    error "🛑 Stopping test execution - alert engine is down"
                    error ""
                    error "🔧 DEBUG STEPS:"
                    error "   1. Check if alert engine process is running: ps aux | grep 'go run'"
                    error "   2. Review alert engine startup logs"
                    error "   3. Verify configuration: CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml"
                    error "   4. Check port availability: lsof -i:8080"
                    error ""
                    cleanup_test_rules
                    exit 1
                    ;;
                "check_log_processing_stats_after")
                    error "💥 CRITICAL INFRASTRUCTURE FAILURE: Kafka consumer is not processing logs!"
                    error "🛑 Stopping test execution - no point in testing alerts if logs aren't being consumed"
                    error ""
                    error "🔧 DEBUG STEPS:"
                    error "   1. Check Kafka broker status: podman ps | grep kafka"
                    error "   2. Verify topic exists: podman exec alert-engine-kafka-e2e kafka-topics --bootstrap-server localhost:9094 --list"
                    error "   3. Check consumer group: podman exec alert-engine-kafka-e2e kafka-consumer-groups --bootstrap-server localhost:9094 --list"
                    error "   4. Review alert engine logs for Kafka connection errors"
                    error "   5. Check if mock log forwarder sent messages successfully"
                    error ""
                    error "📁 Test logs: $LOG_FILE"
                    echo ""
                    cleanup_test_rules
                    exit 1
                    ;;
                *)
                    # Non-critical test failure - continue with remaining tests
                    warning "Test '$test_name' failed but continuing with remaining tests..."
                    ;;
            esac
        fi
        echo ""
    done <<< "$scenarios"
    
    # Cleanup
    cleanup_test_rules
    
    # Final results
    echo "============================================================"
    echo " Comprehensive Test Results"
    echo "============================================================"
    info "📊 Final Results:"
    info "   Total Tests: $total_tests"
    info "   Passed: $passed_tests"
    info "   Failed: $failed_tests"
    local success_rate=$(( (passed_tests * 100) / total_tests ))
    info "   Success Rate: $success_rate%"
    
    if [[ $failed_tests -eq 0 ]]; then
        success "🎉 All comprehensive tests passed!"
        info "🔔 Check your Slack channel (#test-mp-channel) for alert notifications!"
    else
        error "Some tests failed. Check $LOG_FILE for details."
    fi
    
    # Cleanup background processes
    if [[ -n "$log_forwarder_pid" ]] && kill -0 "$log_forwarder_pid" 2>/dev/null; then
        info "Stopping mock log forwarder..."
        kill "$log_forwarder_pid" || true
        success "Mock log forwarder stopped"
    fi
    
    echo "============================================================"
    
    exit $failed_tests
}

# Script entry point
main "$@" 