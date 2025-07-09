# Unit Tests for internal/alerting Package

This directory contains comprehensive unit tests for the `internal/alerting` package of the Alert Engine system. The tests cover all major components including the alert engine, evaluator, and rules management with proper mocking and isolation.

## Overview

The unit tests are organized into the following test files:

- **`engine_test.go`** - Tests for the main alert engine (`Engine`)
- **`evaluator_test.go`** - Tests for the alert evaluator (`Evaluator`, `PerformanceTracker`, `BatchEvaluator`)
- **`rules_test.go`** - Tests for rule management utilities and validation
- **`mocks/`** - Mock implementations for external dependencies
- **`fixtures/`** - Test data fixtures for consistent testing

## Test Coverage

### Engine Tests (`engine_test.go`)

1. **Engine Creation and Initialization**
   - Engine creation with dependencies
   - Rule loading on startup
   - Error handling during initialization

2. **Log Evaluation**
   - Matching rules against log entries
   - Threshold evaluation and alert triggering
   - Disabled rule handling
   - Condition matching (log level, namespace, service, keywords)
   - Counter management and time windows
   - Error handling (store failures, notifier failures)

3. **Rule Management**
   - Adding new rules
   - Updating existing rules
   - Deleting rules
   - Reloading rules from store
   - Timestamp management

4. **Engine Control**
   - Graceful shutdown
   - Error recovery

### Evaluator Tests (`evaluator_test.go`)

1. **Condition Evaluation**
   - Log level matching
   - Namespace filtering
   - Service filtering (via app labels)
   - Keyword matching (case-insensitive, multiple keywords)
   - Complex condition combinations
   - Empty condition handling

2. **Threshold Evaluation**
   - Different operators (gt, gte, lt, lte, eq)
   - Default operator behavior
   - Counter incrementation
   - Store error handling

3. **Rule Testing**
   - Testing rules against sample logs
   - Match rate calculation
   - Trigger probability assessment
   - Empty log handling

4. **Performance Tracking**
   - Evaluation timing
   - Match rate tracking
   - Per-rule metrics
   - Metric aggregation

5. **Batch Processing**
   - Batch evaluation of multiple logs
   - Context cancellation handling
   - Parallel processing

### Rules Tests (`rules_test.go`)

1. **Rule Validation**
   - Required field validation (ID, name)
   - Threshold validation (positive values)
   - Time window validation
   - Operator validation (valid operators)
   - Severity validation (valid levels)
   - Edge cases and error messages

2. **Default Rule Creation**
   - Default rule generation
   - Rule validity
   - Proper configuration
   - Timestamp setting

3. **Rule Templates**
   - Template structure
   - Default values
   - Validity of templates

4. **Rule Statistics**
   - Count aggregations
   - Status distribution
   - Namespace/service distribution
   - Severity distribution
   - Empty rule handling

5. **Rule Filtering**
   - Filter by enabled status
   - Filter by namespace
   - Filter by service
   - Filter by severity
   - Filter by log level
   - Multiple criteria filtering
   - Empty result handling

6. **Utility Functions**
   - Rule ID generation
   - String sanitization
   - Special character handling

## Mock Components

### MockStateStore (`mocks/mock_state_store.go`)

Provides a complete mock implementation of the `StateStore` interface:

- **Rule Management**: Save, get, delete alert rules
- **Counter Management**: Increment and get counters with time windows
- **Alert Status**: Track alert status and timestamps
- **Error Simulation**: Configurable failure modes for testing error scenarios
- **Thread Safety**: Concurrent access support with proper locking
- **Test Utilities**: Helper methods for test setup and verification

### MockNotifier (`mocks/mock_notifier.go`)

Provides a complete mock implementation of the `Notifier` interface:

- **Alert Sending**: Track sent alerts with full details
- **Connection Testing**: Mock connection validation
- **Error Simulation**: Configurable failure modes
- **Alert Tracking**: Count and filter sent alerts
- **Test Utilities**: Helper methods for test verification

## Test Fixtures

### Rule Fixtures (`fixtures/test_rules.json`)

Contains sample alert rules for testing:

