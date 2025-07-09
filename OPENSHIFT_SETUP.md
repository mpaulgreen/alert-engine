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

After installing the operator, it's crucial to validate that it's properly installed before proceeding. Follow these comprehensive validation steps:

#### Check the Subscription Status

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

#### Check the ClusterServiceVersion (CSV)

The CSV represents the actual operator installation:

```bash
# Check CSV status
oc get csv -n amq-streams-kafka

# Get detailed CSV information
oc get csv -n amq-streams-kafka -o wide
```

**Expected Output:**
```
NAME                              DISPLAY                         VERSION   REPLACES   PHASE
amqstreams.v2.7.0-0               AMQ Streams                     2.7.0-0              Succeeded
```

**Key Status to Look For:**
- `PHASE` should be `Succeeded`
- `DISPLAY` should show "AMQ Streams"

#### Check the InstallPlan

```bash
# Check install plan
oc get installplan -n amq-streams-kafka

# Get detailed install plan
oc describe installplan -n amq-streams-kafka
```

**Expected Output:**
```
NAME            CSV                            APPROVAL    APPROVED
install-xxxxx   amqstreams.v2.7.0-0           Manual      true
```

#### Verify Operator Pod is Running

```bash
# Check operator pods
oc get pods -n amq-streams-kafka

# Check operator deployment
oc get deployment -n amq-streams-kafka
```

**Expected Output:**
```
NAME                                          READY   STATUS    RESTARTS   AGE
strimzi-cluster-operator-xxxxxxxxx-xxxxx     1/1     Running   0          2m
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

#### Complete Validation Script

Run this comprehensive validation script:

```bash
#!/bin/bash

echo "=== AMQ Streams Operator Validation ==="
echo ""

echo "1. Checking Subscription..."
oc get subscription amq-streams -n amq-streams-kafka
echo ""

echo "2. Checking ClusterServiceVersion..."
oc get csv -n amq-streams-kafka
echo ""

echo "3. Checking InstallPlan..."
oc get installplan -n amq-streams-kafka
echo ""

echo "4. Checking Operator Pods..."
oc get pods -n amq-streams-kafka
echo ""

echo "5. Checking Operator Deployment..."
oc get deployment -n amq-streams-kafka
echo ""

echo "6. Checking CRDs..."
echo "Kafka CRDs:"
oc get crd | grep kafka | head -5
echo ""

echo "7. Checking Operator Status..."
if oc get csv -n amq-streams-kafka --no-headers | grep -q "Succeeded"; then
    echo "✅ AMQ Streams Operator is successfully installed!"
else
    echo "❌ AMQ Streams Operator installation failed or in progress"
fi

echo ""
echo "8. Operator Version Info..."
oc get csv -n amq-streams-kafka -o jsonpath='{.items[0].spec.version}'
echo ""
```

#### Troubleshooting Common Issues

If the operator is not installed successfully, check these:

**Check for OperatorHub Availability:**
```bash
# Verify OperatorHub is available
oc get catalogsource -n openshift-marketplace | grep redhat-operators
```

**Check Namespace Labels:**
```bash
# Ensure namespace allows operator installation
oc get namespace amq-streams-kafka --show-labels
```

**Check for Installation Errors:**
```bash
# Check events for installation issues
oc get events -n amq-streams-kafka --sort-by='.lastTimestamp'

# Check subscription conditions
oc get subscriptions.operators.coreos.com amq-streams -n amq-streams-kafka -o jsonpath='{.status.conditions}' | jq .
```

**Manual Approval (if needed):**
```bash
# If install plan needs manual approval
oc patch installplan <install-plan-name> -n amq-streams-kafka --type merge --patch '{"spec":{"approved":true}}'
```

#### Quick One-Liner Validation

For a quick check, you can use this one-liner:

```bash
oc get csv -n amq-streams-kafka --no-headers | grep -q "Succeeded" && echo "✅ AMQ Streams Operator Ready" || echo "❌ Installation Failed/In Progress"
```

**⚠️ Important**: Only proceed to the next step after confirming the operator is successfully installed with `Phase: Succeeded`.

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
    version: 3.7.0
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
      inter.broker.protocol.version: "3.7"
      log.message.format.version: "3.7"
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
  name: openshift-logs
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
---
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

## 2. Redis Setup using Redis Enterprise Operator

### Step 2.1: Install Redis Enterprise Operator

**Using OpenShift Web Console:**
1. Navigate to **Operators > OperatorHub**
2. Search for "**Redis Enterprise**"
3. Select **Redis Enterprise Operator provided by Redis** (certified)
4. Click **Install**
5. Configure:
   - **Installation Mode**: Install to specific namespace (create `redis-enterprise`)
   - **Update Channel**: Latest stable version
   - **Update Approval**: Manual

**Using CLI:**
```bash
# Create namespace
oc create namespace redis-enterprise

