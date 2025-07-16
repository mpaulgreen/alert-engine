# Unit Testing Structure for Alert Engine

## Overview

This document outlines the comprehensive unit testing structure for the Alert Engine project. Each package is tested individually with appropriate mocking strategies and test containers where needed.

## Directory Structure

```
alert-engine/
├── internal/
│   ├── alerting/
│   │   ├── engine.go
│   │   ├── evaluator.go
│   │   ├── rules.go
│   │   └── tests/
│   │       ├── engine_test.go
│   │       ├── evaluator_test.go
│   │       ├── rules_test.go
│   │       ├── mocks/
│   │       │   ├── mock_state_store.go
│   │       │   └── mock_notifier.go
│   │       └── fixtures/
│   │           ├── test_rules.json
│   │           └── test_logs.json
│   ├── api/
│   │   ├── handlers.go
│   │   ├── routes.go
│   │   └── tests/
│   │       ├── handlers_test.go
│   │       ├── routes_test.go
│   │       ├── integration_test.go
│   │       ├── mocks/
│   │       │   ├── mock_state_store.go
│   │       │   └── mock_alert_engine.go
│   │       └── fixtures/
│   │           ├── test_requests.json
│   │           └── test_responses.json
│   ├── kafka/
│   │   ├── consumer.go
│   │   ├── processor.go
│   │   └── tests/
│   │       ├── consumer_test.go
│   │       ├── processor_test.go
│   │       ├── integration_test.go
│   │       ├── mocks/
│   │       │   └── mock_alert_engine.go
│   │       ├── fixtures/
│   │       │   └── test_messages.json
│   │       └── testcontainers/
│   │           └── kafka_container.go
│   ├── notifications/
│   │   ├── interfaces.go
│   │   ├── slack.go
│   │   └── tests/
│   │       ├── slack_test.go
│   │       ├── interfaces_test.go
│   │       ├── mocks/
│   │       │   └── mock_http_client.go
│   │       └── fixtures/
│   │           └── test_alerts.json
│   └── storage/
│       ├── redis.go
│       └── tests/
│           ├── redis_test.go
│           ├── integration_test.go
│           ├── fixtures/
│           │   └── test_data.json
│           └── testcontainers/
│               └── redis_container.go
├── pkg/
│   └── models/
│       ├── alert.go
│       ├── log.go
│       └── tests/
│           ├── alert_test.go
│           ├── log_test.go
│           └── fixtures/
│               ├── test_alerts.json
│               └── test_logs.json
├── tests/
│   ├── integration/
│   │   ├── end_to_end_test.go
│   │   ├── kafka_to_slack_test.go
│   │   └── docker-compose.test.yml
│   ├── performance/
│   │   ├── load_test.go
│   │   └── benchmark_test.go
│   └── utils/
│       ├── test_helpers.go
│       └── container_helpers.go
└── scripts/
    ├── run_unit_tests.sh
    ├── run_integration_tests.sh
    └── setup_test_env.sh
```

## Package-Specific Testing Strategies

### 1. pkg/models Package

**Test Focus**: Data structure validation, JSON marshaling/unmarshaling, field validation

**Test Files**:
- `pkg/models/tests/alert_test.go`
- `pkg/models/tests/log_test.go`

**Key Test Cases**:
- JSON serialization/deserialization
- Field validation
- Time handling
- Default values
- Edge cases (empty values, null fields)

**Example Test Structure**:
```go
func TestAlertRule_JSONMarshaling(t *testing.T) {
    // Test marshaling and unmarshaling
}

func TestLogEntry_Validation(t *testing.T) {
    // Test field validation
}

func TestKubernetesInfo_Labels(t *testing.T) {
    // Test label handling
}
```

### 2. internal/alerting Package

**Test Focus**: Alert rule evaluation, engine logic, rule management

**Dependencies**: Mock StateStore, Mock Notifier

**Test Files**:
- `internal/alerting/tests/engine_test.go`
- `internal/alerting/tests/evaluator_test.go`
- `internal/alerting/tests/rules_test.go`

**Key Test Cases**:
- Rule evaluation against log entries
- Threshold logic
- Time window management
- Alert generation
- Rule CRUD operations
- Error handling

**Mock Strategy**:
```go
// Mock StateStore for testing without Redis
type MockStateStore struct {
    rules map[string]models.AlertRule
    counters map[string]int64
    // ... other fields
}

// Mock Notifier for testing without Slack
type MockNotifier struct {
    sentAlerts []models.Alert
    shouldFail bool
}
```

### 3. internal/api Package

**Test Focus**: HTTP handlers, routing, request/response handling

**Dependencies**: Mock StateStore, Mock AlertEngine

**Test Files**:
- `internal/api/tests/handlers_test.go`
- `internal/api/tests/routes_test.go`
- `internal/api/tests/integration_test.go`

**Key Test Cases**:
- HTTP endpoint behavior
- Request validation
- Response formatting
- Error handling
- Authentication/authorization
- CORS headers

