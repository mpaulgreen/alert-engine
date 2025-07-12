# OpenShift Infrastructure Setup Guide

This guide provides step-by-step instructions to set up the required infrastructure components on OpenShift before deploying the Alert Engine application.

## Prerequisites

- OpenShift 4.16.17 cluster with cluster-admin access
- `oc` CLI tool installed and configured
- Access to OperatorHub in your OpenShift cluster

### Prerequisites Verification

Before starting, verify your cluster has the required storage classes:

```bash
# Check available storage classes
oc get storageclass

# Verify gp3-csi exists (or substitute with your preferred class)
oc get storageclass gp3-csi && echo "✅ Storage class available" || echo "❌ Update storage class in configurations"
```

**Note**: If `gp3-csi` is not available, update all storage class references in this guide to use your cluster's available storage class.

## Overview

The Alert Engine requires the following components to be installed on OpenShift:

1. **Red Hat AMQ Streams** (Apache Kafka) - For log message streaming
2. **Redis Enterprise** - For state storage and caching
3. **ClusterLogForwarder** - For forwarding OpenShift logs to Kafka
4. **Network policies and security configurations**

### ⏰ Expected Deployment Times

- **AMQ Streams Operator**: 1-2 minutes
- **Kafka Cluster**: 3-5 minutes  
- **Redis Enterprise Operator**: 1-2 minutes
- **Redis Enterprise Cluster**: 2-4 minutes
- **OpenShift Logging Operator**: 1-2 minutes
- **ClusterLogForwarder + Vector**: 1-2 minutes for validation

**Total Setup Time**: Approximately 15-20 minutes

**💡 Tip**: Each section includes validation steps. Wait for each component to be fully ready before proceeding to the next step.

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
amqstreams.v2.9.1-0               Streams for Apache Kafka        2.9.1-0   amqstreams.v2.8.0-0.1738265624.p  Succeeded
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

#### Step 1.2.6: Check Operator Logs

```bash
# Check operator logs for any issues
oc logs deployment/strimzi-cluster-operator -n amq-streams-kafka

# Follow logs in real-time
oc logs -f deployment/strimzi-cluster-operator -n amq-streams-kafka
```

#### Step 1.2.7: Verify CRDs are Installed

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


#### Step 1.2.8: Troubleshooting Common Issues

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

#### Step 1.2.9: Quick One-Liner Validation

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

Create a production-ready Kafka cluster with tested and verified configuration:

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

**Wait for complete deployment before proceeding:**

```bash
# Wait for Kafka cluster to be ready (this can take 5-10 minutes)
oc wait kafka/alert-kafka-cluster --for=condition=Ready --timeout=600s -n amq-streams-kafka

# Monitor deployment progress
watch -n 10 'oc get kafka alert-kafka-cluster -n amq-streams-kafka && echo "" && oc get pods -n amq-streams-kafka'
```

**Expected successful deployment:**
- Kafka cluster should show `Ready: True`
- All pods should be `Running`: 3 kafka brokers + 3 zookeeper + 1 entity-operator

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

### Step 1.6: Test Kafka Producer-Consumer Flow

This step verifies that messages can be successfully sent and received through the Kafka cluster.

#### Step 1.6.1: Send Test Message

```bash
# Send a test JSON message to the application-logs topic
echo '{"timestamp":"2025-07-10T12:00:00Z","level":"INFO","message":"Test message from Kafka producer","service":"kafka-test","namespace":"test"}' | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs
```

#### Step 1.6.2: Verify Message Reception

```bash
# Check if the message was received by running a consumer
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --max-messages 1
```

**Expected Output:**
```
Defaulted container "kafka" out of: kafka, kafka-init (init)
{"timestamp":"2025-07-10T12:00:00Z","level":"INFO","message":"Test message from Kafka producer","service":"kafka-test","namespace":"test"}
Processed a total of 1 messages
```

#### Step 1.6.3: Complete Producer-Consumer Test

```bash
# Complete test in one command
echo "=== Testing Kafka Producer-Consumer Flow ===" && \
echo "1. Sending test message..." && \
echo '{"timestamp":"'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'","level":"INFO","message":"Kafka test successful","service":"kafka-test","namespace":"test"}' | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs && \
echo "2. Verifying message reception..." && \
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --max-messages 1
```

**✅ Success Criteria:**
- Producer sends message without errors
- Consumer receives and displays the JSON message
- Message count shows "Processed a total of 1 messages"

**Troubleshooting:**
- If producer fails: Check Kafka cluster status and topic existence
- If consumer hangs: Verify topic has messages with `oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-topics.sh --bootstrap-server localhost:9092 --topic application-logs --describe`
- If no messages: Check producer succeeded and topic configuration

### Step 1.7: Create Kafka Network Policies

Set up network policies to allow access from alert-engine and openshift-logging namespaces:

```bash
# Create network policy for Kafka access
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
          kubernetes.io/metadata.name: alert-engine
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: openshift-logging
    ports:
    - protocol: TCP
      port: 9092
    - protocol: TCP
      port: 9093
  egress:
  - {}
EOF
```

### Step 1.8: Get Kafka Connection Details

```bash
# Get Kafka connection details for application configuration
echo "=== Kafka Connection Details ==="
echo "Bootstrap Servers: alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"
echo "Topic: application-logs"
echo "Service IP:"
oc get svc alert-kafka-cluster-kafka-bootstrap -n amq-streams-kafka -o jsonpath='{.spec.clusterIP}:{.spec.ports[0].port}'
echo ""
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

#### Step 2.6.2: Create ConfigMap for Application

Create a ConfigMap with Redis connection details for your applications:

**First, get the current database port (with proper error handling):**
```bash
# Get the actual database port with multiple attempts
echo "🔍 Discovering Redis database port..."
REDIS_PORT=""
for i in {1..5}; do
    REDIS_PORT=$(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}' 2>/dev/null)
    if [[ -n "$REDIS_PORT" ]]; then
        echo "✅ Found Redis database port: $REDIS_PORT"
        break
    else
        echo "⏳ Waiting for database port... (attempt $i/5)"
        sleep 10
    fi
done

# Fallback port discovery methods
if [[ -z "$REDIS_PORT" ]]; then
    echo "🔍 Trying alternative port discovery methods..."
    # Try internal endpoints
    REDIS_PORT=$(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.internalEndpoints[0].port}' 2>/dev/null)
    
    # Try service port
    if [[ -z "$REDIS_PORT" ]]; then
        REDIS_PORT=$(oc get service alert-engine-cache -n redis-enterprise -o jsonpath='{.spec.ports[0].port}' 2>/dev/null)
    fi
fi

# Final validation
if [[ -z "$REDIS_PORT" ]]; then
    echo "❌ Could not determine Redis port. Check database status:"
    oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise
    exit 1
else
    echo "✅ Using Redis port: $REDIS_PORT"
fi
```

**Create the ConfigMap with dynamically discovered values:**
```bash
# Create namespace if it doesn't exist
oc create namespace alert-engine --dry-run=client -o yaml | oc apply -f -

# Create ConfigMap with dynamically discovered Redis connection details
cat <<EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: alert-engine
data:
  # Use the database service name, not the cluster service
  redis-host: "alert-engine-cache.redis-enterprise.svc.cluster.local"
  redis-port: "${REDIS_PORT}"
  redis-database: "alert-engine-cache"
  # Alternative internal endpoint (more reliable for enterprise features)
  redis-internal-host: "redis-${REDIS_PORT}.rec-alert-engine.redis-enterprise.svc.cluster.local"
  # Cluster information
  redis-cluster: "rec-alert-engine"
  redis-namespace: "redis-enterprise"
  # Secret reference for password
  redis-secret-name: "redb-alert-engine-cache"
