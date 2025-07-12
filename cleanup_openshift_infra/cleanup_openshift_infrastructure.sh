#!/bin/bash

set -e

echo "🧹 OpenShift Infrastructure Cleanup Script"
echo "=========================================="
echo ""
echo "This script will remove ALL resources created during the OpenShift infrastructure setup:"
echo "- AMQ Streams / Kafka resources"
echo "- Redis Enterprise resources (with finalizer handling)"
echo "- OpenShift Logging resources"
echo "- Alert Engine test resources"
echo "- Service accounts, RBAC, and network policies"
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to show progress during long operations
show_progress() {
    local message=$1
    local duration=${2:-30}
    
    echo -n "${YELLOW}${message}${NC}"
    for ((i=0; i<duration; i++)); do
        echo -n "."
        sleep 1
    done
    echo ""
}

# Function to wait for resource deletion
wait_for_deletion() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3
    local timeout=${4:-300}
    
    echo "⏳ Waiting for ${resource_type}/${resource_name} deletion..."
    
    local counter=0
    while [[ $counter -lt $timeout ]]; do
        # Check if namespace still exists (for namespaced resources)
        if [[ -n "$namespace" ]]; then
            if ! oc get namespace ${namespace} >/dev/null 2>&1; then
                print_status $GREEN "✅ Namespace ${namespace} deleted - ${resource_type}/${resource_name} deleted"
                return 0
            fi
            
            # Check if resource still exists in the namespace
            if ! oc get ${resource_type} ${resource_name} -n ${namespace} >/dev/null 2>&1; then
                print_status $GREEN "✅ ${resource_type}/${resource_name} deleted successfully"
                return 0
            fi
        else
            # For cluster-scoped resources, check directly
            if ! oc get ${resource_type} ${resource_name} >/dev/null 2>&1; then
                print_status $GREEN "✅ ${resource_type}/${resource_name} deleted successfully"
                return 0
            fi
        fi
        
        # Show progress every 30 seconds
        if [[ $((counter % 30)) -eq 0 && $counter -gt 0 ]]; then
            print_status $YELLOW "Still waiting for ${resource_type}/${resource_name} deletion... (${counter}s elapsed)"
        fi
        
        sleep 10
        counter=$((counter + 10))
    done
    
    # Timeout reached
    print_status $RED "❌ Timeout waiting for ${resource_type}/${resource_name} deletion after ${timeout}s"
    
    # Additional check to see if namespace was deleted during timeout
    if [[ -n "$namespace" ]] && ! oc get namespace ${namespace} >/dev/null 2>&1; then
        print_status $YELLOW "⚠️ Note: Namespace ${namespace} was deleted during wait"
        return 0
    fi
    
    return 1
}

# Function to handle Redis Enterprise finalizer
handle_redis_finalizer() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3
    
    print_status $YELLOW "🔧 Handling Redis Enterprise finalizer for ${resource_type}/${resource_name}"
    
    # Check if namespace exists first
    if ! oc get namespace ${namespace} >/dev/null 2>&1; then
        print_status $GREEN "✅ Namespace ${namespace} doesn't exist - ${resource_type}/${resource_name} already deleted"
        return 0
    fi
    
    # Check if resource exists
    if ! oc get ${resource_type} ${resource_name} -n ${namespace} >/dev/null 2>&1; then
        print_status $GREEN "✅ ${resource_type}/${resource_name} already deleted"
        return 0
    fi
    
    # Try normal deletion first
    print_status $BLUE "🗑️ Attempting normal deletion of ${resource_type}/${resource_name}"
    oc delete ${resource_type} ${resource_name} -n ${namespace} --timeout=60s 2>/dev/null || true
    
    # Wait a bit to see if normal deletion works
    sleep 30
    
    # Check if namespace still exists before proceeding
    if ! oc get namespace ${namespace} >/dev/null 2>&1; then
        print_status $GREEN "✅ Namespace ${namespace} deleted - ${resource_type}/${resource_name} deleted"
        return 0
    fi
    
    # Check if still exists
    if oc get ${resource_type} ${resource_name} -n ${namespace} >/dev/null 2>&1; then
        print_status $YELLOW "⚠️ Resource still exists, removing finalizers..."
        
        # Remove finalizers - check if resource still exists before patching
        if oc get ${resource_type} ${resource_name} -n ${namespace} >/dev/null 2>&1; then
            print_status $YELLOW "🔧 Removing finalizers from ${resource_type}/${resource_name}"
            oc patch ${resource_type} ${resource_name} -n ${namespace} --type='merge' -p='{"metadata":{"finalizers":null}}' 2>/dev/null || true
        fi
        
        # Force delete if still exists
        sleep 10
        if oc get namespace ${namespace} >/dev/null 2>&1 && oc get ${resource_type} ${resource_name} -n ${namespace} >/dev/null 2>&1; then
            print_status $YELLOW "🔨 Force deleting ${resource_type}/${resource_name}"
            oc delete ${resource_type} ${resource_name} -n ${namespace} --force --grace-period=0 2>/dev/null || true
        fi
    fi
    
    # Final verification - only if namespace still exists
    if oc get namespace ${namespace} >/dev/null 2>&1; then
        wait_for_deletion ${resource_type} ${resource_name} ${namespace} 120
    else
        print_status $GREEN "✅ Namespace ${namespace} deleted - ${resource_type}/${resource_name} deleted"
    fi
}

