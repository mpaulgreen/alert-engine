#!/bin/bash
set -e

echo "=== OpenShift Infrastructure Cleanup Script ==="
echo "This script will remove all Alert Engine infrastructure components"
echo "⚠️  WARNING: This will permanently delete all data and configurations!"
echo ""

# Confirmation prompt
read -p "Are you sure you want to proceed with cleanup? (yes/no): " confirm
if [[ "$confirm" != "yes" ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

echo "Starting cleanup process..."
echo ""

# Function to check if resource exists before deletion
check_and_delete() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3
    
    if [[ -n "$namespace" ]]; then
        if oc get "$resource_type" "$resource_name" -n "$namespace" >/dev/null 2>&1; then
            echo "  🗑️  Deleting $resource_type/$resource_name in namespace $namespace"
            oc delete "$resource_type" "$resource_name" -n "$namespace" --ignore-not-found=true
        else
            echo "  ✅ $resource_type/$resource_name not found in namespace $namespace"
        fi
    else
        if oc get "$resource_type" "$resource_name" >/dev/null 2>&1; then
            echo "  🗑️  Deleting $resource_type/$resource_name (cluster-wide)"
            oc delete "$resource_type" "$resource_name" --ignore-not-found=true
        else
            echo "  ✅ $resource_type/$resource_name not found (cluster-wide)"
        fi
    fi
}

# 1. STOP AND REMOVE TEST APPLICATIONS
echo "1. 🛑 Stopping Test Applications..."
check_and_delete "deployment" "continuous-log-generator" "alert-engine"
echo ""

# 2. REMOVE CLUSTERLOGFORWARDER AND LOGGING CONFIGURATION
echo "2. 📝 Removing ClusterLogForwarder and Logging Configuration..."
check_and_delete "clusterlogforwarder" "kafka-forwarder" "openshift-logging"

# Wait for Vector pods to terminate
echo "  ⏳ Waiting for Vector collector pods to terminate..."
sleep 15

# Remove logging service account and RBAC
check_and_delete "serviceaccount" "log-collector" "openshift-logging"
check_and_delete "clusterrolebinding" "log-collector-application-logs" ""
check_and_delete "clusterrolebinding" "log-collector-write-logs" ""
echo ""

# 3. REMOVE REDIS ENTERPRISE RESOURCES
echo "3. 🔴 Removing Redis Enterprise Resources..."

# Remove Redis database first
check_and_delete "redisenterprisedatabase" "alert-engine-cache" "redis-enterprise"

# Wait for database to be removed
echo "  ⏳ Waiting for Redis database to be removed..."
sleep 10

# Remove Redis cluster
check_and_delete "redisenterprisecluster" "rec-alert-engine" "redis-enterprise"

# Wait for cluster to be removed
echo "  ⏳ Waiting for Redis cluster to be removed..."
sleep 20

# Remove Redis network policy
check_and_delete "networkpolicy" "redis-network-policy" "redis-enterprise"

# Remove Redis SCC
check_and_delete "securitycontextconstraints" "redis-enterprise-scc" ""
echo ""

# 4. REMOVE KAFKA RESOURCES
echo "4. 🟡 Removing Kafka Resources..."

# Remove Kafka topics
check_and_delete "kafkatopic" "application-logs" "amq-streams-kafka"

# Remove Kafka cluster
check_and_delete "kafka" "alert-kafka-cluster" "amq-streams-kafka"

# Wait for Kafka cluster to be removed
echo "  ⏳ Waiting for Kafka cluster to be removed..."
sleep 20

# Remove Kafka network policy
check_and_delete "networkpolicy" "kafka-network-policy" "amq-streams-kafka"
echo ""

# 5. REMOVE OPERATORS
echo "5. 🔧 Removing Operators..."

# Remove OpenShift Logging Operator
check_and_delete "subscription" "cluster-logging" "openshift-logging"
check_and_delete "operatorgroup" "cluster-logging" "openshift-logging"

# Remove Redis Enterprise Operator
check_and_delete "subscription" "redis-enterprise-operator" "redis-enterprise"
check_and_delete "operatorgroup" "redis-enterprise-og" "redis-enterprise"

# Remove AMQ Streams Operator
check_and_delete "subscription" "amq-streams" "amq-streams-kafka"
check_and_delete "operatorgroup" "amq-streams-og" "amq-streams-kafka"

# Wait for operators to be removed
echo "  ⏳ Waiting for operators to be removed..."
sleep 30
echo ""

# 6. REMOVE CSVS (ClusterServiceVersions)
echo "6. 📦 Removing ClusterServiceVersions..."

# Get and remove AMQ Streams CSV
AMQ_CSV=$(oc get csv -n amq-streams-kafka --no-headers | grep amq | awk '{print $1}' | head -1)
if [[ -n "$AMQ_CSV" ]]; then
    check_and_delete "csv" "$AMQ_CSV" "amq-streams-kafka"
fi

# Get and remove Redis Enterprise CSV
REDIS_CSV=$(oc get csv -n redis-enterprise --no-headers | grep redis | awk '{print $1}' | head -1)
if [[ -n "$REDIS_CSV" ]]; then
    check_and_delete "csv" "$REDIS_CSV" "redis-enterprise"
fi

# Get and remove Logging CSV
LOGGING_CSV=$(oc get csv -n openshift-logging --no-headers | grep cluster-logging | awk '{print $1}' | head -1)
if [[ -n "$LOGGING_CSV" ]]; then
    check_and_delete "csv" "$LOGGING_CSV" "openshift-logging"
fi

echo ""

# 7. REMOVE ALERT ENGINE RESOURCES
echo "7. 🚨 Removing Alert Engine Resources..."
check_and_delete "configmap" "redis-config" "alert-engine"
check_and_delete "secret" "redis-password" "alert-engine"
check_and_delete "secret" "redb-alert-engine-cache" "alert-engine"
check_and_delete "serviceaccount" "alert-engine-sa" "alert-engine"
check_and_delete "clusterrole" "alert-engine-role" ""
check_and_delete "clusterrolebinding" "alert-engine-binding" ""
echo ""

# 8. REMOVE REMAINING PODS AND RESOURCES
echo "8. 🧹 Removing Remaining Pods and Resources..."

# Force delete any remaining pods
for ns in amq-streams-kafka redis-enterprise openshift-logging alert-engine; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        echo "  🗑️  Force deleting remaining pods in namespace $ns"
        oc delete pods --all -n "$ns" --force --grace-period=0 --ignore-not-found=true
    fi
done

# Remove any remaining PVCs
for ns in amq-streams-kafka redis-enterprise openshift-logging alert-engine; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        echo "  🗑️  Deleting PVCs in namespace $ns"
        oc delete pvc --all -n "$ns" --ignore-not-found=true
    fi
done

# Remove any remaining services
for ns in amq-streams-kafka redis-enterprise openshift-logging alert-engine; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        echo "  🗑️  Deleting services in namespace $ns"
        oc delete svc --all -n "$ns" --ignore-not-found=true
    fi
done

echo ""

# 9. REMOVE INSTALL PLANS
echo "9. 📋 Removing Install Plans..."
for ns in amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        echo "  🗑️  Deleting install plans in namespace $ns"
        oc delete installplan --all -n "$ns" --ignore-not-found=true
    fi
done
echo ""

# 10. FINAL CLEANUP - REMOVE NAMESPACES
echo "10. 🏠 Removing Namespaces..."
echo "  ⏳ Waiting for resources to be fully removed before deleting namespaces..."
sleep 30

# Remove namespaces in order
for ns in alert-engine amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        echo "  🗑️  Deleting namespace $ns"
        oc delete namespace "$ns" --ignore-not-found=true
    else
        echo "  ✅ Namespace $ns not found"
    fi
done

echo ""

# 11. VERIFY CLEANUP
echo "11. ✅ Verifying Cleanup..."
echo ""

# Check if namespaces are gone
echo "Checking namespaces:"
for ns in alert-engine amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        echo "  ❌ Namespace $ns still exists"
    else
        echo "  ✅ Namespace $ns removed"
    fi
done

echo ""

# Check cluster-level resources
echo "Checking cluster-level resources:"
if oc get clusterrole alert-engine-role >/dev/null 2>&1; then
    echo "  ❌ ClusterRole alert-engine-role still exists"
else
    echo "  ✅ ClusterRole alert-engine-role removed"
fi

if oc get clusterrolebinding alert-engine-binding >/dev/null 2>&1; then
    echo "  ❌ ClusterRoleBinding alert-engine-binding still exists"
else
    echo "  ✅ ClusterRoleBinding alert-engine-binding removed"
fi

if oc get securitycontextconstraints redis-enterprise-scc >/dev/null 2>&1; then
    echo "  ❌ SecurityContextConstraints redis-enterprise-scc still exists"
else
    echo "  ✅ SecurityContextConstraints redis-enterprise-scc removed"
fi

echo ""

# Check for any remaining operator resources
echo "Checking for remaining operator resources:"
REMAINING_CSV=$(oc get csv -A | grep -E "(amq|redis|cluster-logging)" | wc -l)
if [[ "$REMAINING_CSV" -gt 0 ]]; then
    echo "  ❌ $REMAINING_CSV operator CSVs still exist"
    oc get csv -A | grep -E "(amq|redis|cluster-logging)"
else
    echo "  ✅ All operator CSVs removed"
fi

echo ""

# Final summary
echo "=== 🎯 Cleanup Summary ==="
echo ""
echo "✅ Test applications stopped and removed"
echo "✅ ClusterLogForwarder and logging configuration removed"
echo "✅ Redis Enterprise cluster and database removed"
echo "✅ Kafka cluster and topics removed"
echo "✅ All operators removed"
echo "✅ Service accounts and RBAC removed"
echo "✅ Network policies removed"
echo "✅ Security context constraints removed"
echo "✅ Persistent volumes and claims removed"
echo "✅ All namespaces removed"
echo ""

# Check if any issues remain
ISSUES_FOUND=0

# Check for stuck namespaces
for ns in alert-engine amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        ((ISSUES_FOUND++))
    fi
done

# Check for remaining cluster resources
if oc get clusterrole alert-engine-role >/dev/null 2>&1; then
    ((ISSUES_FOUND++))
fi

if [[ "$ISSUES_FOUND" -eq 0 ]]; then
    echo "🎉 SUCCESS: All Alert Engine infrastructure has been completely removed!"
    echo "   Your OpenShift cluster is now clean and ready for new deployments."
else
    echo "⚠️  WARNING: $ISSUES_FOUND issues found during cleanup."
    echo "   Some resources may still be terminating or stuck."
    echo "   Manual intervention may be required for complete cleanup."
    echo ""
    echo "🔧 Troubleshooting Commands:"
    echo "   # Force delete stuck namespaces:"
    echo "   oc patch namespace <namespace> -p '{\"metadata\":{\"finalizers\":[]}}' --type=merge"
    echo ""
    echo "   # Check for remaining resources:"
    echo "   oc get csv -A | grep -E 'amq|redis|cluster-logging'"
    echo "   oc get clusterrole | grep -E 'alert-engine|redis|kafka'"
    echo "   oc get clusterrolebinding | grep -E 'alert-engine|redis|kafka'"
fi

echo ""
echo "=== Cleanup Complete ===" 