**Test Strategy**:
```go
func TestHandlers_GetRules(t *testing.T) {
    // Test GET /api/v1/rules endpoint
    router := gin.New()
    mockStore := &MockStateStore{}
    mockEngine := &MockAlertEngine{}
    handlers := NewHandlers(mockStore, mockEngine)
    handlers.SetupRoutes(router)
    
    // Test cases...
}
```

### 4. internal/kafka Package

**Test Focus**: Message consumption, processing, batch handling

**Dependencies**: Mock AlertEngine, Test Kafka container

**Test Files**:
- `internal/kafka/tests/consumer_test.go`
- `internal/kafka/tests/processor_test.go`
- `internal/kafka/tests/integration_test.go`

**Key Test Cases**:
- Message consumption
- JSON deserialization
- Error handling and retries
- Batch processing
- Consumer group management
- Performance metrics

**Container Strategy**:
```go
func setupKafkaContainer(t *testing.T) *kafka.Container {
    // Use testcontainers to spin up Kafka
    // Return configured container
}
```

### 5. internal/notifications Package

**Test Focus**: Notification sending, message formatting, error handling

**Dependencies**: Mock HTTP client

**Test Files**:
- `internal/notifications/tests/slack_test.go`
- `internal/notifications/tests/interfaces_test.go`

**Key Test Cases**:
- Slack message formatting
- Webhook sending
- Error handling
- Rate limiting
- Connection testing
- Message templates

**Mock Strategy**:
```go
type MockHTTPClient struct {
    responses map[string]*http.Response
    requests  []*http.Request
}
```

### 6. internal/storage Package

**Test Focus**: Redis operations, data persistence, error handling

**Dependencies**: Test Redis container

**Test Files**:
- `internal/storage/tests/redis_test.go`
- `internal/storage/tests/integration_test.go`

**Key Test Cases**:
- CRUD operations
- Counter management
- Time-based expiration
- Connection handling
- Error scenarios
- Performance

**Container Strategy**:
```go
func setupRedisContainer(t *testing.T) *redis.Container {
    // Use testcontainers to spin up Redis
    // Return configured container
}
```

## Test Execution with Podman

### Individual Package Testing

Create shell scripts for testing each package:

**scripts/run_unit_tests.sh**:
```bash
#!/bin/bash

# Test individual packages
echo "Testing pkg/models..."
go test -v ./pkg/models/tests/...

echo "Testing internal/alerting..."
go test -v ./internal/alerting/tests/...

echo "Testing internal/api..."
go test -v ./internal/api/tests/...

echo "Testing internal/kafka..."
go test -v ./internal/kafka/tests/...

echo "Testing internal/notifications..."
go test -v ./internal/notifications/tests/...

echo "Testing internal/storage..."
go test -v ./internal/storage/tests/...
```

### Container-based Testing

**docker-compose.test.yml**:
```yaml
version: '3.8'
services:
  redis-test:
    image: redis:7-alpine
    ports:
      - "6380:6379"
    environment:
      - REDIS_PASSWORD=testpass
  
  kafka-test:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper-test:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9093
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    ports:
      - "9093:9092"
    depends_on:
      - zookeeper-test
  
  zookeeper-test:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    ports:
      - "2182:2181"
```

### Podman Testing Commands

```bash
# Start test containers
podman-compose -f docker-compose.test.yml up -d

# Run unit tests for specific package
podman run --rm -v $(pwd):/app -w /app golang:1.21 go test -v ./pkg/models/tests/...

# Run integration tests with containers
podman run --rm --network=host -v $(pwd):/app -w /app golang:1.21 go test -v ./internal/storage/tests/...

# Run all tests
podman run --rm --network=host -v $(pwd):/app -w /app golang:1.21 go test -v ./...

# Cleanup
podman-compose -f docker-compose.test.yml down
```

## Test Organization Best Practices

### 1. Test Structure
- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **End-to-End Tests**: Test complete workflows

### 2. Mock Strategy
- Create interfaces for all external dependencies
- Use dependency injection for testability
- Mock external services (Redis, Kafka, HTTP)

### 3. Test Data Management
- Use fixtures for consistent test data
- Create helper functions for common test scenarios
- Separate test data from test logic

### 4. Performance Testing
- Include benchmarks for critical paths
- Test with realistic data volumes
- Monitor memory usage and goroutine leaks

### 5. Test Categories
Use build tags to categorize tests:
```go
//go:build unit
// +build unit

//go:build integration
// +build integration

//go:build performance
// +build performance
```

## Running Tests

### Unit Tests Only
```bash
go test -tags=unit -v ./...
```

### Integration Tests Only
```bash
go test -tags=integration -v ./...
```

### All Tests
```bash
go test -v ./...
```

### With Coverage
```bash
go test -coverprofile=coverage.out -v ./...
go tool cover -html=coverage.out
```

This structure ensures comprehensive testing of each component while maintaining isolation and enabling efficient testing with podman containers. 