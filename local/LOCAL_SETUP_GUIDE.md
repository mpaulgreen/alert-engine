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

This guide assumes you're working from the `alert-engine/local` directory. All local development files and scripts are located here:

```
alert-engine/
├── local/                          # Local development files
│   ├── LOCAL_SETUP_GUIDE.md        # This guide
│   ├── QUICK_START.md              # Quick setup reference
│   ├── local-setup.sh              # Automated setup script
│   └── test-local-setup.sh         # Validation test script
├── cmd/                            # Source code
├── configs/                        # Configuration files
└── ... (other project files)
```

### Getting Started
```bash
# Navigate to the local directory
cd alert-engine/local

# Run the automated setup
./local-setup.sh
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
  channel: "#alerts"
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
# Navigate to alert-engine directory
cd alert-engine

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
# Terminal 3: Run Alert Engine locally
./alert-engine

# Alternative: Run with go run
go run ./cmd/server/main.go
```

### Expected Output:
```
2025/07/14 12:00:00 Starting Alert Engine...
2025/07/14 12:00:00 Loading configuration from ./configs/config.yaml
2025/07/14 12:00:01 Connected to Redis at localhost:6379
2025/07/14 12:00:01 Connected to Kafka brokers: [localhost:9092]
2025/07/14 12:00:01 Subscribed to topic: application-logs
2025/07/14 12:00:01 Starting HTTP server on :8080
2025/07/14 12:00:01 Starting metrics server on :8081
2025/07/14 12:00:01 Alert Engine started successfully
```

## 🧪 Test Cases and Sanity Checks

### Test 1: Health Check

```bash
# Check if Alert Engine is running
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","timestamp":"2025-07-14T12:00:00Z"}

# Check metrics endpoint
curl http://localhost:8081/metrics
```

### Test 2: Kafka Connectivity Test

```bash
# Send test log to Kafka (this should trigger alerts)
echo '{"timestamp":"2025-07-14T12:00:00Z","level":"ERROR","message":"Test error message for local Alert Engine","service":"test-service","namespace":"alert-engine","user_id":"test-123"}' | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs

# Check Alert Engine logs - you should see it process this message
```

### Test 3: Redis Connectivity Test

```bash
# Check if Alert Engine can write to Redis
curl -X POST http://localhost:8080/api/v1/test-redis \
  -H "Content-Type: application/json" \
  -d '{"test_key": "test_value"}'

# Verify data in Redis via OpenShift
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c get test_key
```

### Test 4: Slack Notification Test

```bash
# Trigger a test alert via API
curl -X POST http://localhost:8080/api/v1/test-alert \
  -H "Content-Type: application/json" \
  -d '{
    "level": "ERROR",
    "message": "Test alert from local Alert Engine",
    "service": "local-test",
    "namespace": "alert-engine"
  }'

# Check your Slack channel for the alert notification
```

### Test 5: End-to-End Alert Flow Test

```bash
# 1. Generate multiple error logs to trigger threshold
for i in {1..5}; do
  echo "{\"timestamp\":\"$(date -Iseconds)\",\"level\":\"ERROR\",\"message\":\"Test error $i from local setup\",\"service\":\"test-service\",\"namespace\":\"alert-engine\",\"sequence\":$i}" | oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs
  sleep 2
done

# 2. Check Alert Engine logs for processing
# 3. Check Slack for alert notification
# 4. Check Redis for alert state
oc exec -n redis-cluster redis-cluster-0 -- redis-cli -c keys "*alert*"
```

### Test 6: API Endpoints Test

```bash
# Get current alert rules
curl http://localhost:8080/api/v1/rules

# Get alert statistics
curl http://localhost:8080/api/v1/stats

# Create a new test rule
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
      "time_window": "1m",
      "operator": "gt"
    },
    "actions": {
      "channel": "#alerts",
      "severity": "low"
    }
  }'
```

## 🔍 Monitoring and Debugging

### Log Monitoring
```bash
# Follow Alert Engine logs
tail -f /tmp/alert-engine-local.log

# Monitor with specific log level
grep "ERROR\|WARN" /tmp/alert-engine-local.log

# Monitor Kafka consumption
grep "kafka" /tmp/alert-engine-local.log
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

✅ **Health Check**: `curl http://localhost:8080/health` returns healthy status  
✅ **Kafka Connection**: Alert Engine logs show successful Kafka subscription  
✅ **Redis Connection**: Alert Engine can store and retrieve data  
✅ **Log Processing**: Test logs are processed and visible in logs  
✅ **Alert Generation**: Error logs trigger alerts based on rules  
✅ **Slack Notifications**: Alerts are sent to your Slack channel  
✅ **API Access**: All API endpoints respond correctly  
✅ **Metrics**: Prometheus metrics are available at `:8081/metrics`

## 🎯 Next Steps

Once your local setup is working:

1. **Customize Alert Rules**: Add your own alert rules via API or config
2. **Test Real Scenarios**: Use your actual log patterns for testing
3. **Performance Testing**: Generate load to test scalability
4. **Integration Testing**: Test with your existing monitoring tools
5. **Development**: Make code changes and test locally before deployment

---

**🎉 Congratulations!** Your Alert Engine is now running locally while connected to your OpenShift infrastructure! 