# Alert Engine Configuration Guide

This directory contains the configuration files for the Alert Engine. The main configuration file is `config.yaml`, which defines all the settings needed to run the Alert Engine in different environments.

## 📋 Table of Contents

- [Overview](#overview)
- [Configuration Structure](#configuration-structure)
- [Server Configuration](#server-configuration)
- [Redis Configuration](#redis-configuration)
- [Kafka Configuration](#kafka-configuration)
- [Slack Notification Configuration](#slack-notification-configuration)
- [Notification Settings](#notification-settings)
- [Alert Processing Configuration](#alert-processing-configuration)
- [Log Processing Configuration](#log-processing-configuration)
- [Monitoring and Metrics](#monitoring-and-metrics)
- [Default Alert Rules](#default-alert-rules)
- [Security Configuration](#security-configuration)
- [Performance Tuning](#performance-tuning)
- [Logging Configuration](#logging-configuration)
- [Environment-Specific Overrides](#environment-specific-overrides)
- [Usage Examples](#usage-examples)
- [Best Practices](#best-practices)

## Overview

The Alert Engine configuration system supports multiple deployment environments:
- **Local Development**: Optimized for local testing with port-forwarding to OpenShift
- **OpenShift Production**: Full cluster deployment with internal service discovery
- **Test Environment**: Isolated testing with dedicated resources

The configuration uses YAML format and supports environment variable substitution for sensitive values.

## Configuration Structure

The configuration is organized into logical sections:

```yaml
# config.yaml structure
server:           # HTTP server settings
redis:            # Redis connection and pool settings
kafka:            # Kafka broker and consumer/producer settings
slack:            # Slack notification configuration
notifications:    # General notification behavior
alerting:         # Alert processing rules and thresholds
log_processing:   # Log ingestion and validation
monitoring:       # Metrics and health check settings
default_rules:    # Pre-configured alert rules
security:         # Authentication and CORS settings
performance:      # Resource limits and optimization
logging:          # Application logging configuration
env_overrides:    # Environment-specific overrides
```

## Server Configuration

Controls the HTTP server that hosts the Alert Engine API.

```yaml
server:
  address: ":8080"           # Server bind address and port
  read_timeout: "30s"        # Maximum time to read request
  write_timeout: "30s"       # Maximum time to write response
  idle_timeout: "60s"        # Maximum idle time for connections
```

**Parameters:**
- `address`: Server listening address (`:8080` means all interfaces on port 8080)
- `read_timeout`: Prevents slow client attacks by limiting request read time
- `write_timeout`: Limits response write time to prevent resource exhaustion
- `idle_timeout`: Keep-alive timeout for HTTP connections

## Redis Configuration

Configures the Redis connection for storing alert rules, state, and caching.

```yaml
redis:
  address: "localhost:6379"     # Redis server address
  password: ""                  # Redis password (empty for no auth)
  database: 0                   # Redis database number (0-15)
  cluster_mode: false           # Enable Redis cluster mode
  max_retries: 3                # Maximum retry attempts
  pool_size: 10                 # Connection pool size
  min_idle_conns: 5             # Minimum idle connections
  dial_timeout: "10s"           # Connection establishment timeout
  read_timeout: "5s"            # Read operation timeout
  write_timeout: "5s"           # Write operation timeout
  pool_timeout: "10s"           # Pool acquisition timeout
  idle_timeout: "5m"            # Idle connection timeout
```

**Parameters:**
- `address`: Redis server endpoint
- `cluster_mode`: Set to `true` for Redis cluster deployments
- `pool_size`: Maximum number of connections in the pool
- `min_idle_conns`: Minimum connections kept open for faster access
- **Timeout Settings**: All timeout values are increased for port-forwarding scenarios

**Local vs Production:**
- **Local**: Uses `localhost:6379` with port-forwarding, `cluster_mode: false`
- **Production**: Uses internal service address, `cluster_mode: true`

## Kafka Configuration

Configures Kafka integration for consuming log messages and producing alerts.

```yaml
kafka:
  brokers:
    - "localhost:9092"              # Kafka broker addresses
  topic: "application-logs"         # Topic to consume logs from
  group_id: "alert-engine-local-group"  # Consumer group ID
  consumer:
    min_bytes: 1024                 # Minimum bytes per fetch
    max_bytes: 1048576             # Maximum bytes per fetch (1MB)
    max_wait: "2s"                 # Maximum wait time for batch
    start_offset: -1               # Starting offset (-1 = latest)
  producer:
    batch_size: 50                 # Messages per batch
    batch_timeout: "2s"            # Batch timeout
```

**Parameters:**
- `brokers`: List of Kafka broker addresses
- `topic`: Kafka topic containing application logs
- `group_id`: Consumer group identifier (should be unique per environment)
- `consumer.min_bytes`: Minimum data per fetch request (improves batching)
- `consumer.max_bytes`: Maximum data per fetch request (prevents memory issues)
- `consumer.start_offset`: `-1` (latest), `0` (earliest), or specific offset
- `producer.batch_size`: Number of messages to batch together
- `producer.batch_timeout`: Maximum time to wait for batch completion

## Slack Notification Configuration

Configures Slack integration for sending alert notifications.

```yaml
slack:
  webhook_url: ""                    # Slack webhook URL (set via env var)
  channel: "#test-mp-channel"        # Default Slack channel
  username: "Alert Engine (Local)"   # Bot username
  icon_emoji: ":warning:"            # Bot icon emoji
  timeout: "30s"                     # Notification timeout
```

**Parameters:**
- `webhook_url`: Slack webhook URL (should be set via `SLACK_WEBHOOK_URL` environment variable)
- `channel`: Default channel for notifications (can be overridden per alert rule)
- `username`: Display name for the bot in Slack
- `icon_emoji`: Emoji icon for the bot
- `timeout`: Maximum time to wait for Slack API response

**Security Note:** The webhook URL should never be hardcoded in the config file. Use the `SLACK_WEBHOOK_URL` environment variable.

## Notification Settings

Controls general notification behavior and rate limiting.

```yaml
notifications:
  enabled: true                      # Enable/disable notifications
  max_retries: 3                     # Maximum retry attempts
  retry_delay: "5s"                  # Delay between retries
  timeout: "30s"                     # Notification timeout
  rate_limit_per_min: 60             # Maximum notifications per minute
  batch_size: 5                      # Notifications per batch
  batch_delay: "2s"                  # Delay between batches
  enable_deduplication: true         # Enable duplicate detection
  deduplication_window: "5m"         # Deduplication time window
```

**Parameters:**
- `enabled`: Master switch for all notifications
- `max_retries`: Number of retry attempts for failed notifications
- `retry_delay`: Exponential backoff delay between retries
- `rate_limit_per_min`: Prevents notification spam
- `batch_size`: Groups notifications to reduce API calls
- `enable_deduplication`: Prevents duplicate alerts within the time window
- `deduplication_window`: Time period for duplicate detection

## Alert Processing Configuration

Controls how alerts are processed and evaluated.

```yaml
alerting:
  enabled: true                      # Enable alert processing
  batch_size: 50                     # Logs processed per batch
  flush_interval: "10s"              # Batch processing interval
  max_rules: 1000                    # Maximum number of alert rules
  default_time_window: "5m"          # Default evaluation window
  default_threshold: 3               # Default threshold count
  cleanup_interval: "1h"             # State cleanup interval
```

**Parameters:**
- `enabled`: Master switch for alert processing
- `batch_size`: Number of logs processed together (affects memory usage)
- `flush_interval`: How often to process accumulated logs
- `max_rules`: Limit on total alert rules (prevents resource exhaustion)
- `default_time_window`: Default time window for rule evaluation
- `default_threshold`: Default threshold if not specified in rule
- `cleanup_interval`: How often to clean up expired alert states

## Log Processing Configuration

Controls log ingestion and validation.

```yaml
log_processing:
  enabled: true                      # Enable log processing
  batch_size: 50                     # Logs processed per batch
  flush_interval: "10s"              # Processing interval
  max_message_size: 1048576          # Maximum log message size (1MB)
  validation:
    require_timestamp: true          # Require timestamp field
    require_level: true              # Require log level field
    require_message: true            # Require message field
    require_namespace: true          # Require namespace field
```

**Parameters:**
- `enabled`: Master switch for log processing
- `batch_size`: Logs processed together (memory vs latency trade-off)
- `flush_interval`: Processing frequency
- `max_message_size`: Prevents oversized messages from consuming memory
- `validation.*`: Required fields for log messages

**Validation Rules:**
- `require_timestamp`: Logs must have a valid timestamp
- `require_level`: Logs must specify a log level (DEBUG, INFO, WARN, ERROR, FATAL)
- `require_message`: Logs must contain a message field
- `require_namespace`: Logs must specify a Kubernetes namespace

## Monitoring and Metrics

Configures health checks, metrics, and profiling.

```yaml
monitoring:
  enabled: true                      # Enable monitoring
  metrics_port: 8081                 # Metrics server port
  health_check_interval: "30s"       # Health check frequency
  log_level: "debug"                 # Application log level
  enable_pprof: true                 # Enable Go profiling
```

**Parameters:**
- `enabled`: Master switch for monitoring features
- `metrics_port`: Port for Prometheus metrics and health endpoints
- `health_check_interval`: How often to run internal health checks
- `log_level`: Application logging level (debug, info, warn, error, fatal)
- `enable_pprof`: Enables Go profiling endpoints (development only)

**Metrics Endpoints:**
- `http://localhost:8081/metrics` - Prometheus metrics
- `http://localhost:8081/health` - Health check
- `http://localhost:8081/debug/pprof/` - Go profiling (if enabled)

## Default Alert Rules

Pre-configured alert rules for common scenarios.

```yaml
default_rules:
  enabled: true                      # Enable default rules
  rules:
    - id: "local-test-error-alert"
      name: "Local Test - Application Error Alert"
      description: "Alert on application errors (local testing)"
      enabled: true
      conditions:
        log_level: "ERROR"
        threshold: 2
        time_window: "2m"
        operator: "gt"
      actions:
        channel: "#alerts"
        severity: "high"
```

**Rule Structure:**
- `id`: Unique identifier for the rule
- `name`: Human-readable name
- `description`: Rule purpose and context
- `enabled`: Enable/disable this specific rule
- `conditions`: When to trigger the alert
- `actions`: What to do when triggered

**Condition Parameters:**
- `log_level`: Log level to match (DEBUG, INFO, WARN, ERROR, FATAL)
- `namespace`: Kubernetes namespace to match
- `service`: Service name to match
- `threshold`: Number of matching logs required
- `time_window`: Time period for threshold evaluation
- `operator`: Comparison operator (gt, gte, lt, lte, eq)

## Security Configuration

Controls authentication and CORS settings.

```yaml
security:
  enable_cors: true                  # Enable CORS headers
  cors_origins: ["*"]                # Allowed CORS origins
  enable_auth: false                 # Enable API authentication
  api_key: ""                        # API key for authentication
```

**Parameters:**
- `enable_cors`: Required for web UI access
- `cors_origins`: List of allowed origins (`["*"]` allows all)
- `enable_auth`: Enable API key authentication
- `api_key`: API key for authenticated requests

**Security Notes:**
- In production, use specific origins instead of `["*"]`
- Enable authentication for production deployments
- API keys should be set via environment variables

## Performance Tuning

Controls resource limits and optimization settings.

```yaml
performance:
  max_concurrent_rules: 50           # Maximum rules evaluated concurrently
  rule_evaluation_timeout: "2s"      # Timeout for rule evaluation
  max_memory_usage: "256Mi"          # Memory usage limit
  gc_percentage: 100                 # Go garbage collection target
```

**Parameters:**
- `max_concurrent_rules`: Limits CPU usage during rule evaluation
- `rule_evaluation_timeout`: Prevents slow rules from blocking others
- `max_memory_usage`: Kubernetes-style memory limit
- `gc_percentage`: Go GC target percentage (100 = default)

**Tuning Guidelines:**
- **Development**: Lower limits for resource-constrained environments
- **Production**: Higher limits for better performance
- **Memory-constrained**: Reduce batch sizes and concurrent rules
- **CPU-constrained**: Increase timeouts and reduce concurrency

## Logging Configuration

Controls application logging output and file rotation.

```yaml
logging:
  level: "debug"                     # Log level
  format: "text"                     # Log format (text, json)
  output: "stdout"                   # Output destination
  file:
    path: "/tmp/alert-engine-local.log"
    max_size: "10MB"                 # Log file size limit
    max_backups: 3                   # Number of backup files
    max_age: "7d"                    # Maximum file age
    compress: true                   # Compress rotated files
```

**Parameters:**
- `level`: Minimum log level to output
- `format`: Log format (`text` for development, `json` for production)
- `output`: Where to write logs (`stdout`, `stderr`, `file`)
- `file.path`: Log file location
- `file.max_size`: File size before rotation
- `file.max_backups`: Number of rotated files to keep
- `file.max_age`: Maximum age of log files
- `file.compress`: Compress rotated files to save space

## Environment-Specific Overrides

Allows different settings per deployment environment.

```yaml
env_overrides:
  development:
    logging:
      level: "debug"
    monitoring:
      enable_pprof: true
    alerting:
      default_threshold: 1           # Very sensitive for testing
  
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

**Environment Types:**
- **development**: Local development settings
- **production**: Production deployment settings
- **test**: Test environment settings

**Override Behavior:**
- Settings in `env_overrides` take precedence over base configuration
- Environment is selected via `ENVIRONMENT` environment variable
- Nested configuration is merged (not replaced)

## Usage Examples

### Setting Environment Variables

```bash
# Set environment for local development
export ENVIRONMENT=development
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
export CONFIG_PATH="./configs/config.yaml"

# Start the Alert Engine
./alert-engine
```

### Custom Configuration

```yaml
# custom-config.yaml
# Override specific settings
server:
  address: ":9090"  # Use different port

kafka:
  brokers:
    - "my-kafka-broker:9092"

notifications:
  rate_limit_per_min: 30  # Reduce notification rate
```

### Production Deployment

```yaml
# production-config.yaml
env_overrides:
  production:
    security:
      enable_auth: true
      cors_origins: ["https://my-dashboard.company.com"]
    performance:
      max_memory_usage: "2Gi"
      max_concurrent_rules: 200
    logging:
      level: "warn"
      format: "json"
```

## Best Practices

### Security
- ✅ Use environment variables for sensitive data (webhook URLs, API keys)
- ✅ Enable authentication in production
- ✅ Use specific CORS origins instead of `["*"]`
- ✅ Set appropriate timeout values to prevent resource exhaustion

### Performance
- ✅ Tune batch sizes based on your log volume
- ✅ Adjust timeouts for your network conditions
- ✅ Monitor memory usage and adjust limits accordingly
- ✅ Use appropriate log levels (debug for development, warn for production)

### Reliability
- ✅ Configure appropriate retry policies
- ✅ Set up health check monitoring
- ✅ Use rate limiting to prevent notification spam
- ✅ Enable deduplication to reduce noise

### Monitoring
- ✅ Enable metrics collection
- ✅ Set up log rotation to prevent disk space issues
- ✅ Monitor alert processing latency
- ✅ Track notification delivery success rates

### Environment Management
- ✅ Use environment-specific overrides
- ✅ Test configuration changes in development first
- ✅ Validate configuration before deployment
- ✅ Document any custom settings or modifications

---

For more information about deploying and running the Alert Engine, see the [LOCAL_SETUP_GUIDE.md](../local/LOCAL_SETUP_GUIDE.md). 