# Install Redis Enterprise Operator subscription
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: redis-enterprise-operator
  namespace: redis-enterprise
spec:
  channel: stable
  name: redis-enterprise-operator-cert
  source: certified-operators
  sourceNamespace: openshift-marketplace
EOF
```

### Step 2.2: Validate Redis Enterprise Operator Installation

After installing the Redis Enterprise operator, validate the installation:

#### Check the Subscription Status

```bash
# Check the Subscription status (use full API resource to avoid conflicts)
oc get subscriptions.operators.coreos.com redis-enterprise-operator -n redis-enterprise

# Get detailed subscription info
oc get subscriptions.operators.coreos.com redis-enterprise-operator -n redis-enterprise -o yaml
```

#### Check the ClusterServiceVersion (CSV)

```bash
# Check CSV status
oc get csv -n redis-enterprise

# Get detailed CSV information
oc get csv -n redis-enterprise -o wide
```

**Expected Output:**
```
NAME                                    DISPLAY                       VERSION   REPLACES   PHASE
redis-enterprise-operator.v6.x.x       Redis Enterprise Operator     6.x.x                Succeeded
```

#### Verify Operator Pod is Running

```bash
# Check operator pods
oc get pods -n redis-enterprise

# Check operator deployment
oc get deployment -n redis-enterprise
```

**Expected Output:**
```
NAME                                       READY   STATUS    RESTARTS   AGE
redis-enterprise-operator-xxxxxxxxx-xxxxx   1/1     Running   0          2m
```

#### Check Operator Logs

```bash
# Check operator logs for any issues
oc logs deployment/redis-enterprise-operator -n redis-enterprise

# Follow logs in real-time
oc logs -f deployment/redis-enterprise-operator -n redis-enterprise
```

#### Verify CRDs are Installed

```bash
# Check for Redis Enterprise CRDs
oc get crd | grep redis
```

**Expected CRDs:**
```
redisenterpriseclusters.app.redislabs.com
redisenterprisedatabases.app.redislabs.com
```

#### Quick Validation

```bash
# One-liner validation
oc get csv -n redis-enterprise --no-headers | grep -q "Succeeded" && echo "✅ Redis Enterprise Operator Ready" || echo "❌ Installation Failed/In Progress"
```

**⚠️ Important**: Only proceed to the next step after confirming the operator is successfully installed with `Phase: Succeeded`.

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
  redisEnterpriseImageSpec:
    imagePullPolicy: IfNotPresent
EOF
```

### Step 2.5: Create Redis Database

```yaml
cat <<EOF | oc apply -f -
apiVersion: app.redislabs.com/v1alpha1
kind: RedisEnterpriseDatabase
metadata:
  name: alert-engine-cache
  namespace: redis-enterprise
spec:
  memorySize: 2GB
  redisEnterpriseCluster:
    name: rec-alert-engine
  type: redis
  persistence: aof
  aofPolicy: appendfsync-every-sec
  redisModule:
  - name: ReJSON
    version: "2.6.6"
  - name: RedisTimeSeries  
    version: "1.10.11"
EOF
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
```

## 3. ClusterLogForwarder Setup

### Step 3.1: Install OpenShift Logging Operator

```bash
# Create namespace for logging
oc create namespace openshift-logging

# Install logging operator
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: cluster-logging
  namespace: openshift-logging
spec:
  channel: stable
  name: cluster-logging
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF
```

### Step 3.2: Validate OpenShift Logging Operator Installation

After installing the logging operator, validate the installation:

#### Check the Subscription Status

```bash
# Check the Subscription status (use full API resource to avoid conflicts)
oc get subscriptions.operators.coreos.com cluster-logging -n openshift-logging

# Get detailed subscription info
oc get subscriptions.operators.coreos.com cluster-logging -n openshift-logging -o yaml
```

#### Check the ClusterServiceVersion (CSV)