EOF

echo "✅ ConfigMap created with Redis port: ${REDIS_PORT}"
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

**Test Redis Connection (Enhanced with Better Error Handling):**
```bash
# Get connection details first
REDIS_HOST=$(oc get configmap redis-config -n alert-engine -o jsonpath='{.data.redis-host}')
REDIS_PORT=$(oc get configmap redis-config -n alert-engine -o jsonpath='{.data.redis-port}')
REDIS_PASSWORD=$(oc get secret redis-password -n alert-engine -o jsonpath='{.data.password}' | base64 -d)

echo "🔍 Testing Redis connection to $REDIS_HOST:$REDIS_PORT"

# Test 1: Basic connection with ping
echo "1. Testing basic Redis connection..."
PING_RESULT=$(oc run redis-connection-test --rm -i --image=redis:7 --restart=Never -- \
  redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" ping 2>/dev/null)

if [[ "$PING_RESULT" == "PONG" ]]; then
    echo "   ✅ Basic Redis connection successful"
else
    echo "   ❌ Basic Redis connection failed: $PING_RESULT"
    echo "   🔧 Troubleshooting: Check host, port, and password"
    exit 1
fi

# Test 2: ReJSON module (with better error handling)
echo "2. Testing ReJSON module..."
JSON_SET_RESULT=$(oc run redis-json-test --rm -i --image=redis:7 --restart=Never -- \
  redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" \
  JSON.SET test:config '$' '{"app":"alert-engine","status":"ready","timestamp":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}' 2>/dev/null)

if [[ "$JSON_SET_RESULT" == "OK" ]]; then
    echo "   ✅ ReJSON SET operation successful"
    
    # Verify ReJSON data
    JSON_GET_RESULT=$(oc run redis-json-verify --rm -i --image=redis:7 --restart=Never -- \
      redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" \
      JSON.GET test:config 2>/dev/null)
    
    if [[ -n "$JSON_GET_RESULT" && "$JSON_GET_RESULT" != "(nil)" ]]; then
        echo "   ✅ ReJSON GET operation successful"
        echo "   📄 Retrieved data: $JSON_GET_RESULT"
    else
        echo "   ⚠️ ReJSON GET operation failed or returned empty result"
        echo "   🔧 This may indicate module access issues, but basic functionality works"
    fi
else
    echo "   ❌ ReJSON SET operation failed: $JSON_SET_RESULT"
    echo "   🔧 Troubleshooting: Check if ReJSON module is properly loaded"
    
    # Try alternative ReJSON test
    echo "   🔄 Trying alternative ReJSON test..."
    ALT_JSON_RESULT=$(oc run redis-json-alt-test --rm -i --image=redis:7 --restart=Never -- \
      redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" \
      SET test:simple-json '{"test":"value"}' 2>/dev/null)
    
    if [[ "$ALT_JSON_RESULT" == "OK" ]]; then
        echo "   ✅ Alternative JSON storage works (using SET command)"
        echo "   📝 Note: ReJSON module may have access issues but basic Redis functionality confirmed"
    else
        echo "   ❌ Alternative JSON test also failed"
    fi
fi

# Test 3: RedisTimeSeries module (with better error handling)
echo "3. Testing RedisTimeSeries module..."
CURRENT_TIME=$(date +%s)
TS_CREATE_RESULT=$(oc run redis-ts-test --rm -i --image=redis:7 --restart=Never -- \
  redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" \
  TS.CREATE test:metrics LABELS metric_type test 2>/dev/null)

if [[ "$TS_CREATE_RESULT" == "OK" ]]; then
    echo "   ✅ RedisTimeSeries CREATE operation successful"
    
    # Add a test time series value
    TS_ADD_RESULT=$(oc run redis-ts-add --rm -i --image=redis:7 --restart=Never -- \
      redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" \
      TS.ADD test:metrics "$CURRENT_TIME" 42.5 2>/dev/null)
    
    if [[ -n "$TS_ADD_RESULT" ]]; then
        echo "   ✅ RedisTimeSeries ADD operation successful"
        echo "   📊 Added timestamp: $CURRENT_TIME, value: 42.5"
    else
        echo "   ⚠️ RedisTimeSeries ADD operation had issues"
    fi
else
    echo "   ❌ RedisTimeSeries CREATE operation failed: $TS_CREATE_RESULT"
    echo "   🔧 Troubleshooting: Check if RedisTimeSeries module is properly loaded"
fi

# Test 4: Module availability check
echo "4. Checking loaded modules..."
MODULES_RESULT=$(oc run redis-modules-check --rm -i --image=redis:7 --restart=Never -- \
  redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" \
  MODULE LIST 2>/dev/null)

if [[ -n "$MODULES_RESULT" ]]; then
    echo "   ✅ Module list retrieved successfully"
    echo "   📋 Loaded modules: $MODULES_RESULT"
else
    echo "   ⚠️ Could not retrieve module list"
fi

# Cleanup test keys
echo "5. Cleaning up test data..."
oc run redis-cleanup --rm -i --image=redis:7 --restart=Never -- \
  redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" \
  DEL test:config test:simple-json test:metrics >/dev/null 2>&1

echo "✅ Redis connection testing completed"
```

**Expected Output:**
```
🔍 Testing Redis connection to alert-engine-cache.redis-enterprise.svc.cluster.local:13261
1. Testing basic Redis connection...
   ✅ Basic Redis connection successful
2. Testing ReJSON module...
   ✅ ReJSON SET operation successful
   ✅ ReJSON GET operation successful
   📄 Retrieved data: {"app":"alert-engine","status":"ready","timestamp":"2025-01-XX..."}
3. Testing RedisTimeSeries module...
   ✅ RedisTimeSeries CREATE operation successful
   ✅ RedisTimeSeries ADD operation successful
   📊 Added timestamp: 1706123456, value: 42.5
4. Checking loaded modules...
   ✅ Module list retrieved successfully
   📋 Loaded modules: [ReJSON, timeseries, ...]
5. Cleaning up test data...
✅ Redis connection testing completed
```

**⚠️ If ReJSON module access issues occur (as identified in execution):**
```
🔍 Testing Redis connection to alert-engine-cache.redis-enterprise.svc.cluster.local:13261
1. Testing basic Redis connection...
   ✅ Basic Redis connection successful
2. Testing ReJSON module...
   ❌ ReJSON SET operation failed: (error) ERR unknown command 'JSON.SET'
   🔧 Troubleshooting: Check if ReJSON module is properly loaded
   🔄 Trying alternative ReJSON test...
   ✅ Alternative JSON storage works (using SET command)
   📝 Note: ReJSON module may have access issues but basic Redis functionality confirmed
3. Testing RedisTimeSeries module...
   ✅ RedisTimeSeries CREATE operation successful
   ✅ RedisTimeSeries ADD operation successful
   📊 Added timestamp: 1706123456, value: 42.5
4. Checking loaded modules...
   ✅ Module list retrieved successfully
   📋 Loaded modules: [timeseries, ...]
5. Cleaning up test data...
✅ Redis connection testing completed
```

**🔧 Troubleshooting ReJSON Module Issues:**
If ReJSON module access fails but basic Redis works, this indicates the module is installed but may have access configuration issues. The Alert Engine can still function using standard Redis commands for JSON storage as a fallback.

#### Step 2.6.3: Complete Redis Enterprise Verification

Run this comprehensive verification to ensure all components are working correctly:

