# Models Package

This package contains the core data models and structures used throughout the Alert Engine application. It defines the foundational types for alerts, log entries, and their related configurations.

## Overview

The models package provides:
- **Alert Models**: `AlertRule`, `Alert`, `AlertStatus`, `AlertConditions`, `AlertActions`
- **Log Models**: `LogEntry`, `KubernetesInfo`, `LogFilter`, `LogStats`, `TimeWindow`
- Comprehensive JSON marshaling/unmarshaling support
- Validation logic for data structures
- Test fixtures for development and testing

## Data Structures

### Alert Models

#### AlertRule
Represents a rule for triggering alerts based on log conditions.

```go
type AlertRule struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Enabled     bool              `json:"enabled"`
    Conditions  AlertConditions   `json:"conditions"`
    Actions     AlertActions      `json:"actions"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}
```

#### Alert
Represents a triggered alert instance.

```go
type Alert struct {
    ID          string    `json:"id"`
    RuleID      string    `json:"rule_id"`
    RuleName    string    `json:"rule_name"`
    LogEntry    LogEntry  `json:"log_entry"`
    Timestamp   time.Time `json:"timestamp"`
    Severity    string    `json:"severity"`
    Status      string    `json:"status"`
    Message     string    `json:"message"`
    Count       int       `json:"count"`
}
```

### Log Models

#### LogEntry
Represents a structured log entry from OpenShift applications.

```go
type LogEntry struct {
    Timestamp     time.Time      `json:"timestamp"`
    Level         string         `json:"level"`
    Message       string         `json:"message"`
    Service       string         `json:"service"`
    Namespace     string         `json:"namespace"`
    Kubernetes    KubernetesInfo `json:"kubernetes"`
    Host          string         `json:"host"`
    Raw           string         `json:"raw,omitempty"`
}
```

#### KubernetesInfo
Contains Kubernetes-specific metadata from OpenShift logs.

```go
type KubernetesInfo struct {
    Namespace     string            `json:"namespace"`
    Pod           string            `json:"pod"`
    Container     string            `json:"container"`
    Labels        map[string]string `json:"labels"`
    // ... additional fields
}
```

### Helper Methods

The `LogEntry` struct provides helper methods for handling OpenShift log format variations:

#### GetNamespace()
Returns the namespace from the log entry, checking multiple possible fields in order:
1. Top-level `namespace` field
2. Top-level `namespace_name` field  
3. `kubernetes.namespace_name` field
4. `kubernetes.namespace` field (fallback)

#### GetPodName()
Returns the pod name, preferring `kubernetes.pod_name` over `kubernetes.pod`.

#### GetContainerName()
Returns the container name, preferring `kubernetes.container_name` over `kubernetes.container`.

These methods handle the field variations found in different OpenShift log formats and versions.

## Testing

The package includes comprehensive unit tests covering:

- JSON marshaling/unmarshaling for all data structures
- Helper method functionality (GetNamespace, GetPodName, GetContainerName)
- Validation logic for alert rules and conditions
- Edge cases and error handling
- Performance testing for concurrent access
- Timezone handling for timestamps

**Test Coverage: 100%** - All executable code paths are tested.

### Running Tests

To run the unit tests for the models package:

```bash
# Run all unit tests for models package
go test -tags unit ./pkg/models -v

# Run tests with coverage (shows 100% coverage)
go test -tags unit ./pkg/models -cover
# Output: coverage: 100.0% of statements

# Run tests and generate detailed coverage report
go test -tags unit ./pkg/models -coverprofile=coverage.out
go tool cover -func=coverage.out  # Shows function-level coverage
go tool cover -html=coverage.out -o coverage.html  # Generates HTML report

# Run specific test functions
go test -tags unit ./pkg/models -run TestAlertRule_JSONMarshaling -v

# Run helper method tests specifically  
go test -tags unit ./pkg/models -run TestLogEntry_Get -v

# Run benchmarks (if any)
go test -tags unit ./pkg/models -bench=. -v
```

## Test Fixtures

The package includes test fixtures in the `fixtures/` directory:

- `test_alerts.json`: Sample alert rules, alerts, and alert statuses
- `test_logs.json`: Sample log entries, filters, stats, and time windows

These fixtures can be used for:
- Development testing
- API testing
- Integration testing
- Documentation examples

### Using Test Fixtures

```bash
# View sample alert data
cat pkg/models/fixtures/test_alerts.json | jq '.alert_rules[0]'

