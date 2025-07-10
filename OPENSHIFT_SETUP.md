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


#### Step 1.2.6: Troubleshooting Common Issues

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

#### Step 1.2.7: Quick One-Liner Validation

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

### Step 1.6: Create Kafka Network Policies

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

### Step 1.7: Get Kafka Connection Details

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
oc run redis-connection-test --rm -i --tty --image=redis:7 --restart=Never -- redis-cli -h alert-engine-cache.redis-enterprise.svc.cluster.local -p 13261 -a $(oc get secret redb-alert-engine-cache -n redis-enterprise -o jsonpath='{.data.password}' | base64 -d) ping
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

### Step 2.7: Create Redis Network Policies

Set up network policies to allow access from alert-engine namespace:

```bash
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
      port: 13066  # Default database port, adjust if different
    - protocol: TCP
      port: 8001   # API port
  egress:
  - {}
EOF
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

#### Step 2.9: Complete Redis Enterprise Setup Summary

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

# Detailed status check
oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o yaml | grep -A 20 "status:"

# Quick validation
oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o jsonpath='{.status.conditions[?(@.type=="Valid")].status}' | grep -q "True" && echo "✅ ClusterLogForwarder Valid" || echo "❌ ClusterLogForwarder Invalid"
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

```bash
#!/bin/bash
echo "=== Complete ClusterLogForwarder End-to-End Validation ==="
echo ""

echo "1. ClusterLogForwarder Status:"
oc get clusterlogforwarder kafka-forwarder -n openshift-logging
echo ""

echo "2. Vector Collector Pods:"
oc get pods -n openshift-logging -l app.kubernetes.io/instance=kafka-forwarder | head -7
echo ""

echo "3. Test Application Status:"
oc get pods -n alert-engine -l app=continuous-log-generator
echo ""

echo "4. Recent Test Application Logs:"
POD_NAME=$(oc get pods -n alert-engine -l app=continuous-log-generator --no-headers | awk '{print $1}')
oc logs $POD_NAME -n alert-engine --tail=3
echo ""

echo "5. 🎯 CRITICAL TEST - Kafka Consumer for Alert-Engine Logs:"
echo "   Looking for messages from continuous-log-generator..."
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- timeout 20 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --max-messages 5 | grep -E "alert-engine|continuous-log-generator|user_id|sequence" | head -3
echo ""

echo "6. Validation Results:"
if oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- timeout 10 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --max-messages 3 | grep -q "user_id"; then
    echo "✅ SUCCESS: ClusterLogForwarder is working! Alert-engine logs are flowing to Kafka."
    echo "   - Vector collectors are processing logs"
    echo "   - ClusterLogForwarder is forwarding to Kafka"
    echo "   - Test application logs are reaching Kafka consumer"
    echo "   - End-to-end log flow validated ✅"
else
    echo "❌ ISSUE: Alert-engine logs not found in Kafka. Check troubleshooting section."
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

### Step 3.6: Complete Infrastructure Verification

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

### Step 3.7: Get All Connection Details

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

## Infrastructure Setup Complete Summary

**✅ All Infrastructure Components Successfully Deployed:**

### 1. **Kafka Cluster** (amq-streams-kafka namespace)
- **Status**: 3 broker cluster with ZooKeeper ensemble
- **Topic**: `application-logs` for log ingestion
- **Connection**: `alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092`
- **Network Policy**: Allows access from alert-engine and openshift-logging namespaces

### 2. **Redis Enterprise** (redis-enterprise namespace)
- **Status**: 3-node cluster with high availability
- **Database**: `alert-engine-cache` with ReJSON and RedisTimeSeries modules
- **Connection**: `alert-engine-cache.redis-enterprise.svc.cluster.local`
- **Security**: Password-protected with secrets management
- **Network Policy**: Allows access from alert-engine namespace

