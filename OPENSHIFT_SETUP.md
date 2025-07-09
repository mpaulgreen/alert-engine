# OpenShift Infrastructure Setup Guide

This guide provides step-by-step instructions to set up the required infrastructure components on OpenShift before deploying the Alert Engine application.

## Prerequisites

- OpenShift 4.16.17 cluster with cluster-admin access
- `oc` CLI tool installed and configured
- Access to OperatorHub in your OpenShift cluster

## Overview

The Alert Engine requires the following components to be installed on OpenShift:

1. **Red Hat AMQ Streams** (Apache Kafka) - For log message streaming
2. **Redis Enterprise** - For state storage and caching
3. **ClusterLogForwarder** - For forwarding OpenShift logs to Kafka
4. **Network policies and security configurations**

## 1. Kafka Setup using Red Hat AMQ Streams

### Step 1.1: Install AMQ Streams Operator

**Using OpenShift Web Console (Recommended):**
1. Navigate to **Operators > OperatorHub** in the OpenShift web console
2. Search for "**Streams for Apache Kafka**"
3. Select the **Red Hat AMQ Streams** operator (certified)
4. Click **Install**
5. Configure installation options:
   - **Update Channel**: `stable` (for OpenShift 4.16.17)
   - **Installation Mode**: Install to specific namespace (create `amq-streams-kafka`)
   - **Update Approval**: Manual (recommended for production)
6. Click **Install** and wait for completion

**Using CLI:**
```bash
# Create namespace
oc create namespace amq-streams-kafka

# Install from OperatorHub
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: amq-streams
  namespace: amq-streams-kafka
spec:
  channel: stable
  name: amq-streams
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF
```

### Step 1.2: Validate AMQ Streams Operator Installation

After installing the operator, it's crucial to validate that it's properly installed before proceeding. **Important**: You may encounter common issues that require troubleshooting - this section includes real-world solutions.

#### Step 1.2.1: Check the Subscription Status

```bash
# Check the Subscription status (use full API resource to avoid conflicts)
oc get subscriptions.operators.coreos.com amq-streams -n amq-streams-kafka -o yaml

# Quick status check
oc get subscriptions.operators.coreos.com amq-streams -n amq-streams-kafka
```

**Expected Output:**
```
NAME          PACKAGE       SOURCE             CHANNEL
amq-streams   amq-streams   redhat-operators   stable
```

#### Step 1.2.2: Check for OperatorGroup (Critical Step)

**Common Issue**: If the operator subscription exists but no CSV is created, check for OperatorGroup:

```bash
# Check if OperatorGroup exists (required for operator installation)
oc get operatorgroup -n amq-streams-kafka
```

**If you see "No resources found"**, you need to create an OperatorGroup:

```bash
# Create required OperatorGroup
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: amq-streams-og
  namespace: amq-streams-kafka
spec:
  targetNamespaces:
  - amq-streams-kafka
EOF
```

**Expected Output:**
```
operatorgroup.operators.coreos.com/amq-streams-og created
```

#### Step 1.2.3: Check the InstallPlan (After OperatorGroup)

Wait a few moments after creating the OperatorGroup, then check:

```bash
# Check install plan
oc get installplan -n amq-streams-kafka

# Check install plan across all namespaces if not found
oc get installplan -A | grep amq-streams
```

**Expected Output:**
```
NAME            CSV                            APPROVAL    APPROVED
install-nnqn7   amqstreams.v2.9.1-0           Automatic   true
```

#### Step 1.2.4: Check the ClusterServiceVersion (CSV)

The CSV represents the actual operator installation:

```bash
# Check CSV status
oc get csv -n amq-streams-kafka

# Look specifically for AMQ Streams
oc get csv -n amq-streams-kafka | grep amq
```

**Expected Output:**
```
NAME                              DISPLAY                         VERSION   REPLACES                           PHASE
amqstreams.v2.9.1-0               Streams for Apache Kafka        2.9.1-0   amqstreams.v2.8.0-0-0.1738265624.p  Succeeded
```

**Key Status to Look For:**
- `PHASE` should be `Succeeded`
- `DISPLAY` should show "Streams for Apache Kafka"

#### Step 1.2.5: Verify Operator Pod is Running

```bash
# Check operator pods
oc get pods -n amq-streams-kafka

# Check operator deployment
oc get deployment -n amq-streams-kafka
```

**Expected Output:**
```
NAME                                                     READY   STATUS    RESTARTS   AGE
amq-streams-cluster-operator-v2.9.1-0-5f8bc76fb8-6p2bl   1/1     Running   0          2m
```

#### Check Operator Logs

```bash
# Check operator logs for any issues
oc logs deployment/strimzi-cluster-operator -n amq-streams-kafka

# Follow logs in real-time
oc logs -f deployment/strimzi-cluster-operator -n amq-streams-kafka
```

#### Verify CRDs are Installed

The AMQ Streams operator should install several Custom Resource Definitions:

```bash
# Check for Kafka CRDs
oc get crd | grep kafka

# Check for Strimzi CRDs  
oc get crd | grep strimzi.io
```

**Expected CRDs:**
```
kafkabridges.kafka.strimzi.io
kafkaconnectors.kafka.strimzi.io
kafkaconnects.kafka.strimzi.io
kafkamirrormaker2s.kafka.strimzi.io
kafkamirrormakers.kafka.strimzi.io
kafkarebalances.kafka.strimzi.io
kafkas.kafka.strimzi.io
kafkatopics.kafka.strimzi.io
kafkausers.kafka.strimzi.io
strimzipodsets.core.strimzi.io
```

#### Step 1.2.6: Complete Validation Script

Run this comprehensive validation script:

