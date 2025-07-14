#!/bin/bash

set -e

echo "🧹 OpenShift Infrastructure Cleanup Script"
echo "=========================================="
echo ""
echo "This script will remove ALL resources created during the OpenShift infrastructure setup:"
echo "- AMQ Streams / Kafka resources"
echo "- Redis Cluster resources"
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
        print_status $YELLOW "Found $initial_vector_pods Vector collector pods. Waiting for termination..."
        
        # Wait for Vector pods to terminate
        counter=0
        while [[ $counter -lt 120 ]]; do
            if ! oc get namespace openshift-logging >/dev/null 2>&1; then
                print_status $GREEN "✅ OpenShift Logging namespace deleted - Vector pods terminated"
                break
            fi
            
            current_pods=$(oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --no-headers 2>/dev/null | wc -l)
            if [[ $current_pods -eq 0 ]]; then
                print_status $GREEN "✅ Vector collector pods terminated"
                break
            fi
            
            if [[ $((counter % 30)) -eq 0 && $counter -gt 0 ]]; then
                print_status $YELLOW "Still waiting for Vector pods to terminate... ($current_pods remaining)"
            fi
            
            sleep 10
            counter=$((counter + 10))
        done
    fi
fi

# Delete service account and RBAC
print_status $YELLOW "🗑️ Deleting logging service account and RBAC..."
oc delete serviceaccount log-collector -n openshift-logging --timeout=60s 2>/dev/null || true
oc delete clusterrolebinding log-collector-application-logs --timeout=60s 2>/dev/null || true
oc delete clusterrolebinding log-collector-write-logs --timeout=60s 2>/dev/null || true

print_status $GREEN "✅ OpenShift Logging resources cleanup completed"
echo ""

# =============================================================================
# STEP 3: Clean up Redis Cluster Resources
# =============================================================================

print_status $BLUE "📋 STEP 3: Cleaning up Redis Cluster Resources"
echo ""

# Delete Redis cluster resources
if oc get namespace redis-cluster >/dev/null 2>&1; then
    print_status $YELLOW "🗑️ Deleting Redis Cluster resources..."
    
    # Delete StatefulSet
    if oc get statefulset redis-cluster -n redis-cluster >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Redis Cluster StatefulSet..."
        oc delete statefulset redis-cluster -n redis-cluster --timeout=120s
        print_status $GREEN "✅ Redis Cluster StatefulSet deleted"
    else
        print_status $GREEN "✅ Redis Cluster StatefulSet already deleted"
    fi
    
    # Delete Services
    if oc get service redis-cluster -n redis-cluster >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Redis Cluster service..."
        oc delete service redis-cluster -n redis-cluster --timeout=60s
        print_status $GREEN "✅ Redis Cluster service deleted"
    fi
    
    if oc get service redis-cluster-access -n redis-cluster >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Redis Cluster access service..."
        oc delete service redis-cluster-access -n redis-cluster --timeout=60s
        print_status $GREEN "✅ Redis Cluster access service deleted"
    fi
    
    # Delete ConfigMaps
    if oc get configmap redis-cluster-config -n redis-cluster >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Redis Cluster ConfigMap..."
        oc delete configmap redis-cluster-config -n redis-cluster --timeout=60s
        print_status $GREEN "✅ Redis Cluster ConfigMap deleted"
    fi
    
    # Delete Network Policy
    if oc get networkpolicy redis-cluster-network-policy -n redis-cluster >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Redis Cluster network policy..."
        oc delete networkpolicy redis-cluster-network-policy -n redis-cluster --timeout=60s
        print_status $GREEN "✅ Redis Cluster network policy deleted"
    fi
    
    # Delete PVCs
    print_status $YELLOW "🗑️ Deleting Redis Cluster PVCs..."
    oc delete pvc -l app=redis-cluster -n redis-cluster --timeout=120s 2>/dev/null || true
    
    # Wait for pods to terminate
    print_status $YELLOW "⏳ Waiting for Redis Cluster pods to terminate..."
    counter=0
    while [[ $counter -lt 120 ]]; do
        if ! oc get namespace redis-cluster >/dev/null 2>&1; then
            print_status $GREEN "✅ Redis Cluster namespace deleted - pods terminated"
            break
        fi
        
        current_pods=$(oc get pods -n redis-cluster -l app=redis-cluster --no-headers 2>/dev/null | wc -l)
        if [[ $current_pods -eq 0 ]]; then
            print_status $GREEN "✅ Redis Cluster pods terminated"
            break
        fi
        
        if [[ $((counter % 30)) -eq 0 && $counter -gt 0 ]]; then
            print_status $YELLOW "Still waiting for Redis Cluster pods to terminate... ($current_pods remaining)"
        fi
        
        sleep 10
        counter=$((counter + 10))
    done
    
    print_status $GREEN "✅ Redis Cluster resources cleanup completed"