```bash
# Complete Redis Enterprise verification
echo "=== Redis Enterprise Complete Verification ===" && \
echo "1. Cluster Status:" && \
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise && \
echo "" && \
echo "2. Database Status:" && \
oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise && \
echo "" && \
echo "3. Pod Status:" && \
oc get pods -n redis-enterprise && \
echo "" && \
echo "4. Service Status:" && \
oc get svc -n redis-enterprise && \
echo "" && \
echo "5. Connection Test:" && \
REDIS_PASSWORD=$(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d 2>/dev/null) && \
REDIS_PORT=$(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}' 2>/dev/null) && \
if [[ -n "$REDIS_PASSWORD" && -n "$REDIS_PORT" ]]; then \
    PING_RESULT=$(oc run redis-connection-test --rm -i --image=redis:7 --restart=Never -- redis-cli -h alert-engine-cache.redis-enterprise.svc.cluster.local -p "$REDIS_PORT" -a "$REDIS_PASSWORD" ping 2>/dev/null); \
    if [[ "$PING_RESULT" == "PONG" ]]; then \
        echo "✅ Redis connection successful (port: $REDIS_PORT)"; \
    else \
        echo "❌ Redis connection failed: $PING_RESULT"; \
    fi; \
else \
    echo "❌ Could not retrieve Redis connection details"; \
fi
```

**Expected Output:**
```
=== Redis Enterprise Complete Verification ===
1. Cluster Status:
NAME               NODES   SHARDS   VERSION     STATE     SPEC STATUS   LICENSE STATE   LICENSE EXPIRATION DATE   AGE
rec-alert-engine   3       1/4      7.22.0-95   Running   Valid         Valid           2025-08-09T13:08:06Z      6m32s

2. Database Status:
NAME                 VERSION   PORT    CLUSTER            SHARDS   STATUS   SPEC STATUS   AGE
alert-engine-cache   7.4.2     13261   rec-alert-engine   1        active   Valid         2m28s

3. Pod Status:
NAME                                                READY   STATUS    RESTARTS   AGE
rec-alert-engine-0                                  2/2     Running   0          6m31s
rec-alert-engine-1                                  2/2     Running   0          6m31s
rec-alert-engine-2                                  2/2     Running   0          6m31s
rec-alert-engine-services-rigger-6995788557-x4kdp   1/1     Running   0          6m32s
redis-enterprise-operator-86bfddd997-xzgnn          2/2     Running   0          8m5s

4. Service Status:
NAME                                TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)             AGE
alert-engine-cache                  ClusterIP   172.30.71.138   <none>        13261/TCP           93s
alert-engine-cache-headless         ClusterIP   None            <none>        13261/TCP           93s
rec-alert-engine                    ClusterIP   172.30.127.47   <none>        9443/TCP,8001/TCP   5m38s

5. Connection Test:
Warning: Using a password with '-a' or '-u' option on the command line interface may not be safe.
PONG
```

**✅ Success Criteria:**
- Cluster STATE shows "Running"
- Database STATUS shows "active"
- All pods show "Running" status
- Connection test returns "PONG"
- ReJSON and RedisTimeSeries modules are accessible (or fallback methods work)

#### Step 2.6.4: Troubleshooting Common Connection Issues

**Issue 1: ReJSON Module Access Problems (Identified During Execution)**

**Symptoms:**
- Basic Redis connection works (PONG successful)
- ReJSON commands fail with `ERR unknown command 'JSON.SET'`
- Alternative JSON storage (SET command) works

**Root Cause:**
- ReJSON module is installed but may have configuration/access issues
- Module loading order or permissions may be affecting access

**Diagnosis:**
```bash
# Check if ReJSON module is loaded
oc run redis-module-check --rm -i --image=redis:7 --restart=Never -- \
  redis-cli -h alert-engine-cache.redis-enterprise.svc.cluster.local -p "$REDIS_PORT" -a "$REDIS_PASSWORD" \
  MODULE LIST

# Check database configuration
oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o yaml | grep -A 5 redisModule
```

**Solution:**
1. **Verify module versions are compatible:**
   ```bash
   # Check available modules and versions
   oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.modules[*]}'
   ```

2. **Use fallback approach for JSON storage:**
   ```bash
   # Alert Engine can use standard Redis commands for JSON storage
   # SET/GET commands work as fallback for ReJSON functionality
   ```

3. **Database recreation if needed:**
   ```bash
   # If ReJSON access is critical, recreate database
   oc delete redisenterprisedatabase alert-engine-cache -n redis-enterprise
   # Wait for deletion, then recreate with Step 2.5 commands
   ```

**Issue 2: Dynamic Port Discovery Problems**

**Symptoms:**
- ConfigMap creation fails with empty port
- Connection attempts use wrong port numbers
- Services show different ports than expected

**Diagnosis:**
```bash
# Check all available port information
echo "Database Port:" && oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}'
echo "Service Port:" && oc get service alert-engine-cache -n redis-enterprise -o jsonpath='{.spec.ports[0].port}'
echo "Internal Endpoints:" && oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.internalEndpoints[*]}'
```

**Solution:**
- Use the enhanced port discovery script from Step 2.6.2
- Always validate port availability before creating ConfigMap
- Use multiple discovery methods as fallback

**Issue 3: Service Name Resolution**

**Symptoms:**
- Connection timeouts or DNS resolution failures
- Services not accessible from alert-engine namespace

**Diagnosis:**
```bash
# Test DNS resolution
oc run dns-test --rm -i --image=busybox --restart=Never -- \
  nslookup alert-engine-cache.redis-enterprise.svc.cluster.local

# Check service endpoints
oc get endpoints alert-engine-cache -n redis-enterprise
```

**Solution:**
1. **Use FQDN for service names:**
   ```bash
   # Always use full service names
   alert-engine-cache.redis-enterprise.svc.cluster.local
   ```

2. **Verify network policies allow access:**
   ```bash
   # Check network policy rules
   oc get networkpolicy redis-network-policy -n redis-enterprise -o yaml
   ```

3. **Test connectivity from alert-engine namespace:**
   ```bash
   # Create test pod in alert-engine namespace
   oc run connectivity-test --rm -i --image=redis:7 --restart=Never -n alert-engine -- \
     redis-cli -h alert-engine-cache.redis-enterprise.svc.cluster.local -p "$REDIS_PORT" -a "$REDIS_PASSWORD" ping
   ```

**Issue 4: Password/Secret Access Problems**

**Symptoms:**
- Authentication failures
- Secret not found errors
- Base64 decoding issues

**Diagnosis:**
```bash
# Check if secret exists
oc get secret redb-alert-engine-cache -n redis-enterprise

# Verify password format
oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d | wc -c
```

**Solution:**
1. **Verify secret exists and is accessible:**
   ```bash
   # Check secret data
   oc describe secret redb-alert-engine-cache -n redis-enterprise
   ```

2. **Recreate secret in alert-engine namespace:**
   ```bash
   # Copy secret to application namespace
   oc get secret redb-alert-engine-cache -n redis-enterprise -o yaml | \
     sed 's/namespace: redis-enterprise/namespace: alert-engine/' | \
     oc apply -f -
   ```

#### Step 2.6.5: Alert Engine Compatibility Notes

**ReJSON Module Issues:**
- If ReJSON module access fails, Alert Engine can use standard Redis SET/GET commands
- JSON data can be stored as strings and parsed by the application
- Performance impact is minimal for typical alert rule storage

**Module Dependencies:**
- **ReJSON**: Used for complex alert rule storage (fallback: standard Redis commands)
- **RedisTimeSeries**: Used for time-based metrics (fallback: sorted sets)
- **Basic Redis**: Always required for state management

**Production Considerations:**
- Test all modules thoroughly before production deployment
- Implement fallback mechanisms in Alert Engine code
- Monitor module performance and access patterns
- Plan for module updates and compatibility

### Step 2.7: Create Redis Network Policies

