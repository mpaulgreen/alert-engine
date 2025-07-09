# Kafka Package Testing Guide

This document provides comprehensive guidance for testing the `internal/kafka` package, including consumer and processor functionality for Kafka message processing and alert engine integration.

## Table of Contents

- [Quick Start](#quick-start)
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Test Categories](#test-categories)
- [Mock Components](#mock-components)
- [Test Fixtures](#test-fixtures)
- [Configuration Testing](#configuration-testing)
- [Performance Testing](#performance-testing)
- [CI/CD Integration](#cicd-integration)
- [Debugging and Troubleshooting](#debugging-and-troubleshooting)
- [Best Practices](#best-practices)

## Quick Start

### Unit Tests Only (Recommended for Development)
```bash
# Run all unit tests - no dependencies required
go test -tags=unit -v ./internal/kafka/tests/...

# Run with coverage
go test -tags=unit -cover ./internal/kafka/tests/...
```

### Integration Tests with Podman (macOS)
```bash
# 1. Start Podman machine
podman machine start podman-machine-default

# 2. Set environment variables (add to your shell profile)
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
export DOCKER_HOST=unix:///var/run/docker.sock
export TESTCONTAINERS_RYUK_DISABLED=true

# 3. Run integration tests
go test -tags=integration -v ./internal/kafka/tests/... -timeout=5m
```

### Using Test Scripts
```bash
# Unit tests (always works)
./scripts/run_unit_tests.sh

# Integration tests (auto-detects Docker/Podman)
./scripts/run_integration_tests.sh
```

### Expected Results
- **Unit Tests**: ~57 tests, all should pass in <30 seconds
- **Integration Tests**: ~5 test suites, all should pass in <3 minutes
- **Performance**: Tests should handle 10+ messages/second

## Overview

The Kafka package test suite provides comprehensive coverage for:
- **Consumer functionality**: Message consumption, group management, health monitoring
- **Processor functionality**: Log processing, validation, metrics, retry logic
- **Batch processing**: High-throughput message processing with batching
- **Error handling**: Network failures, parsing errors, retry scenarios
- **Configuration validation**: Different configuration scenarios and edge cases
- **Integration**: Alert engine integration and concurrent access patterns

## Test Structure

```
internal/kafka/tests/
├── mocks/
│   ├── mock_kafka_reader.go    # Mock Kafka reader implementation
│   └── mock_alert_engine.go    # Mock alert engine implementation
├── fixtures/
│   ├── test_messages.json      # Sample Kafka messages and scenarios
│   └── test_configs.json       # Test configuration examples
├── consumer_test.go            # Consumer functionality tests
├── processor_test.go           # Processor functionality tests
└── README.md                   # This file
```

## Prerequisites

### For Unit Tests
- Go 1.19+ 
- No external dependencies required

### For Integration Tests

#### Docker Setup
- Docker Desktop or Docker Engine
- Docker Compose

#### Podman Setup (macOS)
- Podman Desktop installed
- Podman machine configured and running

##### Initial Podman Setup
```bash
# Install Podman Desktop from https://podman-desktop.io/
# Or install via Homebrew
brew install podman-desktop

# Initialize Podman machine (if not already done)
podman machine init podman-machine-default

# Start Podman machine
podman machine start podman-machine-default

# Verify installation
podman --version
podman machine ls
```

##### Required Environment Variables for Podman
```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
export DOCKER_HOST=unix:///var/run/docker.sock
export TESTCONTAINERS_RYUK_DISABLED=true
```

#### Verification Steps
```bash
# 1. Test unit tests first (should always work)
go test -tags=unit -v ./internal/kafka/tests/...

# 2. Verify Podman setup
podman machine ls
podman ps

# 3. Test environment variables
echo $TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE
echo $DOCKER_HOST  
echo $TESTCONTAINERS_RYUK_DISABLED

# 4. Test simple integration test
go test -tags=integration -v -run TestKafkaContainerHealth ./internal/kafka/tests/... -timeout=3m

# 5. Run all integration tests
go test -tags=integration -v ./internal/kafka/tests/... -timeout=5m
```

## Running Tests

### Unit Tests (No Dependencies Required)
```bash
# Run unit tests only (recommended for development)
go test -tags=unit -v ./internal/kafka/tests/...

# Run unit tests with coverage
go test -tags=unit -cover ./internal/kafka/tests/...

# Run unit tests with race detection
go test -tags=unit -race ./internal/kafka/tests/...
```

### Integration Tests (Requires Docker/Podman)
```bash
# Run integration tests only
go test -tags=integration -v ./internal/kafka/tests/...

# Run integration tests with short timeout
go test -tags=integration -short -v ./internal/kafka/tests/...
```

### All Tests
```bash
# Run all tests (unit + integration)
go test -v ./internal/kafka/tests/...

# Run with verbose output
go test -v ./internal/kafka/tests/...

# Run with coverage
go test -cover ./internal/kafka/tests/...
```

### Specific Test Categories
```bash
# Run consumer tests only
go test -tags=unit -v ./internal/kafka/tests/ -run TestConsumer

# Run processor tests only
go test -tags=unit -v ./internal/kafka/tests/ -run TestProcessor

# Run specific test cases
go test -tags=unit -v ./internal/kafka/tests/ -run TestNewConsumer
go test -tags=unit -v ./internal/kafka/tests/ -run TestLogProcessor_ProcessLogs
```

### Using Test Scripts

#### Unit Test Script
```bash
# Run unit tests using the provided script
./scripts/run_unit_tests.sh

# Run unit tests with coverage
./scripts/run_unit_tests.sh --coverage

# Run unit tests with race detection
./scripts/run_unit_tests.sh --race

# Run unit tests with both coverage and race detection
./scripts/run_unit_tests.sh --coverage --race
```

#### Integration Test Script
```bash
# Run integration tests using the provided script
# (automatically detects and configures Docker/Podman)
./scripts/run_integration_tests.sh

# Run integration tests in container mode
./scripts/run_integration_tests.sh container

# Run integration tests with container logs
./scripts/run_integration_tests.sh --logs

# Run integration tests with cleanup
./scripts/run_integration_tests.sh --cleanup
```

#### Script Features
- **Automatic Detection**: Scripts automatically detect Docker vs Podman
- **Environment Setup**: Configures required environment variables
- **Container Management**: Handles starting/stopping test containers
- **Health Checks**: Verifies container health before running tests
- **Cleanup**: Automatic cleanup of test containers
- **Colored Output**: Provides colored terminal output for better visibility

### Parallel Testing
```bash
# Run tests in parallel (default)
go test -tags=unit -v -parallel 4 ./internal/kafka/tests/...

# Run tests sequentially
go test -tags=unit -v -parallel 1 ./internal/kafka/tests/...
```

### With Race Detection
```bash
# Run tests with race detection
go test -tags=unit -race ./internal/kafka/tests/...
```

### Container-based Testing

#### With Docker
```bash
# Start test containers
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -tags=integration -v ./internal/kafka/tests/...

# Stop test containers
docker-compose -f docker-compose.test.yml down
```

#### With Podman Desktop (macOS)
```bash
# 1. Ensure Podman machine is running
podman machine start podman-machine-default

# 2. Start test containers (Podman Desktop provides Docker Compose compatibility)
docker-compose -f docker-compose.test.yml up -d

# 3. Set environment variables for testcontainers
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
export DOCKER_HOST=unix:///var/run/docker.sock
export TESTCONTAINERS_RYUK_DISABLED=true

# 4. Run integration tests
go test -tags=integration -v ./internal/kafka/tests/... -timeout=5m

# 5. Stop test containers
docker-compose -f docker-compose.test.yml down

# 6. Stop Podman machine (optional)
podman machine stop podman-machine-default
```

#### Quick Commands for Podman
```bash
# All-in-one command for Podman integration tests
podman machine start podman-machine-default && \
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock && \
export DOCKER_HOST=unix:///var/run/docker.sock && \
export TESTCONTAINERS_RYUK_DISABLED=true && \
go test -tags=integration -v ./internal/kafka/tests/... -timeout=5m
```

## Test Categories

### 1. Consumer Tests (`consumer_test.go`)

#### Basic Consumer Operations
- **Consumer Creation**: Tests `NewConsumer()` with various configurations
- **Message Processing**: Tests message reading and processing flow
- **Error Handling**: Tests invalid JSON, network failures, context cancellation
- **Statistics**: Tests `GetStats()` and consumer metrics
- **Health Checks**: Tests `HealthCheck()` functionality
- **Resource Management**: Tests `Close()` and cleanup

#### Consumer Groups
- **Group Creation**: Tests `NewConsumerGroup()` functionality
- **Group Management**: Tests starting/stopping consumer groups
- **Group Statistics**: Tests `GetGroupStats()` for multiple consumers
- **Scalability**: Tests consumer groups with different sizes

#### Message Processor
- **Batch Processing**: Tests `MessageProcessor` with different batch sizes
- **Timing**: Tests flush intervals and batch timeouts
- **Context Handling**: Tests graceful shutdown and context cancellation

#### Configuration Testing
- **Valid Configurations**: Tests various valid configuration scenarios
- **Invalid Configurations**: Tests error handling with invalid configurations
- **Default Configuration**: Tests `DefaultConsumerConfig()` functionality

#### Edge Cases
- **Large Messages**: Tests handling of large message payloads
- **Concurrent Access**: Tests thread safety and concurrent operations
- **Resource Limits**: Tests behavior with zero/negative values

### 2. Processor Tests (`processor_test.go`)

#### Log Processor Operations
- **Processor Creation**: Tests `NewLogProcessor()` with various configurations
- **Log Processing**: Tests `ProcessLogs()` functionality
- **Validation**: Tests log entry validation and sanitization
- **Metrics**: Tests `GetMetrics()` and metric collection
- **Health Monitoring**: Tests `HealthCheck()` with different states

#### Batch Log Processor
- **Batch Creation**: Tests `NewBatchLogProcessor()` functionality
- **Batch Processing**: Tests batch processing with different sizes
- **Flush Behavior**: Tests timer-based and size-based flushing
- **Context Handling**: Tests graceful shutdown and batch cleanup

#### Processor Factory
- **Factory Creation**: Tests `NewProcessorFactory()` functionality
- **Processor Creation**: Tests factory-based processor creation
- **Configuration**: Tests factory with different configurations
- **Error Handling**: Tests factory behavior with invalid configurations

#### Retry Logic
- **Retry Attempts**: Tests retry logic with different attempt counts
- **Retry Delays**: Tests exponential backoff and retry delays
- **Failure Scenarios**: Tests behavior when all retries fail

#### Metrics and Monitoring
- **Metric Collection**: Tests processing metrics and statistics
- **Error Rate Calculation**: Tests error rate computation
- **Health Monitoring**: Tests health check with different scenarios

#### Integration Testing
- **Alert Engine Integration**: Tests integration with alert engine
- **Error Handling**: Tests alert engine panic scenarios
- **Concurrent Access**: Tests thread safety and concurrent operations

## Mock Components

### MockKafkaReader (`mocks/mock_kafka_reader.go`)

A comprehensive mock implementation of the Kafka reader interface for testing consumer functionality.

#### Features:
- **Message Queue**: Simulates Kafka message queue with configurable messages
- **Error Simulation**: Configurable failure modes for testing error handling
- **Statistics**: Mock statistics collection and reporting
- **Timing Control**: Configurable delays for timing-based tests
- **Context Handling**: Proper context cancellation simulation

#### Usage:
```go
// Create mock reader
mockReader := mocks.NewMockKafkaReader()

// Add test messages
message := mocks.CreateTestMessage(0, 100, "test-key", `{"test": "data"}`)
mockReader.AddMessage(message)

// Configure failure behavior
mockReader.SetShouldFail(true, "simulated network error")

// Set read delay for timing tests
mockReader.SetReadDelay(100 * time.Millisecond)

// Reset mock state
mockReader.Reset()
```

#### Available Methods:
- `AddMessage(message)` - Add a message to the queue
- `AddMessages(messages)` - Add multiple messages
- `SetShouldFail(shouldFail, message)` - Configure failure behavior
- `SetReadDelay(delay)` - Set read operation delay
- `Reset()` - Reset mock state
- `GetMessagesCount()` - Get number of messages in queue
- `IsClosed()` - Check if reader is closed
- `IsContextCanceled()` - Check if context was canceled

### MockAlertEngine (`mocks/mock_alert_engine.go`)

A mock implementation of the AlertEngine interface for testing processor functionality.

#### Features:
- **Log Tracking**: Tracks all evaluated log entries
- **Call Counting**: Counts the number of evaluation calls
- **Processing Simulation**: Configurable processing delays
- **Panic Simulation**: Configurable panic behavior for error testing
- **Query Methods**: Methods to inspect processed logs

#### Usage:
```go
// Create mock alert engine
mockEngine := mocks.NewMockAlertEngine()

// Configure processing delay
mockEngine.SetProcessingTime(50 * time.Millisecond)

// Configure panic behavior
mockEngine.SetShouldPanic(true)

// Check evaluation results
count := mockEngine.GetCallCount()
logs := mockEngine.GetEvaluatedLogs()
lastLog := mockEngine.GetLastEvaluatedLog()

// Query specific logs
errorLogs := mockEngine.CountLogsByLevel("ERROR")
prodLogs := mockEngine.CountLogsByNamespace("production")

// Reset mock state
mockEngine.Reset()
```

#### Available Methods:
- `GetEvaluatedLogs()` - Get all evaluated log entries
- `GetCallCount()` - Get number of evaluation calls
- `GetLastEvaluatedLog()` - Get the last processed log
- `SetProcessingTime(duration)` - Set processing delay
- `SetShouldPanic(shouldPanic)` - Configure panic behavior
- `Reset()` - Reset mock state
- `CountLogsByLevel(level)` - Count logs by log level
- `CountLogsByNamespace(namespace)` - Count logs by namespace
- `FindLogByMessage(message)` - Find log by message content

## Test Fixtures

### Test Messages (`fixtures/test_messages.json`)

Contains sample Kafka messages for testing different scenarios:

#### Valid Messages
- **Application Logs**: Production-ready log entries with proper structure
- **Error Messages**: Error logs with various error types
- **Multi-Service**: Messages from different services and namespaces
- **Batch Messages**: Sets of messages for batch processing tests

#### Invalid Messages
- **Malformed JSON**: Invalid JSON structures for error testing
- **Missing Fields**: Messages missing required fields
- **Empty Messages**: Edge cases with empty or null values
- **Invalid Formats**: Incorrectly formatted timestamps and data

#### Configuration Examples
- **Consumer Configs**: Various consumer configuration scenarios
- **Processor Configs**: Different processor configuration setups
- **Error Scenarios**: Common error scenarios and expected behaviors

### Test Configurations (`fixtures/test_configs.json`)

Contains configuration examples for testing:

#### Consumer Configurations
- **Basic Configuration**: Simple single-broker setup
- **Multi-Broker**: Multiple broker configurations
- **High Throughput**: High-performance configurations
- **Minimal Configuration**: Minimal required settings

#### Processor Configurations
- **Default Processor**: Standard processor setup
- **Fast Processor**: Low-latency configuration
- **Reliable Processor**: High-reliability configuration
- **No Retry Processor**: Configuration without retry logic

#### Batch Configurations
- **Small Batch**: Small batch size for testing
- **Large Batch**: Large batch size for performance testing
- **Fast Flush**: Quick flush intervals
- **Slow Flush**: Long flush intervals

#### Validation Configurations
- **Valid Configurations**: Properly configured setups
- **Invalid Configurations**: Configurations with errors
- **Edge Cases**: Boundary value configurations

## Configuration Testing

### Consumer Configuration Tests

```go
// Test valid configuration
config := kafka.ConsumerConfig{
    Brokers:  []string{"localhost:9092"},
    Topic:    "test-topic",
    GroupID:  "test-group",
    MinBytes: 1024,
    MaxBytes: 1048576,
    MaxWait:  time.Second,
}

// Test invalid configuration
invalidConfig := kafka.ConsumerConfig{
    Brokers: []string{}, // Empty brokers
    Topic:   "",         // Empty topic
}
```

### Processor Configuration Tests

```go
// Test default configuration
config := kafka.DefaultProcessorConfig()

// Test custom configuration
customConfig := kafka.ProcessorConfig{
    BatchSize:     50,
    FlushInterval: 2 * time.Second,
    RetryAttempts: 5,
    RetryDelay:    500 * time.Millisecond,
    EnableMetrics: true,
}
```

## Performance Testing

### Throughput Testing
```bash
# Run performance tests
go test -bench=. ./internal/kafka/tests/...

# Run with memory profiling
go test -bench=. -memprofile=mem.prof ./internal/kafka/tests/...

# Run with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./internal/kafka/tests/...
```

### Load Testing
```go
// Example load testing setup
func TestHighVolumeProcessing(t *testing.T) {
    mockEngine := mocks.NewMockAlertEngine()
    processor := kafka.NewLogProcessor([]string{"localhost:9092"}, "test-topic", mockEngine)
    
    // Create many test messages
    messages := make([]models.LogEntry, 1000)
    for i := range messages {
        messages[i] = createTestLogEntry(i)
    }
    
    // Process messages
    start := time.Now()
    for _, msg := range messages {
        mockEngine.EvaluateLog(msg)
    }
    duration := time.Since(start)
    
    // Verify performance
    assert.Less(t, duration, 100*time.Millisecond)
    assert.Equal(t, 1000, mockEngine.GetCallCount())
}
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Kafka Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.19
      
      - name: Run Kafka Tests
        run: |
          go test -v -race -cover ./internal/kafka/tests/...
          
      - name: Run Integration Tests
        run: |
          go test -v -tags=integration ./internal/kafka/tests/...
```

### Test Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./internal/kafka/tests/...

# View coverage report
go tool cover -html=coverage.out

# Coverage threshold check
go test -cover ./internal/kafka/tests/... | grep -E "coverage: [0-9]+\.[0-9]+%" | awk '{print $2}' | grep -E "^[89][0-9]\.[0-9]+%|^100\.0%"
```

## Debugging and Troubleshooting

### Common Issues

#### 1. Mock Configuration Issues
```go
// Problem: Mock not behaving as expected
mockReader := mocks.NewMockKafkaReader()
mockReader.SetShouldFail(true, "test error")

// Solution: Verify mock state
assert.True(t, mockReader.ShouldFail())
assert.Equal(t, "test error", mockReader.GetFailureMessage())
```

#### 2. Timing Issues
```go
// Problem: Tests failing due to timing
time.Sleep(50 * time.Millisecond) // Unreliable

// Solution: Use proper synchronization
done := make(chan bool)
go func() {
    // Test operation
    done <- true
}()
select {
case <-done:
    // Test succeeded
case <-time.After(100 * time.Millisecond):
    t.Fatal("Test timed out")
}
```

#### 3. Context Cancellation
```go
// Problem: Context not being handled properly
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Solution: Verify context is checked
go func() {
    select {
    case <-ctx.Done():
        // Properly handled
    default:
        // Continue processing
    }
}()
```

#### 4. Podman Integration Test Issues

##### Issue: "Cannot connect to Podman" or "No such file or directory"
```bash
# Problem: Podman machine not started
Error: Cannot connect to Podman. Please verify...

# Solution: Start Podman machine
podman machine start podman-machine-default

# Verify machine is running
podman machine ls
```

##### Issue: "Network 'bridge' not found" 
```bash
# Problem: Testcontainers expects 'bridge' network
Error: failed to create network: network bridge not found

# Solution: Environment variables already handle this
export TESTCONTAINERS_RYUK_DISABLED=true
```

##### Issue: "Context deadline exceeded" in integration tests
```bash
# Problem: Tests timing out with Podman containers
Error: context deadline exceeded

# Solution: Use longer timeout and check container startup
go test -tags=integration -v ./internal/kafka/tests/... -timeout=5m

# Check if containers started properly
docker-compose -f docker-compose.test.yml ps
```

##### Issue: "Connection refused" when connecting to Kafka
```bash
# Problem: Kafka not accessible from testcontainers
Error: dial tcp [::1]:9092: connect: connection refused

# Solution: This is expected with our testcontainer setup
# Testcontainers creates its own Kafka instances
# The docker-compose.test.yml containers are separate
```

##### Issue: Integration test assertions failing on context cancellation
```bash
# Problem: Context errors vary between Docker and Podman
Error: assert.Equal: context.DeadlineExceeded != context.Canceled

# Solution: Updated tests now handle both error types
# Tests check for either context.Canceled OR context.DeadlineExceeded
```

#### 5. Environment Setup Verification

```bash
# Verify Podman setup
podman version
podman machine ls
podman ps

# Verify environment variables
echo $TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE
echo $DOCKER_HOST
echo $TESTCONTAINERS_RYUK_DISABLED

# Test basic connectivity
podman run --rm hello-world
```

### Debug Logging
```go
// Enable debug logging in tests
func TestWithDebugLogging(t *testing.T) {
    // Set log level for debugging
    log.SetLevel(log.DebugLevel)
    
    // Your test code here
    
    // Verify log output
    assert.Contains(t, logOutput, "expected log message")
}
```

### Test Isolation
```go
// Ensure tests don't interfere with each other
func TestIsolatedTest(t *testing.T) {
    // Reset mock state
    mockEngine := mocks.NewMockAlertEngine()
    mockEngine.Reset()
    
    // Your test code here
    
    // Verify clean state
    assert.Equal(t, 0, mockEngine.GetCallCount())
}
```

## Best Practices

### 1. Test Organization
- Group related tests using `t.Run()` subtests
- Use descriptive test names that explain the scenario
- Keep tests focused on single functionality
- Use table-driven tests for similar scenarios

### 2. Mock Usage
- Always reset mocks between tests
- Configure mocks to match real behavior
- Use mocks to test error conditions
- Verify mock interactions after tests

### 3. Test Data Management
- Use fixtures for consistent test data
- Create helper functions for common test setup
- Avoid hardcoded values in tests
- Use meaningful test data that reflects real scenarios

### 4. Error Testing
- Test both success and failure scenarios
- Verify error messages and types
- Test edge cases and boundary conditions
- Ensure proper error propagation

### 5. Performance Considerations
- Use short timeouts for faster test execution
- Avoid unnecessary sleeps in tests
- Test with realistic data volumes
- Profile tests to identify bottlenecks

### 6. Maintenance
- Keep tests up-to-date with code changes
- Refactor tests when code structure changes
- Remove obsolete tests
- Update documentation regularly

## Example Test Scenarios

### Consumer Integration Test
```go
func TestConsumerIntegration(t *testing.T) {
    // Setup
    mockEngine := mocks.NewMockAlertEngine()
    config := kafka.DefaultConsumerConfig()
    consumer := kafka.NewConsumer(config, mockEngine)
    
    // Test
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    go func() {
        err := consumer.Start(ctx)
        assert.Equal(t, context.DeadlineExceeded, err)
    }()
    
    // Verify
    time.Sleep(50 * time.Millisecond)
    stats := consumer.GetStats()
    assert.NotNil(t, stats)
}
```

### Processor Error Handling Test
```go
func TestProcessorErrorHandling(t *testing.T) {
    // Setup
    mockEngine := mocks.NewMockAlertEngine()
    mockEngine.SetShouldPanic(true)
    
    processor := kafka.NewLogProcessor([]string{"localhost:9092"}, "test-topic", mockEngine)
    
    // Test should handle panic gracefully
    assert.NotPanics(t, func() {
        processor.ProcessLogs(context.Background())
    })
    
    // Verify error metrics
    metrics := processor.GetMetrics()
    assert.Greater(t, metrics.MessagesFailure, int64(0))
}
```

## Known Limitations and Notes

### Integration Tests
- **Container Startup**: Integration tests may take 1-3 minutes due to Kafka container startup time
- **Podman Compatibility**: Requires Podman Desktop with Docker API compatibility
- **Network Configuration**: Tests create temporary containers with their own network configuration
- **Resource Usage**: Integration tests consume more CPU/memory due to container overhead

### Test Environment
- **Test Isolation**: Each integration test creates its own Kafka container for isolation
- **Port Conflicts**: Tests use dynamic port allocation to avoid conflicts
- **Cleanup**: Containers are automatically cleaned up after tests (with Ryuk disabled for Podman)

### Performance Considerations
- **Parallel Execution**: Unit tests run in parallel by default
- **Timeout Configuration**: Integration tests use longer timeouts (5 minutes)
- **Resource Limits**: Tests are designed to run on development machines

### Build Tags
- **Unit Tests**: Use `//go:build unit` tag - run with `-tags=unit`
- **Integration Tests**: Use `//go:build integration` tag - run with `-tags=integration`
- **Default**: Running without tags includes both unit and integration tests

### Environment Variables
These environment variables are **required** for Podman integration tests:
- `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock`
- `DOCKER_HOST=unix:///var/run/docker.sock`
- `TESTCONTAINERS_RYUK_DISABLED=true`

### Troubleshooting Quick Reference
1. **Unit tests failing**: Check Go version and dependencies
2. **Integration tests timing out**: Increase timeout with `-timeout=5m`
3. **Container connection issues**: Verify Podman machine is running
4. **Environment variables**: Source your shell profile or set in current session

---

For more information about the Kafka package implementation, see the main [README.md](../README.md) file.

For questions or issues, please refer to the project documentation or create an issue in the repository. 