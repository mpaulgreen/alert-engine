# Alert Engine OpenShift Deployment

This directory contains all the necessary Kubernetes/OpenShift manifests to deploy the Alert Engine in production.

## 📋 Prerequisites

Before deploying the Alert Engine, ensure the following infrastructure components are set up according to `alert_engine_infra_setup.md`:

### ✅ Required Infrastructure

1. **Redis Cluster** - Running in `redis-cluster` namespace
   - Service: `redis-cluster-access.redis-cluster.svc.cluster.local:6379`
   
2. **AMQ Streams Kafka** - Running in `amq-streams-kafka` namespace  
   - Bootstrap: `alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092`
   - Topic: `application-logs`
   
3. **ClusterLogForwarder** - Configured to forward logs to Kafka
   - Vector pods collecting logs from all namespaces
   
4. **Slack Webhook URL** - For alert notifications

## 📦 Deployment Components

This deployment includes the following Kubernetes resources:

| Resource | File | Description |
|----------|------|-------------|
| Namespace | `namespace.yaml` | Dedicated namespace for Alert Engine |
| ServiceAccount | `serviceaccount.yaml` | RBAC with cluster read permissions |
| Secret | `secret.yaml` | Slack webhook URL storage |
| ConfigMap | `configmap.yaml` | Production configuration |
| Deployment | `deployment.yaml` | Main application deployment (2 replicas) |
| Service | `service.yaml` | Internal service + OpenShift Route |
| NetworkPolicy | `networkpolicy.yaml` | Network security policies |
| Kustomization | `kustomization.yaml` | Deployment orchestration |

## 🚀 Quick Deployment

### Step 1: Configure Slack Webhook

Update the Secret with your Slack webhook URL:

```bash
# Option 1: Edit the secret.yaml file directly
# Replace the slack-webhook-url value in secret.yaml with your base64 encoded webhook URL

# Option 2: Use kubectl/oc to create the secret
oc create secret generic alert-engine-secrets \
  --from-literal=slack-webhook-url="https://hooks.slack.com/services/YOUR/WEBHOOK/URL" \
  --namespace=alert-engine
```

### Step 2: Build Container Image

The Alert Engine includes a complete container build system following Red Hat standards.

#### Option A: Using the Build Script (Recommended)

```bash
# Navigate to project root (where go.mod is located)
cd ../../

# Build image locally
./deployments/alert-engine/build.sh

# Build with specific version and push to registry
./deployments/alert-engine/build.sh --version v1.0.0 --push

# Build with tests and custom registry
./deployments/alert-engine/build.sh --test --registry quay.io/your-org --push

# See all options
./deployments/alert-engine/build.sh --help
```

#### Option B: Using Makefile

```bash
# From project root, use convenient Makefile targets
make docker-build                    # Build image locally
make docker-push                     # Build and push image  
make docker-test                     # Build with tests
make release-build                   # Full release build with all checks

# See all available targets
make help
```

#### Build Script Features

- ✅ **Red Hat UBI8** base images with Go toolchain
- ✅ **Multi-stage build** for minimal runtime image (UBI8 Micro)
- ✅ **Security hardened** - non-root user, minimal attack surface
- ✅ **OpenShift compatible** - x86_64 architecture, proper labels
- ✅ **Automated testing** - optional test execution before build
- ✅ **Version embedding** - build version and timestamp in binary
- ✅ **Manifest updates** - automatically updates deployment files

#### Container Image Details

| Stage | Base Image | Purpose |
|-------|------------|---------|
| Builder | `registry.access.redhat.com/ubi8/go-toolset:latest` | Compile Go application |
| Runtime | `registry.access.redhat.com/ubi8/ubi-micro:latest` | Minimal runtime container |

The resulting image:
- **Size**: ~50MB (minimal UBI8 Micro + static binary)
- **Security**: Non-root user (UID 1001), read-only filesystem ready
- **Health checks**: Built-in health check endpoint
- **Labels**: Full Red Hat label compliance

### Step 3: Update Container Image Reference

After building, update the deployment to use your image:

#### Automatic Update (Recommended)
The build script automatically updates the deployment files when using `--push`:

```bash
./deployments/alert-engine/build.sh --tag v1.0.0 --push
# Automatically updates deployment.yaml and kustomization.yaml
```

#### Manual Update
Edit `deployment.yaml` and update the container image:

```yaml
containers:
  - name: alert-engine
    image: quay.io/your-registry/alert-engine:v1.0.0  # Update this
```

Or use Kustomize to update the image:

```bash
# Edit kustomization.yaml
images:
  - name: quay.io/your-registry/alert-engine
    newTag: v1.0.0  # Your image tag
```

### Step 4: Deploy Everything

```bash
# Deploy using Kustomize (recommended)
oc apply -k .

# Or deploy individual manifests
oc apply -f namespace.yaml
oc apply -f serviceaccount.yaml
oc apply -f secret.yaml
oc apply -f configmap.yaml
oc apply -f deployment.yaml
oc apply -f service.yaml
oc apply -f networkpolicy.yaml
```

## 🔍 Verification

