# API Package

The `internal/api` package provides the HTTP REST API interface for the Alert Engine. It implements a comprehensive set of endpoints for managing alert rules, retrieving alerts, and monitoring system health.

## Architecture

This package follows a clean architecture pattern with:
- **Handlers**: HTTP request handlers implementing the REST API endpoints
- **Interfaces**: Abstract interfaces for dependency injection (StateStore, AlertEngine)
- **Middleware**: Cross-cutting concerns like CORS, logging, and error handling
- **Testing**: Comprehensive unit and integration test suites

## Package Structure

```
internal/api/
├── handlers.go          # Main HTTP handlers and API logic
├── routes.go           # Route definitions and middleware setup
├── handlers_test.go    # Unit tests for handlers
├── integration_test.go # Integration tests
├── mocks/             # Mock implementations for testing
│   ├── mock_alert_engine.go
│   └── mock_state_store.go
└── fixtures/          # Test data and fixtures
    ├── test_requests.json
    └── test_responses.json
```

## API Endpoints

### Health & System
- `GET /api/v1/health` - Health check endpoint
- `GET /api/v1/metrics` - System metrics
- `GET /api/v1/logs/stats` - Log processing statistics

### Alert Rules Management
- `GET /api/v1/rules` - Get all alert rules
- `GET /api/v1/rules/:id` - Get specific alert rule
- `POST /api/v1/rules` - Create new alert rule
- `PUT /api/v1/rules/:id` - Update existing alert rule
- `DELETE /api/v1/rules/:id` - Delete alert rule
- `GET /api/v1/rules/stats` - Get alert rules statistics
- `GET /api/v1/rules/template` - Get alert rule template
- `GET /api/v1/rules/defaults` - Get default alert rules
- `POST /api/v1/rules/bulk` - Create multiple alert rules
- `POST /api/v1/rules/reload` - Reload all alert rules from storage
- `POST /api/v1/rules/filter` - Filter alert rules by criteria
- `POST /api/v1/rules/test` - Test alert rule against sample logs

### Alerts
- `GET /api/v1/alerts/recent` - Get recent alerts (supports ?limit=N)

### Documentation
- `GET /` - API documentation and status
- `GET /docs` - Detailed API documentation

## Interfaces

### StateStore Interface
```go
type StateStore interface {
    SaveAlertRule(rule models.AlertRule) error
    GetAlertRules() ([]models.AlertRule, error)
    GetAlertRule(id string) (*models.AlertRule, error)
    DeleteAlertRule(id string) error
    SaveAlert(alert models.Alert) error
    GetRecentAlerts(limit int) ([]models.Alert, error)
    GetLogStats() (*models.LogStats, error)
    GetHealthStatus() (bool, error)
    GetMetrics() (map[string]interface{}, error)
    // ... other methods
}
```

### AlertEngine Interface
```go
type AlertEngine interface {
    AddRule(rule models.AlertRule) error
    UpdateRule(rule models.AlertRule) error
    DeleteRule(ruleID string) error
    GetRules() []models.AlertRule
    GetRule(ruleID string) (*models.AlertRule, error)
    ReloadRules() error
}
```

## Testing

This package includes comprehensive test coverage with both unit and integration tests.

### Unit Tests

Run unit tests for the API package:

```bash
# Run unit tests with verbose output
go test ./internal/api/... -v -tags=unit

# Run unit tests with coverage
go test ./internal/api/... -tags=unit -cover

# Run specific test
go test ./internal/api/... -tags=unit -run TestHandlers_Health -v
```

### Integration Tests

Run integration tests that test the complete HTTP server:

```bash
# Run integration tests
go test ./internal/api/... -v -tags=integration

# Run integration tests with race detection
go test ./internal/api/... -tags=integration -race -v
```

### Running All Tests

Use the provided scripts to run comprehensive test suites:

```bash
# Run all unit tests across the project
./scripts/run_unit_tests.sh

# Run all integration tests with container dependencies
./scripts/run_integration_tests.sh
```

## Test Coverage

The package maintains high test coverage across:

