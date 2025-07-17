# NLP based Alert Engine - Phase 0: Foundation & Proof of Concept
### An ALERT SYSTEM been developed using Cursor
A Go-based alert engine for monitoring application logs in OpenShift environments with real-time alerting via Slack.

## Scope

**Goal:** Validate the concept with minimal viable alerting

### Components to Build
- **Simple Log Ingestion** - OpenShift Logging Vector + Kafka pipeline
- **Basic Alert Engine** - Simple rule-based alerting (no NLP engine)
- **Single notification channel** - Slack integration only
- **Minimal UI** - Command Line or simple web form for alert creation

### Deliverables
- Working log pipeline from OpenShift pods to Kafka
- Basic threshold-based alerts (count, keyword matching)
- Slack notification working
- Single hard-coded alert rule validation

### Success Criteria
- Can detect "ERROR" logs exceeding count threshold
- Can send Slack notification within 30 seconds
- No data loss in log pipeline

## Vision

For the long-term vision and NLP-based alert pattern analysis that will guide future development phases, refer to:

**🧠 [NLP Alert Patterns Analysis](inputs/nlp_alert_patterns.md)** - Comprehensive analysis of natural language processing patterns for intelligent log monitoring and advanced alert detection capabilities.

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
├── bin/                           # Binary executables
├── cleanup/                       # Infrastructure cleanup scripts
│   ├── cleanup_openshift_infrastructure.sh
│   └── verify_resources_before_cleanup.sh
├── cmd/                          # Application entry points
├── configs/                      # Configuration files
│   ├── config.yaml              # Main application configuration
│   └── README.md                # Configuration documentation
├── deployments/                 # Deployment manifests
│   └── openshift/
│       ├── deployment.yaml      # OpenShift deployment manifest
│       └── service.yaml         # OpenShift service manifest
├── inputs/                      # Project documentation and analysis
│   ├── coverage_analysis.md     # Test coverage analysis
│   ├── Log Monitoring PRD.pdf   # Product requirements document
│   └── nlp_alert_patterns.md    # NLP pattern analysis
├── internal/                    # Internal application packages
│   ├── alerting/               # Alert evaluation engine
│   │   ├── engine.go           # Main alert evaluation engine
│   │   ├── engine_test.go      # Engine unit tests
│   │   ├── evaluator.go        # Rule evaluation logic
│   │   ├── evaluator_test.go   # Evaluator unit tests
│   │   ├── rules.go            # Rule management and validation
│   │   ├── rules_test.go       # Rules unit tests
│   │   ├── mock_test.go        # Mock setup for tests
│   │   ├── fixtures/           # Test data fixtures
│   │   │   ├── test_logs.json
│   │   │   └── test_rules.json
│   │   ├── mocks/              # Generated mocks
│   │   │   ├── mock_notifier.go
│   │   │   └── mock_state_store.go
│   │   └── README.md           # Alerting package documentation
│   ├── api/                    # HTTP API layer
│   │   ├── handlers.go         # HTTP API handlers
│   │   ├── handlers_test.go    # Handler unit tests
│   │   ├── routes.go           # API route definitions
│   │   ├── integration_test.go # API integration tests
│   │   ├── fixtures/           # Test data fixtures
│   │   │   ├── test_requests.json
│   │   │   └── test_responses.json
│   │   ├── mocks/              # Generated mocks
│   │   │   ├── mock_alert_engine.go
│   │   │   └── mock_state_store.go
│   │   └── README.md           # API package documentation
│   ├── kafka/                  # Kafka integration
│   │   ├── consumer.go         # Kafka consumer implementation
│   │   ├── consumer_test.go    # Consumer unit tests
│   │   ├── processor.go        # Log message processing
│   │   ├── processor_test.go   # Processor unit tests
│   │   ├── integration_test.go # Kafka integration tests
│   │   ├── fixtures/           # Test data fixtures
│   │   │   ├── test_configs.json
│   │   │   └── test_messages.json
│   │   ├── mocks/              # Generated mocks
│   │   │   ├── mock_alert_engine.go
│   │   │   ├── mock_kafka_reader.go
│   │   │   └── mock_state_store.go
│   │   ├── testcontainers/     # Test container setup
│   │   │   └── kafka_container.go
│   │   └── README.md           # Kafka package documentation
│   ├── notifications/          # Notification integrations
│   │   ├── interfaces.go       # Notification interfaces
│   │   ├── interfaces_test.go  # Interface unit tests
│   │   ├── slack.go            # Slack integration
│   │   ├── slack_test.go       # Slack unit tests
│   │   ├── integration_test.go # Notification integration tests
│   │   ├── fixtures/           # Test data fixtures
│   │   │   └── test_alerts.json
│   │   ├── mocks/              # Generated mocks
│   │   │   ├── mock_http_client.go
│   │   │   └── mock_http_server.go
│   │   └── README.md           # Notifications package documentation
│   └── storage/                # Data storage layer
│       ├── redis.go            # Redis storage implementation
│       ├── redis_test.go       # Redis unit tests
│       ├── integration_test.go # Storage integration tests
│       ├── redis_container.go  # Redis test container setup
│       ├── test_data.json      # Test data for storage
│       └── README.md           # Storage package documentation
├── local_e2e/                  # End-to-end testing setup
│   ├── setup/                  # E2E environment setup
│   │   ├── config_local_e2e.yaml
│   │   ├── docker-compose-local-e2e.yml
│   │   ├── mock_log_forwarder.py
│   │   ├── requirements.txt
│   │   ├── setup_local_e2e.sh
│   │   ├── start_alert_engine.sh
│   │   ├── teardown_local_e2e.sh
│   │   ├── test_slack.sh
│   │   └── README.md
│   └── tests/                  # E2E test cases
│       ├── comprehensive_e2e_test_config.json
│       ├── run_e2e_tests.sh
│       └── README.md
├── pkg/                        # Public packages
│   └── models/                 # Data models
│       ├── alert.go            # Alert rule models
│       ├── alert_test.go       # Alert model tests
│       ├── log.go              # Log entry models
│       ├── log_test.go         # Log model tests
│       ├── fixtures/           # Test data fixtures
│       │   ├── test_alerts.json
│       │   └── test_logs.json
│       └── README.md           # Models package documentation
├── scripts/                    # Build and test automation
│   ├── docker-compose.test.yml # Test environment setup
│   ├── run_integration_tests.sh # Integration test runner
│   ├── run_kafka_integration_tests.sh # Kafka-specific test runner
│   ├── run_unit_tests.sh       # Unit test runner
│   ├── test_strategy.md        # Testing strategy documentation
│   └── README.md               # Scripts documentation
├── alert_engine_infra_setup.md # Infrastructure setup guide
├── go.mod                      # Go module definition
├── go.sum                      # Go module checksums
├── main                        # Compiled binary
└── README.md                   # This file
```

## 🚦 Prerequisites

- Go 1.21 or later
- Access to OpenShift/Kubernetes cluster
- Redis instance
- Kafka cluster (Red Hat AMQ Streams)
- Slack workspace with webhook permissions
- Openshift AI

### 📋 Infrastructure Setup

**IMPORTANT**: Before proceeding with the Alert Engine setup, you must first install and configure the required infrastructure components on your OpenShift cluster.

👉 **[OpenShift Infrastructure Setup Guide](alert_engine_infra_setup.md)**

Key infrastructure components to install (15-20 minutes total):
- **Red Hat AMQ Streams**: Install operator and deploy 3-node Kafka cluster with `application-logs` topic
- **Redis Enterprise**: Install operator and create database with ReJSON/TimeSeries modules for state management
- **OpenShift Logging**: Install operator and configure ClusterLogForwarder to route application logs to Kafka
- **RBAC & Security**: Create service accounts, role bindings, and network policies for secure log collection
- **Verification**: Test connectivity between components and validate log forwarding pipeline

**Complete the infrastructure setup before proceeding with the local development or deployment steps below.**

## 🔧 Setup Instructions [To be Updated]

### 1. Local Development Setup

For comprehensive local development setup with end-to-end testing capabilities, refer to:

- **📋 [Local E2E Setup Guide](local_e2e/setup/README.md)** - Complete environment setup with Docker Compose, mock services, and infrastructure
- **🧪 [Local E2E Testing Guide](local_e2e/tests/README.md)** - Running end-to-end tests with real Slack notifications


### 2. Configuration

For detailed configuration instructions including environment variables, configuration files, and deployment settings, refer to:

**📋 [Configuration Guide](configs/README.md)** - Complete configuration documentation with examples for local development, testing, and production deployment.


## 📚 API Documentation

For comprehensive API documentation including endpoints, request/response formats, and usage examples, refer to:

**📋 [API Documentation](internal/api/README.md)** - Complete REST API documentation with detailed endpoint specifications, authentication, and integration examples.

## 🚢 OpenShift Deployment

The Alert Engine provides production-ready OpenShift deployment manifests with comprehensive testing capabilities through mock log generation.

### 🎯 Deployment Components

#### **Alert Engine Production Deployment**
**📁 Location**: [`deployments/alert-engine/`](deployments/alert-engine/)

Production-ready deployment with complete Kubernetes manifests:

- **Container Build System**: Automated build scripts using Red Hat UBI8 base images
- **Security Hardened**: Non-root containers, NetworkPolicies, minimal RBAC permissions
- **High Availability**: Multi-replica deployment with anti-affinity rules and rolling updates
- **Full Integration**: Redis cluster, Kafka (AMQ Streams), and Slack webhook support
- **Monitoring Ready**: Health checks, Prometheus metrics, and comprehensive logging

**Key Features:**
- ✅ **Production Scale**: Optimized for cluster-wide log processing with configurable thresholds
- ✅ **Zero Downtime**: Rolling updates and readiness/liveness probes
- ✅ **Security Compliance**: OpenShift security constraints and network isolation
- ✅ **Resource Management**: Conservative resource requests with horizontal scaling support

#### **MockLogGenerator Testing Deployment**
**📁 Location**: [`deployments/mock/`](deployments/mock/)

Comprehensive log simulation for Alert Engine testing:

- **Realistic Log Patterns**: Generates 19 different alert patterns including payment failures, authentication errors, and database issues
- **OpenShift Integration**: Uses ClusterLogForwarder (Vector) to route logs through standard OpenShift logging pipeline
- **Flexible Modes**: Test mode for E2E validation and continuous mode for ongoing simulation
- **Pattern Coverage**: Supports all alert rule types with configurable generation rates and burst patterns

**Key Features:**
- ✅ **E2E Test Compatibility**: Optimized for automated testing with specific service/log level combinations
- ✅ **Production Simulation**: Realistic log volume and patterns for production-like testing
- ✅ **Standard Log Flow**: Outputs to stdout → Vector → Kafka → Alert Engine pipeline
- ✅ **Configurable Patterns**: Adjustable log generation intervals and alert pattern frequency

### 🚀 Quick Deployment Guide

#### Prerequisites
1. **Complete Infrastructure Setup**: Follow the [Infrastructure Setup Guide](alert_engine_infra_setup.md)
2. **Required Components**: AMQ Streams Kafka, Redis Enterprise, OpenShift Logging with ClusterLogForwarder
3. **Slack Webhook**: Configured webhook URL for alert notifications

#### Production Alert Engine Deployment

```bash
# 1. Build and push container image
cd alert-engine/deployments/alert-engine
./build.sh --version v1.0.0 --push

