# API Package Testing Guide

This document provides comprehensive guidance for testing the `internal/api` package, including HTTP handlers, routing, middleware, and complete API workflow integration testing.

## Table of Contents

- [Quick Start](#quick-start)
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Test Categories](#test-categories)
- [Mock Components](#mock-components)
- [Test Fixtures](#test-fixtures)
- [API Endpoints Testing](#api-endpoints-testing)
- [Performance Testing](#performance-testing)
- [Error Handling Testing](#error-handling-testing)
- [CI/CD Integration](#cicd-integration)
- [Debugging and Troubleshooting](#debugging-and-troubleshooting)
- [Best Practices](#best-practices)

## Quick Start

### Unit Tests Only (Recommended for Development)
```bash
# Run all unit tests - no dependencies required
go test -tags=unit -v ./internal/api/tests/...

# Run with coverage
go test -tags=unit -cover ./internal/api/tests/...
```

### Integration Tests (HTTP Server Testing)
```bash
# Run integration tests with HTTP server
go test -tags=integration -v ./internal/api/tests/... -timeout=5m

# Run with verbose output
go test -tags=integration -v ./internal/api/tests/... -timeout=5m
```

### Using Test Scripts
```bash
# Unit tests (always works)
./scripts/run_unit_tests.sh

# Integration tests (uses HTTP server)
./scripts/run_integration_tests.sh
```

### Expected Results
- **Unit Tests**: ~35 tests, all should pass in <10 seconds
- **Integration Tests**: ~34 test cases, all should pass in <60 seconds
- **All Tests**: ~69 tests (35 unit + 34 integration), all should pass in <60 seconds
- **Performance**: Tests should handle 100+ requests/second

### Important Notes
- **Build Tags Required**: All test files use build tags (`//go:build unit` or `//go:build integration`)
- **Commands without build tags will fail** with "no packages to test" error
- **Always specify build tags** when running tests manually

## Overview

The API package test suite provides comprehensive coverage for:
- **HTTP Handlers**: Request processing, response formatting, error handling
- **Routing**: URL routing, parameter extraction, middleware execution
- **CRUD Operations**: Create, Read, Update, Delete operations for alert rules
- **Authentication**: CORS headers, OPTIONS preflight handling
- **System Monitoring**: Health checks, metrics, log statistics
- **Error Scenarios**: Invalid requests, not found, server errors
- **Performance**: Concurrent requests, throughput, resource management

## Test Structure

```
internal/api/tests/
├── mocks/
│   ├── mock_state_store.go      # Mock StateStore implementation
│   └── mock_alert_engine.go     # Mock AlertEngine implementation
├── fixtures/
│   ├── test_requests.json       # Sample HTTP requests
│   └── test_responses.json      # Sample HTTP responses
├── handlers_test.go             # HTTP handlers unit tests
├── integration_test.go          # Integration tests with HTTP server
└── README.md                    # This file
```

## Prerequisites

### For Unit Tests
- Go 1.19+
- No external dependencies required

### For Integration Tests
- Go 1.19+
- No external dependencies required (uses HTTP test server)

### Required Dependencies
```bash
# Install testing dependencies
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require
go get github.com/stretchr/testify/suite
go get github.com/gin-gonic/gin
```

## Running Tests

### Unit Tests (No Dependencies Required)
```bash
# Run unit tests only (recommended for development)
go test -tags=unit -v ./internal/api/tests/...

# Run unit tests with coverage
go test -tags=unit -cover ./internal/api/tests/...

# Run unit tests with race detection
go test -tags=unit -race ./internal/api/tests/...
```

### Integration Tests (HTTP Server Testing)
```bash
# Run integration tests only
go test -tags=integration -v ./internal/api/tests/...

# Run integration tests with timeout
go test -tags=integration -v ./internal/api/tests/... -timeout=5m
```

### All Tests
```bash
# Run all tests (unit + integration) - use both build tags
go test -tags="unit integration" -v ./internal/api/tests/...

# Run with verbose output
go test -tags="unit integration" -v ./internal/api/tests/...

# Run with coverage
go test -tags="unit integration" -cover ./internal/api/tests/...
```

### Specific Test Categories
```bash
# Run handlers tests only
go test -tags=unit -v ./internal/api/tests/ -run TestHandlers

# Run CRUD workflow tests
go test -tags=integration -v ./internal/api/tests/ -run TestRulesCRUD

# Run health check tests
go test -tags=unit -v ./internal/api/tests/ -run TestHandlers_Health

# Run specific test cases
go test -tags=unit -v ./internal/api/tests/ -run TestHandlers_CreateRule
go test -tags=integration -v ./internal/api/tests/ -run TestAPIIntegrationSuite
```

### Using Test Scripts

#### Unit Test Script
```bash
# Run unit tests using the provided script
./scripts/run_unit_tests.sh

# Run unit tests with coverage
./scripts/run_unit_tests.sh --coverage
```

#### Integration Test Script
```bash
# Run integration tests using the provided script
./scripts/run_integration_tests.sh

# Run integration tests with logs
./scripts/run_integration_tests.sh --logs
```

**Note**: The integration test script may fail due to Docker container startup timeouts (especially Kafka). 
If this happens, use the direct Go test commands instead:
```bash
# Use this instead if script fails
go test -tags=integration -v ./internal/api/tests/... -timeout=5m
```

### Parallel Testing
```bash
# Run tests in parallel (default)
go test -tags=unit -v -parallel 4 ./internal/api/tests/...

# Run tests sequentially
go test -tags=unit -v -parallel 1 ./internal/api/tests/...
```

## Test Categories

### 1. HTTP Handlers Tests (`handlers_test.go`)

#### Health Check Handler
- **Health Status**: Tests `/api/v1/health` endpoint with healthy/unhealthy states
- **Response Format**: Tests JSON response structure and status codes
- **Error Handling**: Tests health check failures and error responses

#### Rule Management Handlers
- **Get Rules**: Tests `GET /api/v1/rules` with different scenarios
- **Get Rule**: Tests `GET /api/v1/rules/{id}` with valid/invalid IDs
- **Create Rule**: Tests `POST /api/v1/rules` with valid/invalid data
- **Update Rule**: Tests `PUT /api/v1/rules/{id}` with various updates
- **Delete Rule**: Tests `DELETE /api/v1/rules/{id}` with cleanup validation

#### System Monitoring Handlers
- **Recent Alerts**: Tests `GET /api/v1/alerts/recent` with pagination
- **Log Statistics**: Tests `GET /api/v1/system/logs/stats` endpoint
- **System Metrics**: Tests `GET /api/v1/system/metrics` endpoint

#### CORS and Middleware
- **CORS Headers**: Tests Cross-Origin Resource Sharing configuration
- **OPTIONS Requests**: Tests preflight request handling
- **Request Validation**: Tests content-type and payload validation

### 2. Integration Tests (`integration_test.go`)

#### Complete CRUD Workflow
- **End-to-End**: Tests complete Create→Read→Update→Delete workflow
- **Data Persistence**: Tests data consistency across operations
- **Error Propagation**: Tests error handling throughout the workflow

#### HTTP Server Testing
- **Real HTTP Requests**: Tests with actual HTTP client and server
- **Status Codes**: Tests proper HTTP status code responses
- **Headers**: Tests response headers and content types
- **Timeouts**: Tests request timeout handling

#### Performance and Load Testing
- **Concurrent Requests**: Tests handling of multiple simultaneous requests
- **Throughput**: Tests requests per second capability
- **Resource Management**: Tests memory and connection cleanup
- **Large Payloads**: Tests handling of large request bodies

#### Error Scenario Testing
- **Invalid JSON**: Tests malformed request handling
- **Not Found**: Tests 404 responses for missing resources
- **Method Not Allowed**: Tests 405 responses for invalid methods
- **Server Errors**: Tests 500 responses for internal failures

## Mock Components

### MockStateStore (`mocks/mock_state_store.go`)
- **Thread-Safe**: Uses sync.RWMutex for concurrent access
- **Configurable Failures**: Can simulate various error conditions
- **Data Management**: Supports CRUD operations for rules and alerts
- **Statistics**: Provides mock log statistics and metrics

#### Key Features:
- Rule storage and retrieval
- Alert management
- Health status simulation
- Counter and status tracking
- Failure simulation controls

### MockAlertEngine (`mocks/mock_alert_engine.go`)
- **Rule Management**: Supports full rule lifecycle operations
- **Validation**: Includes rule validation logic
- **Statistics**: Provides rule statistics and filtering
- **Test Helpers**: Includes utilities for creating test data

#### Key Features:
- Rule CRUD operations
- Rule validation and testing
- Statistics generation
- Sample data creation
- Filtering and search

## Test Fixtures

### Request Fixtures (`fixtures/test_requests.json`)
- **Valid Requests**: Complete valid rule creation/update requests
- **Invalid Requests**: Malformed data for error testing
- **Edge Cases**: Boundary conditions and special scenarios
- **Bulk Operations**: Multiple rule operations

### Response Fixtures (`fixtures/test_responses.json`)
- **Success Responses**: Expected responses for successful operations
- **Error Responses**: Expected error response formats
- **Data Structures**: Complex nested response structures
- **Metadata**: Response metadata and pagination info

## API Endpoints Testing

### Alert Rules API (`/api/v1/rules`)
```bash
# Test all rule endpoints
curl -X GET http://localhost:8080/api/v1/rules
curl -X POST http://localhost:8080/api/v1/rules -d @test_rule.json
curl -X GET http://localhost:8080/api/v1/rules/rule-id
curl -X PUT http://localhost:8080/api/v1/rules/rule-id -d @updated_rule.json
curl -X DELETE http://localhost:8080/api/v1/rules/rule-id
```

### System Monitoring API (`/api/v1/system`)
```bash
# Test system endpoints
curl -X GET http://localhost:8080/api/v1/health
curl -X GET http://localhost:8080/api/v1/system/metrics
curl -X GET http://localhost:8080/api/v1/system/logs/stats
```

### Alerts API (`/api/v1/alerts`)
```bash
# Test alerts endpoints
curl -X GET http://localhost:8080/api/v1/alerts/recent
curl -X GET http://localhost:8080/api/v1/alerts/recent?limit=10
```

## Performance Testing

### Throughput Testing
```bash
# Test API throughput
go test -tags=integration -v ./internal/api/tests/ -run TestBenchmarkEndpoints

# Expected results:
# - Health endpoint: >100 req/s
# - CRUD operations: >50 req/s
# - System endpoints: >75 req/s
```

### Concurrent Testing
```bash
# Test concurrent request handling
go test -tags=integration -v ./internal/api/tests/ -run TestConcurrentRequests

# Expected results:
# - 10 concurrent requests: All succeed
# - No race conditions
# - Proper resource cleanup
```

### Memory and Resource Testing
```bash
# Test resource management
go test -tags=integration -v ./internal/api/tests/ -run TestResourceManagement

# Expected results:
# - No memory leaks
# - Proper connection cleanup
# - Graceful resource deallocation
```

## Error Handling Testing

### HTTP Error Codes
- **400 Bad Request**: Invalid JSON, validation errors
- **404 Not Found**: Missing resources, invalid IDs
- **405 Method Not Allowed**: Invalid HTTP methods
- **500 Internal Server Error**: Server-side failures
- **503 Service Unavailable**: Health check failures

### Error Response Format
```json
{
  "success": false,
  "error": "Detailed error message",
  "message": "Optional user-friendly message"
}
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: API Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.19'
      - name: Run API Unit Tests
        run: go test -tags=unit -v ./internal/api/tests/...
      - name: Run API Integration Tests
        run: go test -tags=integration -v ./internal/api/tests/...
```

### Test Coverage Requirements
- **Unit Tests**: >90% code coverage
- **Integration Tests**: >85% endpoint coverage
- **Error Paths**: >80% error scenario coverage

## Debugging and Troubleshooting

### Common Issues and Solutions

#### 1. Test Failures
```bash
# Check test output for specific failures
go test -tags=unit -v ./internal/api/tests/... | grep FAIL

# Run specific failing test
go test -tags=unit -v ./internal/api/tests/ -run TestHandlers_CreateRule
```

#### 2. Mock Setup Issues
```bash
# Verify mock initialization
go test -tags=unit -v ./internal/api/tests/ -run TestHandlers_Health

# Check mock state between tests
# Ensure proper cleanup in SetupTest methods
```

#### 3. HTTP Server Issues
```bash
# Check server startup
go test -tags=integration -v ./internal/api/tests/ -run TestHealthEndpoint

# Verify port availability
netstat -an | grep :8080
```

#### 4. Race Conditions
```bash
# Run with race detector
go test -tags=unit -race ./internal/api/tests/...

# Check for concurrent access issues
go test -tags=integration -race ./internal/api/tests/...
```

### Test Debugging Commands
```bash
# Enable verbose logging
go test -tags=unit -v ./internal/api/tests/... -test.v

# Add debug output to specific tests
go test -tags=unit -v ./internal/api/tests/ -run TestHandlers_CreateRule -test.v

# Check test timing
go test -tags=unit -v ./internal/api/tests/... -test.timeout=30s
```

### Mock Debugging
```bash
# Check mock state
# Add debug prints in test setup
# Verify mock method calls
# Validate mock responses
```

## Best Practices

### 1. Test Organization
- **Separate Concerns**: Unit tests for handlers, integration tests for workflows
- **Clear Naming**: Use descriptive test names that explain the scenario
- **Setup/Teardown**: Proper test isolation and cleanup

### 2. Mock Usage
- **Realistic Data**: Use realistic test data that matches production scenarios
- **Error Simulation**: Test both success and failure paths
- **State Management**: Ensure mocks are properly reset between tests

### 3. HTTP Testing
- **Status Codes**: Always verify HTTP status codes
- **Response Bodies**: Validate response structure and content
- **Headers**: Check required headers (CORS, Content-Type, etc.)

### 4. Performance Testing
- **Baseline Metrics**: Establish performance baselines
- **Load Testing**: Test with realistic request volumes
- **Resource Monitoring**: Monitor memory and connection usage

### 5. Error Testing
- **All Error Paths**: Test every possible error condition
- **Error Messages**: Verify error messages are helpful
- **Error Codes**: Use consistent error response formats

### 6. Integration Testing
- **End-to-End Workflows**: Test complete user journeys
- **Data Consistency**: Verify data integrity across operations
- **External Dependencies**: Use mocks for external services

## Example Test Scenarios

### Creating a Complete Test Rule
```go
func TestCompleteRuleWorkflow(t *testing.T) {
    // 1. Create rule
    rule := models.AlertRule{
        Name:        "Test Rule",
        Description: "Test rule description",
        Enabled:     true,
        Conditions: models.AlertConditions{
            LogLevel:   "ERROR",
            Threshold:  5,
            TimeWindow: "5m",
            Operator:   "gt",
        },
        Actions: models.AlertActions{
            SlackWebhook: "https://hooks.slack.com/services/test",
            Channel:      "#alerts",
            Severity:     "high",
        },
    }
    
    // 2. Test creation
    // 3. Test retrieval
    // 4. Test updates
    // 5. Test deletion
}
```

### Testing Error Scenarios
```go
func TestErrorHandling(t *testing.T) {
    // Test invalid JSON
    // Test missing required fields
    // Test invalid data types
    // Test server errors
    // Test not found scenarios
}
```

### Performance Testing
```go
func TestPerformance(t *testing.T) {
    // Test concurrent requests
    // Test throughput
    // Test resource usage
    // Test cleanup
}
```

This comprehensive testing approach ensures the API package is robust, performant, and reliable in production environments. 