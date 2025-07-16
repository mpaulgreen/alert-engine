# Alert Engine - Alerting Package

## Overview

The `internal/alerting` package provides the core alert processing functionality for the log monitoring system. It includes the main alert engine, rule evaluation logic, and supporting components for managing alert rules and their execution.

## Package Structure

```
internal/alerting/
├── engine.go           # Main alert engine implementation
├── evaluator.go        # Rule evaluation logic
├── rules.go           # Rule management utilities
├── engine_test.go      # Engine unit tests
├── evaluator_test.go   # Evaluator unit tests
├── rules_test.go       # Rules unit tests
├── fixtures/           # Test data files
│   ├── test_logs.json
│   └── test_rules.json
└── mocks/             # Mock implementations for testing
    ├── mock_notifier.go
    └── mock_state_store.go
```

## Key Components

### Engine (`engine.go`)
The main `AlertEngine` handles:
- Loading and managing alert rules
- Processing incoming log entries
- Triggering alerts when conditions are met
- Managing alert state and persistence

### Evaluator (`evaluator.go`)
The `Evaluator` component provides:
- Rule condition evaluation against log entries
- Threshold-based alert triggering
- Performance tracking for rule evaluations
- Batch processing capabilities

### Rules (`rules.go`)
Rule management utilities including:
- Rule validation
- Default rule creation
- Rule filtering and statistics
- ID generation from rule names

## Testing

### Unit Tests

Run unit tests for the alerting package:

```bash
# From the project root (alert-engine/)
go test ./internal/alerting/... -v -tags=unit
```

Run unit tests with coverage:

```bash
go test ./internal/alerting/... -v -tags=unit -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Using Test Scripts

Run all unit tests using the provided script:

```bash
# From the project root (alert-engine/)
./scripts/run_unit_tests.sh
```

The script provides colored output and runs tests for all packages with proper build tags.

### Test Structure

- **Unit Tests**: Located alongside source files (`*_test.go`)
- **Test Data**: JSON fixtures in `fixtures/` directory
- **Mocks**: Mock implementations in `mocks/` directory
- **Build Tags**: Tests use `//go:build unit` tag

## Usage Examples

### Creating an Alert Engine

```go
package main

import (
    "github.com/log-monitoring/alert-engine/internal/alerting"
    "github.com/log-monitoring/alert-engine/internal/storage"
    "github.com/log-monitoring/alert-engine/internal/notifications"
)

func main() {
    // Initialize dependencies
    stateStore := storage.NewRedisStateStore(redisClient)
    notifier := notifications.NewSlackNotifier(webhookURL)
    
    // Create alert engine
    engine := alerting.NewEngine(stateStore, notifier)
    
    // Process logs
    engine.EvaluateLog(logEntry)
}
```

### Adding Alert Rules

```go
rule := models.AlertRule{
    ID:   "high-error-rate",
    Name: "High Error Rate Alert",
    Conditions: models.AlertConditions{
        LogLevel:   "ERROR",
        Threshold:  10,
        TimeWindow: 5 * time.Minute,
        Operator:   "gt",
    },
    Actions: models.AlertActions{
        Severity: "high",
    },
    Enabled: true,
}

err := engine.AddRule(rule)
```

## Dependencies

### Internal Dependencies
- `pkg/models`: Data models for alerts and logs
- `internal/storage`: State persistence interfaces
- `internal/notifications`: Alert notification interfaces

### External Dependencies
- `github.com/stretchr/testify`: Testing assertions and mocks
- Standard library packages for JSON, time, logging, etc.

## Configuration

The alerting package accepts configuration through:
- Rule definitions (JSON format)
- Engine initialization parameters
- Environment-specific settings

## Development

### Running Tests During Development

```bash
# Run specific test functions
go test ./internal/alerting -v -tags=unit -run TestEngine_EvaluateLog

# Run with verbose output and race detection
go test ./internal/alerting -v -tags=unit -race

# Generate and view coverage report
go test ./internal/alerting -v -tags=unit -coverprofile=alerting.out
go tool cover -func=alerting.out
```

### Adding New Tests

1. Create test files with `_test.go` suffix
2. Add `//go:build unit` build tag
3. Use `package alerting_test` to avoid import cycles
4. Place test data in `fixtures/` directory
5. Create mocks in `mocks/` directory if needed

### Debugging Tests

Enable verbose logging in tests:

```bash
go test ./internal/alerting -v -tags=unit
```

## Performance Considerations

- The engine uses in-memory rule storage for fast evaluation
- Batch processing is available for high-volume scenarios
- Performance metrics are tracked per rule
- Consider using the performance tracker for monitoring

## Error Handling

The package includes comprehensive error handling for:
- Rule validation failures
- State store connectivity issues
- Notification delivery failures
- Malformed log entries

## Contributing

When contributing to this package:

1. Run all tests: `./scripts/run_unit_tests.sh`
2. Ensure coverage remains high (>90%)
3. Add tests for new functionality
4. Update this README for significant changes
5. Follow Go coding standards and conventions 