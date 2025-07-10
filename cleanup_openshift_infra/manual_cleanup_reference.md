# Manual Cleanup Reference Guide

This guide provides manual cleanup commands for troubleshooting and manual removal of Alert Engine infrastructure resources when the automated cleanup script fails or needs manual intervention.

## Quick Start

1. **Check what resources exist**: `./verify_resources_before_cleanup.sh`
2. **Run automated cleanup**: `./cleanup_openshift_infrastructure.sh`
3. **If automation fails**: Use manual commands below

## Manual Cleanup Commands

### 1. Test Applications

```bash
# Remove test applications
oc delete deployment continuous-log-generator -n alert-engine --ignore-not-found=true
oc delete pod --all -n alert-engine --force --grace-period=0
```

### 2. ClusterLogForwarder and Logging

```bash
# Remove ClusterLogForwarder
oc delete clusterlogforwarder kafka-forwarder -n openshift-logging --ignore-not-found=true

# Remove logging service account and RBAC
oc delete serviceaccount log-collector -n openshift-logging --ignore-not-found=true
oc delete clusterrolebinding log-collector-application-logs --ignore-not-found=true
oc delete clusterrolebinding log-collector-write-logs --ignore-not-found=true

# Force delete Vector collector pods
oc delete pods -l app.kubernetes.io/instance=kafka-forwarder -n openshift-logging --force --grace-period=0
```

### 3. Redis Enterprise Resources

```bash
# Remove Redis database (must be done before cluster)
oc delete redisenterprisedatabase alert-engine-cache -n redis-enterprise --ignore-not-found=true

# Wait for database removal, then remove cluster
oc delete redisenterprisecluster rec-alert-engine -n redis-enterprise --ignore-not-found=true

# Remove Redis network policy
oc delete networkpolicy redis-network-policy -n redis-enterprise --ignore-not-found=true

# Remove Redis SCC
oc delete securitycontextconstraints redis-enterprise-scc --ignore-not-found=true

# Force delete stuck Redis pods
oc delete pods --all -n redis-enterprise --force --grace-period=0
```

### 4. Kafka Resources

```bash
# Remove Kafka topics
oc delete kafkatopic application-logs -n amq-streams-kafka --ignore-not-found=true

# Remove Kafka cluster
oc delete kafka alert-kafka-cluster -n amq-streams-kafka --ignore-not-found=true

# Remove Kafka network policy
oc delete networkpolicy kafka-network-policy -n amq-streams-kafka --ignore-not-found=true

# Force delete stuck Kafka pods
oc delete pods --all -n amq-streams-kafka --force --grace-period=0
```

### 5. Operators

```bash
# Remove operator subscriptions
oc delete subscription cluster-logging -n openshift-logging --ignore-not-found=true
oc delete subscription redis-enterprise-operator -n redis-enterprise --ignore-not-found=true
oc delete subscription amq-streams -n amq-streams-kafka --ignore-not-found=true

# Remove operator groups
oc delete operatorgroup cluster-logging -n openshift-logging --ignore-not-found=true
oc delete operatorgroup redis-enterprise-og -n redis-enterprise --ignore-not-found=true
oc delete operatorgroup amq-streams-og -n amq-streams-kafka --ignore-not-found=true

# Remove CSVs (find specific names first)
oc get csv -n amq-streams-kafka | grep amq
oc get csv -n redis-enterprise | grep redis
oc get csv -n openshift-logging | grep cluster-logging

# Delete specific CSVs (replace with actual names)
oc delete csv <csv-name> -n amq-streams-kafka
oc delete csv <csv-name> -n redis-enterprise
oc delete csv <csv-name> -n openshift-logging
```

### 6. Alert Engine Resources

```bash
# Remove Alert Engine application resources
oc delete configmap redis-config -n alert-engine --ignore-not-found=true
oc delete secret redis-password -n alert-engine --ignore-not-found=true
oc delete secret redb-alert-engine-cache -n alert-engine --ignore-not-found=true
oc delete serviceaccount alert-engine-sa -n alert-engine --ignore-not-found=true

# Remove cluster-level RBAC
oc delete clusterrole alert-engine-role --ignore-not-found=true
oc delete clusterrolebinding alert-engine-binding --ignore-not-found=true
```

### 7. Persistent Storage

```bash
# Remove PVCs (WARNING: This deletes all data)
oc delete pvc --all -n amq-streams-kafka
oc delete pvc --all -n redis-enterprise
oc delete pvc --all -n openshift-logging
oc delete pvc --all -n alert-engine

# Remove services
oc delete svc --all -n amq-streams-kafka
oc delete svc --all -n redis-enterprise
oc delete svc --all -n openshift-logging
oc delete svc --all -n alert-engine
```

### 8. Install Plans

```bash
# Remove install plans
oc delete installplan --all -n amq-streams-kafka
oc delete installplan --all -n redis-enterprise
oc delete installplan --all -n openshift-logging
```

### 9. Namespaces

```bash
# Remove namespaces (do this last)
oc delete namespace alert-engine
oc delete namespace amq-streams-kafka
oc delete namespace redis-enterprise
oc delete namespace openshift-logging
```

## Troubleshooting Stuck Resources

### Stuck Namespaces

```bash
# Check namespace status
oc get namespace <namespace> -o yaml | grep -A 20 status

# Force delete stuck namespace
oc patch namespace <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge

# Alternative: Edit namespace directly
oc edit namespace <namespace>
# Remove finalizers array from metadata
```