```bash
#!/bin/bash

echo "=== AMQ Streams Operator Validation ==="
echo ""

echo "1. Checking Subscription..."
oc get subscriptions.operators.coreos.com amq-streams -n amq-streams-kafka
echo ""

echo "2. Checking OperatorGroup..."
oc get operatorgroup -n amq-streams-kafka
echo ""

echo "3. Checking InstallPlan..."
oc get installplan -n amq-streams-kafka
echo ""

echo "4. Checking ClusterServiceVersion..."
oc get csv -n amq-streams-kafka | grep amq
echo ""

echo "5. Checking Operator Pods..."
oc get pods -n amq-streams-kafka
echo ""

echo "6. Checking CRDs..."
echo "Kafka CRDs:"
oc get crd | grep kafka | head -5
echo ""

echo "7. Final Validation..."
if oc get csv -n amq-streams-kafka --no-headers | grep -q "Succeeded"; then
    echo "✅ AMQ Streams Operator is successfully installed!"
    echo "Operator Version: $(oc get csv -n amq-streams-kafka -o jsonpath='{.items[?(@.metadata.name=="amqstreams.v2.9.1-0")].spec.version}' 2>/dev/null || echo 'Check manually')"
else
    echo "❌ AMQ Streams Operator installation failed or in progress"
    echo "Troubleshooting required - check OperatorGroup and InstallPlan status"
fi
echo ""
```

#### Step 1.2.7: Troubleshooting Common Issues

Based on real-world experience, here are the most common issues and their solutions:

**Issue 1: Missing OperatorGroup (Most Common)**

**Symptoms:**
- Subscription exists but no CSV is created
- No InstallPlan appears
- No operator pods running

**Diagnosis:**
```bash
# Check if OperatorGroup exists
oc get operatorgroup -n amq-streams-kafka
```

**Solution:**
```bash
# Create the missing OperatorGroup
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: amq-streams-og
  namespace: amq-streams-kafka
spec:
  targetNamespaces:
  - amq-streams-kafka
EOF
```

**Issue 2: API Resource Conflicts**

**Symptoms:**
- Error: `subscriptions.messaging.knative.dev "amq-streams" not found`

**Solution:**
```bash
# Always use full API resource path
oc get subscriptions.operators.coreos.com amq-streams -n amq-streams-kafka
# NOT: oc get subscription amq-streams -n amq-streams-kafka
```

**Issue 3: Install Plan Not Progressing**

**Diagnosis:**
```bash
# Check install plan details
oc get installplan -n amq-streams-kafka -o yaml
oc describe installplan -n amq-streams-kafka
```

**Solution:**
```bash
# If manual approval needed
oc patch installplan <install-plan-name> -n amq-streams-kafka --type merge --patch '{"spec":{"approved":true}}'
```

**Issue 4: General Debugging**

```bash
# Check OperatorHub availability
oc get catalogsource -n openshift-marketplace | grep redhat-operators

# Check subscription conditions
oc get subscriptions.operators.coreos.com amq-streams -n amq-streams-kafka -o jsonpath='{.status.conditions}' | jq .

# Check events
oc get events -n amq-streams-kafka --sort-by='.lastTimestamp'

# Check package availability
oc get packagemanifest amq-streams -n openshift-marketplace
```

#### Step 1.2.8: Quick One-Liner Validation

For a quick final check, you can use this one-liner:

```bash
oc get csv -n amq-streams-kafka --no-headers | grep -q "Succeeded" && echo "✅ AMQ Streams Operator Ready" || echo "❌ Installation Failed/In Progress"
```

**Expected Output when successful:**
```
✅ AMQ Streams Operator Ready
```

**Complete validation in one command:**
```bash
# Full validation check
echo "Operator Status:" && oc get csv -n amq-streams-kafka | grep amq && echo "Pod Status:" && oc get pods -n amq-streams-kafka
```

**⚠️ Important**: Only proceed to the next step after confirming:
- CSV shows `Phase: Succeeded`
- Operator pod is `Running`
- All CRDs are installed

### Step 1.3: Deploy Kafka Cluster

Create a production-ready Kafka cluster:

```yaml
cat <<EOF | oc apply -f -
apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: alert-kafka-cluster
  namespace: amq-streams-kafka
spec:
  kafka:
    version: 3.9.0
    replicas: 3
    listeners:
      - name: plain
        port: 9092
        type: internal
        tls: false
      - name: tls
        port: 9093
        type: internal
        tls: true
        authentication:
          type: tls
      - name: external
        port: 9094
        type: route
        tls: true
    config:
      offsets.topic.replication.factor: 3
      transaction.state.log.replication.factor: 3
      transaction.state.log.min.isr: 2
      default.replication.factor: 3
      min.insync.replicas: 2
      inter.broker.protocol.version: "3.9"
      log.message.format.version: "3.9"
      auto.create.topics.enable: "false"
    storage:
      type: persistent-claim
      size: 100Gi
      class: gp3-csi  # Adjust based on your storage class
    rack:
      topologyKey: topology.kubernetes.io/zone
    resources:
      requests:
        memory: 2Gi
        cpu: 1000m
      limits:
        memory: 4Gi
        cpu: 2000m
  zookeeper:
    replicas: 3
    storage:
      type: persistent-claim
      size: 20Gi
      class: gp3-csi
    resources:
      requests:
        memory: 1Gi
        cpu: 500m
      limits:
        memory: 2Gi
        cpu: 1000m
  entityOperator:
    topicOperator:
      resources:
        requests:
          memory: 512Mi
          cpu: 200m
        limits:
          memory: 512Mi
          cpu: 500m
    userOperator:
      resources:
        requests:
          memory: 512Mi
          cpu: 200m
        limits:
          memory: 512Mi
          cpu: 500m
EOF
```

### Step 1.4: Create Kafka Topics

Create the required topics for the Alert Engine:

```yaml
cat <<EOF | oc apply -f -
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: application-logs
  namespace: amq-streams-kafka
  labels:
    strimzi.io/cluster: alert-kafka-cluster
spec:
  partitions: 6
  replicas: 3
  config:
    retention.ms: 604800000  # 7 days
    segment.bytes: 1073741824  # 1GB
    cleanup.policy: delete
EOF
```

### Step 1.5: Verify Kafka Installation

