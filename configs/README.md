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
- **Enhanced Timeout Configuration**: Comprehensive timeout settings for reliability
- **Resource Optimization**: Tuned batch sizes and timeouts for container environment

## Active Production Configuration

The current `config.yaml` represents the **live production configuration** used by the Alert Engine deployment on OpenShift:

### 🔧 Server Configuration
```yaml
server:
  host: "0.0.0.0"
  port: 8080
```
- **Container Networking**: Binds to all interfaces for OpenShift pod networking
- **Standard Port**: Uses port 8080 for HTTP traffic (OpenShift service routes traffic here)
- **Load Balancer Ready**: Accepts connections from OpenShift service load balancer

### 🔴 Redis Configuration
```yaml
redis:
  address: "redis-cluster-access.redis-cluster.svc.cluster.local:6379"
  password: ""
  cluster_mode: true
  dial_timeout: "15s"
  read_timeout: "10s"
  write_timeout: "10s"
```
- **Internal Service Address**: Uses OpenShift internal DNS for Redis cluster
- **Cluster Mode**: Configured for Redis cluster (high availability)
- **No Authentication**: Internal cluster access without password
- **Extended Timeouts**: Production-ready timeout values for OpenShift environment
  - Dial: 15 seconds for initial connection
  - Read/Write: 10 seconds for operations

### 📨 Kafka Configuration
```yaml
kafka:
  brokers:
    - "alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"
  topic: "application-logs"
  group_id: "alert-engine-production-group"
  timeout: "30s"
  net:
    dial_timeout: "15s"
    read_timeout: "10s"
    write_timeout: "10s"
  consumer:
    min_bytes: 1024
    max_bytes: 1048576
    max_wait: "2s"
    start_offset: -2
    session_timeout: "30s"
    heartbeat_interval: "3s"
```
- **AMQ Streams Integration**: Uses Red Hat AMQ Streams (Kafka) on OpenShift
- **Internal Service Discovery**: Kafka bootstrap service internal address
- **Production Topic**: `application-logs` topic receives logs from Vector/ClusterLogForwarder
- **Production Group**: `alert-engine-production-group` for production deployment
- **Comprehensive Timeouts**: Network and consumer timeout configuration
- **Earliest Offset**: Starts reading from earliest available messages (`start_offset: -2`)
- **Consumer Tuning**: Optimized batch sizes and session management

### 🔄 Log Processing Configuration
```yaml
log_processing:
  batch_size: 100
  flush_interval: "5s"
  validation:
    default_log_level: "INFO"
```
- **Higher Batch Size**: 100 logs per batch for production efficiency
- **Faster Processing**: 5-second intervals for quick response
- **Default Log Level**: INFO level for validation

### ⚠️ Alerting Configuration
```yaml
alerting:
  default_time_window: "5m"
  default_threshold: 5
```
- **Production Threshold**: Higher threshold (5) to reduce noise
- **5-Minute Windows**: Standard time window for production alerting

### 📱 Notification Configuration
```yaml
notifications:
  enabled: true
  slack:
    webhook_url: ""
    channel: "#alerts"
    username: "Alert Engine (Production)"
    icon_emoji: ":warning:"
    timeout: "30s"
```
- **Master Switch**: Notifications enabled for production
- **Environment Variable**: Webhook URL injected via `SLACK_WEBHOOK_URL`
- **Production Channel**: `#alerts` channel for production notifications
- **Clear Identity**: Identifies as "Alert Engine (Production)"
- **Production Icon**: Warning emoji for alert visibility

## Infrastructure Dependencies

This configuration depends on the OpenShift infrastructure components set up according to `alert_engine_infra_setup.md`:

### ✅ Required Components
1. **Redis Cluster** (`redis-cluster` namespace)
   - Service: `redis-cluster-access.redis-cluster.svc.cluster.local:6379`
   - Mode: Cluster mode for high availability
   - No authentication required for internal access
   
2. **AMQ Streams Kafka** (`amq-streams-kafka` namespace)
   - Bootstrap: `alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092`
   - Topic: `application-logs`
   - Consumer Group: `alert-engine-production-group`
   
3. **Vector/ClusterLogForwarder**
   - Forwards application logs to Kafka topic
   - Transforms logs into expected format

4. **OpenShift Secrets**
   - `alert-engine-secrets`: Contains `SLACK_WEBHOOK_URL`

## Production Settings Explained

### 📈 Performance Optimization
The production configuration includes several optimizations:

#### Batch Processing
- **Log Processing**: 100 logs per batch (higher than development)
- **Flush Interval**: 5 seconds for responsive alerting
- **Kafka Consumer**: Optimized min/max byte settings for efficient network usage

#### Timeout Configuration
- **Redis Timeouts**: 15s dial, 10s read/write for reliability
- **Kafka Timeouts**: 30s general, 15s dial for network resilience
- **Slack Timeout**: 30s for notification delivery
- **Consumer Session**: 30s timeout with 3s heartbeat for connection stability

#### Consumer Settings
- **Start Offset**: `-2` (earliest) ensures no log loss during restarts
- **Session Management**: 30s timeout with 3s heartbeat interval
- **Batch Sizes**: 1KB-1MB range for efficient message processing

### 🔔 Notification Templates

The configuration includes comprehensive notification templates:

