# OpenShift Infrastructure Setup Guide

This guide provides step-by-step instructions to set up the required infrastructure components on OpenShift before deploying the Alert Engine application.

## 📋 Table of Contents

1. [Prerequisites](#prerequisites)
2. [Overview](#overview)
3. [Expected Deployment Times](#-expected-deployment-times)
4. [Step 1: Kafka Setup using Red Hat AMQ Streams](#1-kafka-setup-using-red-hat-amq-streams)
   - [1.1: Install AMQ Streams Operator](#step-11-install-amq-streams-operator)
   - [1.2: Validate AMQ Streams Operator Installation](#step-12-validate-amq-streams-operator-installation)
   - [1.3: Deploy Kafka Cluster](#step-13-deploy-kafka-cluster)
   - [1.4: Create Kafka Topics](#step-14-create-kafka-topics)
   - [1.5: Verify Kafka Installation](#step-15-verify-kafka-installation)
   - [1.6: Test Kafka Producer-Consumer Flow](#step-16-test-kafka-producer-consumer-flow)
   - [1.7: Create Kafka Network Policies](#step-17-create-kafka-network-policies)
   - [1.8: Get Kafka Connection Details](#step-18-get-kafka-connection-details)
5. [Step 2: Redis Cluster (HA) Setup](#2-redis-cluster-ha-setup)
   - [2.1: Create Redis Cluster Namespace](#step-21-create-redis-cluster-namespace)
   - [2.2: Deploy Redis Cluster](#step-22-deploy-redis-cluster)
     - [Troubleshooting Common Redis Cluster Issues](#-troubleshooting-common-redis-cluster-issues)
   - [2.3: Verify Redis Cluster Installation](#step-23-verify-redis-cluster-installation)
   - [2.4: Test Redis Cluster Connectivity](#step-24-test-redis-cluster-connectivity)
   - [2.5: Create Redis Network Policies](#step-25-create-redis-network-policies)
   - [2.6: Get Redis Connection Details](#step-26-get-redis-connection-details)
6. [Step 3: OpenShift Logging and Log Forwarding Setup](#3-openshift-logging-and-log-forwarding-setup)
   - [3.1: Install OpenShift Logging Operator](#step-31-install-openshift-logging-operator)
   - [3.2: Validate OpenShift Logging Operator Installation](#step-32-validate-openshift-logging-operator-installation)
   - [3.3: Create Service Account and RBAC](#step-33-create-service-account-and-rbac)
   - [3.4: Deploy ClusterLogForwarder](#step-34-deploy-clusterlogforwarder-corrected-configuration)
   - [3.5: Deploy Test Application](#step-35-deploy-test-application-for-end-to-end-validation)
   - [3.6: Execute End-to-End Validation](#step-36-execute-end-to-end-validation)
7. [Complete Infrastructure Verification](#-complete-setup-validation-checklist)
8. [Next Steps](#4-next-steps)

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
2. **Redis Cluster (HA)** - For state storage and caching with high availability
3. **ClusterLogForwarder** - For forwarding OpenShift logs to Kafka
4. **Network policies and security configurations**

### ⏰ Expected Deployment Times

- **AMQ Streams Operator**: 1-2 minutes
- **Kafka Cluster**: 3-5 minutes  
- **Redis Cluster (HA)**: 2-3 minutes
- **OpenShift Logging Operator**: 1-2 minutes
- **ClusterLogForwarder + Vector**: 1-2 minutes for validation

**Total Setup Time**: Approximately 10-15 minutes

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
kafkas.kafka.strimzi.io
kafkatopics.kafka.strimzi.io
kafkausers.kafka.strimzi.io
strimzipodsets.core.strimzi.io
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
echo '{"timestamp":"2025-07-12T12:00:00Z","level":"INFO","message":"Test message from Kafka producer","service":"kafka-test","namespace":"test"}' | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs
```

#### Step 1.6.2: Verify Message Reception

```bash
# Check if the message was received by running a consumer
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --max-messages 1
```

**Expected Output:**
```
{"timestamp":"2025-07-12T12:00:00Z","level":"INFO","message":"Test message from Kafka producer","service":"kafka-test","namespace":"test"}
Processed a total of 1 messages
```

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

## 2. Redis Cluster (HA) Setup

This section sets up a highly available Redis cluster for the Alert Engine's state storage and caching needs.

### Step 2.1: Create Redis Cluster Namespace

```bash
# Create namespace for Redis cluster
oc create namespace redis-cluster
```

### Step 2.2: Deploy Redis Cluster

Deploy a highly available Redis cluster with 6 nodes (3 masters + 3 replicas):

```yaml
cat <<EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-cluster-config
  namespace: redis-cluster
data:
  update-node.sh: |
    #!/bin/sh
    REDIS_NODES="/data/nodes.conf"
    if [ -f "\${REDIS_NODES}" ]; then
      sed -i -e "/myself/ s/[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}/\${POD_IP}/" \${REDIS_NODES}
    fi
    exec "\$@"
  redis.conf: |
    cluster-enabled yes
    cluster-require-full-coverage no
    cluster-node-timeout 15000
    cluster-config-file /data/nodes.conf
    cluster-migration-barrier 1
    appendonly yes
    protected-mode no
    bind 0.0.0.0
    port 6379
    tcp-keepalive 60
    maxmemory 256mb
    maxmemory-policy allkeys-lru
---
apiVersion: v1
kind: Service
metadata:
  name: redis-cluster
  namespace: redis-cluster
  labels:
    app: redis-cluster
spec:
  ports:
  - port: 6379
    targetPort: 6379
    name: client
  - port: 16379
    targetPort: 16379
    name: gossip
  clusterIP: None
  selector:
    app: redis-cluster
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis-cluster
  namespace: redis-cluster
spec:
  serviceName: redis-cluster
  replicas: 6
  selector:
    matchLabels:
      app: redis-cluster
  template:
    metadata:
      labels:
        app: redis-cluster
    spec:
      containers:
      - name: redis
        image: redis:7.0-alpine
        ports:
        - containerPort: 6379
          name: client
        - containerPort: 16379
          name: gossip
        command: ["/conf/update-node.sh", "redis-server", "/conf/redis.conf"]
        env:
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        volumeMounts:
        - name: conf
          mountPath: /conf
          readOnly: false
        - name: data
          mountPath: /data
          readOnly: false
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "200m"
      volumes:
      - name: conf
        configMap:
          name: redis-cluster-config
          defaultMode: 0755
          items:
          - key: redis.conf
            path: redis.conf
          - key: update-node.sh
            path: update-node.sh
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: gp3-csi
      resources:
        requests:
          storage: 1Gi
EOF
```

**Wait for Redis cluster pods to be ready:**

```bash
# Wait for all Redis cluster pods to be running
oc wait --for=condition=ready pod -l app=redis-cluster -n redis-cluster --timeout=300s

# Check pod status
oc get pods -n redis-cluster -l app=redis-cluster
```

#### 🔧 Troubleshooting Common Redis Cluster Issues

**If you see CrashLoopBackOff or pods not starting:**

1. **Check pod logs for errors:**
   ```bash
   oc logs redis-cluster-0 -n redis-cluster
   ```

2. **Common Issue: "sed: -i requires an argument" or "Permission denied"**
   
   This indicates the `update-node.sh` script is incorrect. The script should include:
   - Proper file existence check
   - Correct variable escaping
   - Proper exec command
   
   **Resolution:** The configuration above has been updated with the correct script. If you used an older version, delete and recreate the StatefulSet:
   ```bash
   oc delete statefulset redis-cluster -n redis-cluster
   # Then reapply the corrected YAML above
   ```

3. **Check Storage Class availability:**
   ```bash
   oc get storageclass
   # If gp3-csi is not available, update the storageClassName in the YAML
   ```

4. **Verify ConfigMap was created correctly:**
   ```bash
   oc get configmap redis-cluster-config -n redis-cluster -o yaml
   ```

**Expected healthy pod status:**
```
NAME              READY   STATUS    RESTARTS   AGE
redis-cluster-0   1/1     Running   0          2m
redis-cluster-1   1/1     Running   0          2m
redis-cluster-2   1/1     Running   0          2m
redis-cluster-3   1/1     Running   0          2m
redis-cluster-4   1/1     Running   0          2m
redis-cluster-5   1/1     Running   0          2m
```

### Step 2.3: Verify Redis Cluster Installation

**Initialize the Redis cluster:**

```bash
# Get the Redis cluster nodes IP addresses
REDIS_NODES=$(oc get pods -l app=redis-cluster -n redis-cluster -o jsonpath='{range.items[*]}{.status.podIP}:6379 ')

# Display the nodes that will be used for cluster creation
echo "Redis cluster nodes: $REDIS_NODES"

# Initialize the cluster with explicit node list (more reliable than using variables)
oc exec -it redis-cluster-0 -n redis-cluster -- redis-cli --cluster create \
$(oc get pods -l app=redis-cluster -n redis-cluster -o jsonpath='{range.items[*]}{.status.podIP}:6379 ') \
--cluster-replicas 1 --cluster-yes

# Verify cluster status
echo "=== Cluster Status ==="
oc exec -it redis-cluster-0 -n redis-cluster -- redis-cli cluster info

echo "=== Cluster Nodes ==="
oc exec -it redis-cluster-0 -n redis-cluster -- redis-cli cluster nodes
```

**Expected Output:**
```
=== Cluster Status ===
cluster_state:ok
cluster_slots_assigned:16384
cluster_slots_ok:16384
cluster_slots_pfail:0
cluster_slots_fail:0
cluster_known_nodes:6
cluster_size:3

=== Cluster Nodes ===
<node-id> <ip>:6379@16379 master - 0 <timestamp> 1 connected 0-5460
<node-id> <ip>:6379@16379 master - 0 <timestamp> 2 connected 5461-10922
<node-id> <ip>:6379@16379 master - 0 <timestamp> 3 connected 10923-16383
<node-id> <ip>:6379@16379 slave <master-id> 0 <timestamp> 1 connected
<node-id> <ip>:6379@16379 slave <master-id> 0 <timestamp> 2 connected  
<node-id> <ip>:6379@16379 slave <master-id> 0 <timestamp> 3 connected
```

**✅ Success Criteria:**
- `cluster_state:ok` indicates the cluster is healthy
- `cluster_slots_assigned:16384` means all hash slots are assigned
- `cluster_known_nodes:6` confirms all 6 nodes are recognized
- `cluster_size:3` shows 3 master nodes (with 3 replicas)

**❌ Troubleshooting cluster initialization:**

If cluster creation fails:
```bash
# Check if all pods are running
oc get pods -n redis-cluster -l app=redis-cluster

# Check individual pod logs
oc logs redis-cluster-0 -n redis-cluster

# Try manual initialization if automatic fails
oc exec -it redis-cluster-0 -n redis-cluster -- redis-cli --cluster create \
<IP1>:6379 <IP2>:6379 <IP3>:6379 <IP4>:6379 <IP5>:6379 <IP6>:6379 \
--cluster-replicas 1 --cluster-yes
```

### Step 2.4: Test Redis Cluster Connectivity

Test that the Redis cluster is working correctly:

```bash
# Test Redis cluster connectivity and data replication
oc exec -it redis-cluster-0 -n redis-cluster -- redis-cli -c set test-key "Hello Redis Cluster"
oc exec -it redis-cluster-0 -n redis-cluster -- redis-cli -c get test-key

# Test from different nodes
oc exec -it redis-cluster-1 -n redis-cluster -- redis-cli -c get test-key
oc exec -it redis-cluster-2 -n redis-cluster -- redis-cli -c get test-key

# Clean up test data
oc exec -it redis-cluster-0 -n redis-cluster -- redis-cli -c del test-key
```

**Expected Output:**
```
Hello Redis Cluster
```

### Step 2.5: Create Redis Network Policies

Set up network policies to allow access from alert-engine namespace:

```bash
cat <<EOF | oc apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: redis-cluster-network-policy
  namespace: redis-cluster
spec:
  podSelector:
    matchLabels:
      app: redis-cluster
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
      port: 6379
    - protocol: TCP
      port: 16379
  egress:
  - {}
EOF
```

### Step 2.6: Get Redis Connection Details

Create ConfigMap and Service for Alert Engine:

```bash
# Create alert-engine namespace if it doesn't exist
oc create namespace alert-engine --dry-run=client -o yaml | oc apply -f -

# Create service for Redis cluster access
cat <<EOF | oc apply -f -
apiVersion: v1
kind: Service
metadata:
  name: redis-cluster-access
  namespace: redis-cluster
spec:
  type: ClusterIP
  ports:
  - port: 6379
    targetPort: 6379
    protocol: TCP
    name: redis
  selector:
    app: redis-cluster
EOF

# Create ConfigMap with Redis connection details
cat <<EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: alert-engine
data:
  redis-mode: "cluster"
  redis-hosts: "redis-cluster-0.redis-cluster.redis-cluster.svc.cluster.local:6379,redis-cluster-1.redis-cluster.redis-cluster.svc.cluster.local:6379,redis-cluster-2.redis-cluster.redis-cluster.svc.cluster.local:6379,redis-cluster-3.redis-cluster.redis-cluster.svc.cluster.local:6379,redis-cluster-4.redis-cluster.redis-cluster.svc.cluster.local:6379,redis-cluster-5.redis-cluster.redis-cluster.svc.cluster.local:6379"
  redis-cluster-service: "redis-cluster-access.redis-cluster.svc.cluster.local:6379"
  redis-namespace: "redis-cluster"
EOF

# Get Redis connection details
echo "=== Redis Cluster Connection Details ==="
echo "Mode: cluster"
echo "Cluster Nodes:"
oc get pods -l app=redis-cluster -n redis-cluster -o jsonpath='{range .items[*]}{.metadata.name}.redis-cluster.redis-cluster.svc.cluster.local:6379{"\n"}{end}'
echo "Service: redis-cluster-access.redis-cluster.svc.cluster.local:6379"
echo "ConfigMap: redis-config (in alert-engine namespace)"

# Quick validation summary
echo ""
echo "🔍 Quick Redis Cluster Validation:"
echo "1. All pods running: $(oc get pods -l app=redis-cluster -n redis-cluster --no-headers | grep Running | wc -l)/6"
echo "2. Cluster state: $(oc exec -it redis-cluster-0 -n redis-cluster -- redis-cli cluster info | grep cluster_state)"
echo "3. Network policy: $(oc get networkpolicy redis-cluster-network-policy -n redis-cluster --no-headers | wc -l) created"
echo "4. ConfigMap: $(oc get configmap redis-config -n alert-engine --no-headers | wc -l) created"
```

## 3. OpenShift Logging and Log Forwarding Setup

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
# Install the operator using stable-6.2 channel
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

# Check operator pod status
oc get pods -n openshift-logging

# Verify CRDs are installed
oc get crd | grep -E "(logging|clusterlog)"
```

### Step 3.2: Validate OpenShift Logging Operator Installation

```bash
# Quick validation command
oc get csv -n openshift-logging --no-headers | grep cluster-logging | grep -q "Succeeded" && echo "✅ OpenShift Logging Operator Ready" || echo "❌ Installation Failed"
```

### Step 3.3: Create Service Account and RBAC

Create the necessary service account and permissions for log collection:

```bash
# Create service account and RBAC permissions
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

### Step 3.4: Deploy ClusterLogForwarder (Corrected Configuration)

Deploy the ClusterLogForwarder with the **critical RedHat fixes** that resolve the Vector configuration issues:

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

### Step 3.5: Deploy Test Application for End-to-End Validation

Create a continuous log generator to test the complete ClusterLogForwarder flow:

```bash
# Create service account for alert-engine
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
            level_num=\$((RANDOM % 5))
            service_num=\$((RANDOM % 5))
            message_num=\$((RANDOM % 10))
            
            case \$level_num in
              0) level="INFO" ;;
              1) level="WARN" ;;
              2) level="ERROR" ;;
              3) level="DEBUG" ;;
              4) level="TRACE" ;;
            esac
            
            case \$service_num in
              0) service="user-service" ;;
              1) service="payment-service" ;;
              2) service="order-service" ;;
              3) service="inventory-service" ;;
              4) service="notification-service" ;;
            esac
            
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
            
            timestamp=\$(date -Iseconds)
            user_id=\$((RANDOM % 1000 + 1))
            
            echo "[\$timestamp] \$level: \$message | service=\$service | user_id=\$user_id | sequence=\$counter"
            
            counter=\$((counter + 1))
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

### Step 3.6: Execute End-to-End Validation

Wait for logs to flow through the system and verify:

```bash
# Wait for Vector to process logs (60 seconds recommended)
echo "⏳ Waiting 60 seconds for Vector to process and forward logs to Kafka..."
sleep 60

# Check for alert-engine logs in Kafka
echo "🔍 Checking Kafka consumer for alert-engine logs..."
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- timeout 15 bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --max-messages 5
```

## 🎯 Complete Setup Validation Checklist

### Final Pre-Deployment Validation

Run this complete validation before proceeding with Alert Engine deployment:

```bash
#!/bin/bash
echo "=== 🎯 Complete OpenShift Infrastructure Validation ==="

# Kafka verification
echo "1. Kafka Cluster Status:"
KAFKA_READY=$(oc get kafka alert-kafka-cluster -n amq-streams-kafka -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
[[ "$KAFKA_READY" == "True" ]] && echo "   ✅ Kafka cluster ready" || echo "   ❌ Kafka cluster not ready"

# Redis Cluster verification  
echo "2. Redis Cluster Status:"
REDIS_PODS=$(oc get pods -l app=redis-cluster -n redis-cluster --no-headers 2>/dev/null | grep Running | wc -l)
[[ "$REDIS_PODS" -eq 6 ]] && echo "   ✅ Redis cluster ready (6/6 pods)" || echo "   ❌ Redis cluster not ready ($REDIS_PODS/6 pods)"

# ClusterLogForwarder verification
echo "3. ClusterLogForwarder Status:"
# Note: Use full condition type "observability.openshift.io/Valid" instead of just "Valid"
CLF_STATUS=$(oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o jsonpath='{.status.conditions[?(@.type=="observability.openshift.io/Valid")].status}' 2>/dev/null)
[[ "$CLF_STATUS" == "True" ]] && echo "   ✅ ClusterLogForwarder valid" || echo "   ❌ ClusterLogForwarder invalid"

echo ""
echo "📋 CONNECTION DETAILS FOR ALERT ENGINE:"
echo "Kafka: alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"
echo "Redis: redis-cluster-access.redis-cluster.svc.cluster.local:6379 (cluster mode)"
echo "Topic: application-logs"
echo "Namespace: alert-engine"
```

## 4. Next Steps

After completing this infrastructure setup:

1. **Local Testing**: Update your `configs/config.yaml` with the connection details obtained above and test the Alert Engine locally
2. **Deploy to OpenShift**: Use the deployment manifests in `deployments/openshift/` to deploy the Alert Engine application
3. **Configure Monitoring**: Set up Prometheus monitoring for the Alert Engine metrics
4. **Test End-to-End**: Generate test logs and verify alerts are triggered and notifications are sent

---

**🎉 OpenShift Infrastructure Setup Complete!**

You now have:
- ✅ **Kafka Cluster**: Ready for log message streaming
- ✅ **Redis Cluster (HA)**: Ready for state storage and caching
- ✅ **ClusterLogForwarder**: Ready for log collection and forwarding
- ✅ **Network Policies**: Configured for secure access
- ✅ **Service Accounts**: Ready for Alert Engine deployment

**Sample `config.yaml` for Alert Engine:**
```yaml
kafka:
  brokers: ["alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"]
  topic: "application-logs"
  consumer_group: "alert-engine-group"

redis:
  mode: "cluster"
  addresses: [
    "redis-cluster-0.redis-cluster.redis-cluster.svc.cluster.local:6379",
    "redis-cluster-1.redis-cluster.redis-cluster.svc.cluster.local:6379",
    "redis-cluster-2.redis-cluster.redis-cluster.svc.cluster.local:6379"
  ]

kubernetes:
  namespace: "alert-engine"
  service_account: "alert-engine-sa"
```