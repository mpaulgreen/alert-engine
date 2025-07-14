# Unit Tests for pkg/models Package

This directory contains comprehensive unit tests for the `pkg/models` package of the Alert Engine system. The tests cover all data models including JSON serialization/deserialization, field validation, time handling, and edge cases.

## Overview

The unit tests are organized into the following test files:

- **`alert_test.go`** - Tests for alert-related models (`AlertRule`, `Alert`, `AlertStatus`, `AlertConditions`, `AlertActions`)
- **`log_test.go`** - Tests for log-related models (`LogEntry`, `KubernetesInfo`, `LogFilter`, `LogStats`, `TimeWindow`)
- **`fixtures/`** - Test data fixtures in JSON format for consistent testing

## Test Coverage

### Alert Models (`alert_test.go`)

1. **AlertRule JSON Marshaling/Unmarshaling**
   - Complete field preservation
   - Missing optional fields handling
   - Invalid JSON handling

2. **Alert JSON Marshaling/Unmarshaling**
   - Nested structure handling
   - Empty nested structures
   - Complex LogEntry embedding

3. **AlertStatus JSON Marshaling/Unmarshaling**
   - Time handling
   - Status validation

4. **AlertConditions Validation**
   - Valid log levels (DEBUG, INFO, WARN, ERROR, FATAL)
   - Valid operators (gt, lt, eq, contains)
   - Time window parsing (1m, 5m, 1h, 30s)

5. **AlertActions Validation**
   - Valid severity levels (low, medium, high, critical)
   - Slack webhook URL format validation

6. **Edge Cases**
   - Empty structures
   - Nil maps
   - Zero timestamps

### Log Models (`log_test.go`)

1. **LogEntry JSON Marshaling/Unmarshaling**
   - Complete field preservation
   - Missing optional fields
   - Empty Kubernetes info
   - Invalid timestamp handling

2. **KubernetesInfo JSON Marshaling/Unmarshaling**
   - Labels map handling
   - Nil labels
   - Empty labels object

3. **LogFilter JSON Marshaling/Unmarshaling**
   - Time range filtering
   - Omitted fields
   - Empty keyword arrays

4. **LogStats JSON Marshaling/Unmarshaling**
   - Map aggregations
   - Empty maps
   - Null maps

5. **TimeWindow JSON Marshaling/Unmarshaling**
   - Time range handling
   - Zero times

6. **Log Level Validation**
   - Valid log levels
   - Case sensitivity

7. **Timestamp Formatting**
   - RFC3339 format
   - Timezone handling

8. **Edge Cases**
   - Empty log entries
   - Very long messages
   - Special characters and Unicode
   - Large number of labels
   - Concurrent access patterns

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

### Run All Model Tests

```bash
# Run all tests in the models package
go test -tags=unit -v ./pkg/models/tests/...

# Run with coverage
go test -tags=unit -v -cover ./pkg/models/tests/...

# Generate detailed coverage report
go test -tags=unit -v -coverprofile=coverage.out ./pkg/models/tests/...
go tool cover -html=coverage.out -o coverage.html
```

### Run Specific Test Files

```bash
# Run only alert model tests
go test -tags=unit -v ./pkg/models/tests/alert_test.go

# Run only log model tests
go test -tags=unit -v ./pkg/models/tests/log_test.go
```

### Run Specific Test Functions

```bash
# Run specific test function
go test -tags=unit -v -run TestAlertRule_JSONMarshaling ./pkg/models/tests/...

# Run all JSON marshaling tests
go test -tags=unit -v -run ".*JSONMarshaling.*" ./pkg/models/tests/...

# Run all validation tests
go test -tags=unit -v -run ".*Validation.*" ./pkg/models/tests/...
```

### Run with Different Verbosity Levels

```bash
# Basic run
go test -tags=unit ./pkg/models/tests/...

# Verbose output
go test -tags=unit -v ./pkg/models/tests/...

# Short mode (skip long-running tests)
go test -tags=unit -short ./pkg/models/tests/...
```

## Test Categories

The tests are organized into logical categories:

### 1. JSON Marshaling/Unmarshaling Tests
- Verify that all models can be correctly serialized to JSON and deserialized back
- Test preservation of all fields including nested structures
- Handle missing optional fields gracefully

### 2. Field Validation Tests
- Test valid values for enum-like fields (log levels, operators, severities)
- Verify proper handling of time formats
- Test keyword arrays and label maps

