# Kafka Package

This package provides Kafka consumer functionality for processing log messages and integrating with the alert engine. It includes comprehensive unit and integration testing.

## Overview

The kafka package consists of:
- **Consumer**: Kafka message consumer with configurable settings
- **Processor**: Log message processor with batch processing capabilities
- **Factory**: Factory pattern for creating processors with different configurations
- **Mocks**: Mock implementations for testing

## Components

### Consumer (`consumer.go`)
Handles Kafka message consumption with the following features:
- Configurable consumer settings (brokers, topics, group ID, etc.)
- Health checks and statistics
- Graceful shutdown support
- Environment-based configuration

### Processor (`processor.go`)
Processes log messages from Kafka with:
- Single and batch processing modes
- Metrics tracking and error rate monitoring
- State store integration for log statistics
- Configurable retry logic and timeouts
- Health monitoring

### Factory Pattern
The ProcessorFactory provides a clean way to create processors:
- Standard log processors
- Batch log processors
- Configurable processing parameters

## Testing

### Unit Tests

Run unit tests for the kafka package:

```bash
# Run all unit tests
go test -tags=unit ./internal/kafka/ -v

# Run unit tests with coverage
go test -tags=unit ./internal/kafka/ -cover

# Run specific test
go test -tags=unit ./internal/kafka/ -run TestNewLogProcessor -v
```

### Integration Tests

Run integration tests with real Kafka containers:

```bash
# Run integration tests (requires Docker)
go test -tags=integration ./internal/kafka/ -v

# RECOMMENDED: Use the test script for better container management
./scripts/run_kafka_integration_tests.sh

# Run integration tests with race detection (SEQUENTIAL MODE REQUIRED)
./scripts/run_kafka_integration_tests.sh -m race-safe -r

# Alternative: Manual race detection (may have container conflicts)
go test -tags=integration ./internal/kafka/ -race -v -p 1
```

**Important Notes for Race Detection:**
- **Always use `-p 1` (sequential execution)** when running with `-race` flag
- Or use the provided script with `race-safe` mode
- Parallel execution with race detection causes container conflicts

**Container Management:**
- Tests create multiple Kafka containers (7 total across all test functions)
- Parallel execution can cause Docker port conflicts and resource contention
- Use sequential mode (`-p 1`) to avoid container startup/shutdown conflicts

### Kafka Integration Test Script

The package includes a specialized script for running integration tests with different execution modes:

```bash
# Show all available options
./scripts/run_kafka_integration_tests.sh -h

# Safe sequential execution (default, recommended)
./scripts/run_kafka_integration_tests.sh

# Parallel execution (faster but may have conflicts)
./scripts/run_kafka_integration_tests.sh -m parallel

# Race-safe mode with race detection
./scripts/run_kafka_integration_tests.sh -m race-safe -r

# Verbose output with Docker cleanup
./scripts/run_kafka_integration_tests.sh -v -d
```

**Execution Modes:**
- `sequential`: One test at a time, minimal resource conflicts (default)
- `parallel`: Multiple tests simultaneously, faster but may conflict
- `race-safe`: Sequential with optimizations for race detection

### Running All Tests

Use the project-wide test scripts:

```bash
# Run all unit tests across the project
./scripts/run_unit_tests.sh

# Run all integration tests with container dependencies
./scripts/run_integration_tests.sh

# Run Kafka integration tests specifically
./scripts/run_kafka_integration_tests.sh
```

## Usage Examples

### Basic Consumer

```go
package main

import (
    "context"
    "time"
    
    "github.com/log-monitoring/alert-engine/internal/kafka"
    "github.com/log-monitoring/alert-engine/internal/kafka/mocks"
)

func main() {
    // Create consumer configuration
    config := kafka.ConsumerConfig{
        Brokers:  []string{"localhost:9092"},
        Topic:    "application-logs", 
        GroupID:  "alert-engine",
        MinBytes: 1024,
        MaxBytes: 1048576,
        MaxWait:  2 * time.Second,
    }
    
    // Create alert engine (or use mock for testing)
    alertEngine := mocks.NewMockAlertEngine()
    
    // Create consumer
    consumer := kafka.NewConsumer(config, alertEngine)
    
    // Start consuming (blocks until context is cancelled)
    ctx := context.Background()
    err := consumer.Start(ctx)
    if err != nil {
        panic(err)
    }
}
```

### Log Processor with Batch Processing