# Ask for confirmation
echo -e "${RED}⚠️  WARNING: This will delete ALL infrastructure resources!${NC}"
echo ""
read -p "Are you sure you want to proceed? (yes/no): " -r
echo ""

if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    print_status $YELLOW "❌ Cleanup cancelled"
    exit 0
fi

print_status $BLUE "🚀 Starting OpenShift infrastructure cleanup..."
echo ""

# =============================================================================
# PRELIMINARY CHECKS
# =============================================================================

print_status $BLUE "🔍 Performing preliminary checks..."

# Check if we can connect to the OpenShift cluster
if ! oc cluster-info >/dev/null 2>&1; then
    print_status $RED "❌ Cannot connect to OpenShift cluster. Please check your cluster connection and login status."
    print_status $YELLOW "Run 'oc login' first and ensure you have the correct cluster context."
    exit 1
fi

# Check if we have cluster-admin permissions
if ! oc auth can-i '*' '*' --all-namespaces >/dev/null 2>&1; then
    print_status $YELLOW "⚠️ Warning: You may not have cluster-admin permissions. Some operations might fail."
    print_status $YELLOW "Proceeding anyway... Some resources may need manual cleanup."
fi

print_status $GREEN "✅ Preliminary checks completed"
echo ""

# =============================================================================
# STEP 1: Clean up Test Applications and Deployments
# =============================================================================

print_status $BLUE "📋 STEP 1: Cleaning up Test Applications"
echo ""

# Delete test application in alert-engine namespace
if oc get namespace alert-engine >/dev/null 2>&1; then
    if oc get deployment continuous-log-generator -n alert-engine >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting continuous log generator..."
        oc delete deployment continuous-log-generator -n alert-engine --timeout=60s
        print_status $GREEN "✅ Test application deleted"
    else
        print_status $GREEN "✅ Test application already deleted"
    fi
    
    # Delete any test pods that might be running
    print_status $YELLOW "🧹 Cleaning up test pods..."
    oc delete pod -l app=continuous-log-generator -n alert-engine --timeout=60s 2>/dev/null || true
    oc delete pod --all -n alert-engine --timeout=60s 2>/dev/null || true
else
    print_status $GREEN "✅ Alert Engine namespace doesn't exist - test applications already deleted"
fi

echo ""

# =============================================================================
# STEP 2: Clean up ClusterLogForwarder and OpenShift Logging
# =============================================================================

print_status $BLUE "📋 STEP 2: Cleaning up OpenShift Logging Resources"
echo ""

# Delete ClusterLogForwarder
if oc get clusterlogforwarder kafka-forwarder -n openshift-logging >/dev/null 2>&1; then
    print_status $YELLOW "🗑️ Deleting ClusterLogForwarder..."
    oc delete clusterlogforwarder kafka-forwarder -n openshift-logging --timeout=120s
    print_status $GREEN "✅ ClusterLogForwarder deleted"