### 3. Edge Case Tests
- Empty structures
- Nil pointers and maps
- Zero timestamps
- Very long strings
- Special characters and Unicode
- Large data sets

### 4. Error Handling Tests
- Invalid JSON formats
- Invalid field values
- Malformed timestamps

### 5. Concurrent Access Tests
- Thread-safety of marshaling/unmarshaling
- Multiple goroutines accessing same data

## Test Fixtures

The `fixtures/` directory contains JSON test data:

- **`test_alerts.json`** - Sample alert rules, alerts, and alert statuses
- **`test_logs.json`** - Sample log entries, filters, stats, and time windows

These fixtures provide consistent test data and can be used for:
- Integration testing
- Manual testing
- API testing
- Performance testing

## Example Test Run Output

```bash
$ go test -v ./pkg/models/tests/...

=== RUN   TestAlertRule_JSONMarshaling
=== RUN   TestAlertRule_JSONMarshaling/successful_marshal_and_unmarshal
=== RUN   TestAlertRule_JSONMarshaling/unmarshal_with_missing_optional_fields
=== RUN   TestAlertRule_JSONMarshaling/unmarshal_with_invalid_JSON
--- PASS: TestAlertRule_JSONMarshaling (0.00s)
    --- PASS: TestAlertRule_JSONMarshaling/successful_marshal_and_unmarshal (0.00s)
    --- PASS: TestAlertRule_JSONMarshaling/unmarshal_with_missing_optional_fields (0.00s)
    --- PASS: TestAlertRule_JSONMarshaling/unmarshal_with_invalid_JSON (0.00s)

=== RUN   TestAlert_JSONMarshaling
=== RUN   TestAlert_JSONMarshaling/successful_marshal_and_unmarshal
=== RUN   TestAlert_JSONMarshaling/unmarshal_with_empty_nested_structures
--- PASS: TestAlert_JSONMarshaling (0.00s)
    --- PASS: TestAlert_JSONMarshaling/successful_marshal_and_unmarshal (0.00s)
    --- PASS: TestAlert_JSONMarshaling/unmarshal_with_empty_nested_structures (0.00s)

...

PASS
ok      github.com/log-monitoring/alert-engine/pkg/models/tests    0.012s
```

## Performance Testing

For performance testing, you can use benchmarks:

```bash
# Run benchmark tests
go test -tags=unit -bench=. ./pkg/models/tests/...

# Run benchmarks with memory profiling
go test -tags=unit -bench=. -benchmem ./pkg/models/tests/...
```

## Integration with CI/CD

These tests are designed to be run in CI/CD pipelines:

```bash
# CI-friendly test run
go test -tags=unit -v -race -cover ./pkg/models/tests/...

# Generate coverage for CI
go test -tags=unit -v -race -coverprofile=coverage.out -covermode=atomic ./pkg/models/tests/...
```

## Test Data Validation

The test fixtures in the `fixtures/` directory are validated against the actual models to ensure they represent realistic data structures. This helps catch:

- Schema changes that break existing data
- Missing fields in test data
- Invalid field values
- Inconsistent data relationships

## Debugging Tests

If tests fail, you can debug them using:

```bash
# Run tests with detailed output
go test -tags=unit -v -failfast ./pkg/models/tests/...

# Run a specific failing test
go test  -tags=unit -v -run TestSpecificFailingTest ./pkg/models/tests/...
```

## Contributing

When adding new model fields or modifying existing ones:

1. **Update the tests** to cover the new fields
2. **Update the fixtures** with examples of the new data
3. **Add validation tests** for any new constraints
4. **Test edge cases** for the new functionality
5. **Update this README** if new test categories are added

## Best Practices

1. **Use table-driven tests** for testing multiple similar scenarios
2. **Use descriptive test names** that clearly indicate what is being tested
3. **Test both success and failure cases** for each function
4. **Use fixtures** for consistent test data
5. **Keep tests focused** - each test should verify one specific behavior
6. **Use proper assertions** with meaningful error messages
7. **Clean up resources** if tests create any temporary data

## Related Documentation

- [Models Documentation](../README.md) - Overview of the data models
- [API Testing](../../../internal/api/tests/README.md) - API endpoint testing
- [Integration Testing](../../../tests/integration/README.md) - End-to-end testing 