```bash
# Check Kafka cluster status
oc get kafka alert-kafka-cluster -n amq-streams-kafka -o yaml

# Check pods
oc get pods -n amq-streams-kafka

# Verify cluster is ready
oc wait kafka/alert-kafka-cluster --for=condition=Ready --timeout=300s -n amq-streams-kafka

# Check topics
oc get kafkatopic -n amq-streams-kafka
```

**Expected Output when successful:**
```bash
# Kafka cluster should show Ready condition
NAME                  DESIRED KAFKA REPLICAS   DESIRED ZK REPLICAS   READY   WARNINGS
alert-kafka-cluster   3                        3                     True    

# Pods should be running
NAME                                                     READY   STATUS    RESTARTS   AGE
alert-kafka-cluster-kafka-0                             1/1     Running   0          5m
alert-kafka-cluster-kafka-1                             1/1     Running   0          5m
alert-kafka-cluster-kafka-2                             1/1     Running   0          5m
alert-kafka-cluster-zookeeper-0                         1/1     Running   0          6m
alert-kafka-cluster-zookeeper-1                         1/1     Running   0          6m
alert-kafka-cluster-zookeeper-2                         1/1     Running   0          6m
alert-kafka-cluster-entity-operator-xxxxxxxxx-xxxxx     3/3     Running   0          4m
```

**Common Issue: Version Compatibility**

If the Kafka cluster shows `NotReady` status, check for version errors:

```bash
# Check Kafka cluster conditions
oc get kafka alert-kafka-cluster -n amq-streams-kafka -o jsonpath='{.status.conditions}' | jq .

# Check operator logs for version errors
oc logs deployment/amq-streams-cluster-operator-v2.9.1-0 -n amq-streams-kafka | grep -i "version\|error"
```

**Error Example:**
```
UnsupportedKafkaVersionException: Unsupported Kafka.spec.kafka.version: 3.7.0. 
Supported versions are: [3.8.0, 3.9.0]
```

**Solution:** Update the Kafka version in your cluster specification to a supported version (3.9.0 recommended).

## 2. Redis Setup using Redis Enterprise Operator

### Step 2.1: Install Redis Enterprise Operator

**Using OpenShift Web Console:**
1. Navigate to **Operators > OperatorHub**
2. Search for "**Redis Enterprise**"
3. Select **Redis Enterprise Operator provided by Redis** (certified)
4. Click **Install**
5. Configure:
   - **Installation Mode**: Install to specific namespace (create `redis-enterprise`)
   - **Update Channel**: Production (NOT stable)
   - **Update Approval**: Manual

**Using CLI:**
```bash
# Create namespace
oc create namespace redis-enterprise

# Create OperatorGroup first (CRITICAL - prevents most installation issues)
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: redis-enterprise-og
  namespace: redis-enterprise
spec:
  targetNamespaces:
  - redis-enterprise
EOF

# Install Redis Enterprise Operator subscription
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: redis-enterprise-operator
  namespace: redis-enterprise
spec:
  channel: production  # Use 'production' NOT 'stable'
  name: redis-enterprise-operator-cert
  source: certified-operators
  sourceNamespace: openshift-marketplace
EOF
```

### Step 2.2: Validate Redis Enterprise Operator Installation

After installing the Redis Enterprise operator, validate the installation:

#### Step 2.2.1: Check Subscription Status

```bash
# Check the Subscription status (use full API resource to avoid conflicts)
oc get subscriptions.operators.coreos.com redis-enterprise-operator -n redis-enterprise

# Get detailed subscription info
oc get subscriptions.operators.coreos.com redis-enterprise-operator -n redis-enterprise -o yaml
```

**Expected Output:**
```
NAME                        PACKAGE                          SOURCE                CHANNEL
redis-enterprise-operator   redis-enterprise-operator-cert   certified-operators   production
```

#### Step 2.2.2: Check OperatorGroup (CRITICAL)

```bash
# Verify OperatorGroup exists
oc get operatorgroup -n redis-enterprise
```

**Expected Output:**
```
NAME                 AGE
redis-enterprise-og  2m
```

**⚠️ If no OperatorGroup exists, create it:**
```bash
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: redis-enterprise-og
  namespace: redis-enterprise
spec:
  targetNamespaces:
  - redis-enterprise
EOF
```

#### Step 2.2.3: Check InstallPlan

```bash
# Check InstallPlan creation and status
oc get installplan -n redis-enterprise
```

**Expected Output:**
```
NAME            CSV                                      APPROVAL    APPROVED
install-xxxxx   redis-enterprise-operator.v7.22.0-11.2   Automatic   true
```

#### Step 2.2.4: Check ClusterServiceVersion (CSV)

```bash
# Check CSV status
oc get csv -n redis-enterprise

# Get detailed CSV information
oc get csv -n redis-enterprise -o wide
```

**Expected Output:**
```
NAME                                      DISPLAY                       VERSION         PHASE
redis-enterprise-operator.v7.22.0-11.2   Redis Enterprise Operator    7.22.0-11.2     Succeeded
```

#### Step 2.2.5: Verify Operator Pod is Running

```bash
# Check operator pods
oc get pods -n redis-enterprise

# Check operator deployment
oc get deployment -n redis-enterprise
```

**Expected Output:**
```
NAME                                       READY   STATUS    RESTARTS   AGE
redis-enterprise-operator-6f95fb6b4f-97w6b   2/2     Running   0          2m
```

#### Step 2.2.6: Check Operator Logs

```bash
# Check operator logs for any issues
oc logs deployment/redis-enterprise-operator -n redis-enterprise

# Follow logs in real-time
oc logs -f deployment/redis-enterprise-operator -n redis-enterprise
```

#### Step 2.2.7: Verify CRDs are Installed

```bash
# Check for Redis Enterprise CRDs
oc get crd | grep redis

# Check for Redis Enterprise specific CRDs
oc get crd | grep redislabs
```