```go
package main

import (
    "context"
    "time"
    
    "github.com/log-monitoring/alert-engine/internal/kafka"
    "github.com/log-monitoring/alert-engine/internal/kafka/mocks"
)

func main() {
    // Consumer configuration
    consumerConfig := kafka.ConsumerConfig{
        Brokers: []string{"localhost:9092"},
        Topic:   "application-logs",
        GroupID: "alert-engine",
        MinBytes: 1024,
        MaxBytes: 1048576,
        MaxWait:  2 * time.Second,
    }
    
    // Log processing configuration
    logProcessingConfig := kafka.LogProcessingConfig{
        BatchSize:       50,
        FlushInterval:   10 * time.Second,
        RetryAttempts:   3,
        RetryDelay:      1 * time.Second,
        EnableMetrics:   true,
        DefaultLogLevel: "INFO",
    }
    
    // Create mocks
    alertEngine := mocks.NewMockAlertEngine()
    stateStore := mocks.NewMockStateStore()
    
    // Create log processor
    processor := kafka.NewLogProcessor(consumerConfig, logProcessingConfig, alertEngine, stateStore)
    
    // Start processing logs
    ctx := context.Background()
    err := processor.ProcessLogs(ctx)
    if err != nil {
        panic(err)
    }
}
```

### Using Factory Pattern

```go
package main

import (
    "github.com/log-monitoring/alert-engine/internal/kafka"
    "github.com/log-monitoring/alert-engine/internal/kafka/mocks"
)

func main() {
    // Create factory with default configuration
    config := kafka.DefaultProcessorConfig()
    factory := kafka.NewProcessorFactory(config)
    
    // Create mocks
    alertEngine := mocks.NewMockAlertEngine()
    stateStore := mocks.NewMockStateStore()
    
    // Create processor using factory
    processor, err := factory.CreateProcessor(
        []string{"localhost:9092"},
        "application-logs",
        "alert-engine",
        alertEngine,
        stateStore,
    )
    if err != nil {
        panic(err)
    }
    
    // Create batch processor
    batchProcessor, err := factory.CreateBatchProcessor(
        []string{"localhost:9092"},
        "application-logs",
        "alert-engine",
        alertEngine,
        stateStore,
    )
    if err != nil {
        panic(err)
    }
    
    // Use processors...
}
```

## Configuration

### Default Configuration

The package provides sensible defaults:

```go
// Default consumer configuration
config := kafka.DefaultConsumerConfig()
// Brokers: ["127.0.0.1:9094"]
// Topic: "application-logs"
// GroupID: "alert-engine-e2e-fresh-20250716"
// MinBytes: 1024, MaxBytes: 1048576
// MaxWait: 2 seconds

// Default log processing configuration  
logConfig := kafka.DefaultLogProcessingConfig()
// BatchSize: 50
// FlushInterval: 10 seconds
// RetryAttempts: 3, RetryDelay: 1 second
// EnableMetrics: true
// DefaultLogLevel: "INFO"
```

### Environment Variables

Consumer configuration can be overridden with environment variables:

```bash
export KAFKA_BROKERS="broker1:9092,broker2:9092"
export KAFKA_TOPIC="custom-logs"
export KAFKA_GROUP_ID="custom-group"
```

Then use:

```go
config := kafka.DefaultConsumerConfigFromEnv()
```

## Monitoring and Health Checks

### Processor Metrics

```go
metrics := processor.GetMetrics()
fmt.Printf("Messages Processed: %d\n", metrics.MessagesProcessed)
fmt.Printf("Messages Failed: %d\n", metrics.MessagesFailure)
fmt.Printf("Error Rate: %.2f%%\n", metrics.ErrorRate*100)
fmt.Printf("Last Processed: %v\n", metrics.LastProcessed)
```

### Health Checks

```go
// Check processor health
healthy := processor.HealthCheck()
if !healthy {
    // Handle unhealthy state
}

// Check consumer health  
healthy = consumer.HealthCheck()
if !healthy {
    // Handle unhealthy state
}
```

## Testing Utilities

The package includes comprehensive mocks for testing:

### MockAlertEngine

```go
mockEngine := mocks.NewMockAlertEngine()

// Configure mock behavior
mockEngine.SetProcessingTime(50 * time.Millisecond)
mockEngine.SetShouldPanic(false)

// Verify calls
assert.True(t, mockEngine.WasCalled())
assert.Equal(t, 5, mockEngine.GetCallCount())

// Get evaluated logs
logs := mockEngine.GetEvaluatedLogs()
assert.Len(t, logs, 5)
```

### MockStateStore

```go
mockStore := mocks.NewMockStateStore()

// Test log statistics
stats, err := mockStore.GetLogStats()
assert.NoError(t, err)
assert.NotNil(t, stats)
```

## Best Practices

