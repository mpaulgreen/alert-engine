# Alert Engine OpenShift Infrastructure Cleanup

This repository contains comprehensive tools for cleaning up the Alert Engine infrastructure from your OpenShift cluster. The cleanup removes all components installed by the [OpenShift Infrastructure Setup Guide](alert-engine/OPENSHIFT_SETUP.md).

## 🚨 **IMPORTANT WARNING**

**This cleanup will permanently delete:**
- All Alert Engine applications and data
- Kafka clusters and all stored messages
- Redis Enterprise clusters and all cached data
- OpenShift Logging configurations
- All persistent volumes and data
- All operators and configurations

**⚠️ Ensure you have backups of any important data before proceeding!**

## 📋 Components to be Removed

The cleanup process removes all infrastructure components installed by the setup guide:

### 1. **Test Applications**
- `continuous-log-generator` deployment
- All test pods and configurations

### 2. **ClusterLogForwarder & Logging**
- `kafka-forwarder` ClusterLogForwarder
- Vector collector pods (DaemonSet)
- OpenShift Logging Operator
- Service accounts and RBAC

### 3. **Redis Enterprise**
- `alert-engine-cache` database
- `rec-alert-engine` cluster (3 nodes)
- Redis Enterprise Operator
- Security Context Constraints
- All persistent volumes and data

### 4. **Kafka/AMQ Streams**
- `alert-kafka-cluster` (3 brokers + 3 zookeepers)
- `application-logs` topic
- AMQ Streams Operator
- All persistent volumes and data

### 5. **Network & Security**
- Network policies
- Service accounts and RBAC
- Security Context Constraints
- ClusterRoles and ClusterRoleBindings

### 6. **Namespaces**
- `alert-engine`
- `amq-streams-kafka`
- `redis-enterprise`
- `openshift-logging`

## 🛠️ Cleanup Tools

### 1. **Verification Script** (`verify_resources_before_cleanup.sh`)
Checks what infrastructure resources currently exist in your cluster.

```bash
./verify_resources_before_cleanup.sh
```

**Use this to:**
- See what resources exist before cleanup
- Verify cleanup was successful
- Understand the scope of cleanup needed

### 2. **Automated Cleanup Script** (`cleanup_openshift_infrastructure.sh`)
Comprehensive automated cleanup that removes all components in the correct order.

```bash
./cleanup_openshift_infrastructure.sh
```

**Features:**
- ✅ Removes resources in proper dependency order
- ✅ Includes safety confirmations
- ✅ Handles stuck resources gracefully
- ✅ Provides detailed progress feedback
- ✅ Verifies cleanup completion
- ✅ Includes troubleshooting guidance

### 3. **Manual Cleanup Guide** (`manual_cleanup_reference.md`)
Detailed manual commands for troubleshooting and edge cases.

**Use when:**
- Automated script fails
- Resources are stuck
- Need granular control
- Troubleshooting specific issues

## 🚀 Quick Start

### Step 1: Verify Current State
```bash
# Check what resources exist
./verify_resources_before_cleanup.sh
```

### Step 2: Run Automated Cleanup
```bash
# Run the comprehensive cleanup
./cleanup_openshift_infrastructure.sh
```

### Step 3: Verify Cleanup
```bash
# Verify all resources are removed
./verify_resources_before_cleanup.sh
```

## 📊 Expected Cleanup Process

The automated cleanup follows this sequence:

1. **Stop Applications** (1-2 minutes)
   - Stop test applications
   - Wait for pods to terminate

2. **Remove ClusterLogForwarder** (2-3 minutes)
   - Delete ClusterLogForwarder
   - Remove Vector collector pods
   - Clean up logging RBAC

3. **Remove Redis Enterprise** (3-5 minutes)
   - Delete Redis database
   - Delete Redis cluster
   - Remove operator and SCC

4. **Remove Kafka** (3-5 minutes)
   - Delete Kafka topics
   - Delete Kafka cluster
   - Remove operator

5. **Remove Operators** (2-3 minutes)
   - Remove subscriptions
   - Remove CSVs
   - Remove operator groups

6. **Clean Resources** (2-3 minutes)
   - Remove PVCs and services
   - Remove install plans
   - Clean up stuck pods

7. **Remove Namespaces** (1-2 minutes)
   - Delete all namespaces
   - Force remove if stuck

8. **Verify Cleanup** (1 minute)
   - Check all resources removed
   - Report final status

**Total Time: 15-25 minutes**