### Check Deployment Status

```bash
# Check namespace
oc get namespace alert-engine

# Check all resources
oc get all -n alert-engine

# Check pod status and logs
oc get pods -n alert-engine
oc logs -n alert-engine deployment/alert-engine -f

# Check service connectivity
oc get svc -n alert-engine
oc get route -n alert-engine
```

### Verify Complete Log Flow Pipeline

#### Step 1: Check Vector/ClusterLogForwarder Status

```bash
# Check Vector pods are running and collecting logs
oc get pods -n openshift-logging -l component=vector

# Check ClusterLogForwarder configuration
oc get clusterlogforwarder -o yaml

# Verify Vector is processing logs (check recent logs)
oc logs -n openshift-logging -l component=vector --tail=50 | grep "application-logs"
```

#### Step 2: Verify Kafka Topic and Message Flow

```bash
# Check if Kafka topic exists and has recent messages
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe --topic application-logs

# Sample recent messages (should see logs from Vector)
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
  /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic application-logs \
  --max-messages 5 --timeout-ms 10000 \
  --property print.timestamp=true

# Check consumer group status (Alert Engine should be consuming)
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
  /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe --group alert-engine-consumer
```

#### Step 3: Verify Alert Engine Log Processing

```bash
# Check Alert Engine is processing logs
ROUTE_URL=$(oc get route alert-engine -n alert-engine -o jsonpath='{.spec.host}')
curl -s "https://$ROUTE_URL/api/v1/system/logs/stats" | jq '.data.total_logs'

# Should show increasing log count over time
curl -s "https://$ROUTE_URL/api/v1/system/logs/stats" | jq '.'

# Check Alert Engine logs for processing confirmation
oc logs -n alert-engine deployment/alert-engine --tail=100 | grep -E "(processed|kafka|redis)"
```

### Test Alert Engine Functionality

```bash
# Port-forward for local testing
oc port-forward -n alert-engine svc/alert-engine 8080:8080

# Test health endpoints
curl http://localhost:8080/health
curl http://localhost:8080/ready

# Test via OpenShift Route
ROUTE_URL=$(oc get route alert-engine -n alert-engine -o jsonpath='{.spec.host}')
curl https://$ROUTE_URL/health
```

### Create Safe Test Alert Rule

To test alert generation without triggering too many notifications (avoiding Slack 429 errors), create a conservative test rule:

#### Step 1: Create a Low-Frequency Test Rule

```bash
# Create a test rule that triggers on a specific, uncommon pattern
ROUTE_URL=$(oc get route alert-engine -n alert-engine -o jsonpath='{.spec.host}')

curl -X POST "https://$ROUTE_URL/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-verification-rule",
    "name": "🧪 Verification Test Rule",
    "description": "Safe test rule for deployment verification - triggers on uncommon patterns",
    "enabled": true,
    "conditions": {
      "log_level": "ERROR",
      "service": "non-existent-service",
      "threshold": 1,
      "time_window": 3600,
      "operator": "gte"
    },
    "actions": {
      "channel": "#test-alerts",
      "severity": "low"
    }
  }'
```

#### Step 2: Alternative Safe Test Rule (Using Keywords)

```bash
# Create a rule that looks for a very specific test pattern
curl -X POST "https://$ROUTE_URL/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-safe-alert-verification",
    "name": "🔍 Safe Alert Verification",
    "description": "Test rule with unique keyword - safe for production testing",
    "enabled": true,
    "conditions": {
      "keywords": ["ALERT_ENGINE_TEST_VERIFICATION_12345"],
      "threshold": 1,
      "time_window": 1800,
      "operator": "gte"
    },
    "actions": {
      "channel": "#test-alerts",
      "severity": "low"
    }
  }'
```

#### Step 3: Trigger Test Alert Safely

```bash
# Option A: Generate a test log entry with the specific keyword
# (This would need to be done from an application pod that logs to the cluster)

# Option B: Check if the rule exists and is active
curl -s "https://$ROUTE_URL/api/v1/rules" | jq '.data[] | select(.id | contains("test"))'

# Verify rule statistics
curl -s "https://$ROUTE_URL/api/v1/rules/stats" | jq '.'
```

#### Step 4: Monitor Test Results

```bash
# Check if any alerts were generated
curl -s "https://$ROUTE_URL/api/v1/alerts/recent?limit=10" | jq '.data'

# Monitor for a few minutes, then check system metrics
curl -s "https://$ROUTE_URL/api/v1/system/metrics" | jq '.data'

# Check Alert Engine logs for processing activity
oc logs -n alert-engine deployment/alert-engine --tail=50 | grep -E "(alert|rule|notification)"
```

#### Step 5: Clean Up Test Rules

```bash
# Remove test rules after verification
curl -X DELETE "https://$ROUTE_URL/api/v1/rules/test-verification-rule"
curl -X DELETE "https://$ROUTE_URL/api/v1/rules/test-safe-alert-verification"

# Verify cleanup
curl -s "https://$ROUTE_URL/api/v1/rules" | jq '.data | length'
```

### End-to-End Flow Verification Summary