else
    print_status $GREEN "✅ ClusterLogForwarder already deleted"
fi

# Wait for Vector pods to be cleaned up
print_status $YELLOW "⏳ Waiting for Vector collector pods to terminate..."

# Check if openshift-logging namespace exists before checking pods
if ! oc get namespace openshift-logging >/dev/null 2>&1; then
    print_status $GREEN "✅ OpenShift Logging namespace doesn't exist - Vector pods already terminated"
else
    # Check if Vector pods exist at all
    print_status $YELLOW "Checking for Vector collector pods..."
    vector_pods_output=$(oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --no-headers 2>/dev/null)
    if [[ -n "$vector_pods_output" && "$vector_pods_output" != "" ]]; then
        initial_vector_pods=$(echo "$vector_pods_output" | wc -l)
    else
        initial_vector_pods=0
    fi

    if [[ $initial_vector_pods -eq 0 ]]; then
        print_status $GREEN "✅ Vector collector pods already terminated"
    else
        print_status $YELLOW "Found $initial_vector_pods Vector pods to terminate"
        
        # Wait with timeout and progress updates
        timeout_counter=0
        max_timeout=60  # 60 seconds timeout
        
        while [[ $timeout_counter -lt $max_timeout ]]; do
            # Check namespace still exists before querying pods
            if ! oc get namespace openshift-logging >/dev/null 2>&1; then
                print_status $GREEN "✅ OpenShift Logging namespace deleted - Vector pods terminated"
                break
            fi
            
            vector_pods_output=$(oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --no-headers 2>/dev/null)
            if [[ -n "$vector_pods_output" && "$vector_pods_output" != "" ]]; then
                vector_pods=$(echo "$vector_pods_output" | wc -l)
            else
                vector_pods=0
            fi
            
            if [[ $vector_pods -eq 0 ]]; then
                print_status $GREEN "✅ Vector collector pods terminated"
                break
            fi
            
            # Show progress every 15 seconds
            if [[ $((timeout_counter % 15)) -eq 0 ]]; then
                print_status $YELLOW "Still waiting... ($vector_pods pods remaining, ${timeout_counter}s elapsed)"
            fi
            
            sleep 5
            timeout_counter=$((timeout_counter + 5))
        done
        
        # Force cleanup if timeout reached
        if [[ $timeout_counter -ge $max_timeout ]]; then
            # Check namespace still exists before attempting cleanup
            if oc get namespace openshift-logging >/dev/null 2>&1; then
                vector_pods_output=$(oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --no-headers 2>/dev/null)
                if [[ -n "$vector_pods_output" && "$vector_pods_output" != "" ]]; then
                    remaining_pods=$(echo "$vector_pods_output" | wc -l)
                else
                    remaining_pods=0
                fi
                if [[ $remaining_pods -gt 0 ]]; then
                    print_status $YELLOW "⚠️ Timeout reached, force deleting $remaining_pods remaining Vector pods..."
                    oc delete pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --force --grace-period=0 2>/dev/null || true
                    sleep 5
                    print_status $GREEN "✅ Vector collector pods terminated (forced)"
                else
                    print_status $GREEN "✅ Vector collector pods terminated"
                fi
            else
                print_status $GREEN "✅ OpenShift Logging namespace deleted - Vector pods terminated"
            fi
        fi
    fi
fi

# Delete service account and RBAC (check namespace existence first)
if oc get namespace openshift-logging >/dev/null 2>&1; then
    print_status $YELLOW "🗑️ Deleting logging service account and RBAC..."
    oc delete serviceaccount log-collector -n openshift-logging --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ OpenShift Logging namespace doesn't exist - service account and RBAC already deleted"
fi

# Delete cluster-scoped RBAC (these don't require namespace to exist)
print_status $YELLOW "🗑️ Deleting cluster-scoped logging RBAC..."
oc delete clusterrolebinding log-collector-application-logs --timeout=30s 2>/dev/null || true
oc delete clusterrolebinding log-collector-write-logs --timeout=30s 2>/dev/null || true

