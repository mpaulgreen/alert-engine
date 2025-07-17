# MockLogGenerator OpenShift Deployment

This directory contains all the necessary manifests to deploy the MockLogGenerator as a continuous log generator on OpenShift for Alert Engine testing.

## 📋 Overview

The MockLogGenerator generates realistic application logs that simulate 19 different alert patterns, providing comprehensive testing for the Alert Engine. It outputs structured JSON logs to stdout which are automatically collected by OpenShift's ClusterLogForwarder (Vector) and forwarded to Kafka for processing by the Alert Engine.

## 🏗️ Correct Architecture

```
MockLogGenerator Pod (mock-logs namespace)
    ↓ (outputs JSON logs to stdout)
OpenShift ClusterLogForwarder (Vector)
    ↓ (collects & forwards logs)
Kafka Topic (application-logs)
    ↓ (consumed by)
Alert Engine (alert-engine namespace)
    ↓ (processes rules & sends alerts to)
Slack
```

**🎯 Key Architecture Points:**
- ✅ **MockLogGenerator**: Simple log generator outputting to stdout (no Kafka dependencies)
- ✅ **Vector/ClusterLogForwarder**: Handles log collection and Kafka forwarding
- ✅ **Clean Separation**: Each component has a single responsibility
- ✅ **Standard Pattern**: Follows OpenShift logging best practices

## 📁 Files Structure

```
deployments/mock/
├── Dockerfile                    # Container image definition
├── requirements.txt              # Python dependencies
├── mock_log_generator.py        # Main application code
├── configmap.yaml               # Configuration data
├── serviceaccount.yaml          # ServiceAccount and RBAC
├── deployment.yaml              # Main deployment manifest
├── networkpolicy.yaml           # Network security policy
├── kustomization.yaml           # Kustomize configuration
└── README.md                    # This file
```

## 🚀 Prerequisites

Before deploying the MockLogGenerator, ensure you have:

1. ✅ **OpenShift Infrastructure**: Follow the [Infrastructure Setup Guide](../../alert_engine_infra_setup.md)
   - AMQ Streams Kafka cluster deployed
   - ClusterLogForwarder configured
   - OpenShift Logging Operator installed

2. ✅ **MockLogGenerator Namespace**: 
   ```bash
   oc create namespace mock-logs
   ```

3. ✅ **Container Registry Access**: 
   - Build and push the container image to your registry
   - Update the image reference in `deployment.yaml`

## 🔧 Quick Deployment

### Step 1: Build and Push Container Image

```bash
# Navigate to the mock deployment directory
cd alert-engine/deployments/mock

# Build the container image (specify platform for x86_64 OpenShift clusters)
podman build --platform linux/amd64 -t quay.io/your-registry/mock-log-generator:latest .

# Push to your registry
podman push quay.io/your-registry/mock-log-generator:latest

# Note: The --platform flag ensures compatibility with x86_64 OpenShift clusters
# and prevents "Exec format error" when building on ARM64 systems (Apple Silicon Macs)
```

### Step 2: Update Image Reference

Edit `deployment.yaml` and update the image reference:
```yaml
containers:
- name: mock-log-generator
  image: quay.io/your-registry/mock-log-generator:latest  # Update this line
```

### Step 3: Deploy to OpenShift

```bash
# Apply all manifests
oc apply -f configmap.yaml
oc apply -f serviceaccount.yaml
oc apply -f networkpolicy.yaml
oc apply -f deployment.yaml

# Or use kustomize (if kustomization.yaml is available)
oc apply -k .
```

### Step 4: Verify Deployment

```bash
# Check pod status
oc get pods -n mock-logs -l app=mock-log-generator

# Check logs
oc logs -n mock-logs -l app=mock-log-generator -f

# Verify log generation
oc exec -n mock-logs deployment/mock-log-generator -- python3 -c "import json, time; print('Log generator modules available')"
```

## ⚙️ Configuration Options