# 2. Configure Slack webhook
oc create secret generic alert-engine-secrets \
  --from-literal=slack-webhook-url="https://hooks.slack.com/services/YOUR/WEBHOOK/URL" \
  --namespace=alert-engine

# 3. Deploy all components
oc apply -k .

# 4. Verify deployment
oc get pods -n alert-engine
oc logs -n alert-engine deployment/alert-engine -f
```

#### MockLogGenerator Test Deployment

```bash
# 1. Build and push mock container
cd alert-engine/deployments/mock
podman build --platform linux/amd64 -t quay.io/your-registry/mock-log-generator:latest .
podman push quay.io/your-registry/mock-log-generator:latest

# 2. Update image reference in deployment.yaml
# 3. Deploy mock log generator
oc create namespace mock-logs
oc apply -f .

# 4. Verify log generation
oc logs -n mock-logs -l app=mock-log-generator -f
```

### 🔍 Verification & Testing

#### End-to-End Log Flow Verification

```bash
# 1. Check infrastructure status
oc get kafka alert-kafka-cluster -n alert-engine
oc get pods -n openshift-logging -l component=vector

# 2. Verify log generation and forwarding
oc logs -n mock-logs -l app=mock-log-generator --tail=10
oc exec -n alert-engine alert-kafka-cluster-kafka-0 -- \
  /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic application-logs --max-messages 5