**Expected CRDs:**
```
redisenterpriseactiveactivedatabases.app.redislabs.com
redisenterpriseclusters.app.redislabs.com
redisenterprisedatabases.app.redislabs.com  
redisenterpriseremoteclusters.app.redislabs.com
```

#### Step 2.2.8: Complete Validation Script

Run this comprehensive validation script:

```bash
#!/bin/bash

echo "=== Redis Enterprise Operator Validation ==="
echo ""

echo "1. Checking Subscription..."
oc get subscriptions.operators.coreos.com redis-enterprise-operator -n redis-enterprise
echo ""

echo "2. Checking OperatorGroup..."
oc get operatorgroup -n redis-enterprise
echo ""

echo "3. Checking InstallPlan..."
oc get installplan -n redis-enterprise
echo ""

echo "4. Checking ClusterServiceVersion..."
oc get csv -n redis-enterprise | grep redis
echo ""

echo "5. Checking Operator Pods..."
oc get pods -n redis-enterprise
echo ""

echo "6. Checking CRDs..."
echo "Redis Enterprise CRDs:"
oc get crd | grep redislabs | head -4
echo ""

echo "7. Final Validation..."
if oc get csv -n redis-enterprise --no-headers | grep redis | grep -q "Succeeded"; then
    echo "✅ Redis Enterprise Operator is successfully installed!"
    echo "Operator Version: $(oc get csv -n redis-enterprise --no-headers | grep redis | awk '{print $3}')"
else
    echo "❌ Redis Enterprise Operator installation failed or in progress"
    echo "Troubleshooting required - check OperatorGroup, channel name, and InstallPlan status"
fi
echo ""
```

#### Step 2.2.9: Troubleshooting Common Issues

Based on real-world experience, here are the most common issues and their solutions:

**Issue 1: Missing OperatorGroup (Most Common)**

**Symptoms:**
- Subscription exists but no CSV is created
- No InstallPlan appears
- No operator pods running

**Diagnosis:**
```bash
# Check if OperatorGroup exists
oc get operatorgroup -n redis-enterprise
```

**Solution:**
```bash
# Create the missing OperatorGroup
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: redis-enterprise-og
  namespace: redis-enterprise
spec:
  targetNamespaces:
  - redis-enterprise
EOF
```

**Issue 2: Incorrect Channel Name**

**Symptoms:**
- Subscription shows constraint error: "no operators found in channel stable"
- CSV never gets created

**Diagnosis:**
```bash
# Check available channels
oc describe packagemanifest redis-enterprise-operator-cert -n openshift-marketplace | grep "Name:" | grep -A 5 -B 5 "Current CSV"

# Check subscription status
oc describe subscriptions.operators.coreos.com redis-enterprise-operator -n redis-enterprise
```

**Solution:**
```bash
# Delete incorrect subscription
oc delete subscriptions.operators.coreos.com redis-enterprise-operator -n redis-enterprise

# Create new subscription with correct channel
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: redis-enterprise-operator
  namespace: redis-enterprise
spec:
  channel: production  # Use 'production' NOT 'stable'
  name: redis-enterprise-operator-cert
  source: certified-operators
  sourceNamespace: openshift-marketplace
EOF
```

**Issue 3: API Resource Conflicts**

**Symptoms:**
- Error: `subscriptions.messaging.knative.dev "redis-enterprise-operator" not found`

**Solution:**
```bash
# Always use full API resource path
oc get subscriptions.operators.coreos.com redis-enterprise-operator -n redis-enterprise
# NOT: oc get subscription redis-enterprise-operator -n redis-enterprise
```

**Issue 4: General Debugging**

```bash
# Check package availability and channels
oc get packagemanifest -n openshift-marketplace | grep redis
oc describe packagemanifest redis-enterprise-operator-cert -n openshift-marketplace

# Check subscription conditions
oc get subscriptions.operators.coreos.com redis-enterprise-operator -n redis-enterprise -o jsonpath='{.status.conditions}' | jq .

# Check events
oc get events -n redis-enterprise --sort-by='.lastTimestamp'

# Check OperatorHub availability
oc get catalogsource -n openshift-marketplace | grep certified-operators
```

#### Step 2.2.10: Quick One-Liner Validation

For a quick final check, you can use this one-liner:

```bash
oc get csv -n redis-enterprise --no-headers | grep redis | grep -q "Succeeded" && echo "✅ Redis Enterprise Operator Ready" || echo "❌ Installation Failed/In Progress"
```

**Expected Output when successful:**
```
✅ Redis Enterprise Operator Ready
```

**Complete validation in one command:**
```bash
# Full validation check
echo "Operator Status:" && oc get csv -n redis-enterprise | grep redis && echo "Pod Status:" && oc get pods -n redis-enterprise
```

**⚠️ Important**: Only proceed to the next step after confirming:
- CSV shows `Phase: Succeeded`
- Operator pod is `Running` (2/2)
- All CRDs are installed

#### Step 2.2.11: Additional Validation Notes

**Common Channel Names for Redis Enterprise:**
- `production` (recommended, default)
- `preview` (for testing new features)
- Version-specific channels like `7.8.6`, `7.8.4`, etc.

**DO NOT use `stable` - it doesn't exist for Redis Enterprise operator**

**Expected final state:**
```bash
# All should show success
oc get operatorgroup -n redis-enterprise    # Should show: redis-enterprise-og
oc get installplan -n redis-enterprise      # Should show: install-xxxxx with Approved=true
oc get csv -n redis-enterprise              # Should show: Phase=Succeeded
oc get pods -n redis-enterprise             # Should show: Running 2/2
```

### Step 2.3: Create Security Context Constraints