# Delete OpenShift Logging Operator (check namespace existence first)
if oc get namespace openshift-logging >/dev/null 2>&1; then
    print_status $YELLOW "🗑️ Deleting OpenShift Logging Operator..."
    oc delete subscription cluster-logging -n openshift-logging --timeout=60s 2>/dev/null || true
    oc delete csv -n openshift-logging --all --timeout=60s 2>/dev/null || true
else
    print_status $GREEN "✅ OpenShift Logging namespace doesn't exist - operator already deleted"
fi

print_status $GREEN "✅ OpenShift Logging resources cleaned up"
echo ""

# =============================================================================
# STEP 3: Clean up Redis Enterprise Resources
# =============================================================================

print_status $BLUE "📋 STEP 3: Cleaning up Redis Enterprise Resources"
echo ""

# Check if Redis Enterprise namespace exists first
if ! oc get namespace redis-enterprise >/dev/null 2>&1; then
    print_status $GREEN "✅ Redis Enterprise namespace already deleted - skipping Redis cleanup"
else
    # Delete Redis Enterprise Database first (with finalizer handling)
    if oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Redis Enterprise Database..."
        handle_redis_finalizer "redisenterprisedatabase" "alert-engine-cache" "redis-enterprise"
    else
        print_status $GREEN "✅ Redis Enterprise Database already deleted"
    fi

    # Delete Redis Enterprise Cluster (with finalizer handling)
    if oc get redisenterprisecluster rec-alert-engine -n redis-enterprise >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Redis Enterprise Cluster..."
        handle_redis_finalizer "redisenterprisecluster" "rec-alert-engine" "redis-enterprise"
    else
        print_status $GREEN "✅ Redis Enterprise Cluster already deleted"
    fi
fi

# Delete Redis network policy
print_status $YELLOW "🗑️ Deleting Redis network policy..."
if oc get namespace redis-enterprise >/dev/null 2>&1; then
    oc delete networkpolicy redis-network-policy -n redis-enterprise --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ Redis Enterprise namespace doesn't exist - network policy already deleted"
fi

# Delete Redis Enterprise Operator
print_status $YELLOW "🗑️ Deleting Redis Enterprise Operator..."
if oc get namespace redis-enterprise >/dev/null 2>&1; then
    oc delete subscription redis-enterprise-operator -n redis-enterprise --timeout=60s 2>/dev/null || true
    oc delete csv -n redis-enterprise --all --timeout=60s 2>/dev/null || true
else
    print_status $GREEN "✅ Redis Enterprise namespace doesn't exist - operator already deleted"
fi

# Delete Security Context Constraints (cluster-scoped resource)
print_status $YELLOW "🗑️ Deleting Redis Enterprise SCC..."
if oc get securitycontextconstraints redis-enterprise-scc >/dev/null 2>&1; then
    oc delete securitycontextconstraints redis-enterprise-scc --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ Redis Enterprise SCC already deleted"
fi

# Remove SCC policy bindings (check namespace existence first)
if oc get namespace redis-enterprise >/dev/null 2>&1; then
    print_status $YELLOW "🗑️ Removing SCC policy bindings..."
    oc adm policy remove-scc-from-user redis-enterprise-scc -z redis-enterprise-operator -n redis-enterprise 2>/dev/null || true
    oc adm policy remove-scc-from-user redis-enterprise-scc -z rec-alert-engine -n redis-enterprise 2>/dev/null || true
else
    print_status $GREEN "✅ Redis Enterprise namespace doesn't exist - SCC policy bindings already removed"
fi

print_status $GREEN "✅ Redis Enterprise resources cleaned up"
echo ""

# =============================================================================
# STEP 4: Clean up Kafka/AMQ Streams Resources
# =============================================================================

print_status $BLUE "📋 STEP 4: Cleaning up AMQ Streams / Kafka Resources"
echo ""

# Delete Kafka topics
print_status $YELLOW "🗑️ Deleting Kafka topics..."
if oc get namespace amq-streams-kafka >/dev/null 2>&1; then
    oc delete kafkatopic application-logs -n amq-streams-kafka --timeout=60s 2>/dev/null || true
