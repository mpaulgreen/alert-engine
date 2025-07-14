# Storage Package Testing Guide

This document provides comprehensive guidance for testing the `internal/storage` package, including Redis storage operations, data persistence, and container-based integration testing.

## Table of Contents

- [Quick Start](#quick-start)
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Test Categories](#test-categories)
- [Test Fixtures](#test-fixtures)
- [Container Testing](#container-testing)
- [Performance Testing](#performance-testing)
- [CI/CD Integration](#cicd-integration)
- [Debugging and Troubleshooting](#debugging-and-troubleshooting)
- [Best Practices](#best-practices)

## Quick Start

### Unit Tests Only (Recommended for Development)
```bash
# Run all unit tests - no dependencies required
go test -tags=unit -v ./internal/storage/tests/...

# Run with coverage
go test -tags=unit -cover ./internal/storage/tests/...
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
go test -tags=integration -v ./internal/storage/tests/... -timeout=5m
```

### Using Test Scripts
```bash
# Unit tests (always works)
./scripts/run_unit_tests.sh

# Integration tests (auto-detects Docker/Podman)
./scripts/run_integration_tests.sh
```

### Expected Results
- **Unit Tests**: ~13 tests, all should pass in <30 seconds
- **Integration Tests**: ~20 test suites, all should pass in <3 minutes
- **Performance**: Tests should handle 1000+ operations/second

## Overview

The Storage package test suite provides comprehensive coverage for:
- **Redis operations**: CRUD operations for alert rules, alerts, and status data
- **Counter management**: Time-based counters with expiration handling
- **Data persistence**: JSON marshaling/unmarshaling and data integrity
- **Connection management**: Health checks, connection pooling, and error handling
- **Performance**: Bulk operations, concurrent access, and throughput testing
- **Transaction support**: Pipeline operations and atomic transactions

## Test Structure

```
internal/storage/tests/
├── fixtures/
│   └── test_data.json              # Sample test data and scenarios
├── testcontainers/
│   └── redis_container.go          # Redis container setup for integration tests
├── redis_test.go                   # Unit tests with mocked Redis client
├── integration_test.go             # Integration tests with real Redis container
└── README.md                       # This file
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
go test -tags=unit -v ./internal/storage/tests/...

# 2. Verify Podman setup
podman machine ls
podman ps

# 3. Test environment variables
echo $TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE
echo $DOCKER_HOST  
echo $TESTCONTAINERS_RYUK_DISABLED

# 4. Test simple integration test
go test -tags=integration -v -run TestRedisStore_Integration_SaveAndGetAlertRule ./internal/storage/tests/... -timeout=3m

# 5. Run all integration tests
go test -tags=integration -v ./internal/storage/tests/... -timeout=5m
```

## Running Tests

### Unit Tests (No Dependencies Required)
```bash
# Run unit tests only (recommended for development)
go test -tags=unit -v ./internal/storage/tests/...

# Run unit tests with coverage
go test -tags=unit -cover ./internal/storage/tests/...

# Run unit tests with race detection
go test -tags=unit -race ./internal/storage/tests/...
```

### Integration Tests (Requires Docker/Podman)
```bash
# Run integration tests only
go test -tags=integration -v ./internal/storage/tests/...

# Run integration tests with short timeout
go test -tags=integration -short -v ./internal/storage/tests/...
```

### All Tests
```bash
# Run all tests (unit + integration)
go test -tags=unit -v ./internal/storage/tests/...

# Run with verbose output
go test -tags=unit -v ./internal/storage/tests/...

# Run with coverage
go test -tags=unit -cover ./internal/storage/tests/...
```

### Specific Test Categories
```bash
# Run alert rule tests only
go test -tags=unit -v ./internal/storage/tests/ -run "AlertRule"

# Run counter tests only
go test -tags=unit -v ./internal/storage/tests/ -run "Counter"

# Run specific test cases
go test -tags=unit -v ./internal/storage/tests/ -run TestAlertRule_JSONMarshaling
go test -tags=integration -v ./internal/storage/tests/ -run TestRedisStore_Integration_ConcurrentOperations
```

### Using Test Scripts

#### Unit Test Script
```bash
# Run unit tests using the provided script
./scripts/run_unit_tests.sh
```

#### Integration Test Script
```bash
# Run integration tests using the provided script
./scripts/run_integration_tests.sh
```

## Test Categories

### Unit Tests (`redis_test.go`)
- **Alert Rule Operations**: Save, get, update, delete operations
- **Alert Operations**: Alert creation, retrieval, and management
- **Counter Operations**: Increment, get, and time-window management
- **Status Operations**: Alert status tracking and updates
- **Statistics Operations**: Log statistics aggregation and retrieval
- **Bulk Operations**: Batch processing and pipeline operations
- **Error Handling**: Connection errors, data validation, and edge cases

### Integration Tests (`integration_test.go`)
- **Basic CRUD Operations**: Real Redis interactions with full data validation
- **Concurrent Operations**: Multi-goroutine access patterns and race conditions
- **Performance Testing**: Throughput and latency measurements
- **Transaction Operations**: Pipeline operations and atomic transactions
- **Health Monitoring**: Connection health and metrics collection
- **Data Integrity**: JSON marshaling and data persistence verification
- **Error Scenarios**: Network failures and recovery testing

## Test Fixtures

### Test Data (`fixtures/test_data.json`)
The test fixtures include:
- **Alert Rules**: Sample alert rule configurations with various conditions
- **Alerts**: Sample alert instances with different severities and statuses
- **Alert Statuses**: Status tracking data for rules
- **Log Statistics**: Aggregated log data for testing statistics operations
- **Test Scenarios**: Various test case configurations including error scenarios

### Usage in Tests
```go
// Load test data
var testData struct {
    AlertRules []models.AlertRule `json:"alert_rules"`
    Alerts     []models.Alert     `json:"alerts"`
}

// Use in tests
testRule := testData.AlertRules[0]
err := redisStore.SaveAlertRule(testRule)
```

## Container Testing

### Redis Container Setup
The Redis testcontainer provides:
- **Isolated Environment**: Each test gets a fresh Redis instance
- **Automatic Cleanup**: Containers are automatically removed after tests
- **Configuration**: Pre-configured with test-specific settings
- **Health Checks**: Automatic health verification before tests run

### Container Configuration
```go
// Redis container with authentication
container := testcontainers.NewRedisContainer(ctx, t)
store := storage.NewRedisStore(
    container.GetConnectionString(),
    container.GetPassword(),
)
```

### Container Methods
```go
// Basic operations
container.Set("key", "value", time.Hour)
container.Get("key")
container.Del("key")

// Bulk operations
container.LoadTestData(testData)
container.FlushDB()

// Health checks
container.TestRedisAvailability()
container.WaitForConnection(30 * time.Second)
```

## Performance Testing

### Metrics Collected
- **Throughput**: Operations per second for different operation types
- **Latency**: Response times for individual operations
- **Concurrency**: Performance under concurrent access
- **Memory Usage**: Redis memory consumption patterns
- **Connection Pool**: Connection utilization and efficiency

### Performance Test Examples
```bash
# Run performance tests
go test -tags=integration -v ./internal/storage/tests/ -run TestRedisStore_Integration_Performance

# Run with custom duration
go test -tags=integration -v ./internal/storage/tests/ -run TestRedisStore_Integration_Performance -timeout=10m

# Skip performance tests in short mode
go test -tags=integration -short -v ./internal/storage/tests/...
```

### Expected Performance
- **Single Operations**: 10,000+ ops/sec
- **Bulk Operations**: 50,000+ ops/sec
- **Concurrent Access**: Linear scaling up to 10 goroutines
- **Memory Usage**: <1MB for 1000 alert rules

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Storage Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: 1.19
      - name: Run unit tests
        run: go test -tags=unit -v ./internal/storage/tests/...

  integration-tests:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: 1.19
      - name: Run integration tests
        run: go test -tags=integration -v ./internal/storage/tests/...
```

### Local CI Scripts
```bash
# Run CI-like tests locally
./scripts/run_unit_tests.sh
./scripts/run_integration_tests.sh

# Run with coverage
./scripts/run_unit_tests.sh -coverage
./scripts/run_integration_tests.sh -coverage
```

## Debugging and Troubleshooting

### Common Issues

#### Container Startup Problems
```bash
# Check Docker/Podman status
podman machine ls
podman ps

# Verify environment variables
echo $TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE
echo $DOCKER_HOST

# Test container manually
podman run --rm redis:7-alpine redis-cli ping
```

#### Test Failures
```bash
# Run with verbose output
go test -tags=integration -v ./internal/storage/tests/... -run TestFailingTest

# Run single test with debug
go test -tags=integration -v ./internal/storage/tests/... -run TestRedisStore_Integration_SaveAndGetAlertRule -timeout=5m

# Check Redis logs
podman logs <container_id>
```

#### Performance Issues
```bash

# 1. Set environment variables (add to ~/.zshrc or ~/.bashrc)
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
export DOCKER_HOST=unix:///var/run/docker.sock
export TESTCONTAINERS_RYUK_DISABLED=true

# 2. Run profiling command
go test -tags=integration -v ./internal/storage/tests -cpuprofile=cpu.prof -memprofile=mem.prof

# 3. Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
go tool pprof -top cpu.prof  # Top CPU functions
go tool pprof -top mem.prof  # Top memory allocations
go tool pprof -list main cpu.prof    # Show specific function
go tool pprof -http=:8080 cpu.prof # CPU profile in browser
go tool pprof -http=:8080 mem.prof # Memory profile in browser
```

### Debug Environment
```bash
# Set debug environment variables
export TESTCONTAINERS_DEBUG=true
export REDIS_DEBUG=true

# Run tests with debug output
go test -tags=integration -v ./internal/storage/tests/... -run TestRedisStore_Integration_SaveAndGetAlertRule
```

## Best Practices

### Test Organization
1. **Separate Concerns**: Keep unit tests and integration tests separate
2. **Use Build Tags**: Properly tag tests for selective execution
3. **Clean State**: Ensure each test starts with clean state
4. **Parallel Safe**: Make tests safe for parallel execution
5. **Resource Cleanup**: Always cleanup containers and connections

### Data Management
1. **Use Fixtures**: Consistent test data across tests
2. **Randomize IDs**: Use unique IDs to avoid conflicts
3. **Test Boundaries**: Test edge cases and error conditions
4. **Validate Data**: Verify data integrity after operations

### Performance Considerations
1. **Benchmark Critical Paths**: Focus on high-impact operations
2. **Test Concurrency**: Verify behavior under concurrent access
3. **Memory Management**: Monitor memory usage patterns
4. **Connection Pooling**: Test connection pool behavior

### Error Handling
1. **Test Error Scenarios**: Network failures, data corruption, etc.
2. **Validate Error Messages**: Ensure errors are descriptive
3. **Recovery Testing**: Test recovery from various failure modes
4. **Timeout Handling**: Test behavior with various timeout values

### Documentation
1. **Document Test Cases**: Clear descriptions of what each test verifies
2. **Update README**: Keep documentation current with test changes
3. **Code Comments**: Explain complex test logic
4. **Performance Baselines**: Document expected performance characteristics

This comprehensive testing approach ensures the storage package is reliable, performant, and maintainable across different environments and use cases. 