```yaml
cat <<EOF | oc apply -f -
apiVersion: security.openshift.io/v1
kind: SecurityContextConstraints
metadata:
  name: redis-enterprise-scc
allowHostDirVolumePlugin: false
allowHostIPC: false
allowHostNetwork: false
allowHostPID: false
allowHostPorts: false
allowPrivilegedContainer: false
allowedCapabilities: null
defaultAddCapabilities: null
requiredDropCapabilities:
- KILL
- MKNOD
- SETUID
- SETGID
runAsUser:
  type: MustRunAsRange
  uidRangeMin: 1000
  uidRangeMax: 2000
seLinuxContext:
  type: MustRunAs
fsGroup:
  type: RunAsAny
volumes:
- configMap
- downwardAPI
- emptyDir
- persistentVolumeClaim
- projected
- secret
EOF

# Bind SCC to service account
oc adm policy add-scc-to-user redis-enterprise-scc -z redis-enterprise-operator -n redis-enterprise
```

### Step 2.4: Deploy Redis Enterprise Cluster

**⚠️ Important**: Use Red Hat certified images for OpenShift compatibility.

```yaml
cat <<EOF | oc apply -f -
apiVersion: app.redislabs.com/v1
kind: RedisEnterpriseCluster
metadata:
  name: rec-alert-engine
  namespace: redis-enterprise
spec:
  nodes: 3
  persistentSpec:
    enabled: true
    storageClassName: gp3-csi  # Adjust based on your storage class
    volumeSize: 20Gi
  redisEnterpriseNodeResources:
    limits:
      cpu: 2000m
      memory: 4Gi
    requests:
      cpu: 1000m
      memory: 2Gi
  services:
    riggerService:
      serviceType: ClusterIP
    apiService:
      serviceType: ClusterIP
  # CRITICAL: Use Red Hat certified images for OpenShift
  bootstrapperImageSpec:
    repository: registry.connect.redhat.com/redislabs/redis-enterprise-operator
  redisEnterpriseServicesRiggerImageSpec:
    repository: registry.connect.redhat.com/redislabs/services-manager
  redisEnterpriseImageSpec:
    repository: registry.connect.redhat.com/redislabs/redis-enterprise
    imagePullPolicy: IfNotPresent
EOF
```

#### Step 2.4.1: Monitor Cluster Deployment

Wait for the cluster to deploy completely:

```bash
# Monitor cluster status
watch -n 10 'oc get redisenterprisecluster rec-alert-engine -n redis-enterprise'

# Check pod status
oc get pods -n redis-enterprise

# Monitor cluster progression
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.state}'
```

**Expected States Progression:**
1. `BootstrappingFirstPod` → `Initializing` → `Running`

**Expected Final Output:**
```
NAME               NODES   SHARDS   VERSION     STATE     SPEC STATUS   LICENSE STATE   LICENSE EXPIRATION DATE   AGE
rec-alert-engine   3       0/4      7.22.0-95   Running   Valid         Valid           2025-08-08T15:37:04Z      5m
```

#### Step 2.4.2: Troubleshooting Common Issues

**Issue 1: Image Pull Errors**

**Symptoms:**
- Pods show `ImagePullBackOff` or `ErrImagePull`
- Error: `manifest unknown` for Docker Hub images

**Diagnosis:**
```bash
# Check pod errors
oc describe pod rec-alert-engine-0 -n redis-enterprise
```

**Solution:**
- Ensure you're using **Red Hat certified images** (see configuration above)
- Delete and recreate cluster if using incorrect images:
```bash
oc delete redisenterprisecluster rec-alert-engine -n redis-enterprise
# Then recreate with correct images
```

**Issue 2: Cluster Stuck in BootstrappingFirstPod**

**Symptoms:**
- Cluster state remains in `BootstrappingFirstPod` for > 10 minutes
- Pods not reaching Running state

**Diagnosis:**
```bash
# Check cluster events
oc get events -n redis-enterprise --sort-by='.lastTimestamp'

# Check pod logs
oc logs rec-alert-engine-0 -c bootstrapper -n redis-enterprise
```

**Solution:**
- Verify storage class exists and is accessible
- Check node resources and scheduling constraints
- Ensure Security Context Constraints are properly configured

**Issue 3: License Issues**

**Symptoms:**
- License state shows as invalid or expired
- Cluster fails to reach Running state

**Diagnosis:**
```bash
# Check license status
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.licenseStatus}'
```

**Solution:**
- Redis Enterprise operator includes a 30-day trial license
- For production, obtain a proper license from Redis

#### Step 2.4.3: Final Cluster Validation

```bash
#!/bin/bash

echo "=== Redis Enterprise Cluster Validation ==="
echo ""

echo "1. Cluster Status:"
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise
echo ""

echo "2. Pod Status:"
oc get pods -n redis-enterprise
echo ""

echo "3. License Status:"
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.licenseStatus.licenseState}'
echo ""

echo "4. Available Modules:"
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.modules[*].name}'
echo ""

echo "5. Final Validation:"
if [[ $(oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.state}') == "Running" ]]; then
    echo "✅ Redis Enterprise Cluster is Ready!"
    echo "Available Shards: $(oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.licenseStatus.shardsUsage}')"
else
    echo "❌ Redis Enterprise Cluster not ready - check troubleshooting steps above"
fi
echo ""
```

**Expected successful output:**
```
✅ Redis Enterprise Cluster is Ready!
Available Shards: 0/4
```

### Step 2.5: Create Redis Database

**⚠️ Important**: Wait for the Redis Enterprise Cluster to reach `Running` state before creating databases.

#### Step 2.5.1: Check Available Modules and Versions

Before creating the database, check what modules are available:

```bash
# Check available modules and versions
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.modules[*]}'

# Format for easier reading
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{range .status.modules[*]}{.name}: {.versions[0]}{"\n"}{end}'
```

**Expected Output:**
```
ReJSON: 2.8.8
bf: 2.8.6
search: 2.10.17
timeseries: 1.12.6
```

#### Step 2.5.2: Create Database

