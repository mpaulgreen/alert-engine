# Alert Engine Local Setup Guide

This guide will help you run the Alert Engine locally on your Mac while connecting to the Kafka, Redis, and ClusterLogForwarder infrastructure deployed on OpenShift.

## 📋 Prerequisites

### Required Software
1. **Go 1.23+** - Install from [golang.org](https://golang.org/downloads/)
2. **OpenShift CLI (oc)** - Already installed and configured
3. **Git** - For version control

### Verify Prerequisites
```bash
# Verify Go installation
go version

# Verify OpenShift CLI access
oc version

# Verify cluster access
oc whoami
```

## 📁 Directory Structure

This guide assumes you're working from the `alert-engine` directory. The project contains several directories with scripts for different purposes:

```
alert-engine/
├── local/                          # Local development files
│   ├── LOCAL_SETUP_GUIDE.md        # This comprehensive guide
│   ├── QUICK_START.md              # Quick setup reference
│   ├── README.md                   # Local development overview
│   ├── local-setup.sh              # Automated local setup script
│   ├── setup-port-forwards.sh      # OpenShift port forwarding setup
│   ├── start-local.sh              # Start Alert Engine locally
│   └── test-local-setup.sh         # Validation and testing script
├── scripts/                        # Testing and automation scripts
│   ├── README.md                   # Testing documentation
│   ├── run_integration_tests.sh    # Integration test runner
│   └── run_unit_tests.sh           # Unit test runner
├── cleanup/                        # Infrastructure cleanup scripts
│   ├── cleanup_openshift_infrastructure.sh    # Remove OpenShift resources
│   └── verify_resources_before_cleanup.sh     # Pre-cleanup verification
├── cmd/                            # Source code
├── configs/                        # Configuration files
├── deployments/                    # OpenShift deployment manifests
├── internal/                       # Internal Go packages
├── pkg/                            # Public Go packages
└── ... (other project files)
```

### Script Overview

#### Local Development Scripts (`local/`)
- **`local-setup.sh`** - Main setup script that builds the project, creates config files, and sets up environment
- **`setup-port-forwards.sh`** - Sets up port forwarding to OpenShift services (Kafka & Redis)
- **`start-local.sh`** - Starts the Alert Engine with all prerequisite checks
- **`test-local-setup.sh`** - Comprehensive testing script to validate the setup

#### Testing Scripts (`scripts/`)
- **`run_unit_tests.sh`** - Runs all unit tests with coverage reporting
- **`run_integration_tests.sh`** - Runs integration tests with external dependencies

#### Cleanup Scripts (`cleanup/`)
- **`cleanup_openshift_infrastructure.sh`** - Removes all OpenShift resources
- **`verify_resources_before_cleanup.sh`** - Safety check before cleanup

### Getting Started
```bash
# Navigate to the alert-engine directory
cd alert-engine

# Run the automated setup
./local/local-setup.sh
```

## 🔧 OpenShift Infrastructure Connection

### Step 1: Set Up Port Forwarding for OpenShift Services

Since the Alert Engine will run locally, we need to create port forwards to access the OpenShift services:

#### Kafka Port Forward
```bash
# Terminal 1: Forward Kafka bootstrap server
oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092

# Keep this terminal open
```

#### Redis Port Forward  
```bash
# Terminal 2: Forward Redis cluster access
oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379

# Keep this terminal open
```

### Step 2: Verify OpenShift Service Connectivity
```bash
# Test Kafka connectivity (in a new terminal)
echo "test message" | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs

# Test Redis connectivity
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c ping
```

## 🔐 Slack Integration Setup

### Step 1: Create Slack Webhook URL

1. **Go to Slack API**: Visit [api.slack.com](https://api.slack.com/apps)
2. **Create New App**: Click "Create New App" → "From scratch"
3. **App Details**:
   - App Name: `Alert Engine`
   - Workspace: Select your workspace
4. **Enable Incoming Webhooks**:
   - Go to "Incoming Webhooks" in the left sidebar
   - Toggle "Activate Incoming Webhooks" to On
   - Click "Add New Webhook to Workspace"
   - Select the channel (e.g., `#alerts`)
   - Click "Allow"
5. **Copy Webhook URL**: Save the webhook URL (starts with `https://hooks.slack.com/services/...`)

### Step 2: Test Slack Webhook
```bash
# Test your webhook URL
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"Alert Engine test message from local setup!"}' \
  YOUR_WEBHOOK_URL_HERE
```

## ⚙️ Configuration Setup

### Step 1: Update config.yaml for Local + OpenShift

Create/update `configs/config.yaml`:

```yaml
# Alert Engine Configuration - Local + OpenShift Setup
# Updated for connecting to OpenShift infrastructure

# Server configuration
server:
  address: ":8080"
  read_timeout: "30s"
  write_timeout: "30s" 
  idle_timeout: "60s"

# Redis configuration (via port-forward to OpenShift)
redis:
  # Using port-forward to OpenShift Redis cluster
  address: "localhost:6379"
  password: ""
  database: 0
  cluster_mode: false  # Disable cluster mode for local port-forwarding
  max_retries: 3
  pool_size: 10
  min_idle_conns: 5
  dial_timeout: "10s"  # Increased for network latency
  read_timeout: "5s"   # Increased for network latency
  write_timeout: "5s"  # Increased for network latency
  pool_timeout: "10s"  # Increased for network latency
  idle_timeout: "5m"

# Kafka configuration (via port-forward to OpenShift)
kafka:
  brokers:
    - "localhost:9092"  # Port-forwarded to OpenShift Kafka
  topic: "application-logs"
  group_id: "alert-engine-local-group"  # Unique group for local testing
  consumer:
    min_bytes: 1024         # Smaller for testing
    max_bytes: 1048576      # 1MB
    max_wait: "2s"          # Increased for network latency
    start_offset: -1        # Latest offset
  producer:
    batch_size: 50          # Smaller for testing
    batch_timeout: "2s"     # Increased for network latency

# Slack notification configuration
slack:
  webhook_url: ""           # Set via environment variable SLACK_WEBHOOK_URL
  channel: "#test-mp-channel"
  username: "Alert Engine (Local)"
  icon_emoji: ":warning:"
  timeout: "30s"

# Notification settings
notifications:
  enabled: true
  max_retries: 3
  retry_delay: "5s"
  timeout: "30s"
  rate_limit_per_min: 60
  batch_size: 5             # Smaller for testing
  batch_delay: "2s"
  enable_deduplication: true
  deduplication_window: "5m"

# Alert processing configuration
alerting:
  enabled: true
  batch_size: 50            # Smaller for testing
  flush_interval: "10s"     # More frequent for testing
  max_rules: 1000
  default_time_window: "5m"
  default_threshold: 3      # Lower threshold for testing
  cleanup_interval: "1h"

# Log processing configuration
log_processing:
  enabled: true
  batch_size: 50            # Smaller for testing
  flush_interval: "10s"     # More frequent for testing
  max_message_size: 1048576 # 1MB
  validation:
    require_timestamp: true
    require_level: true
    require_message: true
    require_namespace: true

# Monitoring and metrics
monitoring:
  enabled: true
  metrics_port: 8081
  health_check_interval: "30s"
  log_level: "debug"        # Debug for local development
  enable_pprof: true        # Enable profiling for local dev

# Test alert rules for local development
default_rules:
  enabled: true
  rules:
    - id: "local-test-error-alert"
      name: "Local Test - Application Error Alert"
      description: "Alert on application errors (local testing)"
      enabled: true
      conditions:
        log_level: "ERROR"
        threshold: 2          # Lower threshold for testing
        time_window: "2m"     # Shorter window for testing
        operator: "gt"
      actions:
        channel: "#alerts"
        severity: "high"

    - id: "local-test-alert-engine-logs"
      name: "Local Test - Alert Engine Namespace Logs" 
      description: "Alert on alert-engine namespace activity"
      enabled: true
      conditions:
        namespace: "alert-engine"
        log_level: "ERROR"
        threshold: 1          # Very low threshold for testing
        time_window: "1m"     # Short window for testing
        operator: "gt"
      actions:
        channel: "#alerts"
        severity: "medium"

    - id: "local-test-continuous-log-alerts"
      name: "Local Test - Continuous Log Generator Alerts"
      description: "Alert on continuous log generator errors"
      enabled: true
      conditions:
        service: "continuous-log-generator"
        log_level: "ERROR"
        threshold: 3          # Moderate threshold
        time_window: "3m"
        operator: "gt"
      actions:
        channel: "#alerts"
        severity: "medium"

# Security configuration
security:
  enable_cors: true
  cors_origins: ["*"]
  enable_auth: false        # Disable for local development
  api_key: ""

# Performance tuning (relaxed for local development)
performance:
  max_concurrent_rules: 50
  rule_evaluation_timeout: "2s"
  max_memory_usage: "256Mi"
  gc_percentage: 100

# Logging configuration
logging:
  level: "debug"            # Debug for local development
  format: "text"            # Text format for easier reading locally
  output: "stdout"
  file:
    path: "/tmp/alert-engine-local.log"
    max_size: "10MB"
    max_backups: 3
    max_age: "7d"
    compress: true

# Environment-specific overrides
env_overrides:
  development:
    logging:
      level: "debug"
    monitoring:
      enable_pprof: true
    alerting:
      default_threshold: 1  # Very sensitive for testing
  
  production:
    logging:
      level: "warn"
    performance:
      max_memory_usage: "1Gi"
    kafka:
      brokers:
        - "alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"
    redis:
      address: "redis-cluster-access.redis-cluster.svc.cluster.local:6379"
    
  test:
    logging:
      level: "error"
    kafka:
      topic: "test-application-logs"
    redis:
      database: 1 
```

### Step 2: Set Environment Variables

Create a `.env` file (and add it to `.gitignore`):

```bash
# .env file for local development
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
export CONFIG_PATH="./configs/config.yaml"
export LOG_LEVEL="debug"
export ENVIRONMENT="development"
```

## 🚀 Running Alert Engine Locally

### Step 1: Prepare Environment

```bash
# Source environment variables
source .env

# Verify Go modules
go mod tidy
go mod download

# Build the application
go build -o alert-engine ./cmd/server/
```

### Step 2: Start Required Port Forwards

**Important**: These must be running before starting the Alert Engine!

```bash
# Terminal 1: Kafka port forward
oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092

# Terminal 2: Redis port forward  
oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379
```

### Step 3: Run Alert Engine

```bash
# Terminal 3: Run Alert Engine locally (logs to terminal)
./alert-engine

# Alternative: Run with go run
go run ./cmd/server/main.go

# OPTION: Run with file logging (logs to both terminal and file)
./alert-engine 2>&1 | tee /tmp/alert-engine-local.log

# OPTION: Run with file logging only (logs to file only)
./alert-engine > /tmp/alert-engine-local.log 2>&1

# OPTION: Run in background with file logging
nohup ./alert-engine > /tmp/alert-engine-local.log 2>&1 &
```

### Expected Output:
```
2025/07/14 13:08:49 Loaded 0 alert rules
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /api/v1/health            --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).Health-fm (4 handlers)
[GIN-debug] GET    /api/v1/rules             --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).GetRules-fm (4 handlers)
[GIN-debug] POST   /api/v1/rules             --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).CreateRule-fm (4 handlers)
[GIN-debug] GET    /api/v1/rules/stats       --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).GetRuleStats-fm (4 handlers)
[GIN-debug] GET    /api/v1/rules/template    --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).GetRuleTemplate-fm (4 handlers)
[GIN-debug] GET    /api/v1/rules/defaults    --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).GetDefaultRules-fm (4 handlers)
[GIN-debug] POST   /api/v1/rules/bulk        --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).BulkCreateRules-fm (4 handlers)
[GIN-debug] POST   /api/v1/rules/reload      --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).ReloadRules-fm (4 handlers)
[GIN-debug] POST   /api/v1/rules/filter      --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).FilterRules-fm (4 handlers)
[GIN-debug] POST   /api/v1/rules/test        --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).TestRule-fm (4 handlers)
[GIN-debug] GET    /api/v1/rules/:id         --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).GetRule-fm (4 handlers)
[GIN-debug] PUT    /api/v1/rules/:id         --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).UpdateRule-fm (4 handlers)
[GIN-debug] DELETE /api/v1/rules/:id         --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).DeleteRule-fm (4 handlers)
[GIN-debug] GET    /api/v1/alerts/recent     --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).GetRecentAlerts-fm (4 handlers)
[GIN-debug] GET    /api/v1/system/metrics    --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).GetMetrics-fm (4 handlers)
[GIN-debug] GET    /api/v1/system/logs/stats --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).GetLogStats-fm (4 handlers)
[GIN-debug] GET    /                         --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).SetupRoutes.func2 (4 handlers)
[GIN-debug] GET    /docs                     --> github.com/log-monitoring/alert-engine/internal/api.(*Handlers).SetupRoutes.func3 (4 handlers)
2025/07/14 13:08:49 Starting Kafka consumer...
2025/07/14 13:08:49 Starting log processor for topic: application-logs
2025/07/14 13:08:49 Starting HTTP server on :8080
[GIN-debug] Listening and serving HTTP on :8080
```

**Note**: The GIN debug messages show all available API endpoints. This is normal in development mode.

## 📝 Configuring File Logging

By default, the Alert Engine logs to the terminal (stdout). To enable file logging to `/tmp/alert-engine-local.log`, you have several options:

### Option 1: Log to Both Terminal and File (Recommended)
```bash
./alert-engine 2>&1 | tee /tmp/alert-engine-local.log
```
**Benefits**: See logs in real-time AND save to file for later analysis.

### Option 2: Log to File Only
```bash
./alert-engine > /tmp/alert-engine-local.log 2>&1
```
**Benefits**: All output goes to file, cleaner terminal.

### Option 3: Background Process with File Logging
```bash
nohup ./alert-engine > /tmp/alert-engine-local.log 2>&1 &
```
**Benefits**: Runs in background, logs to file, survives terminal closing.

### Option 4: Update start-local.sh for File Logging
Edit `local/start-local.sh` and change the last line from:
```bash
exec ./alert-engine
```
to:
```bash
exec ./alert-engine 2>&1 | tee /tmp/alert-engine-local.log
```

### Configuration File Setup (Future Enhancement)
The `configs/config.yaml` file includes logging configuration:
```yaml
logging:
  level: "debug"
  format: "text"
  output: "stdout"
  file:
    path: "/tmp/alert-engine-local.log"
    max_size: "10MB"
    max_backups: 3
    max_age: "7d"
    compress: true
```

**Note**: This configuration is defined but not currently implemented in the code. Use the shell redirection methods above for now.

## 🧪 Test Cases and Sanity Checks

### Test 1: Health Check

```bash
# Check if Alert Engine is running
curl http://localhost:8080/api/v1/health

# Expected response:
# {"success":true,"data":{"status":"healthy","timestamp":"2025-07-14T12:00:00Z"}}

# Check system metrics endpoint
curl http://localhost:8080/api/v1/system/metrics

# Check root API endpoint
curl http://localhost:8080/
```

### Test 2: Kafka Connectivity Test

```bash
# Send test log to Kafka
echo '{"timestamp":"2025-07-14T12:00:00Z","level":"ERROR","message":"Test error message for local Alert Engine","kubernetes":{"namespace":"alert-engine","pod":"test-pod-123","container":"test-container","labels":{"app":"test-service","version":"1.0.0"}},"host":"test-host"}' | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs

# Verify the message was sent to the topic
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --max-messages 5

# To find your specific test message, you can search for it:
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --timeout-ms 5000 | grep -i "Test error message"

# Expected result: You should see your test message in the output
```

#### 📝 **Expected Message Format**

The Alert Engine expects log messages in this specific JSON structure:

```json
{
  "timestamp": "2025-07-14T12:00:00Z",
  "level": "ERROR",
  "message": "Your log message here",
  "kubernetes": {
    "namespace": "your-namespace",
    "pod": "pod-name",
    "container": "container-name",
    "labels": {
      "app": "your-app",
      "version": "1.0.0"
    }
  },
  "host": "hostname"
}
```

**Required fields:**
- `timestamp`: ISO 8601 timestamp
- `level`: Log level (DEBUG, INFO, WARN, ERROR, FATAL)
- `message`: Log message content (cannot be empty)
- `kubernetes.namespace`: Kubernetes namespace (required for validation)

**Optional fields:**
- `kubernetes.pod`, `kubernetes.container`, `kubernetes.labels`
- `host`: Hostname where log originated

**⚠️ Important Note on Alert Rules:**
When creating alert rules via API, the `time_window` field must be specified in **nanoseconds**:
- 1 minute = `60000000000` (60 seconds × 1,000,000,000 nanoseconds)
- 5 minutes = `300000000000` (300 seconds × 1,000,000,000 nanoseconds)
- 1 hour = `3600000000000` (3600 seconds × 1,000,000,000 nanoseconds)

#### 🚨 **Known Issue: Alert Engine Message Processing**

Due to Kafka port-forwarding limitations, the Alert Engine may show these errors in the logs:
- `[6] Not Leader For Partition` 
- `Error processing message: fetching message: EOF`

**This is expected behavior** and doesn't indicate a problem with your setup.

#### **Why This Happens:**
- The message **is successfully sent** to the Kafka topic ✅
- The Alert Engine **connects** to Kafka initially ✅
- But Kafka redirects consumers to internal broker addresses not accessible through port-forwarding ❌

#### **Verification Steps:**
```bash
# 1. Check if your message reached the topic (should work)
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --max-messages 5

# 1a. To search for your specific test message:
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --timeout-ms 10000 | grep -i "your-search-term"

# 2. Check Alert Engine logs for connection attempts
tail -f /tmp/alert-engine-local.log

# 3. Verify Alert Engine is healthy
curl http://localhost:8080/api/v1/health
```

#### **Solutions:**
1. **For Testing**: Use the verification steps above to confirm connectivity
2. **For Production**: Deploy to OpenShift where internal addresses are accessible
3. **For Local Processing**: Consider using a local Kafka instance instead of port-forwarding

#### **Expected Log Output:**
```
2025/07/14 13:31:15 Starting Kafka consumer...
2025/07/14 13:31:15 Starting log processor for topic: application-logs
2025/07/14 13:31:15 Starting HTTP server on :8080
2025/07/14 13:31:29 Error processing message: fetching message: EOF
```

**Note**: The EOF errors are normal with port-forwarding setups. The Alert Engine will work correctly when deployed to OpenShift.

### Test 3: Redis Connectivity Test

Due to Redis cluster behavior with port-forwarding, we need to connect to the specific Redis node that handles our key range. Here's the step-by-step process:

#### Step 1: Understand Redis Cluster Setup
```bash
# View Redis cluster topology
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c CLUSTER NODES

# Expected output shows 3 masters with different slot ranges:
# - Node 1: slots 0-5460
# - Node 2: slots 5461-10922  
# - Node 3: slots 10923-16383
```

#### Step 2: Find Which Node Handles Alert Rule Keys
```bash
# Test which slot our alert rule keys hash to
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c CLUSTER KEYSLOT "alert_rule:test-rule-1"
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c CLUSTER KEYSLOT "alert_rule:test-rule-2"
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c CLUSTER KEYSLOT "alert_rule:test-rule-3"

# Example output:
# alert_rule:test-rule-1 -> Slot: 11846 (handled by node 3: 10923-16383)
# alert_rule:test-rule-2 -> Slot: 7717  (handled by node 2: 5461-10922)
# alert_rule:test-rule-3 -> Slot: 2012  (handled by node 1: 0-5460)
```

#### Step 3: Map Slot Ranges to Redis Pods
```bash
# Find which pod handles the slot range you need (e.g., 5461-10922)
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c CLUSTER NODES | grep "5461-10922"

# Example output:
# 40416fd64086e4d635458905477f26e83a637a45 10.130.2.19:6379@16379 master - 0 1752517324113 2 connected 5461-10922

# Find which pod has this IP address
oc get pods -n redis-cluster -o wide | grep "10.130.2.19"

# Example output:
# redis-cluster-1   1/1     Running   0   6h36m   10.130.2.19   ...
```

#### Step 4: Connect to the Correct Redis Node
```bash
# Stop any existing Redis port-forward
pkill -f "oc port-forward.*redis"

# Connect to the specific Redis node that handles your keys
# (In this example, redis-cluster-1 handles slots 5461-10922)
oc port-forward -n redis-cluster redis-cluster-1 6379:6379 &

# Test connection
echo "PING" | nc localhost 6379
# Should return: +PONG
```

#### Step 5: Test Redis Operations with Correct Keys
```bash
# Find a rule ID that hashes to the correct slot range
for i in {1..10}; do
  key="alert_rule:test-rule-$i"
  slot=$(oc exec -n redis-cluster redis-cluster-1 -- redis-cli -c CLUSTER KEYSLOT "$key")
  echo "Key: $key -> Slot: $slot"
  if [ "$slot" -ge 5461 ] && [ "$slot" -le 10922 ]; then
    echo "✅ Found key in correct range: $key (slot $slot)"
    correct_rule_id="test-rule-$i"
    break
  fi
done

echo "Use rule ID: $correct_rule_id"
```

#### Step 6: Test Alert Rule Creation
```bash
# Create a rule with an ID that hashes to the correct Redis node
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-rule-2",
    "name": "Redis Connectivity Test Rule",
    "description": "Test rule to verify Redis connectivity",
    "enabled": true,
    "conditions": {
      "log_level": "ERROR",
      "namespace": "test-namespace",
      "keywords": ["test"],
      "threshold": 1,
      "time_window": 60000000000,
      "operator": "gt"
    },
    "actions": {
      "channel": "#alerts",
      "severity": "medium"
    }
  }'

# Expected success response:
# {"success":true,"message":"Rule created successfully","data":{"id":"test-rule-2",...}}
```

#### Step 7: Verify Rule Storage
```bash
# Check if the rule was stored successfully
curl http://localhost:8080/api/v1/rules

# Should show the created rule in the response
```

#### Step 8: Test Rule Evaluation
```bash
# Test the rule with sample log data
curl -X POST http://localhost:8080/api/v1/rules/test \
  -H "Content-Type: application/json" \
  -d '{
    "rule": {
      "id": "test-rule-2",
      "conditions": {
        "log_level": "ERROR",
        "namespace": "test-namespace",
        "keywords": ["test"],
        "threshold": 1,
        "time_window": 60000000000,
        "operator": "gt"
      }
    },
    "sample_logs": [
      {
        "timestamp": "2025-07-14T12:00:00Z",
        "level": "ERROR",
        "message": "Test error message",
        "kubernetes": {
          "namespace": "test-namespace",
          "pod": "test-pod",
          "container": "test-container"
        },
        "host": "test-host"
      }
    ]
  }'

# Expected success response:
# {"success":true,"data":{"matched_logs":1,"would_trigger":true,"match_rate":1}}
```

#### Understanding the Results

**✅ Success Indicators:**
- Rule creation returns `{"success":true}`
- Rule appears in GET `/api/v1/rules` response
- Rule evaluation shows `"would_trigger":true`

**❌ Failure Indicators:**
- `"MOVED <slot> <ip>:6379"` errors indicate wrong Redis node
- `"connection refused"` indicates port-forward issues
- `"got 4 elements in cluster info"` indicates cluster parsing errors

#### Troubleshooting Redis Connectivity

**Issue**: Getting `MOVED` redirects
**Solution**: Use the process above to connect to the correct Redis node

**Issue**: Port-forward timeouts
**Solution**: 
```bash
# Restart port-forward
pkill -f "oc port-forward.*redis"
oc port-forward -n redis-cluster redis-cluster-1 6379:6379 &
```

**Issue**: Different rule IDs failing
**Solution**: Each rule ID hashes to a different slot - use the key slot calculator above

#### Expected Limitations

- **Kafka Consumer**: Will show "Not Leader For Partition" errors (expected with port-forwarding)
- **Some Rule IDs**: May fail if they hash to different Redis nodes
- **Performance**: Slower than direct cluster access due to port-forwarding overhead

The Redis connectivity test is successful when you can create and retrieve alert rules without `MOVED` errors.

### Test 4: Slack Notification Test

Since there's no direct test-alert endpoint, we test Slack notifications by creating an alert rule and triggering it with log messages:

```bash
# Step 1: Create a test alert rule
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "slack-test-rule",
    "name": "Slack Test Rule",
    "description": "Test rule for Slack notifications",
    "enabled": true,
    "conditions": {
      "log_level": "ERROR",
      "namespace": "alert-engine",
      "keywords": ["slack", "test"],
      "threshold": 1,
      "time_window": 60000000000,
      "operator": "gt"
    },
    "actions": {
      "channel": "#alerts",
      "severity": "medium"
    }
  }'

# Step 2: Trigger the alert by sending a matching log message
echo '{"timestamp":"2025-07-14T12:00:00Z","level":"ERROR","message":"Slack test message for notification","kubernetes":{"namespace":"alert-engine","pod":"test-pod-123","container":"test-container","labels":{"app":"slack-test"}},"host":"test-host"}' | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs

# Step 3: Check your Slack channel for the alert notification
# Step 4: Clean up - delete the test rule
curl -X DELETE http://localhost:8080/api/v1/rules/slack-test-rule
```

### Test 5: End-to-End Alert Flow Test

```bash
# 1. Generate multiple error logs to trigger threshold
for i in {1..5}; do
  echo "{\"timestamp\":\"$(date -Iseconds)\",\"level\":\"ERROR\",\"message\":\"Test error $i from local setup\",\"kubernetes\":{\"namespace\":\"alert-engine\",\"pod\":\"test-pod-$i\",\"container\":\"test-container\",\"labels\":{\"app\":\"test-service\",\"sequence\":\"$i\"}},\"host\":\"test-host\"}" | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs
  sleep 2
done

# 2. Check Alert Engine logs for processing
# 3. Check Slack for alert notification
# 4. Check Redis for alert state
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c keys "*alert*"
```

### Test 6: API Endpoints Test

#### Available API Endpoints

**Health & System:**
- `GET /api/v1/health` - System health check
- `GET /api/v1/system/metrics` - System performance metrics  
- `GET /api/v1/system/logs/stats` - Log processing statistics

**Alert Rules:**
- `GET /api/v1/rules` - Get all alert rules
- `POST /api/v1/rules` - Create new alert rule
- `GET /api/v1/rules/{id}` - Get specific alert rule
- `PUT /api/v1/rules/{id}` - Update alert rule
- `DELETE /api/v1/rules/{id}` - Delete alert rule
- `GET /api/v1/rules/stats` - Get rule statistics
- `POST /api/v1/rules/test` - Test rule against sample logs

**Alerts:**
- `GET /api/v1/alerts/recent` - Get recent alert instances

#### Testing Common Endpoints

```bash
# 1. Check system health
curl http://localhost:8080/api/v1/health

# 2. Get current alert rules
curl http://localhost:8080/api/v1/rules

# 3. Get rule statistics
curl http://localhost:8080/api/v1/rules/stats

# 4. Get recent alerts
curl http://localhost:8080/api/v1/alerts/recent

# 5. Get system metrics
curl http://localhost:8080/api/v1/system/metrics

# 6. Create a new test rule
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-rule-local",
    "name": "Test Rule for Local Development",
    "description": "Testing rule creation via API",
    "enabled": true,
    "conditions": {
      "log_level": "WARN",
      "keywords": ["test", "local"],
      "threshold": 1,
      "time_window": 60000000000,
      "operator": "gt"
    },
    "actions": {
      "channel": "#alerts",
      "severity": "low"
    }
  }'

# 7. Test a rule against sample logs
curl -X POST http://localhost:8080/api/v1/rules/test \
  -H "Content-Type: application/json" \
  -d '{
    "rule": {
      "name": "Test Rule",
      "conditions": {
        "log_level": "ERROR",
        "keywords": ["test"]
      }
    },
    "sample_logs": [
      {
        "level": "ERROR",
        "message": "Test error message",
        "timestamp": "2025-07-14T12:00:00Z"
      }
    ]
  }'
```

## 🔍 Monitoring and Debugging

### Log Monitoring
```bash
# METHOD 1: Real-time monitoring (if running in terminal)
# Simply check the terminal where you started ./alert-engine
# - All log messages appear in real-time
# - Look for Kafka message processing logs
# - Watch for error messages or alerts being triggered

# METHOD 2: Follow log file (if using file logging - see configuration section)
tail -f /tmp/alert-engine-local.log

# METHOD 3: Monitor with specific log levels
grep "ERROR\|WARN" /tmp/alert-engine-local.log

# METHOD 4: Monitor Kafka message processing
grep -i "kafka\|processing\|message" /tmp/alert-engine-local.log

# METHOD 5: Live log filtering (while Alert Engine runs)
# In a separate terminal:
tail -f /tmp/alert-engine-local.log | grep -E "ERROR|WARN|Kafka|Processing"

# Check if Alert Engine is actively processing logs
ps aux | grep alert-engine
lsof -p <PID> | grep log  # Replace <PID> with actual process ID
```

### Connection Monitoring
```bash
# Monitor port forwards
lsof -i :9092  # Kafka port forward
lsof -i :6379  # Redis port forward
lsof -i :8080  # Alert Engine server
lsof -i :8081  # Alert Engine metrics
```

### Performance Monitoring
```bash
# Check memory usage
ps aux | grep alert-engine

# Monitor Go runtime metrics
curl http://localhost:8081/debug/pprof/heap
curl http://localhost:8081/debug/pprof/goroutine?debug=1
```

## 🚨 Troubleshooting

### Common Issues and Solutions

#### Issue 1: Port Forward Connection Lost
```bash
# Symptoms: "connection refused" errors
# Solution: Restart port forwards
oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092
oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379
```

#### Issue 2: Kafka Consumer Lag  
```bash
# Check consumer group status
oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-consumer-groups.sh --bootstrap-server localhost:9092 --group alert-engine-local-group --describe
```

#### Issue 3: Redis Connection Issues
```bash
# Test Redis connectivity
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c ping
# Should return PONG
```

#### Issue 4: Slack Notifications Not Working
```bash
# Test webhook directly
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"Test from troubleshooting"}' \
  $SLACK_WEBHOOK_URL
```

## 📊 Success Criteria

Your local setup is working correctly if:

✅ **Health Check**: `curl http://localhost:8080/api/v1/health` returns healthy status  
✅ **Kafka Topic Access**: You can send messages to the `application-logs` topic  
✅ **Redis Connection**: Alert Engine can store and retrieve data  
✅ **API Access**: All API endpoints respond correctly  
✅ **Metrics**: System metrics are available at `:8080/api/v1/system/metrics`
✅ **File Logging**: Logs are saved to `/tmp/alert-engine-local.log` (if configured)

**Expected Limitations in Local Setup:**
⚠️ **Kafka Consumer**: May show "Not Leader For Partition" or "EOF" errors due to port-forwarding limitations  
⚠️ **Message Processing**: Full message processing works in OpenShift deployment, not local port-forwarding  
⚠️ **Alert Generation**: Limited by Kafka consumer issues in local setup  
⚠️ **Slack Notifications**: Dependent on successful message processing

## 🎯 Next Steps

Once your local setup is working:

1. **Customize Alert Rules**: Add your own alert rules via API or config
2. **Test Real Scenarios**: Use your actual log patterns for testing
3. **Performance Testing**: Generate load to test scalability
4. **Integration Testing**: Test with your existing monitoring tools
5. **Development**: Make code changes and test locally before deployment

---

**🎉 Congratulations!** Your Alert Engine is now running locally while connected to your OpenShift infrastructure! 