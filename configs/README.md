# Alert Engine OpenShift Production Configuration Guide

This directory contains the configuration files for the Alert Engine optimized for **OpenShift Production Deployment**. The main configuration file is `config.yaml`, which defines the active production settings for running the Alert Engine on OpenShift using the infrastructure components set up in `alert_engine_infra_setup.md`.

## 📋 Table of Contents

- [Overview](#overview)
- [Active Production Configuration](#active-production-configuration)
- [Infrastructure Dependencies](#infrastructure-dependencies)
- [Production Settings Explained](#production-settings-explained)
- [Environment Variables](#environment-variables)
- [Notification Configuration](#notification-configuration)
- [Default Rules Analysis](#default-rules-analysis)
- [Performance Characteristics](#performance-characteristics)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)

## Overview

This configuration is specifically designed for **OpenShift Container Platform** deployment with the following production characteristics:

- **Container-Optimized**: Uses `0.0.0.0:8080` for container networking
- **Redis Cluster Mode**: Configured for Redis cluster deployment on OpenShift
- **Internal Service Discovery**: Uses OpenShift internal service addresses
- **Production Logging**: Structured logging with appropriate levels
- **Slack Integration**: Enterprise Slack webhook integration
- **Rate Limiting**: Built-in notification rate limiting to prevent spam
- **Resource Optimization**: Tuned batch sizes and timeouts for container environment

## Active Production Configuration

The current `config.yaml` represents the **live production configuration** used by the Alert Engine deployment on OpenShift:

### 🔧 Server Configuration
```yaml
server:
  address: "0.0.0.0:8080"
```
- **Container Networking**: Binds to all interfaces for OpenShift pod networking
- **Standard Port**: Uses port 8080 for HTTP traffic (OpenShift service routes traffic here)
- **Load Balancer Ready**: Accepts connections from OpenShift service load balancer

### 🔴 Redis Configuration
```yaml
redis:
  address: "redis-cluster-access.redis-cluster.svc.cluster.local:6379"
  cluster_mode: true
  max_retries: 3
  pool_size: 10
  dial_timeout: "10s"
  read_timeout: "5s"
  write_timeout: "5s"
```
- **Internal Service Address**: Uses OpenShift internal DNS for Redis cluster
- **Cluster Mode**: Configured for Redis cluster (high availability)
- **Connection Pooling**: Optimized pool settings for container resource limits
- **Timeout Handling**: Production-ready timeout values

### 📨 Kafka Configuration
```yaml
kafka:
  brokers:
    - "alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"
  topic: "application-logs"
  group_id: "alert-engine-group"
```
- **AMQ Streams Integration**: Uses Red Hat AMQ Streams (Kafka) on OpenShift
- **Internal Service Discovery**: Kafka bootstrap service internal address
- **Production Topic**: `application-logs` topic receives logs from Vector/ClusterLogForwarder
- **Consumer Group**: Unique group ID for alert engine instances

### 📱 Slack Integration
```yaml
slack:
  webhook_url: ""  # Set via SLACK_WEBHOOK_URL environment variable
  channel: "#test-mp-channel"
  username: "Alert Engine"
  icon_emoji: ":rotating_light:"
  timeout: "30s"
```
- **Environment Variable**: Webhook URL injected via OpenShift Secret
- **Default Channel**: Production test channel for alert notifications
- **Bot Identity**: Clear identification as Alert Engine
- **Timeout**: 30-second timeout for Slack API calls

## Infrastructure Dependencies

This configuration depends on the OpenShift infrastructure components set up according to `alert_engine_infra_setup.md`:

### ✅ Required Components
1. **Redis Cluster** (`redis-cluster` namespace)
   - Service: `redis-cluster-access.redis-cluster.svc.cluster.local:6379`
   - Mode: Cluster mode for high availability
   
2. **AMQ Streams Kafka** (`amq-streams-kafka` namespace)
   - Bootstrap: `alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092`
   - Topic: `application-logs`
   
3. **Vector/ClusterLogForwarder**
   - Forwards application logs to Kafka topic
   - Transforms logs into expected format

4. **OpenShift Secrets**
   - `alert-engine-secrets`: Contains `SLACK_WEBHOOK_URL`

## Production Settings Explained

### �� Log Processing
```yaml
log_processing:
  enabled: true
  batch_size: 50
  flush_interval: "10s"
```
- **Batch Processing**: Processes 50 logs at a time for efficiency
- **10-Second Intervals**: Balances latency vs throughput
- **Production Optimized**: Settings tested with high log volumes

### ⚠️ Alert Processing
```yaml
alerting:
  enabled: true
  batch_size: 50
  flush_interval: "10s"
  default_time_window: "5m"
  default_threshold: 3
```
- **Threshold Protection**: Default threshold of 3 prevents noise
- **5-Minute Windows**: Reasonable time window for production alerting
- **Batch Efficiency**: Matches log processing batch size

### 🔔 Notification Settings
```yaml
notifications:
  enabled: true
  slack:
    webhook_url: ""  # From environment
    channel: "#test-mp-channel"
    username: "Alert Engine"
    icon_emoji: ":rotating_light:"
    timeout: "30s"
```
- **Master Switch**: Can disable all notifications via config
- **Webhook Security**: URL never stored in config, only environment
- **Timeout Protection**: 30-second limit prevents hanging requests

### 📊 Monitoring Configuration
```yaml
monitoring:
  enabled: true
  log_level: "info"
```
- **Production Logging**: Info level balances detail vs volume
- **Health Endpoints**: `/health` and `/ready` for OpenShift health checks

## Environment Variables

The production deployment uses these environment variables (managed by OpenShift):

### 🔐 Required Secrets
```bash
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T027F3GAJ/B095UD6BECT/...
```
- **Source**: OpenShift Secret `alert-engine-secrets`
- **Key**: `slack-webhook-url` (base64 encoded)
- **Security**: Never stored in config files

### ⚙️ Configuration Override
```bash
CONFIG_PATH=./configs/config.yaml
```
- **Default Path**: Points to this production configuration
- **Override Capability**: Can point to custom config files

### 🏷️ Environment Detection
```bash
ENVIRONMENT=production
```
- **Deployment Context**: Identifies production vs development
- **Config Behavior**: May trigger environment-specific settings

## Notification Configuration

### 📢 Slack Webhook Setup
The production Slack integration uses:

1. **Enterprise Webhook**: `https://hooks.slack.com/services/T027F3GAJ/B095UD6BECT/...`
2. **Default Channel**: `#test-mp-channel` for production alerts
3. **Bot Identity**: "Alert Engine" with `:rotating_light:` emoji
4. **Timeout**: 30-second timeout for reliable delivery

### 🚨 Alert Template
```yaml
templates:
  alert_message: |
    🚨 **{{.Severity | upper}} ALERT** - {{.RuleName}}
    
    **Service:** {{.ServiceName}}
    **Namespace:** {{.Namespace}}
    **Time:** {{.Timestamp}}
    **Count:** {{.Count}} occurrences in {{.TimeWindow}}
    
    **Message:** {{.Message}}
```

### 🎨 Severity Configuration
```yaml
severity:
  emojis:
    low: "🟡"
    medium: "🟠"
    high: "🔴"
    critical: "🚨"
  colors:
    low: "#ffeb3b"
    medium: "#ff9800"
    high: "#f44336"
    critical: "#9c27b0"
```

## Default Rules Analysis

The production configuration includes one default rule for testing:

### 🧪 Test Rule
```yaml
default_rules:
  enabled: true
  rules:
    - id: "high-error-rate"
      name: "High Error Rate Detection"
      enabled: true
      conditions:
        log_level: "ERROR"
        threshold: 2
        time_window: "2m"
        operator: "gt"
      actions:
        channel: "#test-mp-channel"
        severity: "high"
```

**Purpose**: Validates end-to-end alert processing and Slack integration
**Conservative Settings**: 2 errors in 2 minutes prevents spam
**Test Channel**: Uses dedicated test channel for safety

## Performance Characteristics

### 📈 Throughput Metrics
Based on production testing:
- **Log Processing**: ~2,500 logs processed in complete e2e test
- **Alert Generation**: ~10 alerts generated from test patterns
- **Service Detection**: 14 different services identified
- **Processing Time**: Complete 21-test suite in ~2-3 minutes

### 💾 Resource Usage
Container resource requirements:
- **Memory**: ~100-200MB under normal load
- **CPU**: Minimal CPU usage with batch processing
- **Network**: Kafka and Redis connection pooling
- **Storage**: Temporary state only (no persistent storage)

### ⚡ Response Times
- **Log Ingestion**: <1 second per batch (50 logs)
- **Alert Evaluation**: <2 seconds per rule set
- **Slack Notifications**: <30 seconds per alert
- **Health Checks**: <100ms response time

## Security Considerations

### 🔒 Secrets Management
- **Webhook URLs**: Stored in OpenShift Secrets, never in config
- **API Keys**: Environment variable injection only
- **TLS**: Skip verification for internal OpenShift webhooks
- **Network**: Internal service communication only

### 🛡️ Access Control
- **Service Account**: Dedicated service account for Alert Engine
- **RBAC**: Minimal required permissions
- **Network Policies**: Restricted to required services only
- **Container Security**: Non-root container execution

### 🔐 Data Protection
- **Log Data**: Processed in memory, not persisted
- **Alert Rules**: Stored in Redis cluster
- **Temporary State**: TTL-based cleanup
- **Audit Trail**: All actions logged

## Troubleshooting

### 🔍 Common Issues

#### Slack Notifications Not Working
```bash
# Check webhook URL is set
oc get secret alert-engine-secrets -o yaml
echo "..." | base64 -d

# Check logs for errors
oc logs -f deployment/alert-engine | grep -i slack
```

#### Redis Connection Issues
```bash
# Test Redis connectivity
oc exec deployment/alert-engine -- wget -qO- redis-cluster-access.redis-cluster.svc.cluster.local:6379

# Check Redis cluster status
oc exec -n redis-cluster deployment/redis-cluster -- redis-cli cluster info
```

#### Kafka Consumer Lag
```bash
# Check Kafka connectivity
oc exec deployment/alert-engine -- wget -qO- alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092

# Monitor consumer group
oc exec -n amq-streams-kafka deployment/kafka-cluster -- bin/kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group alert-engine-group
```

### 📊 Performance Monitoring
```bash
# Monitor alert processing
oc logs -f deployment/alert-engine | grep "alerts generated"

# Check resource usage
oc top pod -l app=alert-engine

# Health check endpoints
oc port-forward svc/alert-engine 8080:8080
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### 🔧 Configuration Validation
```bash
# Validate config syntax
CONFIG_PATH=./configs/config.yaml go run cmd/server/main.go --validate-config

# Test rule loading
curl -X GET http://localhost:8080/api/rules
```

---

This configuration represents the **active production deployment** tested and validated on OpenShift with Redis cluster, AMQ Streams Kafka, and Slack integration. For deployment instructions, see `../deployments/alert-engine/README.md`.
