# Alert Engine - Phase 0: Foundation & Proof of Concept

A Go-based alert engine for monitoring application logs in OpenShift environments with real-time alerting via Slack.

## 🚀 Overview

The Alert Engine is a cloud-native solution designed to monitor application logs from OpenShift/Kubernetes environments, evaluate them against configurable alert rules, and send notifications to Slack channels. This implementation represents Phase 0 of a comprehensive log monitoring system.

### Key Features

- **Real-time Log Processing**: Consumes log messages from Kafka streams
- **Flexible Alert Rules**: Configurable rules based on log level, namespace, service, keywords, and thresholds
- **Slack Integration**: Rich notification messages with severity-based formatting
- **High Performance**: Redis-backed state management with horizontal scaling support
- **Cloud-Native**: Designed for OpenShift/Kubernetes with proper RBAC and security
- **RESTful API**: Full API for managing alert rules and monitoring system status

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐    ┌─────────────┐
│   OpenShift     │    │   AMQ        │    │   Alert         │    │   Slack     │
│   Pods/Logs     │───▶│   Streams    │───▶│   Engine        │───▶│   Webhook   │
│                 │    │   (Kafka)    │    │   (Go Service)  │    │             │
└─────────────────┘    └──────────────┘    └─────────────────┘    └─────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   OpenShift     │    │   Redis      │    │   REST API      │
│   Logging       │    │   (State)    │    │   (Management)  │
│   (Vector)      │    │              │    │                 │
└─────────────────┘    └──────────────┘    └─────────────────┘
```

## 🛠️ Technology Stack

- **Language**: Go 1.21+
- **Framework**: Gin (HTTP router)
- **Message Streaming**: Apache Kafka (Red Hat AMQ Streams)
- **State Storage**: Redis
- **Container Platform**: OpenShift 4.12+
- **Notifications**: Slack Webhooks
- **Monitoring**: Prometheus metrics

## 📁 Project Structure

```
alert-engine/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── alerting/
│   │   ├── engine.go              # Main alert evaluation engine
│   │   ├── rules.go               # Rule management and validation
│   │   └── evaluator.go           # Rule evaluation logic
│   ├── kafka/
│   │   ├── consumer.go            # Kafka consumer implementation
│   │   └── processor.go           # Log message processing
│   ├── notifications/
│   │   ├── interfaces.go          # Notification interfaces
│   │   └── slack.go              # Slack integration
│   ├── storage/
│   │   └── redis.go              # Redis storage implementation
│   └── api/
│       ├── handlers.go           # HTTP API handlers
│       └── routes.go             # API route definitions
├── pkg/
│   └── models/
│       ├── alert.go              # Alert rule models
│       └── log.go                # Log entry models
├── configs/
│   └── config.yaml               # Application configuration
├── deployments/
│   └── openshift/
│       ├── deployment.yaml       # OpenShift deployment manifest
│       └── service.yaml          # OpenShift service manifest
├── go.mod                        # Go module definition
└── README.md                     # This file
```

## 🚦 Prerequisites

- Go 1.21 or later
- Access to OpenShift/Kubernetes cluster
- Redis instance
- Kafka cluster (Red Hat AMQ Streams)
- Slack workspace with webhook permissions

### 📋 Infrastructure Setup

**IMPORTANT**: Before proceeding with the Alert Engine setup, you must first install and configure the required infrastructure components on your OpenShift cluster.

👉 **[OpenShift Infrastructure Setup Guide](OPENSHIFT_SETUP.md)**

Key infrastructure components to install (15-20 minutes total):
- **Red Hat AMQ Streams**: Install operator and deploy 3-node Kafka cluster with `application-logs` topic
- **Redis Enterprise**: Install operator and create database with ReJSON/TimeSeries modules for state management
- **OpenShift Logging**: Install operator and configure ClusterLogForwarder to route application logs to Kafka
- **RBAC & Security**: Create service accounts, role bindings, and network policies for secure log collection
- **Verification**: Test connectivity between components and validate log forwarding pipeline

**Complete the infrastructure setup before proceeding with the local development or deployment steps below.**

## 🔧 Setup Instructions [To be Updated]

### 1. Local Development Setup

```bash
# Clone the repository
git clone <repository-url>
cd alert-engine

# Initialize Go module dependencies
go mod tidy

# Run locally (requires Redis and Kafka)
go run cmd/server/main.go
```

### 2. Configuration

#### Environment Variables

```bash
export SERVER_ADDRESS=":8080"
export REDIS_ADDRESS="localhost:6379"
export REDIS_PASSWORD=""
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="application-logs"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
export LOG_LEVEL="info"
```

#### Configuration File

Edit `configs/config.yaml` to customize the application settings:

```yaml
server:
  address: ":8080"

redis:
  address: "localhost:6379"
  password: ""

kafka:
  brokers: ["localhost:9092"]
  topic: "application-logs"

slack:
  webhook_url: ""
  channel: "#alerts"
```


## 📚 API Documentation

### Base URL

- Local: `http://localhost:8080`
- OpenShift: `https://alert-engine-log-monitoring.apps.your-cluster.com`

### Endpoints

#### Health Check
```bash
GET /api/v1/health
```

#### Alert Rules Management

```bash
# Get all rules
GET /api/v1/rules

# Create new rule
POST /api/v1/rules
Content-Type: application/json

{
  "name": "Database Error Alert",
  "description": "Alert on database connection errors",
  "enabled": true,
  "conditions": {
    "log_level": "ERROR",
    "keywords": ["database", "connection"],
    "threshold": 5,
    "time_window": "5m",
    "operator": "gt"
  },
  "actions": {
    "channel": "#infrastructure",
    "severity": "high"
  }
}

# Get specific rule
GET /api/v1/rules/{id}

# Update rule
PUT /api/v1/rules/{id}

# Delete rule
DELETE /api/v1/rules/{id}

# Get rule statistics
GET /api/v1/rules/stats

# Test rule
POST /api/v1/rules/test
```

