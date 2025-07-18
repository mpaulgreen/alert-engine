#!/bin/bash

# OpenShift Infrastructure Validation Script for Alert Engine
# This script validates the setup of Kafka, Redis Cluster, and OpenShift Logging

set -e  # Exit on error
set -o pipefail  # Exit on pipe failure

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
KAFKA_NAMESPACE="amq-streams-kafka"
REDIS_NAMESPACE="redis-cluster"
LOGGING_NAMESPACE="openshift-logging"
ALERT_ENGINE_NAMESPACE="alert-engine"

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

log_step() {
    echo -e "\n${BLUE}======================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}======================================${NC}"
}

# Function to check if resource exists
resource_exists() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3
    
    if [ -n "$namespace" ]; then
        oc get "$resource_type" "$resource_name" -n "$namespace" &>/dev/null
    else
        oc get "$resource_type" "$resource_name" &>/dev/null
    fi
}

# Function to get resource status
get_resource_status() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3
    local condition=$4
    
    if [ -n "$condition" ]; then
        oc get "$resource_type" "$resource_name" -n "$namespace" -o jsonpath="{.status.conditions[?(@.type=='$condition')].status}" 2>/dev/null || echo "Unknown"
    else
        oc get "$resource_type" "$resource_name" -n "$namespace" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown"
    fi
}