```yaml
cat <<EOF | oc apply -f -
apiVersion: app.redislabs.com/v1alpha1
kind: RedisEnterpriseDatabase
metadata:
  name: alert-engine-cache
  namespace: redis-enterprise
spec:
  memorySize: 1GB  # Use 1GB to avoid shard allocation issues
  redisEnterpriseCluster:
    name: rec-alert-engine
  type: redis
  persistence: aofEverySecond  # Use correct persistence value
  redisModule:
  - name: ReJSON
    version: "2.8.8"          # Use available version
  - name: timeseries          # Use 'timeseries' not 'RedisTimeSeries'
    version: "1.12.6"         # Use available version
EOF
```

#### Step 2.5.3: Monitor Database Creation

```bash
# Monitor database creation
watch -n 5 'oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise'

# Check database status
oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise
```

**Expected Output:**
```
NAME                 VERSION   PORT    CLUSTER            SHARDS   STATUS   SPEC STATUS   AGE
alert-engine-cache   7.4.2     13066   rec-alert-engine   1        active   Valid         30s
```

#### Step 2.5.4: Troubleshooting Database Creation Issues

**Issue 1: Cannot Allocate Nodes for Shards**

**Symptoms:**
- Error: `Cannot allocate nodes for shards`
- Database creation fails

**Diagnosis:**
```bash
# Check cluster shard usage
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.licenseStatus.shardsUsage}'

# Check cluster resource limits
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.licenseStatus.shardsLimit}'
```

**Solution:**
- Reduce memory size (try 1GB instead of 2GB)
- Check if cluster has available shards
- Verify cluster is in `Running` state

**Issue 2: Invalid Persistence Value**

**Symptoms:**
- Error: `Unsupported value: "aof"`
- Database validation fails

**Valid persistence values:**
- `disabled`
- `aofEverySecond` (recommended)
- `aofAlways`
- `snapshotEvery1Hour`
- `snapshotEvery6Hour`
- `snapshotEvery12Hour`

**Issue 3: Module Version Errors**

**Symptoms:**
- Module version not available
- Database creation fails

**Solution:**
```bash
# Check available module versions
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.modules[?(@.name=="ReJSON")].versions[*]}'
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.modules[?(@.name=="timeseries")].versions[*]}'
```

**Issue 4: Cluster Not Ready**

**Symptoms:**
- Error: `could not get cluster object`
- Connection refused errors

**Solution:**
```bash
# Verify cluster is running
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise

# Wait for cluster to be ready
until [[ $(oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.state}') == "Running" ]]; do
  echo "Waiting for cluster to be ready..."
  sleep 10
done
```

#### Step 2.5.5: Database Validation Script

```bash
#!/bin/bash

echo "=== Redis Enterprise Database Validation ==="
echo ""

echo "1. Cluster Status:"
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise
echo ""

echo "2. Database Status:"
oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise
echo ""

echo "3. Connection Information:"
DATABASE_PORT=$(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.redisEnterpriseCluster}')
echo "Database Port: $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}')"
echo ""

echo "4. Final Validation:"
if [[ $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.status}') == "active" ]]; then
    echo "✅ Redis Enterprise Database is Ready!"
    echo "Connection Details:"
    echo "  - Name: alert-engine-cache"
    echo "  - Port: $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}')"
    echo "  - Modules: ReJSON, RedisTimeSeries"
    echo "  - Persistence: AOF Every Second"
else
    echo "❌ Redis Enterprise Database not ready - check troubleshooting steps above"
fi
echo ""
```

**Expected successful output:**
```
✅ Redis Enterprise Database is Ready!
Connection Details:
  - Name: alert-engine-cache
  - Port: 13066
  - Modules: ReJSON, RedisTimeSeries
  - Persistence: AOF Every Second
```

### Step 2.6: Verify Redis Installation

```bash
# Check Redis cluster status
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise

# Check Redis database
oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise

# Check pods
oc get pods -n redis-enterprise

# Get Redis connection info
oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d
echo  # Add newline after password

# Get database port
oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}'
echo  # Add newline after port

# Get full connection string
echo "Connection details:"
echo "Host: rec-alert-engine.redis-enterprise.svc.cluster.local"
echo "Port: $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}')"
echo "Password: $(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d)"
```

#### Step 2.6.1: Test Redis Connection

```bash
# Test Redis connection from within the cluster
oc run redis-test --rm -i --tty --image=redis:7 --restart=Never -- redis-cli -h rec-alert-engine.redis-enterprise.svc.cluster.local -p $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}') -a $(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d)
```

#### Step 2.6.2: Test Redis Modules

```bash
# Test ReJSON module
oc run redis-test --rm -i --tty --image=redis:7 --restart=Never -- redis-cli -h rec-alert-engine.redis-enterprise.svc.cluster.local -p $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}') -a $(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d) JSON.SET test $ '{"message":"Hello Redis Enterprise!"}'

# Test RedisTimeSeries module
oc run redis-test --rm -i --tty --image=redis:7 --restart=Never -- redis-cli -h rec-alert-engine.redis-enterprise.svc.cluster.local -p $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}') -a $(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d) TS.CREATE test-metric
```

**Expected Output:**
```
OK
OK
```

#### Step 2.6.3: Create ConfigMap for Application

Create a ConfigMap with Redis connection details for your applications:

**First, get the current database port:**
```bash
# Get the actual database port
REDIS_PORT=$(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.internalEndpoints[0].port}')
echo "Database Port: $REDIS_PORT"
```

**Create the ConfigMap with actual values:**
```yaml
# Create namespace if it doesn't exist
oc create namespace alert-engine --dry-run=client -o yaml | oc apply -f -

# Create ConfigMap with correct Redis connection details
cat <<EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: alert-engine
data:
  # Use the database service name, not the cluster service
  redis-host: "alert-engine-cache.redis-enterprise.svc.cluster.local"
  redis-port: "13066"
  redis-database: "alert-engine-cache"
  # Alternative internal endpoint (more reliable for enterprise features)
  redis-internal-host: "redis-13066.rec-alert-engine.redis-enterprise.svc.cluster.local"
  # Cluster information
  redis-cluster: "rec-alert-engine"
  redis-namespace: "redis-enterprise"
  # Secret reference for password
  redis-secret-name: "redb-alert-engine-cache"
EOF
```