### Environment Variables (ConfigMap)

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_MODE` | `continuous` | Generation mode (`continuous` or `test`) |
| `LOG_LEVEL` | `INFO` | Application log level |
| `LOG_GENERATION_INTERVAL` | `1.0` | Seconds between normal logs |
| `BURST_INTERVAL` | `10` | Cycles between alert pattern bursts |
| `CLUSTER_NAME` | `openshift-cluster` | Optional cluster identifier |

**📝 Note**: No Kafka configuration needed - logs are output to stdout for Vector/ClusterLogForwarder collection

### 🧪 E2E Test Mode Alignment

**Test Mode** (`LOG_MODE=test`): Optimized for e2e test compatibility
- ✅ **Threshold Alignment**: Uses `threshold=1` + small buffer for reliable alert triggering
- ✅ **Service Matching**: Generates specific service/log level combinations expected by e2e tests:
  - `payment-service` → ERROR logs (high_error_rate pattern)
  - `user-service` → WARN logs (high_warn_rate pattern)  
  - `database-service` → FATAL logs (database_errors pattern)
  - `authentication-api` → ERROR logs (authentication_failures pattern)
- ✅ **Pattern Priority**: Tests 12 key patterns from `comprehensive_e2e_test_config.json`
- ✅ **Keyword Precision**: Ensures exact keyword matching for test pattern recognition
- ✅ **Full Coverage**: Supports all 19 patterns in both test and continuous modes

**Continuous Mode** (`LOG_MODE=continuous`): Production-style simulation
- Uses realistic production thresholds (5, 10, 15, 25+)
- Rotates through all 19 supported patterns randomly
- Balanced normal vs alert log generation

### Modifying Configuration

Edit the ConfigMap and restart the deployment:

```bash
# Edit configuration
oc edit configmap mock-log-generator-config -n mock-logs

# Restart deployment to pick up changes
oc rollout restart deployment/mock-log-generator -n mock-logs
```

## 🔍 Monitoring and Troubleshooting

### Check Pod Status

```bash
# Get pod status
oc get pods -n mock-logs -l app=mock-log-generator

# Describe pod for events
oc describe pod -n mock-logs -l app=mock-log-generator

# Check resource usage
oc top pod -n mock-logs -l app=mock-log-generator
```

### View Logs

```bash
# Follow application logs
oc logs -n mock-logs -l app=mock-log-generator -f

# Get recent logs
oc logs -n mock-logs -l app=mock-log-generator --tail=100

# Check previous container logs (if pod restarted)
oc logs -n mock-logs -l app=mock-log-generator --previous
```

### Verify Log Flow

```bash
# Check if logs are being generated by MockLogGenerator
oc logs -n mock-logs -l app=mock-log-generator --tail 20

# Check if logs are being collected by Vector
oc logs -n openshift-logging -l component=vector -f | grep mock-logs

# Check ClusterLogForwarder status
oc get clusterlogforwarder -n openshift-logging -o yaml
```

### 🔍 Verifying Logs Reach Kafka

Use these commands to comprehensively verify that logs are flowing from MockLogGenerator to Kafka:

#### Step 1: Check Infrastructure Status

```bash
# Verify Kafka cluster is ready
oc get kafka alert-kafka-cluster -n alert-engine -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'

# Check ClusterLogForwarder status
oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o jsonpath='{.status.conditions[?(@.type=="Valid")].status}'

# Verify Vector pods are running
oc get pods -n openshift-logging -l component=vector | grep kafka-forwarder
```

#### Step 2: Verify Log Generation

```bash
# Check MockLogGenerator pod is running and generating logs
oc get pods -n mock-logs -l app=mock-log-generator

# View recent logs being generated
oc logs -n mock-logs -l app=mock-log-generator --tail=10 -f
```

#### Step 3: Read Logs from Kafka Topic

```bash
# Connect to Kafka cluster and read from application-logs topic
oc exec -n alert-engine alert-kafka-cluster-kafka-0 -- /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic application-logs \
  --from-beginning \
  --max-messages 10