```bash
# Check CSV status
oc get csv -n openshift-logging

# Get detailed CSV information
oc get csv -n openshift-logging -o wide
```

**Expected Output:**
```
NAME                                    DISPLAY                       VERSION   REPLACES   PHASE
cluster-logging.v5.x.x                 Red Hat OpenShift Logging     5.x.x                Succeeded
```

#### Verify Operator Pod is Running

```bash
# Check operator pods
oc get pods -n openshift-logging

# Check operator deployment
oc get deployment -n openshift-logging
```

**Expected Output:**
```
NAME                                       READY   STATUS    RESTARTS   AGE
cluster-logging-operator-xxxxxxxxx-xxxxx    1/1     Running   0          2m
```

#### Check Operator Logs

```bash
# Check operator logs for any issues
oc logs deployment/cluster-logging-operator -n openshift-logging
```

#### Verify CRDs are Installed

```bash
# Check for logging CRDs
oc get crd | grep logging
```

**Expected CRDs:**
```
clusterlogforwarders.logging.coreos.com
clusterloggings.logging.coreos.com
```

#### Quick Validation

```bash
# One-liner validation
oc get csv -n openshift-logging --no-headers | grep -q "Succeeded" && echo "✅ OpenShift Logging Operator Ready" || echo "❌ Installation Failed/In Progress"
```

**⚠️ Important**: Only proceed to the next step after confirming the operator is successfully installed with `Phase: Succeeded`.

### Step 3.3: Create ClusterLogging Instance

```yaml
cat <<EOF | oc apply -f -
apiVersion: logging.coreos.com/v1
kind: ClusterLogging
metadata:
  name: instance
  namespace: openshift-logging
spec:
  managementState: Managed
  logStore:
    type: lokistack
    lokistack:
      name: logging-loki
  collection:
    type: vector
  visualization:
    type: ocp-console
EOF
```

### Step 3.4: Create ClusterLogForwarder

```yaml
cat <<EOF | oc apply -f -
apiVersion: logging.coreos.com/v1
kind: ClusterLogForwarder
metadata:
  name: kafka-log-forwarder
  namespace: openshift-logging
spec:
  outputs:
  - name: kafka-openshift-logs
    type: kafka
    url: tcp://alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092
    kafka:
      topic: openshift-logs
      brokers:
      - alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092
  - name: kafka-application-logs
    type: kafka
    url: tcp://alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092
    kafka:
      topic: application-logs
      brokers:
      - alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092
  pipelines:
  - name: forward-openshift-logs
    inputRefs:
    - infrastructure
    - audit
    outputRefs:
    - kafka-openshift-logs
    filterRefs: []
  - name: forward-application-logs
    inputRefs:
    - application
    outputRefs:
    - kafka-application-logs
    filterRefs: []
  - name: forward-to-default
    inputRefs:
    - infrastructure
    - audit
    - application
    outputRefs:
    - default
EOF
```

### Step 3.5: Verify Log Forwarding

```bash
# Check ClusterLogForwarder status
oc get clusterlogforwarder kafka-log-forwarder -n openshift-logging -o yaml

# Check vector pods (log collectors)
oc get pods -n openshift-logging -l component=collector

# Verify logs are being forwarded to Kafka
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic openshift-logs --from-beginning --max-messages 5
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
oc get clusterlogforwarder kafka-log-forwarder -n openshift-logging
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

1. **Subscription API Conflicts**: If you get `subscriptions.messaging.knative.dev` error, use the full API resource:
   ```bash
   # Wrong (may cause conflicts)
   oc get subscription amq-streams -n amq-streams-kafka
   
   # Correct (explicit API group)
   oc get subscriptions.operators.coreos.com amq-streams -n amq-streams-kafka
   ```

2. **Debugging API Resource Conflicts**:
   ```bash
   # Check all subscription resources available
   oc api-resources | grep subscription
   
   # Check what subscriptions exist in the namespace
   oc get subscriptions.operators.coreos.com -n amq-streams-kafka
   
   # Check if there are knative subscriptions (causing the conflict)
   oc get subscriptions.messaging.knative.dev -A 2>/dev/null || echo 'No knative subscriptions found'
   ```

3. **Kafka not ready**: Check storage class and resource limits
4. **Redis connection issues**: Verify SCC and network policies
5. **Log forwarding not working**: Check vector collector pods in openshift-logging namespace
6. **Permission denied**: Ensure proper RBAC and SCC configurations

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