# Prerequisites check
check_prerequisites() {
    log_step "CHECKING PREREQUISITES"
    
    # Check if oc CLI is available
    if ! command -v oc &> /dev/null; then
        log_error "oc CLI tool is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we're logged into OpenShift
    if ! oc whoami &>/dev/null; then
        log_error "Not logged into OpenShift cluster. Please login first."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Validate Kafka infrastructure
validate_kafka() {
    log_step "VALIDATING KAFKA INFRASTRUCTURE"
    
    local kafka_status=0
    
    # Check namespace
    if resource_exists namespace "$KAFKA_NAMESPACE"; then
        log_success "✅ Kafka namespace ($KAFKA_NAMESPACE) exists"
    else
        log_error "❌ Kafka namespace ($KAFKA_NAMESPACE) not found"
        kafka_status=1
    fi
    
    # Check AMQ Streams operator
    local csv_status=$(oc get csv -n "$KAFKA_NAMESPACE" --no-headers 2>/dev/null | grep "amqstreams" | awk '{print $6}' || echo "NotFound")
    if [[ "$csv_status" == "Succeeded" ]]; then
        log_success "✅ AMQ Streams operator is ready"
    else
        log_error "❌ AMQ Streams operator status: $csv_status"
        kafka_status=1
    fi
    
    # Check Kafka cluster
    if resource_exists kafka alert-kafka-cluster "$KAFKA_NAMESPACE"; then
        local kafka_ready=$(get_resource_status kafka alert-kafka-cluster "$KAFKA_NAMESPACE" Ready)
        if [[ "$kafka_ready" == "True" ]]; then
            log_success "✅ Kafka cluster is ready"
        else
            log_error "❌ Kafka cluster not ready (status: $kafka_ready)"
            kafka_status=1
        fi
    else
        log_error "❌ Kafka cluster not found"
        kafka_status=1
    fi
    
    # Check Kafka topic
    if resource_exists kafkatopic application-logs "$KAFKA_NAMESPACE"; then
        log_success "✅ Kafka topic 'application-logs' exists"
    else
        log_error "❌ Kafka topic 'application-logs' not found"
        kafka_status=1
    fi
    
    # Check Kafka pods
    local kafka_pods_ready=$(oc get pods -n "$KAFKA_NAMESPACE" -l strimzi.io/cluster=alert-kafka-cluster --no-headers 2>/dev/null | grep -c Running || echo 0)
    local kafka_pods_total=$(oc get pods -n "$KAFKA_NAMESPACE" -l strimzi.io/cluster=alert-kafka-cluster --no-headers 2>/dev/null | wc -l || echo 0)
    
    if [[ $kafka_pods_ready -gt 0 ]] && [[ $kafka_pods_ready -eq $kafka_pods_total ]]; then
        log_success "✅ Kafka pods running ($kafka_pods_ready/$kafka_pods_total)"
    else
        log_error "❌ Kafka pods not all running ($kafka_pods_ready/$kafka_pods_total)"
        kafka_status=1
    fi
    
    # Check network policy
    if resource_exists networkpolicy kafka-network-policy "$KAFKA_NAMESPACE"; then
        log_success "✅ Kafka network policy exists"
    else
        log_warning "⚠️  Kafka network policy not found"
    fi
    
    # Test Kafka connectivity
    log_info "Testing Kafka connectivity..."
    if oc exec -n "$KAFKA_NAMESPACE" alert-kafka-cluster-kafka-0 -- timeout 5 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --max-messages 0 &>/dev/null; then
        log_success "✅ Kafka connectivity test passed"
    else
        log_warning "⚠️  Kafka connectivity test failed or topic empty"
    fi
    
    return $kafka_status
}

# Validate Redis infrastructure
validate_redis() {
    log_step "VALIDATING REDIS CLUSTER INFRASTRUCTURE"
    
    local redis_status=0
    
    # Check namespace
    if resource_exists namespace "$REDIS_NAMESPACE"; then
        log_success "✅ Redis namespace ($REDIS_NAMESPACE) exists"
    else
        log_error "❌ Redis namespace ($REDIS_NAMESPACE) not found"
        redis_status=1
    fi
    
    # Check Redis pods
    local redis_pods_running=$(oc get pods -l app=redis-cluster -n "$REDIS_NAMESPACE" --no-headers 2>/dev/null | grep -c Running || echo 0)
    if [[ $redis_pods_running -eq 6 ]]; then
        log_success "✅ Redis cluster pods running (6/6)"
    else
        log_error "❌ Redis cluster pods not all running ($redis_pods_running/6)"
        redis_status=1
    fi
    
    # Check Redis services
    if resource_exists service redis-cluster "$REDIS_NAMESPACE"; then
        log_success "✅ Redis cluster service exists"
    else
        log_error "❌ Redis cluster service not found"
        redis_status=1
    fi
    
    if resource_exists service redis-cluster-access "$REDIS_NAMESPACE"; then
        log_success "✅ Redis cluster access service exists"
    else
        log_error "❌ Redis cluster access service not found"
        redis_status=1
    fi
    
    # Check Redis ConfigMap
    if resource_exists configmap redis-cluster-config "$REDIS_NAMESPACE"; then
        log_success "✅ Redis cluster ConfigMap exists"
    else
        log_error "❌ Redis cluster ConfigMap not found"
        redis_status=1
    fi
    
    # Check network policy
    if resource_exists networkpolicy redis-cluster-network-policy "$REDIS_NAMESPACE"; then
        log_success "✅ Redis network policy exists"
    else
        log_warning "⚠️  Redis network policy not found"
    fi
    
    # Test Redis cluster connectivity
    if [[ $redis_pods_running -eq 6 ]]; then
        log_info "Testing Redis cluster connectivity..."
        local cluster_info=$(oc exec -n "$REDIS_NAMESPACE" redis-cluster-0 -- redis-cli cluster info 2>/dev/null || echo "cluster_state:fail")
        
        if [[ $cluster_info == *"cluster_state:ok"* ]]; then
            log_success "✅ Redis cluster is healthy"
        else
            log_error "❌ Redis cluster is not healthy"
            redis_status=1
        fi
        
        # Test basic operations
        if oc exec -n "$REDIS_NAMESPACE" redis-cluster-0 -- redis-cli -c set validation-test "OK" &>/dev/null; then
            local test_result=$(oc exec -n "$REDIS_NAMESPACE" redis-cluster-0 -- redis-cli -c get validation-test 2>/dev/null || echo "FAIL")
            if [[ "$test_result" == "OK" ]]; then
                log_success "✅ Redis cluster connectivity test passed"
                oc exec -n "$REDIS_NAMESPACE" redis-cluster-0 -- redis-cli -c del validation-test &>/dev/null
            else
                log_error "❌ Redis cluster connectivity test failed"
                redis_status=1
            fi
        else
            log_error "❌ Redis cluster connectivity test failed"
            redis_status=1
        fi
    fi
    
    # Check Alert Engine Redis config
    if resource_exists configmap redis-config "$ALERT_ENGINE_NAMESPACE"; then
        log_success "✅ Alert Engine Redis ConfigMap exists"
    else
        log_warning "⚠️  Alert Engine Redis ConfigMap not found"
    fi
    
    return $redis_status
}

# Validate OpenShift Logging infrastructure
validate_logging() {
    log_step "VALIDATING OPENSHIFT LOGGING INFRASTRUCTURE"
    
    local logging_status=0
    
    # Check namespace
    if resource_exists namespace "$LOGGING_NAMESPACE"; then
        log_success "✅ Logging namespace ($LOGGING_NAMESPACE) exists"
    else
        log_error "❌ Logging namespace ($LOGGING_NAMESPACE) not found"
        logging_status=1
    fi
    
    # Check OpenShift Logging operator
    local csv_status=$(oc get csv -n "$LOGGING_NAMESPACE" --no-headers 2>/dev/null | grep "cluster-logging" | awk '{print $6}' || echo "NotFound")
    if [[ "$csv_status" == "Succeeded" ]]; then
        log_success "✅ OpenShift Logging operator is ready"
    else
        log_error "❌ OpenShift Logging operator status: $csv_status"
        logging_status=1
    fi
    
    # Check service account
    if resource_exists serviceaccount log-collector "$LOGGING_NAMESPACE"; then
        log_success "✅ Log collector service account exists"
    else
        log_error "❌ Log collector service account not found"
        logging_status=1
    fi
    
    # Check ClusterLogForwarder
    if resource_exists clusterlogforwarder kafka-forwarder "$LOGGING_NAMESPACE"; then
        log_success "✅ ClusterLogForwarder exists"
        
        # Check ClusterLogForwarder status
        local clf_status=$(oc get clusterlogforwarder kafka-forwarder -n "$LOGGING_NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="observability.openshift.io/Valid")].status}' 2>/dev/null || echo "Unknown")
        if [[ "$clf_status" == "True" ]]; then
            log_success "✅ ClusterLogForwarder is valid"
        else
            log_warning "⚠️  ClusterLogForwarder status: $clf_status"
        fi
    else
        log_error "❌ ClusterLogForwarder not found"
        logging_status=1
    fi
    
    # Check for Vector pods (log forwarder implementation)
    local vector_pods=$(oc get pods -A -l app.kubernetes.io/component=collector --no-headers 2>/dev/null | grep -c Running || echo 0)
    if [[ $vector_pods -gt 0 ]]; then
        log_success "✅ Vector log collector pods running ($vector_pods)"
    else
        log_warning "⚠️  Vector log collector pods not found or not running"
    fi
    
    return $logging_status
}

# Validate Alert Engine namespace and resources
validate_alert_engine() {
    log_step "VALIDATING ALERT ENGINE NAMESPACE AND RESOURCES"
    
    local alert_engine_status=0
    
    # Check namespace
    if resource_exists namespace "$ALERT_ENGINE_NAMESPACE"; then
        log_success "✅ Alert Engine namespace ($ALERT_ENGINE_NAMESPACE) exists"
    else
        log_error "❌ Alert Engine namespace ($ALERT_ENGINE_NAMESPACE) not found"
        alert_engine_status=1
    fi
    
    # Check service account
    if resource_exists serviceaccount alert-engine-sa "$ALERT_ENGINE_NAMESPACE"; then
        log_success "✅ Alert Engine service account exists"
    else
        log_error "❌ Alert Engine service account not found"
        alert_engine_status=1
    fi
    
    # Check ClusterRole and ClusterRoleBinding
    if resource_exists clusterrole alert-engine-role; then
        log_success "✅ Alert Engine ClusterRole exists"
    else
        log_error "❌ Alert Engine ClusterRole not found"
        alert_engine_status=1
    fi
    
    if resource_exists clusterrolebinding alert-engine-binding; then
        log_success "✅ Alert Engine ClusterRoleBinding exists"
    else
        log_error "❌ Alert Engine ClusterRoleBinding not found"
        alert_engine_status=1
    fi
    
    # Check test log generator
    if resource_exists deployment continuous-log-generator "$ALERT_ENGINE_NAMESPACE"; then
        local log_gen_pods=$(oc get pods -l app=continuous-log-generator -n "$ALERT_ENGINE_NAMESPACE" --no-headers 2>/dev/null | grep -c Running || echo 0)
        if [[ $log_gen_pods -gt 0 ]]; then
            log_success "✅ Continuous log generator is running"
        else
            log_warning "⚠️  Continuous log generator not running"
        fi
    else
        log_warning "⚠️  Continuous log generator deployment not found"
    fi
    
    return $alert_engine_status
}

# Test end-to-end connectivity
test_e2e_connectivity() {
    log_step "TESTING END-TO-END CONNECTIVITY"
    
    log_info "Testing complete pipeline: Log Generator → Vector → Kafka → Alert Engine"
    
    # Check if logs are flowing to Kafka
    log_info "Checking for logs in Kafka topic 'application-logs'..."
    local kafka_logs=$(oc exec -n "$KAFKA_NAMESPACE" alert-kafka-cluster-kafka-0 -- timeout 10 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --max-messages 3 2>/dev/null || echo "")
    
    if [[ -n "$kafka_logs" ]]; then
        log_success "✅ Logs are flowing to Kafka topic"
        
        # Show sample logs
        echo -e "${BLUE}Sample logs from Kafka:${NC}"
        echo "$kafka_logs" | head -3
    else
        log_warning "⚠️  No logs found in Kafka topic (logs may still be processing)"
    fi
    
    # Connection details
    log_step "CONNECTION DETAILS"
    echo "Kafka Bootstrap Server: alert-kafka-cluster-kafka-bootstrap.$KAFKA_NAMESPACE.svc.cluster.local:9092"
    echo "Kafka Topic: application-logs"
    echo "Redis Cluster: redis-cluster-access.$REDIS_NAMESPACE.svc.cluster.local:6379"
    echo "Alert Engine Namespace: $ALERT_ENGINE_NAMESPACE"
    echo "Service Account: alert-engine-sa"
}

# Generate connection configuration
generate_config() {
    log_step "SAMPLE CONFIGURATION FOR ALERT ENGINE"
    
    cat <<EOF
# Sample config.yaml for Alert Engine deployment
kafka:
  brokers: ["alert-kafka-cluster-kafka-bootstrap.$KAFKA_NAMESPACE.svc.cluster.local:9092"]
  topic: "application-logs"
  consumer_group: "alert-engine-group"
  timeout: 30s

redis:
  mode: "cluster"
  addresses: [
    "redis-cluster-0.redis-cluster.$REDIS_NAMESPACE.svc.cluster.local:6379",
    "redis-cluster-1.redis-cluster.$REDIS_NAMESPACE.svc.cluster.local:6379",
    "redis-cluster-2.redis-cluster.$REDIS_NAMESPACE.svc.cluster.local:6379",
    "redis-cluster-3.redis-cluster.$REDIS_NAMESPACE.svc.cluster.local:6379",
    "redis-cluster-4.redis-cluster.$REDIS_NAMESPACE.svc.cluster.local:6379",
    "redis-cluster-5.redis-cluster.$REDIS_NAMESPACE.svc.cluster.local:6379"
  ]
  timeout: 5s

kubernetes:
  namespace: "$ALERT_ENGINE_NAMESPACE"
  service_account: "alert-engine-sa"

alerting:
  rules_file: "/etc/config/alert_rules.yaml"
  
notifications:
  slack:
    webhook_url: "YOUR_SLACK_WEBHOOK_URL"
    channel: "#alerts"
    
logging:
  level: "info"
  format: "json"
EOF
}

# Main validation function
main() {
    echo "=========================================="
    echo "OpenShift Infrastructure Validation for Alert Engine"
    echo "=========================================="
    echo ""
    
    check_prerequisites
    
    local overall_status=0
    
    # Run all validations
    validate_kafka
    local kafka_result=$?
    
    validate_redis
    local redis_result=$?
    
    validate_logging
    local logging_result=$?
    
    validate_alert_engine
    local alert_engine_result=$?
    
    # Calculate overall status
    overall_status=$((kafka_result + redis_result + logging_result + alert_engine_result))
    
    # Test end-to-end connectivity
    test_e2e_connectivity
    
    # Generate configuration
    generate_config
    
    # Final summary
    log_step "VALIDATION SUMMARY"
    
    if [[ $kafka_result -eq 0 ]]; then
        log_success "✅ Kafka Infrastructure: PASSED"
    else
        log_error "❌ Kafka Infrastructure: FAILED"
    fi
    
    if [[ $redis_result -eq 0 ]]; then
        log_success "✅ Redis Infrastructure: PASSED"
    else
        log_error "❌ Redis Infrastructure: FAILED"
    fi
    
    if [[ $logging_result -eq 0 ]]; then
        log_success "✅ Logging Infrastructure: PASSED"
    else
        log_error "❌ Logging Infrastructure: FAILED"
    fi
    
    if [[ $alert_engine_result -eq 0 ]]; then
        log_success "✅ Alert Engine Namespace: PASSED"
    else
        log_error "❌ Alert Engine Namespace: FAILED"
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log_success "🎉 Overall Infrastructure Validation: PASSED"
        log_info "Your OpenShift infrastructure is ready for Alert Engine deployment!"
    else
        log_error "❌ Overall Infrastructure Validation: FAILED"
        log_info "Please check the failed components and re-run the setup script if needed."
        exit 1
    fi
}

# Check if script is being sourced or executed
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 