## 🔧 Troubleshooting

### Common Issues

**Issue 1: Namespaces Stuck in Terminating**
```bash
# Check namespace status
oc get namespace <namespace> -o yaml | grep finalizers

# Force delete
oc patch namespace <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

**Issue 2: Operators Won't Uninstall**
```bash
# Check CSV status
oc get csv -n <namespace>

# Force delete
oc delete csv <csv-name> -n <namespace> --force --grace-period=0
```

**Issue 3: Custom Resources Stuck**
```bash
# Check resource status
oc get <resource-type> <resource-name> -n <namespace> -o yaml

# Remove finalizers
oc patch <resource-type> <resource-name> -n <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

**Issue 4: PVCs Won't Delete**
```bash
# Force delete pods first
oc delete pods --all -n <namespace> --force --grace-period=0

# Then delete PVCs
oc delete pvc --all -n <namespace> --force --grace-period=0
```

### Manual Cleanup Commands

If automated cleanup fails, use the manual cleanup guide:

```bash
# View detailed manual commands
cat manual_cleanup_reference.md

# Or use individual commands from the guide
```

### Nuclear Option

⚠️ **Last resort for completely stuck resources:**

```bash
# Force delete everything (USE WITH EXTREME CAUTION)
for ns in alert-engine amq-streams-kafka redis-enterprise openshift-logging; do
    if oc get namespace "$ns" >/dev/null 2>&1; then
        oc delete all --all -n "$ns" --force --grace-period=0
        oc patch namespace "$ns" -p '{"metadata":{"finalizers":[]}}' --type=merge
        oc delete namespace "$ns" --force --grace-period=0
    fi
done
```

## ✅ Success Criteria

Cleanup is successful when:

1. **All namespaces removed:**
   ```bash
   oc get namespaces | grep -E "(alert-engine|amq-streams|redis-enterprise|openshift-logging)"
   # Should return no results
   ```

2. **No operator CSVs remain:**
   ```bash
   oc get csv -A | grep -E "(amq|redis|cluster-logging)"
   # Should return no results
   ```

3. **Cluster-level resources removed:**
   ```bash
   oc get clusterrole | grep -E "(alert-engine|redis|kafka)"
   oc get clusterrolebinding | grep -E "(alert-engine|redis|kafka)"
   oc get securitycontextconstraints | grep redis
   # Should return no results
   ```

4. **Verification script confirms clean state:**
   ```bash
   ./verify_resources_before_cleanup.sh
   # Should show "No Alert Engine resources found"
   ```

## 📚 File Reference

| File | Purpose |
|------|---------|
| `verify_resources_before_cleanup.sh` | Pre-cleanup verification |
| `cleanup_openshift_infrastructure.sh` | Automated cleanup script |
| `manual_cleanup_reference.md` | Manual cleanup commands |
| `README_CLEANUP.md` | This guide |

## 🔒 Safety Features

All cleanup tools include:

- **Confirmation prompts** before destructive operations
- **Resource existence checks** before deletion attempts
- **Graceful handling** of missing resources
- **Progress feedback** throughout the process
- **Verification steps** to confirm successful cleanup
- **Troubleshooting guidance** for common issues

## 💾 Backup Recommendations

Before running cleanup, consider backing up:

1. **Application configurations:**
   ```bash
   oc get configmap -n alert-engine -o yaml > alert-engine-configs.yaml
   ```

2. **Kafka topics (if needed):**
   ```bash
   # Export important messages from Kafka topics
   oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
     bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 \
     --topic application-logs --from-beginning > kafka-backup.json
   ```

3. **Redis data (if needed):**
   ```bash
   # Export Redis data
   oc exec -n redis-enterprise <redis-pod> -- redis-cli BGSAVE
   ```

## 🤝 Support

If you encounter issues:

1. **Check the troubleshooting section** above
2. **Review the manual cleanup guide** for specific commands
3. **Consult OpenShift documentation** for operator-specific procedures
4. **Contact your OpenShift administrator** for cluster-level issues

## 🎯 Next Steps

After successful cleanup:

1. **Verify cluster health:**
   ```bash
   oc get nodes
   oc get pods -A | grep -E "(Error|CrashLoopBackOff|Pending)"
   ```

2. **Check available storage:**
   ```bash
   oc get pv | grep Available
   ```

3. **Your cluster is now clean** and ready for new deployments!

---

**⚠️ Remember: Always test cleanup procedures in non-production environments first!** 