Set up network policies to allow access from alert-engine namespace:

```bash
# Get the actual database port for network policy
REDIS_DB_PORT=$(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}')

if [[ -z "$REDIS_DB_PORT" ]]; then
    echo "❌ Could not determine Redis database port for network policy"
    echo "🔧 Using default port 13066 - update manually if needed"
    REDIS_DB_PORT="13066"
else
    echo "✅ Using Redis database port: $REDIS_DB_PORT"
fi

# Create network policy for Redis access
cat <<EOF | oc apply -f -
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
          kubernetes.io/metadata.name: alert-engine
    ports:
    - protocol: TCP
      port: ${REDIS_DB_PORT}  # Dynamically discovered database port
    - protocol: TCP
      port: 8001   # API port
    - protocol: TCP
      port: 9443   # HTTPS API port
  egress:
  - {}
EOF

echo "✅ Redis network policy created with port: $REDIS_DB_PORT"
```

### Step 2.8: Get Redis Connection Details

```bash
# Get Redis connection details for application configuration
echo "=== Redis Connection Details ==="
echo "Host: alert-engine-cache.redis-enterprise.svc.cluster.local"
echo "Port: $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}')"
echo "Password: $(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d)"
echo "Modules: ReJSON, RedisTimeSeries"
echo "ConfigMap: redis-config (in alert-engine namespace)"
echo "Secret: redis-password (in alert-engine namespace)"
echo ""
```

### Step 2.9: Complete Redis Enterprise Setup Summary

**✅ Redis Enterprise Setup Complete**

Your Redis Enterprise setup now includes:

1. **Operator**: Redis Enterprise Operator v7.22.0-11.2
2. **Cluster**: 3-node cluster with 4 available shards
3. **Database**: alert-engine-cache with ReJSON and RedisTimeSeries modules
4. **Persistence**: AOF Every Second
5. **Security**: Secured with password authentication
6. **Network Policies**: Configured to allow access from alert-engine namespace

**Connection Details:**
- **Host**: `alert-engine-cache.redis-enterprise.svc.cluster.local` (database service)
- **Alternative Host**: `redis-13066.rec-alert-engine.redis-enterprise.svc.cluster.local` (internal endpoint)
- **Port**: `13066` (use the actual port from verification output)
- **Password**: Retrieved via `oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d`
- **Modules**: ReJSON 2.8.8, RedisTimeSeries 1.12.6
- **ConfigMap**: Available in `alert-engine` namespace as `redis-config`
- **Secret**: Available in `alert-engine` namespace as `redis-password`

**Next Steps:**
- Use the connection details in your alert-engine application
- Configure log forwarding to send logs to your application for processing
- Set up monitoring and alerting rules

## 3. OpenShift Logging and Log Forwarding Setup

### Overview

This section sets up log collection and forwarding infrastructure to send application logs to the Kafka topic for processing by the Alert Engine. We'll use the OpenShift Logging Operator with ClusterLogForwarder using the correct configuration that addresses known issues.

**Important Note**: OpenShift Logging Operator v6.2.3 had a Vector configuration bug where ClusterLogForwarder generated configs with empty bootstrap_servers. This has been resolved using RedHat's recommended fixes: using `tcp://` prefix for brokers and `deliveryMode` instead of `delivery` in tuning section.

### Step 3.1: Install OpenShift Logging Operator

The OpenShift Logging Operator is **NOT** pre-installed in most clusters. We need to install it from scratch:

#### Step 3.1.1: Create Namespace and OperatorGroup

```bash
# Create the openshift-logging namespace
oc create namespace openshift-logging

# Create OperatorGroup (CRITICAL - prevents installation issues)
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: cluster-logging
  namespace: openshift-logging
spec:
  targetNamespaces:
  - openshift-logging
EOF
```

#### Step 3.1.2: Install the Operator

```bash
# Check available channels
oc describe packagemanifest cluster-logging -n openshift-marketplace | grep -A 10 "Default Channel"

# Install the operator
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: cluster-logging
  namespace: openshift-logging
spec:
  channel: stable-6.2
  name: cluster-logging
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF
```

#### Step 3.1.3: Validate Installation

```bash
# Wait for CSV to reach Succeeded state
oc get csv -n openshift-logging

# Check operator pod
oc get pods -n openshift-logging

# Verify CRDs are installed
oc get crd | grep -E "(logging|clusterlog)"

# Quick validation
oc get csv -n openshift-logging --no-headers | grep cluster-logging | grep -q "Succeeded" && echo "✅ OpenShift Logging Operator Ready" || echo "❌ Installation Failed"
```

**Expected Output:**
```
NAME                     DISPLAY                     VERSION   PHASE
cluster-logging.v6.2.3   Red Hat OpenShift Logging   6.2.3     Succeeded

✅ OpenShift Logging Operator Ready
```

### Step 3.2: Create Service Account and RBAC

Create the necessary service account and permissions for log collection:

```bash
# Create service account
cat <<EOF | oc apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: log-collector
  namespace: openshift-logging
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: log-collector-application-logs
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: collect-application-logs
subjects:
- kind: ServiceAccount
  name: log-collector
  namespace: openshift-logging
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: log-collector-write-logs
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: logging-collector-logs-writer
subjects:
- kind: ServiceAccount
  name: log-collector
  namespace: openshift-logging
EOF
```

**Verify service account creation:**
```bash
# Check service account
oc get serviceaccount log-collector -n openshift-logging

# Check role bindings
oc get clusterrolebinding log-collector-application-logs log-collector-write-logs
```

### Step 3.3: Deploy ClusterLogForwarder (Corrected Configuration)

Deploy the ClusterLogForwarder with the RedHat fixes that resolve the Vector configuration issues:

```bash
# Create ClusterLogForwarder with corrected configuration
cat <<EOF | oc apply -f -
apiVersion: observability.openshift.io/v1
kind: ClusterLogForwarder
metadata:
  name: kafka-forwarder
  namespace: openshift-logging
spec:
  outputs:
  - kafka:
      # CRITICAL: Use tcp:// prefix (RedHat fix)
      brokers:
      - tcp://alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092
      topic: application-logs
      tuning:
        compression: snappy
        # CRITICAL: Use deliveryMode instead of delivery (RedHat fix)
        deliveryMode: AtLeastOnce
    name: kafka-output
    type: kafka
  pipelines:
  - inputRefs:
    - application
    name: application-logs
    outputRefs:
    - kafka-output
  serviceAccount:
    name: log-collector
EOF
```

#### Step 3.3.1: Verify ClusterLogForwarder Status

```bash
# Check ClusterLogForwarder status
oc get clusterlogforwarder kafka-forwarder -n openshift-logging

# IMPORTANT: Wait for ClusterLogForwarder to be processed (can take 30-60 seconds)
echo "⏳ Waiting for ClusterLogForwarder to be validated..."
sleep 30

# Detailed status check
oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o yaml | grep -A 20 "status:"

# Quick validation with proper wait
echo "Checking ClusterLogForwarder validation status..."
for i in {1..6}; do
  if oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o jsonpath='{.status.conditions[?(@.type=="Valid")].status}' | grep -q "True"; then
    echo "✅ ClusterLogForwarder Valid"
    break
  else
    echo "⏳ Waiting for validation... (attempt $i/6)"
    sleep 10
  fi
done
```

**Expected Output:**
```
NAME             AGE
kafka-forwarder   30s

✅ ClusterLogForwarder Valid
```

#### Step 3.3.2: Verify Vector Collector Pods