1. **Use Factory Pattern**: Prefer using ProcessorFactory for creating processors
2. **Configure Timeouts**: Set appropriate timeouts for your use case
3. **Monitor Metrics**: Regularly check processor metrics and health
4. **Graceful Shutdown**: Always handle context cancellation properly
5. **Error Handling**: Implement proper error handling and retry logic
6. **Testing**: Use the provided mocks for unit testing

## Dependencies

- `github.com/segmentio/kafka-go` - Kafka client library
- `github.com/testcontainers/testcontainers-go` - For integration testing
- `github.com/stretchr/testify` - Testing framework

## Error Handling

The package implements robust error handling:
- Connection failures are retried automatically
- Invalid messages are logged but don't stop processing
- Context cancellation is handled gracefully
- Metrics track error rates for monitoring

## Performance Considerations

- Batch processing reduces overhead for high-volume scenarios
- Configurable batch sizes and flush intervals
- Health checks detect performance degradation
- Error rate monitoring helps identify issues early

## Troubleshooting Integration Tests

### Container Start/Stop Issues

**Problem:** Containers keep starting and stopping when running `go test -tags=integration ./internal/kafka/ -race -v`

**Root Cause:** 
- Multiple Kafka containers (7 total) created simultaneously
- Race detector + parallel execution causes resource conflicts
- Docker port conflicts and container lifecycle race conditions

**Solutions:**
1. **Use the test script (recommended):**
   ```bash
   ./scripts/run_kafka_integration_tests.sh -m race-safe -r
   ```

2. **Force sequential execution:**
   ```bash
   go test -tags=integration ./internal/kafka/ -race -v -p 1
   ```

3. **Run without race detection first:**
   ```bash
   go test -tags=integration ./internal/kafka/ -v -p 1
   ```

### Docker Issues

**Problem:** `docker: command not found` during cleanup

**Cause:** Script tries to use `docker` command but you're using Podman or Docker is not in PATH

**Solution:** ✅ **FIXED** - The script now automatically detects and uses either `docker` or `podman`. No action needed - testcontainers handles cleanup automatically.

**Problem:** `unable to find network with name or ID bridge: network not found` (Podman)

**Cause:** Testcontainers expects a "bridge" network but Podman uses "podman" as the default bridge network

**Solutions:**
1. **Use the test script (recommended - auto-configures for Podman):**
   ```bash
   # The script automatically detects and configures Podman
   ./scripts/run_kafka_integration_tests.sh -m race-safe
   ```

2. **Manual Podman configuration (if needed):**
   ```bash
   # Set environment variables before running tests
   export TESTCONTAINERS_RYUK_DISABLED=true
   export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=unix:///run/podman/podman.sock
   export DOCKER_HOST=unix:///run/podman/podman.sock
   
   # Then run tests
   go test -tags=integration ./internal/kafka/ -v -p 1
   ```

3. **Use Docker instead of Podman:**
   ```bash
   # Install Docker Desktop and ensure Docker daemon is running
   # Then run tests normally
   ./scripts/run_kafka_integration_tests.sh
   ```

4. **Alternative Podman configuration:**
   ```bash
   # Enable Podman Docker compatibility socket
   systemctl --user enable podman.socket
   export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
   export TESTCONTAINERS_RYUK_DISABLED=true
   ```

**Problem:** Docker containers not cleaning up properly

**Solution:** Use the cleanup option:
```bash
./scripts/run_kafka_integration_tests.sh -d  # Clean before running
```

**Problem:** "Container creation failed" or port conflicts

**Solutions:**
1. Stop existing Kafka containers (works with docker or podman):
   ```bash
   # For Docker
   docker ps -a | grep kafka | awk '{print $1}' | xargs docker rm -f
   
   # For Podman  
   podman ps -a | grep kafka | awk '{print $1}' | xargs podman rm -f
   ```

2. Use sequential mode to avoid port conflicts:
   ```bash
   ./scripts/run_kafka_integration_tests.sh -m sequential
   ```

### Performance Issues

**Problem:** Tests are very slow

**Cause:** Each test function creates its own container (~30 seconds startup time)

**Optimization:** The updated tests use shared containers where possible to reduce startup overhead.

### Race Detection Issues

**Problem:** Race conditions detected in test code

**Solution:** The tests are designed to handle expected race conditions (like context cancellation). Use the `race-safe` mode which has better timeout handling.

**Problem:** Spurious race detection failures

**Solution:** Race detection with testcontainers can have timing issues. If tests pass without `-race` but fail with it, use:
```bash
./scripts/run_kafka_integration_tests.sh -m race-safe -r -v
```