# For real-time monitoring of new logs
oc exec -n alert-engine alert-kafka-cluster-kafka-0 -- /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic application-logs
```

#### 📋 Expected Results

**Infrastructure Status:**
- Kafka cluster should return: `True`
- ClusterLogForwarder should return: `True`  
- Vector pods should show multiple `kafka-forwarder-*` pods in `Running` state

**Log Verification:**
- MockLogGenerator logs should show JSON format with various alert patterns
- Kafka topic should contain logs with double-nesting structure (Vector metadata containing MockLogGenerator JSON)

#### 📄 Example Log from Kafka

Here's an actual log from the `application-logs` Kafka topic showing the complete flow:

```json
{
  "@timestamp": "2024-01-15T10:30:45.123456Z",
  "openshift": {
    "cluster_name": "openshift-cluster",
    "labels": {
      "app": "mock-log-generator",
      "deployment": "mock-log-generator",
      "pod-template-hash": "abc123def4"
    }
  },
  "kubernetes": {
    "container_name": "mock-log-generator", 
    "namespace_name": "mock-logs",
    "pod_name": "mock-log-generator-abc123def4-xyz89",
    "host": "worker-1.example.com",
    "annotations": {
      "app.openshift.io/name": "mock-log-generator"
    },
    "container_id": "cri-o://1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z",
    "pod_ip": "10.128.2.45",
    "pod_owner": "ReplicaSet/mock-log-generator-abc123def4"
  },
  "message": "{\"timestamp\":\"2024-01-15T10:30:45.123Z\",\"level\":\"ERROR\",\"service\":\"payment-service\",\"host\":\"mock-log-generator-abc123def4-xyz89\",\"hostname\":\"mock-log-generator-abc123def4-xyz89\",\"log_source\":\"application\",\"log_type\":\"structured\",\"message\":\"Payment processing failed for user ID 12345: Credit card declined\",\"error_code\":\"PAYMENT_DECLINED\",\"user_id\":\"12345\",\"transaction_id\":\"txn_67890abcde\",\"amount\":\"99.99\",\"currency\":\"USD\",\"payment_method\":\"credit_card\",\"kubernetes\":{\"namespace\":\"mock-logs\",\"pod_name\":\"mock-log-generator-abc123def4-xyz89\",\"container_name\":\"mock-log-generator\",\"host\":\"worker-1.example.com\",\"annotations\":{\"app.openshift.io/name\":\"mock-log-generator\"},\"container_id\":\"cri-o://1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z\",\"pod_ip\":\"10.128.2.45\",\"pod_owner\":\"ReplicaSet/mock-log-generator-abc123def4\"},\"raw\":\"{\\\"timestamp\\\":\\\"2024-01-15T10:30:45.123Z\\\",\\\"level\\\":\\\"ERROR\\\",\\\"service\\\":\\\"payment-service\\\",\\\"message\\\":\\\"Payment processing failed for user ID 12345: Credit card declined\\\"}\"}",
  "stream": "stdout",
  "time": "2024-01-15T10:30:45.123456789Z",
  "tag": "kubernetes.var.log.containers.mock-log-generator-abc123def4-xyz89_mock-logs_mock-log-generator-1a2b3c4d5e6f.log"
}
```

**🔍 Log Structure Analysis:**
- **Outer layer**: Vector/ClusterLogForwarder metadata (OpenShift, Kubernetes info)
- **Inner `message` field**: Complete MockLogGenerator JSON log (as string)
- **Double-nesting**: This structure allows Alert Engine to extract the original log while preserving OpenShift metadata
- **Alert Pattern**: This example shows a `payment_failures` pattern with ERROR level and payment-specific metadata

#### ✅ Success Criteria

Your log flow is working correctly if you see:
1. **Infrastructure**: All Kafka and Vector components are `Ready`/`Running`
2. **Generation**: MockLogGenerator producing JSON logs with various alert patterns
3. **Kafka Delivery**: Logs appearing in `application-logs` topic with the double-nested structure above
4. **Pattern Recognition**: Alert Engine can process these logs and trigger appropriate alerts

### Debug Pod Health

```bash
# Check pod status and health
oc describe pod -n mock-logs -l app=mock-log-generator