#### System Monitoring

```bash
# Get recent alerts
GET /api/v1/alerts/recent?limit=50

# Get system metrics
GET /api/v1/system/metrics

# Get log processing statistics
GET /api/v1/system/logs/stats
```

### API Response Format

```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": { ... },
  "error": null
}
```

## 🎯 Usage Examples

### Creating Alert Rules

#### Basic Error Alert
```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Application Errors",
    "description": "Alert on application errors",
    "enabled": true,
    "conditions": {
      "log_level": "ERROR",
      "threshold": 10,
      "time_window": "5m"
    },
    "actions": {
      "channel": "#alerts",
      "severity": "high"
    }
  }'
```

#### Service-Specific Alert
```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Payment Service Issues",
    "description": "Monitor payment service for errors",
    "enabled": true,
    "conditions": {
      "log_level": "ERROR",
      "namespace": "payment-service",
      "service": "payment-api",
      "keywords": ["timeout", "failed"],
      "threshold": 3,
      "time_window": "2m"
    },
    "actions": {
      "channel": "#payments",
      "severity": "critical"
    }
  }'
```

### Monitoring System Health

```bash
# Check system health
curl http://localhost:8080/api/v1/health

# Get processing metrics
curl http://localhost:8080/api/v1/system/metrics

# View recent alerts
curl http://localhost:8080/api/v1/alerts/recent?limit=10
```

## 🔍 Monitoring & Observability

### Metrics

The application exposes Prometheus metrics on port 8081:

- `alert_rules_total`: Total number of alert rules
- `alerts_triggered_total`: Total number of alerts triggered
- `kafka_messages_processed_total`: Total Kafka messages processed
- `redis_operations_total`: Total Redis operations

### Logging

Structured JSON logs are written to stdout with configurable levels:
- `DEBUG`: Detailed debugging information
- `INFO`: General operational messages
- `WARN`: Warning conditions
- `ERROR`: Error conditions

### Health Checks

- **Liveness Probe**: `/api/v1/health`
- **Readiness Probe**: `/api/v1/health`

## 🛡️ Security Considerations

### Authentication & Authorization

- Service Account: `alert-engine`
- RBAC: Minimal permissions for ConfigMaps, Secrets, and Pods
- Network Policies: Restricted ingress/egress traffic

### Secret Management

- Slack webhook URLs stored in Kubernetes Secrets
- Redis passwords stored in Kubernetes Secrets
- Environment variable injection from Secrets

### Container Security

- Non-root user execution (UID 1001)
- Read-only filesystem
- Security context constraints
- Resource limits and requests

## 📈 Performance & Scaling

### Horizontal Pod Autoscaler

The deployment includes HPA configuration:
- Min replicas: 3
- Max replicas: 10
- CPU threshold: 70%
- Memory threshold: 80%

### Resource Requirements

**Requests:**
- Memory: 256Mi
- CPU: 100m

**Limits:**
- Memory: 512Mi
- CPU: 500m

### Optimization Tips

1. **Batch Processing**: Configure appropriate batch sizes for log processing
2. **Redis Connection Pooling**: Tune Redis connection pool settings
3. **Kafka Consumer Groups**: Use multiple consumer instances for high throughput
4. **Rule Optimization**: Minimize complex keyword matching

## 🐛 Troubleshooting

### Common Issues

#### 1. Connection Issues

```bash
# Check Redis connectivity
oc exec -it deployment/alert-engine -n log-monitoring -- sh
# Inside container:
redis-cli -h redis -p 6379 ping

# Check Kafka connectivity
oc get kafka -n log-monitoring
```

#### 2. Alert Rules Not Triggering

```bash
# Check rule configuration
curl http://localhost:8080/api/v1/rules

# Verify log processing
curl http://localhost:8080/api/v1/system/logs/stats

# Check recent alerts
curl http://localhost:8080/api/v1/alerts/recent
```

#### 3. Slack Notifications Not Working

```bash
# Verify webhook URL
oc get secret slack-secret -n log-monitoring -o yaml

# Test webhook manually
curl -X POST https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK \
  -H "Content-Type: application/json" \
  -d '{"text": "Test message"}'
```

### Log Analysis

```bash
# View application logs
oc logs -f deployment/alert-engine -n log-monitoring

# Filter error logs
oc logs deployment/alert-engine -n log-monitoring | grep ERROR

# Monitor in real-time
oc logs -f deployment/alert-engine -n log-monitoring --since=1h
```

## 🔄 Development Workflow

### Local Development

1. Start dependencies (Redis, Kafka)
2. Set environment variables
3. Run application: `go run cmd/server/main.go`
4. Test with curl or Postman

### Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Test specific package
go test ./internal/alerting
```

### Building

```bash
# Build binary
go build -o bin/alert-engine cmd/server/main.go

# Build container image
podman build -t alert-engine:latest .
```

## 📋 TODO: Future Enhancements

- [ ] Email notification support
- [ ] Web UI for rule management
- [ ] Advanced rule templates
- [ ] Machine learning-based anomaly detection
- [ ] Integration with external ticketing systems
- [ ] Multi-tenant support
- [ ] Historical alert analysis

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Make changes and test thoroughly
4. Commit changes: `git commit -am 'Add new feature'`
5. Push to branch: `git push origin feature/new-feature`
6. Submit a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:
- Create an issue in the repository
- Contact the development team
- Check the troubleshooting section above

---

**Alert Engine v1.0.0** - Phase 0: Foundation & Proof of Concept 