- **Valid Rules**: Various rule configurations for positive testing
- **Invalid Rules**: Rules with validation errors for negative testing
- **Filter Examples**: Rules for testing filtering functionality
- **Edge Cases**: Rules with boundary conditions

### Log Fixtures (`fixtures/test_logs.json`)

Contains sample log entries for testing:

- **Matching Logs**: Logs that should trigger various rules
- **Non-matching Logs**: Logs that should not trigger rules
- **Batch Logs**: Multiple logs for batch processing tests
- **Edge Case Logs**: Logs with special characters, empty fields, etc.
- **Performance Logs**: Logs for performance testing

## Prerequisites

Before running the tests, ensure you have:

1. **Go 1.21+** installed
2. **Dependencies** installed:
   ```bash
   go mod tidy
   ```

## Running the Tests

### Install Dependencies

First, make sure all dependencies are installed:

```bash
cd alert-engine
go mod tidy
```

### Run All Alerting Tests

```bash
# Run all tests in the alerting package
go test -v ./internal/alerting/tests/...

# Run with coverage
go test -v -cover ./internal/alerting/tests/...

# Generate detailed coverage report
go test -v -coverprofile=coverage.out ./internal/alerting/tests/...
go tool cover -html=coverage.out -o coverage.html
```

### Run Specific Test Files

```bash
# Run only engine tests
go test -v ./internal/alerting/tests/engine_test.go ./internal/alerting/tests/mocks/*.go

# Run only evaluator tests
go test -v ./internal/alerting/tests/evaluator_test.go ./internal/alerting/tests/mocks/*.go

# Run only rules tests
go test -v ./internal/alerting/tests/rules_test.go
```

### Run Specific Test Functions

```bash
# Run specific test function
go test -v -run TestEngine_EvaluateLog ./internal/alerting/tests/...

# Run all engine tests
go test -v -run "TestEngine_.*" ./internal/alerting/tests/...

# Run all validation tests
go test -v -run ".*Validation.*" ./internal/alerting/tests/...
```

### Run with Different Verbosity Levels

```bash
# Basic run
go test ./internal/alerting/tests/...

# Verbose output
go test -v ./internal/alerting/tests/...

# Short mode (skip long-running tests)
go test -short ./internal/alerting/tests/...

# Run with race detection
go test -race ./internal/alerting/tests/...
```

## Test Categories

### 1. Unit Tests
- Test individual functions and methods in isolation
- Use mocks for all external dependencies
- Focus on business logic correctness

### 2. Integration Tests
- Test interaction between components
- Use real interfaces but mock external systems
- Verify data flow and error propagation

### 3. Error Handling Tests
- Test failure scenarios
- Verify proper error messages
- Ensure graceful degradation

### 4. Edge Case Tests
- Test boundary conditions
- Handle empty/null inputs
- Verify special character handling

### 5. Performance Tests
- Benchmark critical paths
- Test with realistic data volumes
- Monitor memory usage

## Example Test Execution

```bash
$ go test -v ./internal/alerting/tests/...

=== RUN   TestNewEngine
=== RUN   TestNewEngine/creates_engine_successfully
=== RUN   TestNewEngine/loads_existing_rules_on_startup
=== RUN   TestNewEngine/handles_store_error_during_rule_loading
--- PASS: TestNewEngine (0.00s)
    --- PASS: TestNewEngine/creates_engine_successfully (0.00s)
    --- PASS: TestNewEngine/loads_existing_rules_on_startup (0.00s)
    --- PASS: TestNewEngine/handles_store_error_during_rule_loading (0.00s)

=== RUN   TestEngine_EvaluateLog
=== RUN   TestEngine_EvaluateLog/evaluates_log_against_matching_rule
=== RUN   TestEngine_EvaluateLog/skips_disabled_rules
=== RUN   TestEngine_EvaluateLog/does_not_trigger_alert_when_conditions_don't_match
=== RUN   TestEngine_EvaluateLog/does_not_trigger_alert_when_threshold_not_met
=== RUN   TestEngine_EvaluateLog/handles_notifier_error
=== RUN   TestEngine_EvaluateLog/handles_counter_increment_error
--- PASS: TestEngine_EvaluateLog (0.01s)
    --- PASS: TestEngine_EvaluateLog/evaluates_log_against_matching_rule (0.00s)
    --- PASS: TestEngine_EvaluateLog/skips_disabled_rules (0.00s)
    --- PASS: TestEngine_EvaluateLog/does_not_trigger_alert_when_conditions_don't_match (0.00s)
    --- PASS: TestEngine_EvaluateLog/does_not_trigger_alert_when_threshold_not_met (0.00s)
    --- PASS: TestEngine_EvaluateLog/handles_notifier_error (0.00s)
    --- PASS: TestEngine_EvaluateLog/handles_counter_increment_error (0.00s)

...

PASS
ok      github.com/log-monitoring/alert-engine/internal/alerting/tests 0.045s
```