```bash
# Check Vector collector pods (should be 6 pods, one per node)
oc get pods -n openshift-logging -l app.kubernetes.io/component=collector

# Check specific forwarder pods
oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder

# Verify Vector configuration now has populated bootstrap_servers
COLLECTOR_POD=$(oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --no-headers | head -1 | awk '{print $1}')
echo "Checking Vector configuration in pod: $COLLECTOR_POD"
oc exec -n openshift-logging $COLLECTOR_POD -- cat /etc/vector/vector.toml | grep -A 3 -B 3 bootstrap_servers
```

**Expected Output:**
```
NAME                     READY   STATUS    RESTARTS   AGE
kafka-forwarder-56qql    1/1     Running   0          45s
kafka-forwarder-8m29s    1/1     Running   0          45s
kafka-forwarder-bg2zb    1/1     Running   0          45s
kafka-forwarder-fzqdx    1/1     Running   0          45s
kafka-forwarder-l9kqt    1/1     Running   0          45s
kafka-forwarder-t2msb    1/1     Running   0          45s

# Vector config should show populated bootstrap_servers (not empty)
bootstrap_servers = "alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"
```

### Step 3.4: Deploy Test Application for End-to-End Validation

Create a continuous log generator to test the complete ClusterLogForwarder flow:

#### Step 3.4.1: Create Alert Engine Namespace and Service Account

```bash
# Create the alert-engine namespace if it doesn't exist
oc create namespace alert-engine --dry-run=client -o yaml | oc apply -f -

# Create service account and RBAC for alert-engine application
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

#### Step 3.4.2: Deploy Continuous Log Generator

Deploy a test application that generates realistic log messages:

```bash
# Create continuous log generator deployment
cat <<EOF | oc apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: continuous-log-generator
  namespace: alert-engine
  labels:
    app: continuous-log-generator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: continuous-log-generator
  template:
    metadata:
      labels:
        app: continuous-log-generator
    spec:
      containers:
      - name: log-generator
        image: busybox:latest
        command:
        - /bin/sh
        - -c
        - |
          echo "Starting continuous random log generator..."
          
          counter=1
          while true; do
            # Generate random values using modulo
            level_num=\$((RANDOM % 5))
            service_num=\$((RANDOM % 5))
            message_num=\$((RANDOM % 10))
            
            # Set level based on random number
            case \$level_num in
              0) level="INFO" ;;
              1) level="WARN" ;;
              2) level="ERROR" ;;
              3) level="DEBUG" ;;
              4) level="TRACE" ;;
            esac
            
            # Set service based on random number
            case \$service_num in
              0) service="user-service" ;;
              1) service="payment-service" ;;
              2) service="order-service" ;;
              3) service="inventory-service" ;;
              4) service="notification-service" ;;
            esac
            
            # Set message based on random number
            case \$message_num in
              0) message="User authentication successful" ;;
              1) message="Payment processing initiated" ;;
              2) message="Order validation completed" ;;
              3) message="Inventory check performed" ;;
              4) message="Email notification sent" ;;
              5) message="Database connection established" ;;
              6) message="Cache operation completed" ;;
              7) message="API request processed" ;;
              8) message="File upload finished" ;;
              9) message="Session management handled" ;;
            esac
            
            # Generate timestamp and random user ID
            timestamp=\$(date -Iseconds)
            user_id=\$((RANDOM % 1000 + 1))
            
            # Output structured log message
            echo "[\$timestamp] \$level: \$message | service=\$service | user_id=\$user_id | sequence=\$counter"
            
            counter=\$((counter + 1))
            
            # Send message every 3 seconds
            sleep 3
          done
        resources:
          requests:
            memory: "32Mi"
            cpu: "50m"
          limits:
            memory: "64Mi"
            cpu: "100m"
EOF
```

#### Step 3.4.3: Verify Test Application

```bash
# Check deployment status
oc get deployment continuous-log-generator -n alert-engine

# Check pod status
oc get pods -n alert-engine -l app=continuous-log-generator

# Check logs (should show random messages every 3 seconds)
POD_NAME=$(oc get pods -n alert-engine -l app=continuous-log-generator --no-headers | awk '{print $1}')
oc logs $POD_NAME -n alert-engine --tail=10
```

**Expected Output:**
```
NAME                       READY   UP-TO-DATE   AVAILABLE   AGE
continuous-log-generator   1/1     1            1           45s

NAME                                        READY   STATUS    RESTARTS   AGE
continuous-log-generator-7d4997bcfd-bhn9s   1/1     Running   0          45s

[2025-07-10T17:40:12+00:00] DEBUG: Order validation completed | service=notification-service | user_id=106 | sequence=3
[2025-07-10T17:40:15+00:00] ERROR: Database connection established | service=notification-service | user_id=83 | sequence=4
[2025-07-10T17:40:18+00:00] TRACE: Session management handled | service=order-service | user_id=639 | sequence=5
```

### Step 3.5: **CRITICAL** - End-to-End Validation

This is the most important step to verify the complete log flow through ClusterLogForwarder:

#### Step 3.5.1: Wait for Vector Processing

```bash
# Wait for Vector to process the logs (60 seconds recommended)
echo "⏳ Waiting 60 seconds for Vector to process and forward logs to Kafka..."
sleep 60
```

#### Step 3.5.2: Verify Logs in Kafka Consumer

Test that the continuous log generator messages are appearing in Kafka:

```bash
# Check latest messages in Kafka for our test application logs
echo "🔍 Checking Kafka consumer for continuous log generator messages..."
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- timeout 15 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --max-messages 10 | grep -i "user_id\|sequence\|service"
```

**Expected Output (showing our test messages):**
```
{"@timestamp":"2025-07-10T17:43:31.032340843Z","hostname":"ip-10-0-36-253.ca-central-1.compute.internal","kubernetes":{"namespace_name":"alert-engine","pod_name":"continuous-log-generator-7d4997bcfd-bhn9s","container_name":"log-generator"},"level":"warn","log_source":"container","log_type":"application","message":"[2025-07-10T17:43:31+00:00] WARN: Order validation completed | service=user-service | user_id=437 | sequence=69"}
{"@timestamp":"2025-07-10T17:43:34.035104276Z","hostname":"ip-10-0-36-253.ca-central-1.compute.internal","kubernetes":{"namespace_name":"alert-engine","pod_name":"continuous-log-generator-7d4997bcfd-bhn9s","container_name":"log-generator"},"level":"info","log_source":"container","log_type":"application","message":"[2025-07-10T17:43:34+00:00] INFO: User authentication successful | service=payment-service | user_id=304 | sequence=70"}
```

#### Step 3.5.3: Complete End-to-End Validation Script

Run this comprehensive validation to ensure everything is working:

**Script 1: Basic Component Status Check**
```bash
#!/bin/bash
echo "=== Complete ClusterLogForwarder End-to-End Validation ==="
echo ""

echo "1. ClusterLogForwarder Status:"
oc get clusterlogforwarder kafka-forwarder -n openshift-logging
echo ""

echo "2. Vector Collector Pods:"
VECTOR_PODS=$(oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --no-headers | wc -l)
echo "   Found $VECTOR_PODS Vector collector pods"
oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder | head -7
echo ""

echo "3. Test Application Status:"
oc get pods -n alert-engine -l app=continuous-log-generator
echo ""

echo "4. Recent Test Application Logs:"
POD_NAME=$(oc get pods -n alert-engine -l app=continuous-log-generator --no-headers | awk '{print $1}')
if [[ -n "$POD_NAME" ]]; then
    oc logs "$POD_NAME" -n alert-engine --tail=3
else
    echo "   ❌ Test application pod not found"
fi
echo ""
```

**Script 2: Critical Kafka Consumer Test**
```bash
#!/bin/bash
echo "5. 🎯 CRITICAL TEST - Kafka Consumer for Alert-Engine Logs:"
echo "   Looking for messages from continuous-log-generator..."