After running the above steps, you should confirm:

1. ✅ **Vector/ClusterLogForwarder**: Collecting logs from cluster pods
2. ✅ **Kafka**: Receiving transformed logs on `application-logs` topic  
3. ✅ **Alert Engine**: Consuming from Kafka and processing logs
4. ✅ **Redis**: Storing alert rules and state data
5. ✅ **Slack Integration**: Webhook configured and ready (test rules safely created)
6. ✅ **API Endpoints**: Health, rules, alerts, and metrics responding correctly

### Expected Results

- **Log Processing**: `total_logs` count should be > 0 and increasing
- **Consumer Group**: Alert Engine should show up as active consumer
- **Alert Rules**: Test rules created and visible via API
- **System Health**: All endpoints returning healthy status
- **No 429 Errors**: Conservative test rules prevent Slack rate limiting

## ⚙️ Configuration

### Production Configuration

The `configmap.yaml` contains production-optimized settings:

- **Redis**: Cluster mode enabled, connects to Redis cluster
- **Kafka**: Production group ID, optimized batch processing  
- **Alerting**: Higher thresholds (5 errors in 5m) to reduce noise
- **Slack**: Production channel `#alerts`
- **Processing**: Higher batch sizes for production scale

### Environment Variables

The deployment uses these environment variables:

| Variable | Value | Source |
|----------|-------|--------|
| `CONFIG_PATH` | `/etc/alert-engine/config.yaml` | Hardcoded |
| `SLACK_WEBHOOK_URL` | `<webhook-url>` | Secret |
| `LOG_LEVEL` | `INFO` | Hardcoded |
| `ENVIRONMENT` | `production` | Hardcoded |

## 🔒 Security Features

### RBAC Permissions

The Alert Engine has minimal required permissions:

- **Read-only** access to pods, services, namespaces
- **Read-only** access to events and deployments
- **No write** permissions to cluster resources

### Network Security

NetworkPolicy restricts traffic to:

- **Ingress**: Only from OpenShift routers and monitoring
- **Egress**: Only to Kafka, Redis, DNS, and HTTPS endpoints

### Pod Security

- Runs as non-root user
- Read-only root filesystem
- Security context constraints compliant
- Drops all capabilities

## 📊 Monitoring

### Health Checks

- **Liveness Probe**: `/health` endpoint (30s interval)
- **Readiness Probe**: `/ready` endpoint (10s interval)

### Metrics

- **Prometheus**: Metrics exposed on `/metrics` endpoint
- **Annotations**: Configured for automatic scraping

### Logs

```bash
# Stream logs from all Alert Engine pods
oc logs -n alert-engine -l app.kubernetes.io/name=alert-engine -f

# View logs from specific pod
oc logs -n alert-engine <pod-name> -f
```

## 🔧 Troubleshooting

### Common Issues

1. **Pod not starting**
   ```bash
   oc describe pod -n alert-engine <pod-name>
   oc logs -n alert-engine <pod-name>
   ```

2. **Cannot connect to Redis**
   ```bash
   # Test Redis connectivity from Alert Engine pod
   oc exec -n alert-engine <pod-name> -- nc -zv redis-cluster-access.redis-cluster.svc.cluster.local 6379
   ```

3. **Cannot connect to Kafka**
   ```bash
   # Test Kafka connectivity
   oc exec -n alert-engine <pod-name> -- nc -zv alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local 9092
   ```

4. **Slack notifications not working**
   ```bash
   # Check secret
   oc get secret alert-engine-secrets -n alert-engine -o yaml
   
   # Test webhook URL manually
   curl -X POST -H 'Content-type: application/json' \
     --data '{"text":"Test from Alert Engine"}' \
     <your-webhook-url>
   ```

### Resource Scaling

```bash
# Scale replicas
oc scale deployment alert-engine -n alert-engine --replicas=3

# Update resource limits
oc patch deployment alert-engine -n alert-engine -p '{"spec":{"template":{"spec":{"containers":[{"name":"alert-engine","resources":{"limits":{"cpu":"1000m","memory":"1Gi"}}}]}}}}'
```

## 🔄 Updates

### Rolling Updates

```bash
# Update image
oc set image deployment/alert-engine alert-engine=quay.io/your-registry/alert-engine:v1.1.0 -n alert-engine

# Update configuration
oc apply -f configmap.yaml
oc rollout restart deployment/alert-engine -n alert-engine
```

### Rollback

```bash
# View rollout history
oc rollout history deployment/alert-engine -n alert-engine

# Rollback to previous version
oc rollout undo deployment/alert-engine -n alert-engine
```

## 🧹 Cleanup

```bash
# Remove Alert Engine deployment
oc delete -k .

# Or remove individual resources
oc delete namespace alert-engine
```

## 📝 Notes

- **High Availability**: Configured with 2 replicas and anti-affinity rules
- **Zero Downtime**: Rolling updates with `maxUnavailable: 0`
- **Resource Efficiency**: Conservative resource requests with appropriate limits
- **Security**: Follows OpenShift security best practices
- **Monitoring**: Integrated with OpenShift monitoring stack 