**Create a Secret reference for the password:**
```yaml
# Option 1: Copy the password to your application namespace
oc get secret redb-alert-engine-cache -n redis-enterprise -o yaml | \
  sed 's/namespace: redis-enterprise/namespace: alert-engine/' | \
  oc apply -f -

# Option 2: Create a new secret with just the password
cat <<EOF | oc apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: redis-password
  namespace: alert-engine
type: Opaque
data:
  password: $(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}')
EOF
```

**Verify the ConfigMap:**
```bash
# Check ConfigMap contents
oc get configmap redis-config -n alert-engine -o yaml

# Test the connection details
echo "Redis connection details:"
echo "Host: $(oc get configmap redis-config -n alert-engine -o jsonpath='{.data.redis-host}')"
echo "Port: $(oc get configmap redis-config -n alert-engine -o jsonpath='{.data.redis-port}')"
echo "Password: $(oc get secret redis-password -n alert-engine -o jsonpath='{.data.password}' | base64 -d)"
```

**Test Redis Connection:**
```bash
# Test Redis connection using ConfigMap values
REDIS_HOST=$(oc get configmap redis-config -n alert-engine -o jsonpath='{.data.redis-host}')
REDIS_PORT=$(oc get configmap redis-config -n alert-engine -o jsonpath='{.data.redis-port}')
REDIS_PASSWORD=$(oc get secret redis-password -n alert-engine -o jsonpath='{.data.password}' | base64 -d)

# Test connection with ping
oc run redis-connection-test --rm -i --tty --image=redis:7 --restart=Never -- redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD ping

# Test ReJSON module
oc run redis-json-test --rm -i --tty --image=redis:7 --restart=Never -- redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD JSON.SET test:config $ '{"app":"alert-engine","status":"ready"}'

# Test RedisTimeSeries module
oc run redis-ts-test --rm -i --tty --image=redis:7 --restart=Never -- redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD TS.CREATE test:metrics
```

**Expected Output:**
```
PONG
OK
OK
```

#### Step 2.6.4: Complete Redis Enterprise Setup Summary

**✅ Redis Enterprise Setup Complete**

Your Redis Enterprise setup now includes:

1. **Operator**: Redis Enterprise Operator v7.22.0-11.2
2. **Cluster**: 3-node cluster with 4 available shards
3. **Database**: alert-engine-cache with ReJSON and RedisTimeSeries modules
4. **Persistence**: AOF Every Second
5. **Security**: Secured with password authentication

**Connection Details:**
- **Host**: `alert-engine-cache.redis-enterprise.svc.cluster.local` (database service)
- **Alternative Host**: `redis-13066.rec-alert-engine.redis-enterprise.svc.cluster.local` (internal endpoint)
- **Port**: `13066`
- **Password**: Retrieved via `oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d`
- **Modules**: ReJSON 2.8.8, RedisTimeSeries 1.12.6
- **ConfigMap**: Available in `alert-engine` namespace as `redis-config`
- **Secret**: Available in `alert-engine` namespace as `redis-password`

**Next Steps:**
- Use the connection details in your alert-engine application
- Configure log forwarding to send logs to your application for processing
- Set up monitoring and alerting rules

## 3. ClusterLogForwarder Setup

**⚠️ Important Note**: In most OpenShift clusters, the OpenShift Logging Operator and logging infrastructure are already installed and configured. This section has been streamlined to focus only on creating the additional ClusterLogForwarder needed for the alert-engine.

### Prerequisites Check

The OpenShift Logging Operator is typically already installed in most OpenShift clusters. Let's verify this:

```bash
# Check if OpenShift Logging Operator is already installed
oc get csv -n openshift-logging | grep cluster-logging

# Check if logging infrastructure is already deployed
oc get deployment -n openshift-logging
```

**Expected Output:**
```
cluster-logging.v6.x.x    Red Hat OpenShift Logging    6.x.x    Succeeded
```

If the operator is already installed (which is common), you can skip the installation steps and proceed directly to creating the ClusterLogForwarder.

### Step 3.1: Create ClusterLogForwarder for Kafka

Since there's already a ClusterLogForwarder named "logging" in the cluster, we'll create a new one specifically for the alert-engine with a different name:

```yaml
cat <<EOF | oc apply -f -
apiVersion: observability.openshift.io/v1
kind: ClusterLogForwarder
metadata:
  name: kafka-alert-forwarder
  namespace: openshift-logging
spec:
  serviceAccount:
    name: collector
  outputs:
  - name: kafka-application-logs
    type: kafka
    kafka:
      topic: application-logs
      brokers:
      - alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092
  pipelines:
  - name: forward-application-logs
    inputRefs:
    - application
    outputRefs:
    - kafka-application-logs
    filterRefs: []
EOF
```

**Key Configuration Details:**
- **serviceAccount: collector** - Required field that uses the existing logging collector service account
- **No `url` field** - For kafka outputs, only `brokers` array is needed under the `kafka` section
- **Application logs only** - Focuses on `application` input (pod logs) which is what you need for application monitoring, not infrastructure/audit logs
- **Separate from existing forwarder** - This ClusterLogForwarder is separate from the existing "logging" forwarder that sends logs to LokiStack. Both can coexist, allowing logs to be sent to both destinations.

### Step 3.2: Verify Log Forwarding

```bash
# Check ClusterLogForwarder status
oc get clusterlogforwarder kafka-alert-forwarder -n openshift-logging -o yaml

# Check vector pods (log collectors)
oc get pods -n openshift-logging -l component=collector

# Verify logs are being forwarded to Kafka
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --max-messages 5
```

## 4. Network Policies and Security

### Step 4.1: Create Network Policies