else
    print_status $GREEN "✅ AMQ Streams namespace doesn't exist - Kafka topics already deleted"
fi

# Delete Kafka cluster
if oc get namespace amq-streams-kafka >/dev/null 2>&1; then
    if oc get kafka alert-kafka-cluster -n amq-streams-kafka >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Kafka cluster..."
        oc delete kafka alert-kafka-cluster -n amq-streams-kafka --timeout=300s
        wait_for_deletion "kafka" "alert-kafka-cluster" "amq-streams-kafka" 300
    else
        print_status $GREEN "✅ Kafka cluster already deleted"
    fi
else
    print_status $GREEN "✅ AMQ Streams namespace doesn't exist - Kafka cluster already deleted"
fi

# Delete Kafka network policy
print_status $YELLOW "🗑️ Deleting Kafka network policy..."
if oc get namespace amq-streams-kafka >/dev/null 2>&1; then
    oc delete networkpolicy kafka-network-policy -n amq-streams-kafka --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ AMQ Streams namespace doesn't exist - network policy already deleted"
fi

# Delete AMQ Streams Operator
print_status $YELLOW "🗑️ Deleting AMQ Streams Operator..."
if oc get namespace amq-streams-kafka >/dev/null 2>&1; then
    oc delete subscription amq-streams -n amq-streams-kafka --timeout=60s 2>/dev/null || true
    oc delete csv -n amq-streams-kafka --all --timeout=60s 2>/dev/null || true
else
    print_status $GREEN "✅ AMQ Streams namespace doesn't exist - operator already deleted"
fi

print_status $GREEN "✅ AMQ Streams / Kafka resources cleaned up"
echo ""

# =============================================================================
# STEP 5: Clean up Alert Engine Resources
# =============================================================================

print_status $BLUE "📋 STEP 5: Cleaning up Alert Engine Resources"
echo ""

# Delete ConfigMaps and Secrets
print_status $YELLOW "🗑️ Deleting ConfigMaps and Secrets..."
if oc get namespace alert-engine >/dev/null 2>&1; then
    oc delete configmap redis-config -n alert-engine --timeout=30s 2>/dev/null || true
    oc delete secret redis-password -n alert-engine --timeout=30s 2>/dev/null || true
    oc delete secret redb-alert-engine-cache -n alert-engine --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ Alert Engine namespace doesn't exist - ConfigMaps and Secrets already deleted"
fi

# Delete Service Account and RBAC
print_status $YELLOW "🗑️ Deleting Alert Engine service account and RBAC..."
if oc get namespace alert-engine >/dev/null 2>&1; then
    oc delete serviceaccount alert-engine-sa -n alert-engine --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ Alert Engine namespace doesn't exist - service account already deleted"
fi

# Delete cluster-scoped RBAC (these don't require namespace to exist)
print_status $YELLOW "🗑️ Deleting Alert Engine cluster-scoped RBAC..."
if oc get clusterrole alert-engine-role >/dev/null 2>&1; then
    oc delete clusterrole alert-engine-role --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ Alert Engine cluster role already deleted"
fi

if oc get clusterrolebinding alert-engine-binding >/dev/null 2>&1; then
    oc delete clusterrolebinding alert-engine-binding --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ Alert Engine cluster role binding already deleted"
fi

print_status $GREEN "✅ Alert Engine resources cleaned up"
echo ""

# =============================================================================
# STEP 6: Clean up Namespaces
# =============================================================================

print_status $BLUE "📋 STEP 6: Cleaning up Namespaces"
echo ""

# Delete namespaces in order
namespaces=("alert-engine" "redis-enterprise" "amq-streams-kafka" "openshift-logging")