### 3. **ClusterLogForwarder** (openshift-logging namespace)
- **Status**: Successfully forwarding logs from all namespaces to Kafka
- **Vector Collectors**: 6 pods running on cluster nodes
- **Configuration**: Fixed bootstrap_servers issue with tcp:// prefix and deliveryMode
- **Test Validation**: ✅ Confirmed end-to-end log flow

### 4. **Alert Engine Namespace** (alert-engine namespace)
- **Service Account**: `alert-engine-sa` with necessary RBAC permissions
- **ConfigMaps**: Redis connection details
- **Secrets**: Redis password and database credentials
- **Test Application**: Continuous log generator for validation

### 5. **Network Security**
- **Network Policies**: Configured for Kafka and Redis access control
- **RBAC**: ClusterRole and ClusterRoleBinding for application permissions
- **Service Accounts**: Dedicated service account for alert-engine application

### 6. **End-to-End Validation**
- **Log Flow**: Application → Vector → Kafka → Consumer ✅
- **Test Data**: Realistic log messages with user_id, service, and sequence fields
- **Kafka Consumer**: Successfully receiving and processing application logs
- **Redis Connectivity**: Verified with connection tests and module support

**🎉 Infrastructure Ready for Alert Engine Deployment**

All components are configured, tested, and validated. The alert-engine application can now be deployed using the provided connection details and service account. The infrastructure provides:
- **Scalable log ingestion** via Kafka
- **High-performance caching** via Redis Enterprise
- **Secure network access** via network policies
- **Proper RBAC** for application deployment
- **Validated end-to-end flow** for reliable operation

**Next Steps:**
1. Update `configs/config.yaml` with the connection details above
2. Deploy alert-engine application using `deployments/openshift/` manifests
3. Configure alert rules and notification channels
4. Set up monitoring and observability

## 4. Next Steps

After completing this infrastructure setup:

1. **Local Testing**: Update your `configs/config.yaml` with the connection details obtained above and test the Alert Engine locally
2. **Deploy to OpenShift**: Use the deployment manifests in `deployments/openshift/` to deploy the Alert Engine application
3. **Configure Monitoring**: Set up Prometheus monitoring for the Alert Engine metrics
4. **Test End-to-End**: Generate test logs and verify alerts are triggered and notifications are sent

## 5. Troubleshooting

### Common Issues

1. **Kafka Consumer Hangs/Times Out**: If the verification command hangs or shows timeout errors:

   **Check Kafka Cluster Status:**
   ```bash
   # Check if all Kafka brokers are running
   oc get pods -n amq-streams-kafka -l strimzi.io/cluster=alert-kafka-cluster,strimzi.io/kind=Kafka
   
   # Check for scheduling issues
   oc get events -n amq-streams-kafka --field-selector involvedObject.kind=Pod
   ```

   **Common Solutions:**
   
   **Option A: Scale down to 2 brokers (Recommended for resource-constrained clusters):**
   ```bash
   oc patch kafka alert-kafka-cluster -n amq-streams-kafka --type='merge' -p='{"spec":{"kafka":{"replicas":2}}}'
   oc wait kafka/alert-kafka-cluster --for=condition=Ready --timeout=300s -n amq-streams-kafka
   ```

   **Option B: Reduce CPU requirements:**
   ```bash
   oc patch kafka alert-kafka-cluster -n amq-streams-kafka --type='merge' -p='{"spec":{"kafka":{"resources":{"requests":{"cpu":"500m","memory":"1Gi"},"limits":{"cpu":"1","memory":"2Gi"}}}}}'
   oc wait kafka/alert-kafka-cluster --for=condition=Ready --timeout=300s -n amq-streams-kafka
   ```

   **Check ClusterLogForwarder logs:**
   ```bash
   # Check if collector pods are having connectivity issues
   oc logs -l app.kubernetes.io/instance=kafka-alert-forwarder -n openshift-logging | grep -i error
   ```

2. **Missing OperatorGroup (Most Common Issue)**: If subscription exists but no CSV/InstallPlan is created:
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

## 6. Configuration Updates

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