- **Unit Tests**: Test individual handler functions with mocked dependencies
- **Integration Tests**: Test complete HTTP workflows with real server instances
- **Error Scenarios**: Test error conditions and edge cases
- **Performance Tests**: Benchmark critical endpoints
- **Concurrent Tests**: Test thread safety and concurrent access

Current test coverage includes:
- ✅ Health endpoint testing
- ✅ CRUD operations for alert rules
- ✅ Alert retrieval and filtering
- ✅ System metrics and statistics
- ✅ Rule statistics and templates
- ✅ Default rules and bulk operations
- ✅ Rule testing and filtering
- ✅ Rule reloading functionality
- ✅ Error handling and validation
- ✅ CORS and security headers
- ✅ Concurrent request handling
- ✅ Large payload processing
- ✅ Rate limiting simulation

## Usage Example

### Creating a Handler Instance

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/log-monitoring/alert-engine/internal/api"
    "github.com/log-monitoring/alert-engine/internal/storage"
    "github.com/log-monitoring/alert-engine/internal/alerting"
)

func main() {
    // Initialize dependencies
    store := storage.NewRedisStore(/* config */)
    engine := alerting.NewEngine(/* config */)
    
    // Create handlers
    handlers := api.NewHandlers(store, engine)
    
    // Setup routes
    router := gin.New()
    handlers.SetupRoutes(router)
    
    // Start server
    router.Run(":8080")
}
```

### Making API Requests

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Get all rules
curl http://localhost:8080/api/v1/rules

# Create a new rule
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Error Rate",
    "pattern": "ERROR",
    "threshold": 10,
    "window": "5m",
    "enabled": true
  }'

# Get recent alerts
curl http://localhost:8080/api/v1/alerts/recent?limit=10

# Get rule statistics
curl http://localhost:8080/api/v1/rules/stats

# Get rule template
curl http://localhost:8080/api/v1/rules/template

# Get default rules
curl http://localhost:8080/api/v1/rules/defaults

# Test a rule against sample logs
curl -X POST http://localhost:8080/api/v1/rules/test \
  -H "Content-Type: application/json" \
  -d '{
    "rule": {
      "name": "Test Rule",
      "conditions": {"log_level": "ERROR", "threshold": 1}
    },
    "sample_logs": [
      {"level": "ERROR", "message": "Test error", "timestamp": "2023-01-01T00:00:00Z"}
    ]
  }'

# Bulk create rules
curl -X POST http://localhost:8080/api/v1/rules/bulk \
  -H "Content-Type: application/json" \
  -d '[
    {"name": "Rule 1", "conditions": {"log_level": "ERROR"}},
    {"name": "Rule 2", "conditions": {"log_level": "WARN"}}
  ]'

# Reload all rules
curl -X POST http://localhost:8080/api/v1/rules/reload

# Filter rules
curl -X POST http://localhost:8080/api/v1/rules/filter \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "severity": "high"}'
```

## Development

### Adding New Endpoints

1. Add handler method to `handlers.go`
2. Register route in `routes.go`
3. Add corresponding tests in `handlers_test.go`
4. Update integration tests if needed

### Running Tests During Development

```bash
# Watch mode for unit tests (requires entr or similar)
find . -name "*.go" | entr -r go test ./internal/api/... -tags=unit -v

# Quick test during development
go test ./internal/api/... -tags=unit -run TestHandlers_YourNewTest -v
```

## Dependencies

- **Gin**: HTTP web framework
- **Testify**: Testing utilities and assertions
- **Standard Library**: HTTP, JSON, and context handling

## Build Tags

The package uses build tags to separate test types:
- `//go:build unit` - Unit tests with mocked dependencies
- `//go:build integration` - Integration tests with real server instances

## Performance Characteristics

- **Health Endpoint**: ~21,000 requests/second (measured on local machine)
- **Rule CRUD Operations**: Sub-millisecond response times with proper caching
- **Alert Retrieval**: Optimized for recent alerts with configurable limits
- **Concurrent Safety**: All handlers are thread-safe and support concurrent access

## Error Handling

The API uses standardized error responses:

```json
{
  "success": false,
  "error": "Detailed error message",
  "message": "User-friendly message"
}
```

All endpoints return appropriate HTTP status codes and structured error responses for consistent client handling. 