## Test Data Usage

### Using Fixtures

The test fixtures can be loaded and used in tests:

```go
func TestWithFixtures(t *testing.T) {
    // Load test rules from fixture
    var rulesData struct {
        TestRules []models.AlertRule `json:"test_rules"`
    }
    
    fixtureData, err := os.ReadFile("fixtures/test_rules.json")
    require.NoError(t, err)
    
    err = json.Unmarshal(fixtureData, &rulesData)
    require.NoError(t, err)
    
    // Use rules in test
    for _, rule := range rulesData.TestRules {
        err := alerting.ValidateRule(rule)
        assert.NoError(t, err)
    }
}
```

### Mock Configuration

Configure mocks for different test scenarios:

```go
func TestErrorScenario(t *testing.T) {
    mockStore := mocks.NewMockStateStore()
    mockNotifier := mocks.NewMockNotifier()
    
    // Configure store to fail
    mockStore.SetShouldFail(true, "database connection failed")
    
    engine := alerting.NewEngine(mockStore, mockNotifier)
    
    // Test error handling
    err := engine.AddRule(testRule)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "database connection failed")
}
```

## Performance Testing

For performance testing, use benchmarks:

```bash
# Run benchmark tests
go test -bench=. ./internal/alerting/tests/...

# Run benchmarks with memory profiling
go test -bench=. -benchmem ./internal/alerting/tests/...

# Profile specific functions
go test -bench=BenchmarkEvaluateLog -cpuprofile=cpu.prof ./internal/alerting/tests/...
```

## Integration with CI/CD

These tests are designed to run in CI/CD pipelines:

```bash
# CI-friendly test run
go test -v -race -cover ./internal/alerting/tests/...

# Generate coverage for CI
go test -v -race -coverprofile=coverage.out -covermode=atomic ./internal/alerting/tests/...
```

## Debugging Tests

If tests fail, debug using:

```bash
# Run tests with detailed output
go test -v -failfast ./internal/alerting/tests/...

# Run a specific failing test
go test -v -run TestSpecificFailingTest ./internal/alerting/tests/...

# Enable detailed logging
GOLOG_v=2 go test -v ./internal/alerting/tests/...
```

## Best Practices

1. **Use Mocks**: Always use mocks for external dependencies
2. **Test Isolation**: Each test should be independent
3. **Error Testing**: Test both success and failure cases
4. **Edge Cases**: Test boundary conditions and edge cases
5. **Clear Names**: Use descriptive test names
6. **Setup/Teardown**: Clean up after tests
7. **Parallel Tests**: Use `t.Parallel()` where appropriate
8. **Table-Driven Tests**: Use table-driven tests for multiple scenarios

## Contributing

When adding new functionality:

1. **Add Tests**: Write tests for new features
2. **Update Mocks**: Extend mocks if needed
3. **Update Fixtures**: Add new test data as needed
4. **Document**: Update this README for new test categories
5. **Coverage**: Maintain high test coverage

## Related Documentation

- [Alerting Package](../README.md) - Overview of the alerting system
- [Models Testing](../../pkg/models/tests/README.md) - Model unit tests
- [Integration Testing](../../../tests/integration/README.md) - End-to-end testing
- [Testing Structure](../../../TESTING_STRUCTURE.md) - Overall testing strategy 