# 3. Test Alert Engine processing
ROUTE_URL=$(oc get route alert-engine -n alert-engine -o jsonpath='{.spec.host}')
curl -s "https://$ROUTE_URL/api/v1/system/logs/stats" | jq '.'
```

#### Safe Alert Rule Testing

```bash
# Create conservative test rule to avoid Slack rate limiting
curl -X POST "https://$ROUTE_URL/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-verification-rule",
    "name": "🧪 Deployment Verification",
    "description": "Safe test rule for deployment verification",
    "enabled": true,
    "conditions": {
      "log_level": "ERROR",
      "service": "test-service-verification",
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

### 📊 Production Features

- **High Availability**: Multi-replica deployment with pod anti-affinity
- **Monitoring Integration**: Prometheus metrics and OpenShift monitoring
- **Security Hardening**: NetworkPolicies, minimal RBAC, non-root containers
- **Resource Management**: Configurable CPU/memory limits with horizontal scaling
- **Zero Downtime Updates**: Rolling deployment strategy with health checks

### 🔧 Customization & Scaling

- **Log Generation Rate**: Adjustable via MockLogGenerator ConfigMap
- **Alert Thresholds**: Configurable per-rule via Alert Engine ConfigMap
- **Resource Scaling**: Horizontal pod autoscaling support
- **Network Security**: Customizable NetworkPolicies for environment-specific requirements

For detailed deployment instructions, troubleshooting, and advanced configuration, refer to:
- **🚀 [Alert Engine Deployment Guide](deployments/alert-engine/README.md)** - Complete production deployment documentation
- **🧪 [MockLogGenerator Guide](deployments/mock/README.md)** - Comprehensive testing and log simulation setup




**Alert Engine v1.0.0** - Phase 0: Foundation & Proof of Concept 