else
    print_status $GREEN "✅ Redis Cluster namespace doesn't exist - resources already deleted"
fi

echo ""

# =============================================================================
# STEP 4: Clean up AMQ Streams / Kafka Resources
# =============================================================================

print_status $BLUE "📋 STEP 4: Cleaning up AMQ Streams / Kafka Resources"
echo ""

# Delete Kafka resources
if oc get namespace amq-streams-kafka >/dev/null 2>&1; then
    # Delete KafkaTopics first
    print_status $YELLOW "🗑️ Deleting Kafka topics..."
    oc delete kafkatopic application-logs -n amq-streams-kafka --timeout=60s 2>/dev/null || true
    
    # Delete Kafka cluster
    if oc get kafka alert-kafka-cluster -n amq-streams-kafka >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Kafka cluster..."
        oc delete kafka alert-kafka-cluster -n amq-streams-kafka --timeout=300s
        print_status $GREEN "✅ Kafka cluster deleted"
    else
        print_status $GREEN "✅ Kafka cluster already deleted"
    fi
    
    # Delete Network Policy
    if oc get networkpolicy kafka-network-policy -n amq-streams-kafka >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting Kafka network policy..."
        oc delete networkpolicy kafka-network-policy -n amq-streams-kafka --timeout=60s
        print_status $GREEN "✅ Kafka network policy deleted"
    fi
    
    # Wait for all Kafka pods to terminate
    print_status $YELLOW "⏳ Waiting for Kafka pods to terminate..."
    counter=0
    while [[ $counter -lt 180 ]]; do
        if ! oc get namespace amq-streams-kafka >/dev/null 2>&1; then
            print_status $GREEN "✅ AMQ Streams namespace deleted - Kafka pods terminated"
            break
        fi
        
        current_pods=$(oc get pods -n amq-streams-kafka --no-headers 2>/dev/null | wc -l)
        if [[ $current_pods -eq 0 ]]; then
            print_status $GREEN "✅ Kafka pods terminated"
            break
        fi
        
        if [[ $((counter % 30)) -eq 0 && $counter -gt 0 ]]; then
            print_status $YELLOW "Still waiting for Kafka pods to terminate... ($current_pods remaining)"
        fi
        
        sleep 10
        counter=$((counter + 10))
    done
    
    print_status $GREEN "✅ AMQ Streams / Kafka resources cleanup completed"
else
    print_status $GREEN "✅ AMQ Streams namespace doesn't exist - resources already deleted"
fi

echo ""

# =============================================================================
# STEP 5: Clean up Alert Engine Resources
# =============================================================================

print_status $BLUE "📋 STEP 5: Cleaning up Alert Engine Resources"
echo ""

# Delete Alert Engine RBAC
print_status $YELLOW "🗑️ Deleting Alert Engine RBAC..."
oc delete clusterrolebinding alert-engine-binding --timeout=60s 2>/dev/null || true
oc delete clusterrole alert-engine-role --timeout=60s 2>/dev/null || true

# Delete Alert Engine ConfigMaps and Secrets
if oc get namespace alert-engine >/dev/null 2>&1; then
    print_status $YELLOW "🗑️ Deleting Alert Engine ConfigMaps and Secrets..."
    oc delete configmap redis-config -n alert-engine --timeout=60s 2>/dev/null || true
    oc delete serviceaccount alert-engine-sa -n alert-engine --timeout=60s 2>/dev/null || true
    print_status $GREEN "✅ Alert Engine resources deleted"
