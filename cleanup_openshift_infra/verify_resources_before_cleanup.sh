#!/bin/bash

echo "=== Pre-Cleanup Resource Verification ==="
echo "This script checks what Alert Engine infrastructure resources exist in the cluster"
echo ""

# Function to check resource existence
check_resource() {
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

# Function to count resources
count_resources() {
    local resource_type=$1
    local namespace=$2
    
    if [[ -n "$namespace" ]]; then
        if oc get namespace "$namespace" >/dev/null 2>&1; then
            count=$(oc get "$resource_type" -n "$namespace" --no-headers 2>/dev/null | wc -l)
            echo "  $count $resource_type(s) found in namespace $namespace"
        else
            echo "  Namespace $namespace not found"
        fi
    else
        count=$(oc get "$resource_type" --no-headers 2>/dev/null | wc -l)
        echo "  $count $resource_type(s) found cluster-wide"
    fi
}

echo "1. 🏠 Checking Namespaces..."
for ns in alert-engine amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        echo "  ✅ Namespace $ns exists"
    else
        echo "  ❌ Namespace $ns does not exist"
    fi
done
echo ""

echo "2. 🚨 Checking Alert Engine Resources..."
check_resource "deployment" "continuous-log-generator" "alert-engine"
check_resource "configmap" "redis-config" "alert-engine"
check_resource "secret" "redis-password" "alert-engine"
check_resource "secret" "redb-alert-engine-cache" "alert-engine"
check_resource "serviceaccount" "alert-engine-sa" "alert-engine"
check_resource "clusterrole" "alert-engine-role" ""
check_resource "clusterrolebinding" "alert-engine-binding" ""
echo ""

echo "3. 🟡 Checking Kafka Resources..."
check_resource "kafka" "alert-kafka-cluster" "amq-streams-kafka"
check_resource "kafkatopic" "application-logs" "amq-streams-kafka"
check_resource "networkpolicy" "kafka-network-policy" "amq-streams-kafka"
check_resource "subscription" "amq-streams" "amq-streams-kafka"
check_resource "operatorgroup" "amq-streams-og" "amq-streams-kafka"
echo ""

echo "4. 🔴 Checking Redis Enterprise Resources..."
check_resource "redisenterprisecluster" "rec-alert-engine" "redis-enterprise"
check_resource "redisenterprisedatabase" "alert-engine-cache" "redis-enterprise"
check_resource "networkpolicy" "redis-network-policy" "redis-enterprise"
check_resource "subscription" "redis-enterprise-operator" "redis-enterprise"
check_resource "operatorgroup" "redis-enterprise-og" "redis-enterprise"
check_resource "securitycontextconstraints" "redis-enterprise-scc" ""
echo ""

echo "5. 📝 Checking Logging Resources..."
check_resource "clusterlogforwarder" "kafka-forwarder" "openshift-logging"
check_resource "serviceaccount" "log-collector" "openshift-logging"
check_resource "subscription" "cluster-logging" "openshift-logging"
check_resource "operatorgroup" "cluster-logging" "openshift-logging"
check_resource "clusterrolebinding" "log-collector-application-logs" ""
check_resource "clusterrolebinding" "log-collector-write-logs" ""
echo ""

echo "6. 📦 Checking Operator CSVs..."
echo "  AMQ Streams CSVs:"
oc get csv -n amq-streams-kafka --no-headers 2>/dev/null | grep amq | awk '{print "    " $1 " - " $4}'
echo "  Redis Enterprise CSVs:"
oc get csv -n redis-enterprise --no-headers 2>/dev/null | grep redis | awk '{print "    " $1 " - " $4}'
echo "  Logging CSVs:"
oc get csv -n openshift-logging --no-headers 2>/dev/null | grep cluster-logging | awk '{print "    " $1 " - " $4}'
echo ""

echo "7. 🔧 Checking Running Pods..."
for ns in alert-engine amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        echo "  Namespace $ns:"
        pod_count=$(oc get pods -n "$ns" --no-headers 2>/dev/null | wc -l)
        running_pods=$(oc get pods -n "$ns" --no-headers 2>/dev/null | grep Running | wc -l)
        echo "    Total pods: $pod_count"
        echo "    Running pods: $running_pods"
        if [[ "$pod_count" -gt 0 ]]; then
            echo "    Pod details:"
            oc get pods -n "$ns" --no-headers 2>/dev/null | awk '{print "      " $1 " - " $3}'
        fi
    fi
done
echo ""

echo "8. 💾 Checking Persistent Volumes and Claims..."
for ns in amq-streams-kafka redis-enterprise openshift-logging alert-engine; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        pvc_count=$(oc get pvc -n "$ns" --no-headers 2>/dev/null | wc -l)
        if [[ "$pvc_count" -gt 0 ]]; then
            echo "  Namespace $ns has $pvc_count PVC(s):"
            oc get pvc -n "$ns" --no-headers 2>/dev/null | awk '{print "    " $1 " - " $2}'
        fi
    fi
done
echo ""

echo "9. 🌐 Checking Services..."
for ns in amq-streams-kafka redis-enterprise openshift-logging alert-engine; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        svc_count=$(oc get svc -n "$ns" --no-headers 2>/dev/null | wc -l)
        if [[ "$svc_count" -gt 0 ]]; then
            echo "  Namespace $ns has $svc_count service(s):"
            oc get svc -n "$ns" --no-headers 2>/dev/null | awk '{print "    " $1 " - " $2 ":" $5}'
        fi
    fi
done
echo ""

echo "10. 📋 Checking Install Plans..."
for ns in amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        ip_count=$(oc get installplan -n "$ns" --no-headers 2>/dev/null | wc -l)
        if [[ "$ip_count" -gt 0 ]]; then
            echo "  Namespace $ns has $ip_count install plan(s):"
            oc get installplan -n "$ns" --no-headers 2>/dev/null | awk '{print "    " $1 " - " $3}'
        fi
    fi
done
echo ""

# Summary
echo "=== 📊 Summary ==="
echo ""

total_resources=0
for ns in alert-engine amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        ns_resources=$(oc get all -n "$ns" --no-headers 2>/dev/null | wc -l)
        echo "Namespace $ns: $ns_resources resources"
        total_resources=$((total_resources + ns_resources))
    fi
done

# Check cluster-level resources
cluster_resources=0
if oc get clusterrole alert-engine-role >/dev/null 2>&1; then
    cluster_resources=$((cluster_resources + 1))
fi
if oc get clusterrolebinding alert-engine-binding >/dev/null 2>&1; then
    cluster_resources=$((cluster_resources + 1))
fi
if oc get securitycontextconstraints redis-enterprise-scc >/dev/null 2>&1; then
    cluster_resources=$((cluster_resources + 1))
fi

echo "Cluster-level resources: $cluster_resources"
echo "Total resources to cleanup: $((total_resources + cluster_resources))"

echo ""
if [[ "$total_resources" -gt 0 ]] || [[ "$cluster_resources" -gt 0 ]]; then
    echo "🔄 Resources found that need cleanup. You can proceed with the cleanup script."
    echo "   Run: ./cleanup_openshift_infrastructure.sh"
else
    echo "🎉 No Alert Engine resources found. The cluster appears to be clean."
fi
echo ""
echo "=== Verification Complete ===" 