### Stuck Pods

```bash
# Force delete stuck pods
oc delete pod <pod-name> -n <namespace> --force --grace-period=0

# If still stuck, patch the pod
oc patch pod <pod-name> -n <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

### Stuck PVCs

```bash
# Force delete stuck PVCs
oc delete pvc <pvc-name> -n <namespace> --force --grace-period=0

# If still stuck, patch the PVC
oc patch pvc <pvc-name> -n <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

### Stuck Operators

```bash
# Force delete stuck CSVs
oc delete csv <csv-name> -n <namespace> --force --grace-period=0

# Remove operator finalizers
oc patch csv <csv-name> -n <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

### Stuck Custom Resources

```bash
# For stuck Kafka clusters
oc patch kafka alert-kafka-cluster -n amq-streams-kafka -p '{"metadata":{"finalizers":[]}}' --type=merge

# For stuck Redis clusters
oc patch redisenterprisecluster rec-alert-engine -n redis-enterprise -p '{"metadata":{"finalizers":[]}}' --type=merge

# For stuck Redis databases
oc patch redisenterprisedatabase alert-engine-cache -n redis-enterprise -p '{"metadata":{"finalizers":[]}}' --type=merge
```

## Verification Commands

### Check Remaining Resources

```bash
# Check all namespaces
oc get namespaces | grep -E "(alert-engine|amq-streams|redis-enterprise|openshift-logging)"

# Check cluster-level resources
oc get clusterrole | grep -E "(alert-engine|redis|kafka)"
oc get clusterrolebinding | grep -E "(alert-engine|redis|kafka)"
oc get securitycontextconstraints | grep redis

# Check remaining CSVs
oc get csv -A | grep -E "(amq|redis|cluster-logging)"

# Check remaining operators
oc get subscription -A | grep -E "(amq|redis|cluster-logging)"
oc get operatorgroup -A | grep -E "(amq|redis|cluster-logging)"
```

### Check Specific Resources

```bash
# Check Kafka resources
oc get kafka -A
oc get kafkatopic -A

# Check Redis resources
oc get redisenterprisecluster -A
oc get redisenterprisedatabase -A

# Check logging resources
oc get clusterlogforwarder -A
```

## Nuclear Option - Complete Cleanup

⚠️ **WARNING: This will forcefully remove ALL resources, potentially causing data loss**

```bash
# Force delete all resources in namespaces
for ns in alert-engine amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        echo "Force cleaning namespace $ns"
        oc delete all --all -n "$ns" --force --grace-period=0
        oc delete pvc --all -n "$ns" --force --grace-period=0
        oc delete secret --all -n "$ns" --force --grace-period=0
        oc delete configmap --all -n "$ns" --force --grace-period=0
        oc delete serviceaccount --all -n "$ns" --force --grace-period=0
        oc delete rolebinding --all -n "$ns" --force --grace-period=0
        oc delete role --all -n "$ns" --force --grace-period=0
        oc delete networkpolicy --all -n "$ns" --force --grace-period=0
        
        # Force delete namespace
        oc patch namespace "$ns" -p '{"metadata":{"finalizers":[]}}' --type=merge
        oc delete namespace "$ns" --force --grace-period=0
    fi
done

# Remove cluster-level resources
oc delete clusterrole alert-engine-role --ignore-not-found=true
oc delete clusterrolebinding alert-engine-binding --ignore-not-found=true
oc delete clusterrolebinding log-collector-application-logs --ignore-not-found=true
oc delete clusterrolebinding log-collector-write-logs --ignore-not-found=true
oc delete securitycontextconstraints redis-enterprise-scc --ignore-not-found=true
```

## Post-Cleanup Verification

```bash
# Run verification script
./verify_resources_before_cleanup.sh

# Check cluster is clean
oc get namespaces | grep -E "(alert-engine|amq-streams|redis-enterprise|openshift-logging)"
oc get csv -A | grep -E "(amq|redis|cluster-logging)"
oc get clusterrole | grep -E "(alert-engine|redis|kafka)"
oc get clusterrolebinding | grep -E "(alert-engine|redis|kafka)"

# Should return no results if cleanup successful
```

## Common Issues and Solutions

### Issue 1: Namespace Stuck in Terminating

**Cause**: Finalizers preventing deletion
**Solution**: Remove finalizers manually

```bash
oc patch namespace <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

### Issue 2: Operator Won't Uninstall

**Cause**: CSV or subscription stuck
**Solution**: Force delete operator resources

```bash
oc delete csv <csv-name> -n <namespace> --force --grace-period=0
oc delete subscription <subscription-name> -n <namespace> --force --grace-period=0
```

### Issue 3: Custom Resources Stuck

**Cause**: Controller not responding
**Solution**: Patch finalizers

```bash
oc patch <resource-type> <resource-name> -n <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

### Issue 4: PVCs Won't Delete

**Cause**: Pod still using PVC
**Solution**: Force delete pods first

```bash
oc delete pods --all -n <namespace> --force --grace-period=0
oc delete pvc --all -n <namespace> --force --grace-period=0
```

## Support

If manual cleanup fails, consider:
1. Contacting your OpenShift administrator
2. Checking OpenShift documentation for specific operator removal procedures
3. Using OpenShift support channels

## Safety Notes

- Always backup important data before cleanup
- Test cleanup procedures in non-production environments first
- Monitor cluster health during cleanup process
- Be cautious with force deletion commands 