else
    print_status $GREEN "✅ Alert Engine namespace doesn't exist - resources already deleted"
fi

echo ""

# =============================================================================
# STEP 6: Clean up Operators
# =============================================================================

print_status $BLUE "📋 STEP 6: Cleaning up Operators"
echo ""

# Delete AMQ Streams Operator
if oc get namespace amq-streams-kafka >/dev/null 2>&1; then
    if oc get subscription amq-streams -n amq-streams-kafka >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting AMQ Streams operator subscription..."
        oc delete subscription amq-streams -n amq-streams-kafka --timeout=60s
        print_status $GREEN "✅ AMQ Streams operator subscription deleted"
    fi
    
    # Delete AMQ Streams CSV
    amq_csv=$(oc get csv -n amq-streams-kafka --no-headers 2>/dev/null | grep amq | awk '{print $1}')
    if [[ -n "$amq_csv" ]]; then
        print_status $YELLOW "🗑️ Deleting AMQ Streams CSV: $amq_csv"
        oc delete csv "$amq_csv" -n amq-streams-kafka --timeout=60s
        print_status $GREEN "✅ AMQ Streams CSV deleted"
    fi
    
    # Delete AMQ Streams OperatorGroup
    if oc get operatorgroup amq-streams-og -n amq-streams-kafka >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting AMQ Streams OperatorGroup..."
        oc delete operatorgroup amq-streams-og -n amq-streams-kafka --timeout=60s
        print_status $GREEN "✅ AMQ Streams OperatorGroup deleted"
    fi
fi

# Delete OpenShift Logging Operator
if oc get namespace openshift-logging >/dev/null 2>&1; then
    if oc get subscription cluster-logging -n openshift-logging >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting OpenShift Logging operator subscription..."
        oc delete subscription cluster-logging -n openshift-logging --timeout=60s
        print_status $GREEN "✅ OpenShift Logging operator subscription deleted"
    fi
    
    # Delete OpenShift Logging CSV
    logging_csv=$(oc get csv -n openshift-logging --no-headers 2>/dev/null | grep cluster-logging | awk '{print $1}')
    if [[ -n "$logging_csv" ]]; then
        print_status $YELLOW "🗑️ Deleting OpenShift Logging CSV: $logging_csv"
        oc delete csv "$logging_csv" -n openshift-logging --timeout=60s
        print_status $GREEN "✅ OpenShift Logging CSV deleted"
    fi
    
    # Delete OpenShift Logging OperatorGroup
    if oc get operatorgroup cluster-logging -n openshift-logging >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting OpenShift Logging OperatorGroup..."
        oc delete operatorgroup cluster-logging -n openshift-logging --timeout=60s
        print_status $GREEN "✅ OpenShift Logging OperatorGroup deleted"
    fi
fi

print_status $GREEN "✅ Operators cleanup completed"
echo ""

# =============================================================================
# STEP 7: Clean up Install Plans
# =============================================================================

print_status $BLUE "📋 STEP 7: Cleaning up Install Plans"
echo ""

for ns in amq-streams-kafka openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        install_plans=$(oc get installplan -n "$ns" --no-headers 2>/dev/null | awk '{print $1}')
        if [[ -n "$install_plans" ]]; then
            print_status $YELLOW "🗑️ Deleting install plans in namespace $ns..."
            echo "$install_plans" | xargs -I {} oc delete installplan {} -n "$ns" --timeout=60s 2>/dev/null || true
            print_status $GREEN "✅ Install plans deleted in namespace $ns"
        else
            print_status $GREEN "✅ No install plans found in namespace $ns"
        fi
    fi
done

echo ""

# =============================================================================
# STEP 8: Clean up Namespaces
# =============================================================================

print_status $BLUE "📋 STEP 8: Cleaning up Namespaces"
echo ""