# Check if application is running properly
oc exec -n mock-logs deployment/mock-log-generator -- ps aux | grep python
```

## 🎯 Log Patterns Generated

The MockLogGenerator generates logs for these 19 alert patterns:

1. **high_error_rate** - High volume of ERROR level logs
2. **payment_failures** - Payment processing failures
3. **database_errors** - Database connection and query errors
4. **authentication_failures** - User authentication failures
5. **service_timeouts** - Service timeout scenarios
6. **critical_namespace_alerts** - Critical production alerts
7. **inventory_warnings** - Inventory stock warnings
8. **notification_failures** - Notification delivery failures
9. **high_warn_rate** - High volume of WARNING logs
10. **audit_issues** - Security audit violations
11. **checkout_payment_failed** - Checkout process failures
12. **inventory_stock_unavailable** - Stock availability issues
13. **email_smtp_failed** - Email delivery failures
14. **redis_connection_refused** - Cache connection failures
15. **message_queue_full** - Message queue overflow
16. **timeout_any_service** - General timeout scenarios
17. **slow_query** - Database performance issues
18. **deadlock_detected** - Database deadlock scenarios
19. **cross_service_errors** - Service communication failures

## 🔧 Customization

### Adjusting Log Generation Rate

To increase or decrease log generation:

```bash
# Edit ConfigMap
oc patch configmap mock-log-generator-config -n mock-logs -p '{"data":{"LOG_GENERATION_INTERVAL":"0.5"}}' # Faster
oc patch configmap mock-log-generator-config -n mock-logs -p '{"data":{"LOG_GENERATION_INTERVAL":"2.0"}}' # Slower

# Restart deployment
oc rollout restart deployment/mock-log-generator -n mock-logs
```

### Changing Alert Pattern Frequency

```bash
# More frequent alert patterns
oc patch configmap mock-log-generator-config -n mock-logs -p '{"data":{"BURST_INTERVAL":"5"}}'

# Less frequent alert patterns  
oc patch configmap mock-log-generator-config -n mock-logs -p '{"data":{"BURST_INTERVAL":"20"}}'
```

### Running in Test Mode

To run a one-time test of all patterns:

```bash
# Patch deployment to use test mode
oc patch deployment mock-log-generator -n mock-logs -p '{"spec":{"template":{"spec":{"containers":[{"name":"mock-log-generator","args":["/app/mock_log_generator.py","--mode","test"]}]}}}}'

# This will run once and complete, testing all 19 patterns
```

## 🧹 Cleanup

To remove the MockLogGenerator deployment:

```bash
# Delete all resources
oc delete -f deployment.yaml
oc delete -f networkpolicy.yaml
oc delete -f serviceaccount.yaml
oc delete -f configmap.yaml

# Or delete by label
oc delete all,configmap,serviceaccount,networkpolicy -n mock-logs -l app=mock-log-generator
```

## 🔐 Security

The deployment includes several security measures:

- **Non-root user**: Runs as user ID 1001
- **Read-only root filesystem**: Where possible
- **Dropped capabilities**: All Linux capabilities dropped
- **Network policies**: Restricts ingress/egress traffic
- **RBAC**: Minimal required permissions
- **Security contexts**: OpenShift security constraints compliant

## 📊 Integration with Alert Engine

Once deployed, the MockLogGenerator will:

1. Generate logs that appear in OpenShift application logs (from `mock-logs` namespace)
2. Logs are collected by ClusterLogForwarder (Vector)
3. Forwarded to Kafka topic `application-logs`
4. Consumed by Alert Engine (deployed in `alert-engine` namespace) for processing
5. Trigger alerts based on the 19 defined patterns
6. Send notifications to configured Slack channels

## 🆘 Support

For issues or questions:

1. Check the [main Alert Engine documentation](../../README.md)
2. Review the [infrastructure setup guide](../../alert_engine_infra_setup.md)
3. Examine pod logs and events for troubleshooting
4. Verify Kafka and ClusterLogForwarder status

---

**🎉 MockLogGenerator Ready!**

Your continuous log generator is now deployed and will provide realistic test data for comprehensive Alert Engine testing in your OpenShift environment. 