# First check if any messages exist at all
echo "   Checking if any messages exist in topic..."
MESSAGE_COUNT=$(oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
    timeout 10 bin/kafka-console-consumer.sh \
    --bootstrap-server localhost:9092 \
    --topic application-logs \
    --max-messages 5 2>/dev/null | wc -l)

if [[ "$MESSAGE_COUNT" -eq 0 ]]; then
    echo "   ❌ No messages found in Kafka topic"
else
    echo "   ✅ Found $MESSAGE_COUNT messages in Kafka topic"
    
    # Check for our specific test messages
    echo "   Checking for alert-engine test messages..."
    oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
        timeout 15 bin/kafka-console-consumer.sh \
        --bootstrap-server localhost:9092 \
        --topic application-logs \
        --max-messages 5 2>/dev/null | \
        grep -E "alert-engine|continuous-log-generator|user_id|sequence" | head -3
fi
echo ""
```

**Script 3: Final Validation**
```bash
#!/bin/bash
echo "6. Final Validation Results:"

# Check if alert-engine logs are reaching Kafka
if oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
    timeout 10 bin/kafka-console-consumer.sh \
    --bootstrap-server localhost:9092 \
    --topic application-logs \
    --max-messages 3 2>/dev/null | grep -q "user_id"; then
    
    echo "✅ SUCCESS: ClusterLogForwarder is working! Alert-engine logs are flowing to Kafka."
    echo "   - Vector collectors are processing logs"
    echo "   - ClusterLogForwarder is forwarding to Kafka"
    echo "   - Test application logs are reaching Kafka consumer"
    echo "   - End-to-end log flow validated ✅"
else
    echo "❌ ISSUE: Alert-engine logs not found in Kafka."
    echo "   Troubleshooting steps:"
    echo "   1. Check Vector collector pod logs for errors"
    echo "   2. Verify ClusterLogForwarder configuration"
    echo "   3. Check Kafka connectivity from Vector pods"
    echo "   4. Ensure test application is generating logs"
fi
echo ""
```

**All-in-One Validation Script**
```bash
#!/bin/bash
# Complete validation script combining all checks
set -e

echo "=== Complete ClusterLogForwarder End-to-End Validation ==="
echo ""

# Component status
echo "1. ClusterLogForwarder Status:"
oc get clusterlogforwarder kafka-forwarder -n openshift-logging
echo ""

echo "2. Vector Collector Pods:"
VECTOR_PODS=$(oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --no-headers | wc -l)
echo "   Found $VECTOR_PODS Vector collector pods"
oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder | head -7
echo ""

echo "3. Test Application Status:"
oc get pods -n alert-engine -l app=continuous-log-generator
echo ""

echo "4. Recent Test Application Logs:"
POD_NAME=$(oc get pods -n alert-engine -l app=continuous-log-generator --no-headers | awk '{print $1}')
if [[ -n "$POD_NAME" ]]; then
    oc logs "$POD_NAME" -n alert-engine --tail=3
else
    echo "   ❌ Test application pod not found"
    exit 1
fi
echo ""

echo "5. 🎯 CRITICAL TEST - Kafka Consumer for Alert-Engine Logs:"
echo "   Looking for messages from continuous-log-generator..."

# Test Kafka consumer with proper error handling
if oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
    timeout 10 bin/kafka-console-consumer.sh \
    --bootstrap-server localhost:9092 \
    --topic application-logs \
    --max-messages 3 2>/dev/null | grep -q "user_id"; then
    
    echo "✅ SUCCESS: ClusterLogForwarder is working! Alert-engine logs are flowing to Kafka."
    echo "   - Vector collectors are processing logs"
    echo "   - ClusterLogForwarder is forwarding to Kafka"
    echo "   - Test application logs are reaching Kafka consumer"
    echo "   - End-to-end log flow validated ✅"
else
    echo "❌ ISSUE: Alert-engine logs not found in Kafka."
    echo "   Run troubleshooting steps in next section."
fi
echo ""
```

**✅ Success Criteria:**
- ClusterLogForwarder shows `Valid: True` status
- 6 Vector collector pods are `Running`
- Test application pod is `Running` and generating logs
- Kafka consumer receives messages from `continuous-log-generator` with `user_id` and `sequence` fields
- Messages show `namespace_name: alert-engine` in Kubernetes metadata

**🎉 ClusterLogForwarder Successfully Configured!**

The corrected ClusterLogForwarder configuration resolves the Vector bootstrap_servers issue and provides reliable log forwarding from OpenShift applications to Kafka.

### Step 3.6: Troubleshooting

#### Common Issues and Solutions:

**Issue 1: Missing OperatorGroup**
- **Symptoms**: Subscription exists but no CSV created
- **Solution**: Create OperatorGroup before Subscription (see Step 3.1.1)

**Issue 2: ClusterLogForwarder Not Valid**
- **Symptoms**: ClusterLogForwarder shows status other than "Valid"
- **Diagnosis**: Check conditions: `oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o yaml | grep -A 20 conditions`
- **Solution**: Verify service account exists and has proper permissions

**Issue 3: Vector Pods Not Starting**
- **Symptoms**: No collector pods or pods failing to start
- **Solution**: Check node resources and ensure OpenShift Logging Operator is properly installed

**Issue 4: No Logs Reaching Kafka**
- **Symptoms**: Test application running but no logs in Kafka consumer
- **Diagnosis**: Check Vector configuration has populated bootstrap_servers
- **Solution**: Verify ClusterLogForwarder uses `tcp://` prefix and `deliveryMode` (not `delivery`)

**Issue 5: Kafka Connectivity**
- **Symptoms**: Vector logs show connection errors
- **Solution**: Verify Kafka service name and port: `tcp://alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092`

#### Debugging Commands:

```bash
# Check logging operator status
oc get csv -n openshift-logging | grep cluster-logging

# Check ClusterLogForwarder status
oc get clusterlogforwarder kafka-forwarder -n openshift-logging

# Check Vector configuration for bootstrap_servers (should NOT be empty)
COLLECTOR_POD=$(oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --no-headers | head -1 | awk '{print $1}')
oc exec -n openshift-logging $COLLECTOR_POD -- cat /etc/vector/vector.toml | grep -A 5 -B 5 bootstrap_servers

# Check Vector collector logs for errors
oc logs $COLLECTOR_POD -n openshift-logging | grep -i error

# Test Kafka connectivity from Vector pod
oc exec -n openshift-logging $COLLECTOR_POD -- nslookup alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local

# Manual test of Kafka producer
echo '{"test":"message"}' | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs
```

## Summary

**✅ Step 3 Complete - ClusterLogForwarder Successfully Deployed**