for namespace in "${namespaces[@]}"; do
    if oc get namespace $namespace >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting namespace: $namespace"
        
        # Start deletion without background process to avoid hanging
        print_status $BLUE "🗑️ Initiating deletion of namespace $namespace"
        oc delete namespace $namespace --timeout=60s 2>/dev/null || true
        
        # Monitor deletion progress
        timeout_counter=0
        max_wait=180  # 3 minutes
        
        while [[ $timeout_counter -lt $max_wait ]]; do
            if ! oc get namespace $namespace >/dev/null 2>&1; then
                print_status $GREEN "✅ Namespace $namespace deleted successfully"
                break
            fi
            
            if [[ $((timeout_counter % 30)) -eq 0 && $timeout_counter -gt 0 ]]; then
                print_status $YELLOW "Still waiting for namespace $namespace deletion... (${timeout_counter}s elapsed)"
                
                # Check if namespace is stuck in terminating state
                namespace_status=$(oc get namespace $namespace -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
                if [[ "$namespace_status" == "Terminating" ]]; then
                    print_status $YELLOW "⚠️ Namespace $namespace is in Terminating state"
                fi
            fi
            
            sleep 10
            timeout_counter=$((timeout_counter + 10))
        done
        
        # Handle stuck namespace
        if oc get namespace $namespace >/dev/null 2>&1; then
            print_status $YELLOW "⚠️ Namespace $namespace is stuck, attempting finalizer removal..."
            
            # Check if namespace is stuck in terminating state
            namespace_status=$(oc get namespace $namespace -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
            if [[ "$namespace_status" == "Terminating" ]]; then
                print_status $YELLOW "🔧 Namespace is in Terminating state, removing stuck resources..."
                
                # Try to remove finalizers from stuck resources
                for resource in $(oc api-resources --namespaced=true -o name 2>/dev/null | grep -E "(redis|kafka|strimzi)" | head -10); do
                    if oc get $resource -n $namespace --no-headers 2>/dev/null | grep -q .; then
                        print_status $YELLOW "🔧 Removing finalizers from $resource resources"
                        oc get $resource -n $namespace --no-headers 2>/dev/null | awk '{print $1}' | while read name; do
                            if [[ -n "$name" ]]; then
                                oc patch $resource $name -n $namespace --type='merge' -p='{"metadata":{"finalizers":null}}' 2>/dev/null || true
                            fi
                        done
                    fi
                done
                
                # Force patch namespace finalizers
                sleep 10
                if oc get namespace $namespace >/dev/null 2>&1; then
                    print_status $YELLOW "🔨 Force removing namespace finalizers for $namespace"
                    oc patch namespace $namespace --type='merge' -p='{"metadata":{"finalizers":null}}' 2>/dev/null || true
                fi
                
                # Final check with shorter timeout
                final_timeout=60
                final_counter=0
                while [[ $final_counter -lt $final_timeout ]]; do
                    if ! oc get namespace $namespace >/dev/null 2>&1; then
                        print_status $GREEN "✅ Namespace $namespace deleted successfully"
                        break
                    fi
                    sleep 5
                    final_counter=$((final_counter + 5))
                done
                
                if oc get namespace $namespace >/dev/null 2>&1; then
                    print_status $YELLOW "⚠️ Namespace $namespace may still exist but deletion is in progress"
                    print_status $YELLOW "This is normal for namespaces with complex resources - check manually later"
                fi
            else
                print_status $YELLOW "⚠️ Namespace $namespace deletion may be in progress"
            fi
        fi
        
    else
        print_status $GREEN "✅ Namespace $namespace already deleted"
    fi
done

print_status $GREEN "✅ All namespaces processed"
echo ""

# =============================================================================
# STEP 7: Clean up Operator Groups and Install Plans
# =============================================================================

print_status $BLUE "📋 STEP 7: Cleaning up Operator Groups and Install Plans"
echo ""

# Clean up any remaining operator groups
print_status $YELLOW "🗑️ Cleaning up operator groups..."
if oc get namespace amq-streams-kafka >/dev/null 2>&1; then
    oc delete operatorgroup amq-streams-og -n amq-streams-kafka --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ AMQ Streams namespace doesn't exist - operator group already deleted"
fi

if oc get namespace redis-enterprise >/dev/null 2>&1; then
    oc delete operatorgroup redis-enterprise-og -n redis-enterprise --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ Redis Enterprise namespace doesn't exist - operator group already deleted"
fi

if oc get namespace openshift-logging >/dev/null 2>&1; then
    oc delete operatorgroup cluster-logging -n openshift-logging --timeout=30s 2>/dev/null || true
else
    print_status $GREEN "✅ OpenShift Logging namespace doesn't exist - operator group already deleted"
fi

# Clean up install plans
print_status $YELLOW "🗑️ Cleaning up install plans..."
# Use a safer approach to check install plans without failing if namespaces don't exist
for namespace in amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace $namespace >/dev/null 2>&1; then
        # Check if there are any install plans in this namespace
        install_plans=$(oc get installplan -n $namespace --no-headers 2>/dev/null | grep -E "(amq-streams|redis-enterprise|cluster-logging)" | awk '{print $1}' || echo "")
        if [[ -n "$install_plans" ]]; then
            print_status $YELLOW "🗑️ Deleting install plans in namespace $namespace..."
            echo "$install_plans" | while read plan_name; do
                if [[ -n "$plan_name" ]]; then
                    oc delete installplan $plan_name -n $namespace --timeout=30s 2>/dev/null || true
                fi
            done
        fi
    fi
done

print_status $GREEN "✅ Operator groups and install plans cleaned up"
echo ""

# =============================================================================
# STEP 8: Clean up Custom Resource Definitions (Optional)
# =============================================================================

print_status $BLUE "📋 STEP 8: Cleaning up Custom Resource Definitions"
echo ""

print_status $YELLOW "❓ Do you want to delete CRDs? (This will affect other operator installations)"
read -p "Delete CRDs? (yes/no): " -r
echo ""

if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    print_status $YELLOW "🗑️ Deleting Custom Resource Definitions..."
    
    # Delete AMQ Streams CRDs
    print_status $YELLOW "🗑️ Checking for AMQ Streams CRDs..."
    amq_crds=$(oc get crd -o name 2>/dev/null | grep -E "(kafka|strimzi)" || echo "")
    if [[ -n "$amq_crds" ]]; then
        echo "$amq_crds" | while read crd; do
            if [[ -n "$crd" ]]; then
                print_status $YELLOW "🗑️ Deleting $crd"
                oc delete $crd --timeout=60s 2>/dev/null || true
            fi
        done
    else
        print_status $GREEN "✅ No AMQ Streams CRDs found"
    fi
    
    # Delete Redis Enterprise CRDs
    print_status $YELLOW "🗑️ Checking for Redis Enterprise CRDs..."
    redis_crds=$(oc get crd -o name 2>/dev/null | grep -E "(redis|redislabs)" || echo "")
    if [[ -n "$redis_crds" ]]; then
        echo "$redis_crds" | while read crd; do
            if [[ -n "$crd" ]]; then
                print_status $YELLOW "🗑️ Deleting $crd"
                oc delete $crd --timeout=60s 2>/dev/null || true
            fi
        done
    else
        print_status $GREEN "✅ No Redis Enterprise CRDs found"
    fi
    
    # Delete OpenShift Logging CRDs
    print_status $YELLOW "🗑️ Checking for OpenShift Logging CRDs..."
    logging_crds=$(oc get crd -o name 2>/dev/null | grep -E "(logging|clusterlogforwarder)" || echo "")
    if [[ -n "$logging_crds" ]]; then
        echo "$logging_crds" | while read crd; do
            if [[ -n "$crd" ]]; then
                print_status $YELLOW "🗑️ Deleting $crd"
                oc delete $crd --timeout=60s 2>/dev/null || true
            fi
        done
    else
        print_status $GREEN "✅ No OpenShift Logging CRDs found"
    fi
    
    print_status $GREEN "✅ Custom Resource Definitions cleaned up"
else
    print_status $YELLOW "⏭️ Skipping CRD cleanup"
fi

echo ""

# =============================================================================
# STEP 9: Final Verification
# =============================================================================

print_status $BLUE "📋 STEP 9: Final Verification"
echo ""

# Check if any resources remain
print_status $YELLOW "🔍 Checking for remaining resources..."

# Check namespaces
remaining_namespaces=()
for namespace in "${namespaces[@]}"; do
    if oc get namespace $namespace >/dev/null 2>&1; then
        remaining_namespaces+=($namespace)
    fi
done

if [[ ${#remaining_namespaces[@]} -gt 0 ]]; then
    print_status $YELLOW "⚠️ Some namespaces still exist: ${remaining_namespaces[*]}"
    print_status $YELLOW "These may be in terminating state, check manually if needed"
else
    print_status $GREEN "✅ All namespaces successfully deleted"
fi

# Check for remaining CRDs
print_status $YELLOW "🔍 Checking for remaining CRDs..."
crd_output=$(oc get crd -o name 2>/dev/null | grep -E "(kafka|strimzi|redis|redislabs|logging|clusterlogforwarder)" 2>/dev/null || echo "")
if [[ -n "$crd_output" ]]; then
    remaining_crds=$(echo "$crd_output" | wc -l)
    print_status $YELLOW "⚠️ Some CRDs still exist ($remaining_crds found)"
    print_status $YELLOW "These may be used by other operators or installations"
    print_status $BLUE "To check: oc get crd | grep -E '(kafka|strimzi|redis|redislabs|logging|clusterlogforwarder)'"
else
    print_status $GREEN "✅ No related CRDs found"
fi

# Check for remaining PVCs
print_status $YELLOW "🔍 Checking for remaining PVCs..."
pvc_output=$(oc get pvc -A --no-headers 2>/dev/null | grep -E "(kafka|redis|zookeeper)" 2>/dev/null || echo "")
if [[ -n "$pvc_output" ]]; then
    remaining_pvcs=$(echo "$pvc_output" | wc -l)
    print_status $YELLOW "⚠️ Some Persistent Volume Claims still exist ($remaining_pvcs found)"
    print_status $YELLOW "These may need manual cleanup to avoid storage costs"
    print_status $BLUE "To check: oc get pvc -A | grep -E '(kafka|redis|zookeeper)'"
else
    print_status $GREEN "✅ No related PVCs found"
fi

# Check for remaining pods in terminating state
print_status $YELLOW "🔍 Checking for pods in terminating state..."
pods_output=$(oc get pods -A --no-headers 2>/dev/null | grep -i terminating 2>/dev/null || echo "")
if [[ -n "$pods_output" ]]; then
    terminating_pods=$(echo "$pods_output" | wc -l)
    print_status $YELLOW "⚠️ Some pods are still in terminating state ($terminating_pods found)"
    print_status $YELLOW "These should eventually be cleaned up automatically"
    print_status $BLUE "To check: oc get pods -A | grep -i terminating"
else
    print_status $GREEN "✅ No pods in terminating state found"
fi

echo ""

# =============================================================================
# COMPLETION
# =============================================================================

print_status $GREEN "🎉 OpenShift Infrastructure Cleanup Complete!"
echo ""
print_status $BLUE "Summary of cleaned up resources:"
echo "  ✅ Test applications and deployments"
echo "  ✅ ClusterLogForwarder and Vector collectors"
echo "  ✅ OpenShift Logging Operator"
echo "  ✅ Redis Enterprise databases and clusters (with finalizer handling)"
echo "  ✅ Redis Enterprise Operator"
echo "  ✅ Kafka clusters and topics"
echo "  ✅ AMQ Streams Operator"
echo "  ✅ Service accounts and RBAC"
echo "  ✅ Network policies"
echo "  ✅ Security Context Constraints"
echo "  ✅ ConfigMaps and Secrets"
echo "  ✅ Namespaces"
echo "  ✅ Operator groups and install plans"
echo ""

if [[ ${#remaining_namespaces[@]} -gt 0 ]]; then
    print_status $YELLOW "⚠️ Note: Some resources may still be terminating. Check cluster status in a few minutes."
else
    print_status $GREEN "✅ Your OpenShift cluster is now clean and ready for fresh deployments!"
fi

echo ""
print_status $BLUE "To verify the cleanup, run:"
echo "  oc get namespaces | grep -E '(amq-streams|redis-enterprise|openshift-logging|alert-engine)'"
echo "  oc get crd | grep -E '(kafka|strimzi|redis|redislabs|logging)'"
echo ""

print_status $GREEN "Cleanup completed successfully! 🎉" 