#!/bin/bash

# OpenShift Utilities Library
# Shared functions for OpenShift resource management and validation

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default namespaces
KAFKA_NAMESPACE="${KAFKA_NAMESPACE:-amq-streams-kafka}"
REDIS_NAMESPACE="${REDIS_NAMESPACE:-redis-cluster}"
LOGGING_NAMESPACE="${LOGGING_NAMESPACE:-openshift-logging}"
ALERT_ENGINE_NAMESPACE="${ALERT_ENGINE_NAMESPACE:-alert-engine}"

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

# Function to count resources in a namespace
count_resources() {
    local resource_type=$1
    local namespace=$2
    
    if [[ -n "$namespace" ]]; then
        if oc get namespace "$namespace" >/dev/null 2>&1; then
            count=$(oc get "$resource_type" -n "$namespace" --no-headers 2>/dev/null | wc -l)
            echo "$count"
        else
            echo "0"
        fi
    else
        count=$(oc get "$resource_type" --no-headers 2>/dev/null | wc -l)
        echo "$count"
    fi
}

# Function to check basic resource existence (for cleanup verification)
check_resource_simple() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3
    
    if [[ -n "$namespace" ]]; then
        if oc get "$resource_type" "$resource_name" -n "$namespace" >/dev/null 2>&1; then
            echo "  ✅ Found: $resource_type/$resource_name in namespace $namespace"
            return 0
        else
            echo "  ❌ Not found: $resource_type/$resource_name in namespace $namespace"
            return 1
        fi
    else
        if oc get "$resource_type" "$resource_name" >/dev/null 2>&1; then
            echo "  ✅ Found: $resource_type/$resource_name (cluster-wide)"
            return 0
        else
            echo "  ❌ Not found: $resource_type/$resource_name (cluster-wide)"
            return 1
        fi
    fi
}

# Function to wait for resource condition
wait_for_condition() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3
    local condition=$4
    local timeout=${5:-300}
    
    log_info "Waiting for $resource_type/$resource_name to be $condition (timeout: ${timeout}s)..."
    
    if oc wait "$resource_type/$resource_name" --for="condition=$condition" --timeout="${timeout}s" -n "$namespace" 2>/dev/null; then
        log_success "$resource_type/$resource_name is $condition"
        return 0
    else
        log_error "Timeout waiting for $resource_type/$resource_name to be $condition"
        return 1
    fi
}

# Function to check OpenShift prerequisites
check_openshift_prerequisites() {
    local requires_admin=${1:-true}
    
    # Check if oc CLI is available
    if ! command -v oc &> /dev/null; then
        log_error "oc CLI tool is not installed or not in PATH"
        return 1
    fi
    
    # Check if we're logged into OpenShift
    if ! oc whoami &>/dev/null; then
        log_error "Not logged into OpenShift cluster. Please login first."
        return 1
    fi
    
    # Check cluster-admin permissions if required
    if [[ "$requires_admin" == "true" ]]; then
        if ! oc auth can-i '*' '*' --all-namespaces &>/dev/null; then
            log_error "Cluster-admin permissions required"
            return 1
        fi
    fi
    
    return 0
}

# Function to get CSV status for an operator
get_operator_csv_status() {
    local namespace=$1
    local operator_pattern=$2
    
    oc get csv -n "$namespace" --no-headers 2>/dev/null | grep "$operator_pattern" | awk '{print $6}' || echo "NotFound"
}

# Function to check namespace existence
check_namespace() {
    local namespace=$1
    
    if oc get namespace "$namespace" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to get pod count by label
get_pod_count() {
    local namespace=$1
    local label_selector=$2
    local status_filter=${3:-""}
    
    if [[ -n "$status_filter" ]]; then
        oc get pods -l "$label_selector" -n "$namespace" --no-headers 2>/dev/null | grep "$status_filter" | wc -l || echo 0
    else
        oc get pods -l "$label_selector" -n "$namespace" --no-headers 2>/dev/null | wc -l || echo 0
    fi
}

# Function to validate storage class
validate_storage_class() {
    local storage_class=$1
    
    if oc get storageclass "$storage_class" &>/dev/null; then
        return 0
    else
        log_warning "Storage class '$storage_class' not found"
        log_info "Available storage classes:"
        oc get storageclass --no-headers -o custom-columns=NAME:.metadata.name
        return 1
    fi
}

# Function to get connection details for a service
get_service_connection() {
    local service_name=$1
    local namespace=$2
    local port=${3:-""}
    
    if [[ -n "$port" ]]; then
        echo "$service_name.$namespace.svc.cluster.local:$port"
    else
        local service_port=$(oc get svc "$service_name" -n "$namespace" -o jsonpath='{.spec.ports[0].port}' 2>/dev/null)
        echo "$service_name.$namespace.svc.cluster.local:$service_port"
    fi
}

# Function to test Kafka connectivity
test_kafka_connectivity() {
    local kafka_namespace=$1
    local kafka_cluster_name=${2:-"alert-kafka-cluster"}
    local topic=${3:-"application-logs"}
    
    # Test basic connectivity
    if oc exec -n "$kafka_namespace" "${kafka_cluster_name}-kafka-0" -- timeout 5 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic "$topic" --from-beginning --max-messages 0 &>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Function to test Redis cluster connectivity
test_redis_connectivity() {
    local redis_namespace=$1
    local redis_pod=${2:-"redis-cluster-0"}
    
    # Test cluster info
    local cluster_info=$(oc exec -n "$redis_namespace" "$redis_pod" -- redis-cli cluster info 2>/dev/null || echo "cluster_state:fail")
    
    if [[ $cluster_info == *"cluster_state:ok"* ]]; then
        return 0
    else
        return 1
    fi
}

# Function to generate sample configuration
generate_alert_engine_config() {
    local kafka_namespace=$1
    local redis_namespace=$2
    local alert_engine_namespace=$3
    
    cat <<EOF
# Sample config.yaml for Alert Engine deployment
kafka:
  brokers: ["alert-kafka-cluster-kafka-bootstrap.$kafka_namespace.svc.cluster.local:9092"]
  topic: "application-logs"
  consumer_group: "alert-engine-group"
  timeout: 30s

redis:
  mode: "cluster"
  addresses: [
    "redis-cluster-0.redis-cluster.$redis_namespace.svc.cluster.local:6379",
    "redis-cluster-1.redis-cluster.$redis_namespace.svc.cluster.local:6379",
    "redis-cluster-2.redis-cluster.$redis_namespace.svc.cluster.local:6379",
    "redis-cluster-3.redis-cluster.$redis_namespace.svc.cluster.local:6379",
    "redis-cluster-4.redis-cluster.$redis_namespace.svc.cluster.local:6379",
    "redis-cluster-5.redis-cluster.$redis_namespace.svc.cluster.local:6379"
  ]
  timeout: 5s

kubernetes:
  namespace: "$alert_engine_namespace"
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