# View sample log data
cat pkg/models/fixtures/test_logs.json | jq '.log_entries[0]'
```

## Validation

The models package provides validation for:

### AlertRule Validation
- **ID**: Must be non-empty
- **Name**: Must be non-empty
- **Threshold**: Must be positive
- **TimeWindow**: Must be positive duration
- **Operator**: Must be one of "gt", "gte", "lt", "eq"
- **Severity**: Must be one of "low", "medium", "high", "critical"

### LogEntry Validation
- **Timestamp**: Must be valid time
- **Level**: Common levels are "DEBUG", "INFO", "WARN", "ERROR", "FATAL"
- **Namespace**: Supports multiple field formats for OpenShift compatibility

## Usage Examples

### Creating an Alert Rule

```go
rule := AlertRule{
    ID:          "db-error-rule",
    Name:        "Database Error Alert",
    Description: "Alert on database connection errors",
    Enabled:     true,
    Conditions: AlertConditions{
        LogLevel:   "ERROR",
        Namespace:  "production",
        Keywords:   []string{"database", "connection", "failed"},
        Threshold:  3,
        TimeWindow: 5 * time.Minute,
        Operator:   "gt",
    },
    Actions: AlertActions{
        SlackWebhook: "https://hooks.slack.com/services/...",
        Channel:      "#alerts",
        Severity:     "high",
    },
}
```

### Processing Log Entries

```go
logEntry := LogEntry{
    Timestamp: time.Now(),
    Level:     "ERROR",
    Message:   "Database connection failed",
    Kubernetes: KubernetesInfo{
        NamespaceName: "production",  // OpenShift format
        PodName:       "user-service-abc123",
        ContainerName: "user-service",
        Labels: map[string]string{
            "app": "user-service",
            "version": "1.2.3",
        },
    },
}

// Helper methods handle field variations automatically
namespace := logEntry.GetNamespace()     // Returns "production"
podName := logEntry.GetPodName()         // Returns "user-service-abc123"
container := logEntry.GetContainerName() // Returns "user-service"
```

### Handling Different Log Formats

```go
// Legacy format
legacyLog := LogEntry{
    Namespace: "staging",
    Kubernetes: KubernetesInfo{
        Pod:       "api-service-xyz789",
        Container: "api-service",
    },
}

// Modern OpenShift format  
modernLog := LogEntry{
    Kubernetes: KubernetesInfo{
        NamespaceName: "staging",
        PodName:       "api-service-xyz789", 
        ContainerName: "api-service",
    },
}

// Both return the same values using helper methods
assert.Equal(t, legacyLog.GetNamespace(), modernLog.GetNamespace())
assert.Equal(t, legacyLog.GetPodName(), modernLog.GetPodName())
assert.Equal(t, legacyLog.GetContainerName(), modernLog.GetContainerName())
```

## Performance Considerations

- All structures support concurrent access for read operations
- JSON marshaling is optimized for frequent serialization
- Timestamp handling preserves timezone information
- Large label maps are supported efficiently

## Dependencies

This package has minimal external dependencies:
- Standard Go library (`time`, `encoding/json`)
- Testing dependencies: `github.com/stretchr/testify`

## Package Structure

```
pkg/models/
├── README.md              # This documentation
├── alert.go              # Alert-related data structures
├── log.go                # Log-related data structures
├── alert_test.go         # Unit tests for alert models
├── log_test.go           # Unit tests for log models
└── fixtures/             # Test data fixtures
    ├── test_alerts.json  # Sample alert data
    └── test_logs.json    # Sample log data
```

## Contributing

When adding new models or modifying existing ones:

1. Update the corresponding Go structures
2. Add or update JSON tags for serialization
3. Write comprehensive unit tests
4. Update test fixtures if needed
5. Document any validation rules
6. Ensure backward compatibility for API changes

## Version Compatibility

This package is designed to be compatible with:
- Go 1.19+
- OpenShift 4.x log formats
- Kubernetes metadata structures
- Standard JSON serialization patterns 