```yaml
# Network policy for Kafka access
cat <<EOF | oc apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kafka-network-policy
  namespace: amq-streams-kafka
spec:
  podSelector:
    matchLabels:
      strimzi.io/cluster: alert-kafka-cluster
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: alert-engine  # Your application namespace
    - namespaceSelector:
        matchLabels:
          name: openshift-logging
    ports:
    - protocol: TCP
      port: 9092
    - protocol: TCP
      port: 9093
  egress:
  - {}
---
# Network policy for Redis access
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: redis-network-policy
  namespace: redis-enterprise
spec:
  podSelector:
    matchLabels:
      app: redis-enterprise
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: alert-engine  # Your application namespace
    ports:
    - protocol: TCP
      port: 6379
    - protocol: TCP
      port: 8001
  egress:
  - {}
EOF
```

### Step 4.2: Create Service Accounts and RBAC

```yaml
cat <<EOF | oc apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: alert-engine-sa
  namespace: alert-engine
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: alert-engine-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints", "persistentvolumeclaims", "events", "configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "daemonsets", "replicasets", "statefulsets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["monitoring.coreos.com"]
  resources: ["servicemonitors"]
  verbs: ["get", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: alert-engine-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: alert-engine-role
subjects:
- kind: ServiceAccount
  name: alert-engine-sa
  namespace: alert-engine
EOF
```

## 5. Verification and Testing

### Step 5.1: Complete Infrastructure Verification

```bash
#!/bin/bash
echo "=== Verifying Kafka Setup ==="
oc get kafka alert-kafka-cluster -n amq-streams-kafka
oc get kafkatopic -n amq-streams-kafka
echo ""

echo "=== Verifying Redis Setup ==="
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise
oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise
echo ""

echo "=== Verifying Log Forwarding ==="
oc get clusterlogforwarder kafka-alert-forwarder -n openshift-logging
echo ""

echo "=== Getting Service Endpoints ==="
echo "Kafka Bootstrap Servers:"
oc get svc alert-kafka-cluster-kafka-bootstrap -n amq-streams-kafka
echo ""
echo "Redis Service:"
oc get svc -n redis-enterprise -l app=redis-enterprise-database
echo ""

echo "=== All infrastructure components are ready! ==="
```

### Step 5.2: Get Connection Details

```bash
# Kafka connection details
echo "Kafka Bootstrap Servers:"
oc get svc alert-kafka-cluster-kafka-bootstrap -n amq-streams-kafka -o jsonpath='{.spec.clusterIP}:{.spec.ports[0].port}'

# Redis connection details
echo "Redis Host:"
oc get svc -n redis-enterprise -l app=redis-enterprise-database -o jsonpath='{.items[0].spec.clusterIP}:{.items[0].spec.ports[0].port}'

# Redis password
echo "Redis Password:"
oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d
```

## 6. Next Steps

After completing this infrastructure setup:

1. **Local Testing**: Update your `configs/config.yaml` with the connection details obtained above and test the Alert Engine locally
2. **Deploy to OpenShift**: Use the deployment manifests in `deployments/openshift/` to deploy the Alert Engine application
3. **Configure Monitoring**: Set up Prometheus monitoring for the Alert Engine metrics
4. **Test End-to-End**: Generate test logs and verify alerts are triggered and notifications are sent

## Troubleshooting

### Common Issues

1. **Missing OperatorGroup (Most Common Issue)**: If subscription exists but no CSV/InstallPlan is created:
   ```bash
   # Check if OperatorGroup exists
   oc get operatorgroup -n amq-streams-kafka
   
   # If missing, create it
   cat <<EOF | oc apply -f -
   apiVersion: operators.coreos.com/v1
   kind: OperatorGroup
   metadata:
     name: amq-streams-og
     namespace: amq-streams-kafka
   spec:
     targetNamespaces:
     - amq-streams-kafka
   EOF
   ```

2. **Subscription API Conflicts**: If you get `subscriptions.messaging.knative.dev` error, use the full API resource:
   ```bash
   # Wrong (may cause conflicts)
   oc get subscription amq-streams -n amq-streams-kafka
   
   # Correct (explicit API group)
   oc get subscriptions.operators.coreos.com amq-streams -n amq-streams-kafka
   ```

3. **Debugging API Resource Conflicts**:
   ```bash
   # Check all subscription resources available
   oc api-resources | grep subscription
   
   # Check what subscriptions exist in the namespace
   oc get subscriptions.operators.coreos.com -n amq-streams-kafka
   
   # Check if there are knative subscriptions (causing the conflict)
   oc get subscriptions.messaging.knative.dev -A 2>/dev/null || echo 'No knative subscriptions found'
   ```

4. **Kafka Version Compatibility**: If Kafka cluster shows NotReady status:
   ```bash
   # Check for version errors in operator logs
   oc logs deployment/amq-streams-cluster-operator-v2.9.1-0 -n amq-streams-kafka | grep -i "version\|error"
   
   # Expected error: UnsupportedKafkaVersionException
   # Solution: Update Kafka spec to use supported version (3.9.0 or 3.8.0)
   ```

5. **Kafka not ready**: Check storage class and resource limits
6. **Redis connection issues**: Verify SCC and network policies
7. **Log forwarding not working**: Check vector collector pods in openshift-logging namespace
8. **Permission denied**: Ensure proper RBAC and SCC configurations

### Useful Commands

```bash
# Check all operator statuses
oc get csv -A | grep -E "(amq-streams|redis|logging)"

# Check pod logs
oc logs -f deployment/strimzi-cluster-operator -n amq-streams-kafka
oc logs -f deployment/redis-enterprise-operator -n redis-enterprise

# Monitor log forwarding
oc logs -f daemonset/collector -n openshift-logging
```

## Configuration Updates

Once the infrastructure is deployed, update your Alert Engine configuration file (`configs/config.yaml`) with the actual connection details:

```yaml
kafka:
  brokers: ["alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"]
  
redis:
  host: "<redis-service-ip>"
  port: 6379
  password: "<redis-password>"
```

This completes the OpenShift infrastructure setup. You can now proceed with local testing and then deploy the Alert Engine application to OpenShift. 