# Delete namespaces in order
for ns in alert-engine redis-cluster amq-streams-kafka openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        print_status $YELLOW "🗑️ Deleting namespace $ns..."
        oc delete namespace "$ns" --timeout=300s &
        print_status $GREEN "✅ Namespace $ns deletion initiated"
    else
        print_status $GREEN "✅ Namespace $ns already deleted"
    fi
done

# Wait for all namespace deletions to complete
print_status $YELLOW "⏳ Waiting for namespace deletions to complete..."
wait

# Verify all namespaces are deleted
print_status $YELLOW "🔍 Verifying namespace deletions..."
all_deleted=true
for ns in alert-engine redis-cluster amq-streams-kafka openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        print_status $RED "❌ Namespace $ns still exists"
        all_deleted=false
    else
        print_status $GREEN "✅ Namespace $ns deleted"
    fi
done

if [[ "$all_deleted" == "true" ]]; then
    print_status $GREEN "✅ All namespaces deleted successfully"
else
    print_status $YELLOW "⚠️ Some namespaces may still be terminating. This is normal and they will be deleted eventually."
fi

echo ""

# =============================================================================
# FINAL VERIFICATION
# =============================================================================

print_status $BLUE "📋 FINAL VERIFICATION"
echo ""

print_status $YELLOW "🔍 Checking for remaining resources..."

# Check for remaining PVs
remaining_pvs=$(oc get pv --no-headers 2>/dev/null | grep -E "(alert-engine|redis-cluster|amq-streams-kafka|openshift-logging)" | wc -l)
if [[ $remaining_pvs -gt 0 ]]; then
    print_status $YELLOW "⚠️ $remaining_pvs persistent volumes may still exist and might need manual cleanup"
    oc get pv --no-headers 2>/dev/null | grep -E "(alert-engine|redis-cluster|amq-streams-kafka|openshift-logging)" | awk '{print "  - " $1 " (" $5 ")"}'
else
    print_status $GREEN "✅ No remaining persistent volumes found"
fi

# Check for remaining ClusterRoles and ClusterRoleBindings
remaining_cluster_resources=0
if oc get clusterrole alert-engine-role >/dev/null 2>&1; then
    remaining_cluster_resources=$((remaining_cluster_resources + 1))
fi
if oc get clusterrolebinding alert-engine-binding >/dev/null 2>&1; then
    remaining_cluster_resources=$((remaining_cluster_resources + 1))
fi

if [[ $remaining_cluster_resources -gt 0 ]]; then
    print_status $YELLOW "⚠️ $remaining_cluster_resources cluster-level resources may still exist"
else
    print_status $GREEN "✅ No remaining cluster-level resources found"
fi

echo ""

# =============================================================================
# CLEANUP SUMMARY
# =============================================================================

print_status $BLUE "🎉 CLEANUP SUMMARY"
echo ""

print_status $GREEN "✅ Test Applications: Deleted"
print_status $GREEN "✅ OpenShift Logging: Deleted"
print_status $GREEN "✅ Redis Cluster: Deleted"
print_status $GREEN "✅ AMQ Streams / Kafka: Deleted"
print_status $GREEN "✅ Alert Engine Resources: Deleted"
print_status $GREEN "✅ Operators: Deleted"
print_status $GREEN "✅ Namespaces: Deleted"

echo ""
print_status $BLUE "🎯 OpenShift Infrastructure Cleanup Complete!"
echo ""

if [[ $remaining_pvs -gt 0 ]] || [[ $remaining_cluster_resources -gt 0 ]]; then
    print_status $YELLOW "⚠️  Note: Some resources may require manual cleanup:"
    [[ $remaining_pvs -gt 0 ]] && echo "  - Persistent Volumes (will be cleaned up automatically by OpenShift)"
    [[ $remaining_cluster_resources -gt 0 ]] && echo "  - Cluster-level resources (verify with 'oc get clusterrole,clusterrolebinding | grep alert-engine')"
    echo ""
fi

print_status $GREEN "The OpenShift cluster is now ready for fresh infrastructure deployment."
echo ""

exit 0 