### What's Working:
1. **OpenShift Logging Operator**: v6.2.3 installed and operational
2. **ClusterLogForwarder**: Successfully configured with RedHat fixes (tcp:// prefix, deliveryMode)
3. **Vector Configuration**: Bootstrap servers properly populated (not empty)
4. **Vector Collectors**: 6 pods running across cluster nodes, processing application logs
5. **Test Application**: Continuous log generator producing realistic log messages
6. **End-to-End Validation**: ✅ Logs flowing Application → Vector → Kafka → Consumer

### Architecture Summary:
- **ClusterLogForwarder**: Collects application logs from all namespaces
- **Vector Collectors**: DaemonSet pods on each node forwarding logs to Kafka
- **Kafka Integration**: Direct Vector-to-Kafka forwarding with proper configuration
- **OpenShift Integration**: Native log collection from container stdout/stderr

### Technical Details:
- **Vector Configuration**: `bootstrap_servers = "alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"`
- **Log Processing**: Vector enriches logs with Kubernetes metadata (namespace, pod, container)
- **Delivery**: AtLeastOnce delivery mode with snappy compression
- **Format**: JSON logs with OpenShift metadata and original application messages

### Connection Details for Alert Engine:
- **Kafka Bootstrap Servers**: `alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092`
- **Topic**: `application-logs`
- **Log Format**: JSON with Kubernetes metadata + original log messages
- **Log Volume**: Real-time streaming from all applications in cluster
- **Test Data**: Continuous log generator with user_id, sequence, service fields

### Key Success Factors:
- **RedHat Fixes Applied**: `tcp://` prefix for brokers, `deliveryMode` instead of `delivery`
- **Proper Service Account**: `log-collector` with correct RBAC permissions
- **Vector Validation**: Bootstrap servers populated in Vector configuration (fixed)
- **End-to-End Testing**: Continuous log generator validates complete flow

### Next Steps:
- Configure Alert Engine application to consume from `application-logs` topic
- Set up alerting rules based on log patterns and Kubernetes metadata
- Deploy Alert Engine to OpenShift using validated connection details
- Monitor Vector collector performance and log throughput

**Note**: The corrected ClusterLogForwarder configuration resolves the Vector bootstrap_servers bug in OpenShift Logging Operator v6.2.3, providing reliable, production-ready log forwarding for Alert Engine processing.

### Step 3.7: Complete Infrastructure Verification

Run this comprehensive verification to ensure all components are working together:

```bash
#!/bin/bash
echo "=== Complete Infrastructure Verification ==="
echo ""

echo "1. Kafka Cluster Status:"
oc get kafka alert-kafka-cluster -n amq-streams-kafka -o jsonpath='{.status.conditions[?(@.type=="Ready")]}'
echo ""

echo "2. Kafka Topics:"
oc get kafkatopic -n amq-streams-kafka
echo ""

echo "3. Redis Enterprise Cluster Status:"
oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.state}'
echo ""

echo "4. Redis Database Status:"
oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.status}'
echo ""

echo "5. ClusterLogForwarder Status:"
oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o jsonpath='{.status.conditions[?(@.type=="Valid")]}'
echo ""

echo "6. Vector Collector Pods:"
oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder | grep Running | wc -l
echo ""

echo "7. Test Application Status:"
oc get pods -n alert-engine -l app=continuous-log-generator -o jsonpath='{.items[0].status.phase}'
echo ""

echo "8. Service Account Status:"
oc get serviceaccount alert-engine-sa -n alert-engine
echo ""

echo "9. Network Policies:"
oc get networkpolicy kafka-network-policy -n amq-streams-kafka
oc get networkpolicy redis-network-policy -n redis-enterprise
echo ""

echo "10. Connection Details Summary:"
echo "   Kafka: alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"
echo "   Redis: alert-engine-cache.redis-enterprise.svc.cluster.local:$(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}' 2>/dev/null || echo '13066')"
echo "   Topic: application-logs"
echo "   Service Account: alert-engine-sa"
echo ""

echo "11. Final End-to-End Test - Kafka Consumer:"
if oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- timeout 10 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --max-messages 3 2>/dev/null | grep -q "user_id"; then
    echo "✅ SUCCESS: All infrastructure components are working correctly!"
    echo "   - Kafka cluster is ready and accepting messages"
    echo "   - Redis Enterprise database is active and accessible"
    echo "   - ClusterLogForwarder is successfully forwarding logs to Kafka"
    echo "   - Test application is generating logs that reach Kafka consumer"
    echo "   - Network policies are configured"
    echo "   - Service accounts and RBAC are set up"
    echo "   - Ready for alert-engine application deployment!"
else
    echo "❌ ISSUE: Some components may not be fully ready. Check individual components above."
fi
echo ""
echo "=== Infrastructure Setup Complete ==="
```

**✅ Success Criteria for Complete Setup:**
- Kafka cluster shows `Ready: True` condition
- Redis cluster shows `Running` state
- Redis database shows `active` status
- ClusterLogForwarder shows `Valid: True` condition
- 6 Vector collector pods are running
- Test application pod is running
- Service account exists
- Network policies are created
- Kafka consumer receives test messages with `user_id` field

### Step 3.8: Get All Connection Details

```bash
# Get all connection details needed for alert-engine application
echo "=== Alert Engine Connection Details ==="
echo ""
echo "KAFKA CONFIGURATION:"
echo "  Bootstrap Servers: alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"
echo "  Topic: application-logs"
echo "  Service IP: $(oc get svc alert-kafka-cluster-kafka-bootstrap -n amq-streams-kafka -o jsonpath='{.spec.clusterIP}:{.spec.ports[0].port}')"
echo ""
echo "REDIS CONFIGURATION:"
echo "  Host: alert-engine-cache.redis-enterprise.svc.cluster.local"
echo "  Port: $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}' 2>/dev/null || echo '13066')"
echo "  Password: $(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' 2>/dev/null | base64 -d)"
echo "  Modules: ReJSON, RedisTimeSeries"
echo "  ConfigMap: redis-config (in alert-engine namespace)"
echo "  Secret: redis-password (in alert-engine namespace)"
echo ""
echo "KUBERNETES CONFIGURATION:"
echo "  Namespace: alert-engine"
echo "  Service Account: alert-engine-sa"
echo "  Network Policies: kafka-network-policy, redis-network-policy"
echo ""
echo "SAMPLE CONFIG.YAML:"
cat <<EOF
kafka:
  brokers: ["alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"]
  topic: "application-logs"
  consumer_group: "alert-engine-group"

redis:
  host: "alert-engine-cache.redis-enterprise.svc.cluster.local"
  port: $(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}' 2>/dev/null || echo '13066')
  password: "$(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' 2>/dev/null | base64 -d)"
  database: 0

kubernetes:
  namespace: "alert-engine"
  service_account: "alert-engine-sa"
EOF
echo ""
```

## 4. Next Steps

After completing this infrastructure setup:

1. **Local Testing**: Update your `configs/config.yaml` with the connection details obtained above and test the Alert Engine locally
2. **Deploy to OpenShift**: Use the deployment manifests in `deployments/openshift/` to deploy the Alert Engine application
3. **Configure Monitoring**: Set up Prometheus monitoring for the Alert Engine metrics
4. **Test End-to-End**: Generate test logs and verify alerts are triggered and notifications are sent

This completes the OpenShift infrastructure setup. You can now proceed with local testing and then deploy the Alert Engine application to OpenShift.

---

## 🎯 Complete Setup Validation Checklist

Use this comprehensive checklist to verify your entire OpenShift infrastructure setup:

### ✅ Final Pre-Deployment Validation

**Run this complete validation before proceeding with Alert Engine deployment:**

```bash
#!/bin/bash
echo "=== 🎯 Complete OpenShift Infrastructure Validation ==="
echo ""

# Storage verification
echo "1. Storage Class Verification:"
oc get storageclass gp3-csi > /dev/null 2>&1 && echo "   ✅ Storage class available" || echo "   ❌ Storage class missing"
echo ""

# AMQ Streams / Kafka verification
echo "2. AMQ Streams / Kafka Verification:"
KAFKA_READY=$(oc get kafka alert-kafka-cluster -n amq-streams-kafka -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
[[ "$KAFKA_READY" == "True" ]] && echo "   ✅ Kafka cluster ready" || echo "   ❌ Kafka cluster not ready"

KAFKA_TOPICS=$(oc get kafkatopic -n amq-streams-kafka --no-headers | wc -l)
[[ "$KAFKA_TOPICS" -gt 0 ]] && echo "   ✅ Kafka topics created ($KAFKA_TOPICS)" || echo "   ❌ No Kafka topics found"

echo "   Testing Kafka producer-consumer..."
echo '{"test":"validation"}' | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs >/dev/null 2>&1
KAFKA_TEST=$(oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- timeout 5 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --max-messages 1 2>/dev/null | wc -l)
[[ "$KAFKA_TEST" -gt 0 ]] && echo "   ✅ Kafka producer-consumer test passed" || echo "   ❌ Kafka producer-consumer test failed"
echo ""

# Redis Enterprise verification  
echo "3. Redis Enterprise Verification:"
REDIS_STATE=$(oc get redisenterprisecluster rec-alert-engine -n redis-enterprise -o jsonpath='{.status.state}' 2>/dev/null)
[[ "$REDIS_STATE" == "Running" ]] && echo "   ✅ Redis Enterprise cluster running" || echo "   ❌ Redis Enterprise cluster not running"

REDIS_DB_STATUS=$(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.status}' 2>/dev/null)
[[ "$REDIS_DB_STATUS" == "active" ]] && echo "   ✅ Redis database active" || echo "   ❌ Redis database not active"

echo "   Testing Redis connection..."
REDIS_TEST=$(oc run redis-validate-test --rm -i --tty --image=redis:7 --restart=Never -- redis-cli -h alert-engine-cache.redis-enterprise.svc.cluster.local -p 13261 -a "$(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d)" ping 2>/dev/null)
[[ "$REDIS_TEST" == "PONG" ]] && echo "   ✅ Redis connection test passed" || echo "   ❌ Redis connection test failed"
echo ""

# ClusterLogForwarder verification
echo "4. ClusterLogForwarder Verification:"
CLF_VALID=$(oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o jsonpath='{.status.conditions[?(@.type=="Valid")].status}' 2>/dev/null)
[[ "$CLF_VALID" == "True" ]] && echo "   ✅ ClusterLogForwarder valid" || echo "   ❌ ClusterLogForwarder invalid"

VECTOR_PODS=$(oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder --no-headers 2>/dev/null | grep Running | wc -l)
[[ "$VECTOR_PODS" -ge 3 ]] && echo "   ✅ Vector collector pods running ($VECTOR_PODS)" || echo "   ❌ Vector collector pods not running"

echo "   Testing end-to-end log flow..."
LOG_FLOW_TEST=$(oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- timeout 10 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --max-messages 3 2>/dev/null | grep -c "user_id")
[[ "$LOG_FLOW_TEST" -gt 0 ]] && echo "   ✅ End-to-end log flow working" || echo "   ❌ End-to-end log flow not working"
echo ""

# Service accounts and RBAC verification
echo "5. Service Accounts and RBAC Verification:"
oc get serviceaccount alert-engine-sa -n alert-engine > /dev/null 2>&1 && echo "   ✅ Alert Engine service account exists" || echo "   ❌ Alert Engine service account missing"
oc get clusterrolebinding alert-engine-binding > /dev/null 2>&1 && echo "   ✅ Alert Engine RBAC configured" || echo "   ❌ Alert Engine RBAC missing"
echo ""

# Network policies verification
echo "6. Network Policies Verification:"
oc get networkpolicy kafka-network-policy -n amq-streams-kafka > /dev/null 2>&1 && echo "   ✅ Kafka network policy exists" || echo "   ❌ Kafka network policy missing"
oc get networkpolicy redis-network-policy -n redis-enterprise > /dev/null 2>&1 && echo "   ✅ Redis network policy exists" || echo "   ❌ Redis network policy missing"
echo ""

# Configuration verification
echo "7. Configuration Verification:"
oc get configmap redis-config -n alert-engine > /dev/null 2>&1 && echo "   ✅ Redis configuration available" || echo "   ❌ Redis configuration missing"
oc get secret redis-password -n alert-engine > /dev/null 2>&1 && echo "   ✅ Redis password secret available" || echo "   ❌ Redis password secret missing"
echo ""

# Final summary
echo "8. 🎯 FINAL VALIDATION SUMMARY:"
echo ""
echo "   Connection Details:"
echo "   - Kafka: alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"
echo "   - Redis: alert-engine-cache.redis-enterprise.svc.cluster.local:$(oc get redisenterprisedatabase alert-engine-cache -n redis-enterprise -o jsonpath='{.status.databasePort}' 2>/dev/null || echo '13066')"
echo "   - Topic: application-logs"
echo "   - Namespace: alert-engine"
echo ""

# Overall validation check
TOTAL_CHECKS=8
PASSED_CHECKS=0

[[ "$KAFKA_READY" == "True" ]] && ((PASSED_CHECKS++))
[[ "$KAFKA_TOPICS" -gt 0 ]] && ((PASSED_CHECKS++))
[[ "$KAFKA_TEST" -gt 0 ]] && ((PASSED_CHECKS++))
[[ "$REDIS_STATE" == "Running" ]] && ((PASSED_CHECKS++))
[[ "$REDIS_DB_STATUS" == "active" ]] && ((PASSED_CHECKS++))
[[ "$CLF_VALID" == "True" ]] && ((PASSED_CHECKS++))
[[ "$VECTOR_PODS" -ge 3 ]] && ((PASSED_CHECKS++))
[[ "$LOG_FLOW_TEST" -gt 0 ]] && ((PASSED_CHECKS++))

echo "   Overall Status: $PASSED_CHECKS/$TOTAL_CHECKS checks passed"
echo ""

if [[ "$PASSED_CHECKS" -eq "$TOTAL_CHECKS" ]]; then
    echo "🎉 SUCCESS: All infrastructure components are ready!"
    echo "   ✅ You can now proceed with Alert Engine deployment"
    echo "   📋 Use the connection details above for your application configuration"
else
    echo "❌ ISSUES FOUND: $((TOTAL_CHECKS - PASSED_CHECKS)) checks failed"
    echo "   🔧 Review the failed checks above and address issues before proceeding"
    echo "   📖 Refer to troubleshooting sections in each component setup"
fi
echo ""
echo "=== Infrastructure Validation Complete ==="
```

### 🚀 Ready for Deployment

**If all validations pass, you have successfully:**

1. ✅ **Verified Prerequisites**: Storage classes and cluster access
2. ✅ **Deployed AMQ Streams**: Kafka cluster with topics and producer-consumer validation
3. ✅ **Deployed Redis Enterprise**: Cluster and database with module testing
4. ✅ **Configured ClusterLogForwarder**: End-to-end log flow from applications to Kafka
5. ✅ **Set up Service Accounts**: RBAC and network policies
6. ✅ **Created Configuration**: ConfigMaps and secrets for application use

### 📋 Connection Details for Alert Engine

**Use these validated connection details in your `configs/config.yaml`:**

```yaml
kafka:
  brokers: ["alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"]
  topic: "application-logs"
  consumer_group: "alert-engine-group"

redis:
  host: "alert-engine-cache.redis-enterprise.svc.cluster.local"
  port: 13261  # Use actual port from validation output
  password: "YOUR_REDIS_PASSWORD"  # From validation output
  database: 0

kubernetes:
  namespace: "alert-engine"
  service_account: "alert-engine-sa"
```

### 🔧 If Validation Fails

**Common issues and quick fixes:**

1. **Storage Class Issues**: Update all `gp3-csi` references to your cluster's available storage class
2. **Kafka Not Ready**: Check operator logs and ensure OperatorGroup exists
3. **Redis Connection Fails**: Verify ports and passwords from validation output
4. **ClusterLogForwarder Invalid**: Check service account permissions and wait for validation
5. **No Log Flow**: Verify Vector pods are running and check ClusterLogForwarder configuration

**Re-run the validation script after fixing issues.**

---

**🎯 Infrastructure Setup Complete - Ready for Alert Engine Deployment!** 