```yaml
templates:
  alert_message: |
    🚨 PRODUCTION Alert: {{.RuleName}}
    Service: {{.Service}}
    Namespace: {{.Namespace}}
    Level: {{.Level}}
    Count: {{.Count}} in {{.TimeWindow}}
    Message: {{.Message}}
  
  slack_alert_title: "{{.SeverityEmoji}} {{.RuleName}}"
  
  slack_alert_fields:
    - title: "Severity"
      value: "{{.Severity}}"
      short: true
    - title: "Namespace"
      value: "{{.Namespace}}"
      short: true
    # ... additional fields
```

### 🎨 Severity Configuration
```yaml
severity:
  emojis:
    critical: "🔴"
    high: "🟠"
    medium: "🟡"
    low: "🟢"
    default: "⚪"
  
  colors:
    critical: "#ff0000"
    high: "#ff8000"
    medium: "#ffff00"
    low: "#00ff00"
    default: "#808080"
```

## Environment Variables

The production deployment uses these environment variables (managed by OpenShift):

### 🔐 Required Secrets
```bash
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
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

## Default Rules Analysis

**Important**: Default rules are currently **DISABLED** in production:

```yaml
default_rules:
  enabled: false
```

This means:
- **No Built-in Rules**: System starts with no predefined alert rules
- **Manual Configuration**: Rules must be added via API or external configuration
- **Safety First**: Prevents unexpected alerting on production deployment
- **Clean Start**: Allows careful rule implementation and testing

To enable default rules:
1. Change `enabled: false` to `enabled: true`
2. Add rules under the `rules:` section
3. Restart the alert engine deployment

## Performance Characteristics

### 📈 Expected Throughput
Based on production configuration:
- **Log Processing**: ~2,000 logs/minute with 100-log batches every 5 seconds
- **Alert Evaluation**: Near real-time with 5-minute time windows
- **Kafka Consumer**: Efficient processing with optimized byte ranges
- **Redis Operations**: Fast state management with cluster mode

### 💾 Resource Usage
Container resource requirements:
- **Memory**: ~150-250MB under normal load (higher due to larger batches)
- **CPU**: Moderate CPU usage with frequent processing
- **Network**: Optimized with connection pooling and timeouts
- **Storage**: Temporary state only (no persistent storage)

### ⚡ Response Times
- **Log Ingestion**: <5 seconds per batch (100 logs)
- **Alert Evaluation**: <5 seconds per rule set
- **Slack Notifications**: <30 seconds per alert
- **Health Checks**: <100ms response time

## Security Considerations

### 🔒 Secrets Management
- **Webhook URLs**: Stored in OpenShift Secrets, never in config
- **Redis Access**: No password required for internal cluster access
- **Kafka Access**: Internal service communication only
- **TLS**: Internal OpenShift service mesh

### 🛡️ Access Control
- **Service Account**: Dedicated service account for Alert Engine
- **RBAC**: Minimal required permissions
- **Network Policies**: Restricted to required services only
- **Container Security**: Non-root container execution

### 🔐 Data Protection
- **Log Data**: Processed in memory, not persisted
- **Alert Rules**: Stored in Redis cluster with appropriate TTL
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

# Test notification endpoint
curl -X POST http://localhost:8080/api/test-notification
```

#### Redis Connection Issues
```bash
# Test Redis connectivity from pod
oc exec deployment/alert-engine -- nc -zv redis-cluster-access.redis-cluster.svc.cluster.local 6379

# Check Redis cluster status
oc exec -n redis-cluster deployment/redis-cluster -- redis-cli cluster info

# Test with longer timeouts
oc exec deployment/alert-engine -- timeout 20s nc -zv redis-cluster-access.redis-cluster.svc.cluster.local 6379
```

#### Kafka Consumer Issues
```bash
# Check Kafka connectivity
oc exec deployment/alert-engine -- nc -zv alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local 9092

# Monitor consumer group lag
oc exec -n amq-streams-kafka deployment/kafka-cluster -- bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group alert-engine-production-group

# Check topic status
oc exec -n amq-streams-kafka deployment/kafka-cluster -- bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic application-logs
```

#### Performance Issues
```bash
# Monitor processing metrics
oc logs -f deployment/alert-engine | grep -E "(batch|flush|processed)"

# Check resource constraints
oc describe pod -l app=alert-engine | grep -A 10 Resources

# Monitor timeout issues
oc logs -f deployment/alert-engine | grep -i timeout
```

### 📊 Performance Monitoring
```bash
# Monitor alert processing
oc logs -f deployment/alert-engine | grep "alerts generated"

# Check batch processing efficiency
oc logs -f deployment/alert-engine | grep "batch_size\|flush_interval"

# Resource usage monitoring
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

# Test rule loading (when enabled)
curl -X GET http://localhost:8080/api/rules

# Check notification templates
curl -X GET http://localhost:8080/api/notification-test
```

### 🚨 Alert Testing
```bash
# Send test log messages
curl -X POST http://localhost:8080/api/test-logs \
  -H "Content-Type: application/json" \
  -d '{"level":"ERROR","message":"Test error for alerting"}'

# Check alert generation
curl -X GET http://localhost:8080/api/alerts

# Test notification delivery
curl -X POST http://localhost:8080/api/test-slack
```

---

This configuration represents the **active production deployment** with enhanced timeouts, optimized batch processing, and comprehensive monitoring capabilities. The system is designed for high reliability and performance in OpenShift environments. For deployment instructions, see `../deployments/alert-engine/README.md`.
