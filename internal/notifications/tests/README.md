# Notifications Package Testing Guide

This document provides comprehensive guidance for testing the `internal/notifications` package, including Slack notification functionality, message formatting, and integration testing with mock HTTP servers.

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
go test -tags=unit -v ./internal/notifications/tests/...

# Run with coverage
go test -tags=unit -cover ./internal/notifications/tests/...
```

### Integration Tests (Mock HTTP Server)
```bash
# Run integration tests with mock HTTP server
go test -tags=integration -v ./internal/notifications/tests/... -timeout=3m

# Run with verbose output
go test -tags=integration -v ./internal/notifications/tests/... -timeout=3m
```

### Using Test Scripts
```bash
# Unit tests (always works)
./scripts/run_unit_tests.sh

# Integration tests (uses mock HTTP server)
./scripts/run_integration_tests.sh
```

### Expected Results
- **Unit Tests**: ~25 tests, all should pass in <10 seconds
- **Integration Tests**: ~30 test cases, all should pass in <30 seconds
- **Performance**: Tests should handle 50+ notifications/second

## Overview

The Notifications package test suite provides comprehensive coverage for:
- **Slack Notifier**: Message formatting, webhook sending, configuration, error handling
- **Interface Compliance**: Tests that implementations follow the Notifier interface
- **Message Formatting**: Rich Slack message formatting with attachments and fields
- **Error Handling**: Network failures, HTTP errors, rate limiting, timeout scenarios
- **Configuration Validation**: Different configuration scenarios and validation
- **Integration Testing**: Real HTTP requests to mock Slack servers

## Test Structure

```
internal/notifications/tests/
├── mocks/
│   ├── mock_http_client.go       # Mock HTTP client for unit tests
│   └── mock_http_server.go       # Mock HTTP server for integration tests
├── fixtures/
│   └── test_alerts.json          # Sample alerts and test scenarios
├── slack_test.go                 # Slack notifier unit tests
├── interfaces_test.go            # Interface and data structure tests
├── integration_test.go           # Integration tests with mock HTTP server
└── README.md                     # This file
```

## Prerequisites

### For Unit Tests
- Go 1.19+
- No external dependencies required

### For Integration Tests
- Go 1.19+
- No external dependencies required (uses mock HTTP server)

### Required Dependencies
```bash
# Install testing dependencies
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require
```

## Running Tests

### Unit Tests (No Dependencies Required)
```bash
# Run unit tests only (recommended for development)
go test -tags=unit -v ./internal/notifications/tests/...

# Run unit tests with coverage
go test -tags=unit -cover ./internal/notifications/tests/...

# Run unit tests with race detection
go test -tags=unit -race ./internal/notifications/tests/...
```

### Integration Tests (Mock HTTP Server)
```bash
# Run integration tests only
go test -tags=integration -v ./internal/notifications/tests/...

# Run integration tests with short timeout
go test -tags=integration -short -v ./internal/notifications/tests/...
```

### All Tests
```bash
# Run all tests (unit + integration)
go test -tags=unit,integration -v ./internal/notifications/tests/...

# Run with verbose output
go test -tags=unit,integration -v ./internal/notifications/tests/...

# Run with coverage
go test -tags=unit,integration -cover ./internal/notifications/tests/...
```

### Specific Test Categories
```bash
# Run Slack notifier tests only
go test -tags=unit -v ./internal/notifications/tests/ -run TestSlackNotifier

# Run interface tests only
go test -tags=unit -v ./internal/notifications/tests/ -run TestNotification

# Run integration tests only
go test -tags=integration -v ./internal/notifications/tests/ -run TestSlackNotifierIntegration

# Run specific test cases
go test -tags=unit -v ./internal/notifications/tests/ -run TestNewSlackNotifier
go test -tags=unit -v ./internal/notifications/tests/ -run TestSlackNotifier_SendAlert
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
```

#### Integration Test Script
```bash
# Run integration tests using the provided script
./scripts/run_integration_tests.sh

# Run integration tests with container logs
./scripts/run_integration_tests.sh --logs
```

### Parallel Testing
```bash
# Run tests in parallel (default)
go test -tags=unit -v -parallel 4 ./internal/notifications/tests/...

# Run tests sequentially
go test -tags=unit -v -parallel 1 ./internal/notifications/tests/...
```

### With Race Detection
```bash
# Run tests with race detection
go test -tags=unit -race ./internal/notifications/tests/...
```

## Test Categories

### 1. Slack Notifier Tests (`slack_test.go`)

#### Basic Slack Operations
- **Notifier Creation**: Tests `NewSlackNotifier()` with various configurations
- **Message Sending**: Tests `SendAlert()` functionality with different alerts
- **Error Handling**: Tests disabled notifier, invalid configurations
- **Configuration**: Tests `SetConfig()`, `GetConfig()`, custom settings
- **Validation**: Tests `ValidateConfig()` with valid/invalid configurations

#### Message Formatting
- **Severity Handling**: Tests emoji and color mapping for different severities
- **Alert Formatting**: Tests rich message formatting with attachments and fields
- **Service Extraction**: Tests service name extraction from Kubernetes labels
- **Message Limits**: Tests message length limits and truncation

#### Custom Messages
- **Simple Messages**: Tests `SendSimpleMessage()` functionality
- **Alert Summaries**: Tests `CreateAlertSummary()` with multiple alerts
- **Custom Messages**: Tests `SendCustomMessage()` with custom payloads

#### Configuration Management
- **Default Configuration**: Tests `DefaultNotificationConfig()` values
- **Custom Configuration**: Tests configuration setting and retrieval
- **Channel Settings**: Tests custom channel, username, icon settings

### 2. Interface Tests (`interfaces_test.go`)

#### Data Structure Tests
- **NotificationChannel**: Tests channel creation and JSON marshaling
- **NotificationTemplate**: Tests template structure and variables
- **NotificationResult**: Tests success/failure result tracking
- **NotificationHistory**: Tests historical notification tracking
- **NotificationStats**: Tests statistics collection and reporting

#### Configuration Tests
- **NotificationConfig**: Tests configuration structure and defaults
- **NotificationFilter**: Tests filtering criteria and time ranges
- **QueueItem**: Tests notification queue item structure
- **TimeRange**: Tests time range validation and JSON handling

#### Interface Compliance
- **Notifier Interface**: Tests that SlackNotifier implements Notifier interface
- **Method Availability**: Tests all required interface methods are available

### 3. Integration Tests (`integration_test.go`)

#### Mock HTTP Server Testing
- **Successful Requests**: Tests successful HTTP requests to mock Slack server
- **Error Handling**: Tests various HTTP error scenarios (400, 403, 404, 500)
- **Rate Limiting**: Tests 429 rate limit response handling
- **Timeout Handling**: Tests request timeout scenarios

#### Message Validation
- **JSON Structure**: Tests that sent messages have correct JSON structure
- **Required Fields**: Tests presence of required Slack webhook fields
- **Attachment Structure**: Tests rich attachment formatting
- **Field Validation**: Tests field structure and content

#### Scenario Testing
- **Multiple Alerts**: Tests sending multiple alerts sequentially
- **Custom Configuration**: Tests custom channel, username, icon settings
- **Message Formatting**: Tests different severity levels and formatting
- **Kubernetes Info**: Tests Kubernetes information formatting

## Mock Components

### MockHTTPClient (`mocks/mock_http_client.go`)

A comprehensive mock implementation of HTTP client for testing Slack requests.

#### Features:
- **Request Tracking**: Captures all HTTP requests for inspection
- **Response Simulation**: Configurable HTTP responses with status codes
- **Error Simulation**: Configurable failure modes for testing error handling
- **Timing Control**: Configurable delays and timeouts
- **Slack Scenarios**: Pre-configured Slack-specific test scenarios

#### Usage:
```go
// Create mock HTTP client
mockClient := mocks.NewMockHTTPClient()

// Set up success scenario
mockClient.SetupSlackScenario("success")

// Set up error scenario
mockClient.SetupSlackScenario("server_error")

// Set up rate limiting
mockClient.SetupSlackScenario("rate_limit")

// Configure custom response
mockClient.SetStatusCode(http.StatusOK)
mockClient.SetResponseBody("ok")

// Configure timeout
mockClient.SetSimulateTimeout(true)

// Reset mock state
mockClient.Reset()
```

#### Available Methods:
- `SetupSlackScenario(scenario)` - Configure predefined Slack scenarios
- `SetStatusCode(code)` - Set HTTP status code
- `SetResponseBody(body)` - Set response body
- `SetResponseDelay(delay)` - Set response delay
- `SetSimulateTimeout(bool)` - Enable/disable timeout simulation
- `GetCallCount()` - Get number of requests made
- `GetLastRequestBody()` - Get last request body
- `Reset()` - Reset mock state

### MockSlackServer (`mocks/mock_http_server.go`)

A mock HTTP server that simulates Slack webhook endpoints for integration testing.

#### Features:
- **Real HTTP Server**: Uses httptest.Server for realistic HTTP handling
- **Request Validation**: Validates incoming webhook requests
- **Response Scenarios**: Configurable response scenarios
- **Request Tracking**: Captures all requests for inspection
- **Webhook Simulation**: Simulates real Slack webhook behavior

#### Usage:
```go
// Create mock Slack server
mockServer := mocks.NewMockSlackServer()
defer mockServer.Close()

// Set up success scenario
mockServer.SetupSlackScenario("success")

// Set up error scenario
mockServer.SetupSlackScenario("server_error")

// Get webhook URL
webhookURL := mockServer.GetWebhookURL()

// Validate webhook payload
err := mockServer.ValidateSlackWebhookPayload(requestBody)

// Check requests
assert.Equal(t, 1, mockServer.GetCallCount())
lastRequest := mockServer.GetLastRequest()
```

#### Available Methods:
- `SetupSlackScenario(scenario)` - Configure predefined scenarios
- `GetWebhookURL()` - Get mock webhook URL
- `ValidateSlackWebhookPayload(body)` - Validate webhook payload
- `GetCallCount()` - Get number of requests received
- `GetLastRequest()` - Get last HTTP request
- `GetLastRequestBody()` - Get last request body
- `Reset()` - Reset server state

## Test Fixtures

### Test Alerts (`fixtures/test_alerts.json`)

Contains comprehensive test data for different testing scenarios:

#### Sample Alerts
- **Critical Alerts**: High-severity alerts with full context
- **Various Severities**: Alerts with different severity levels
- **Kubernetes Context**: Alerts with detailed Kubernetes information
- **Error Scenarios**: Alerts for testing error handling

#### Slack Messages
- **Formatted Messages**: Expected Slack message structures
- **Test Messages**: Connection test messages
- **Custom Messages**: Various message format examples

#### Configuration Examples
- **Valid Configurations**: Working notification configurations
- **Invalid Configurations**: Configurations for testing validation
- **Custom Settings**: Various channel and user settings

#### Test Scenarios
- **Single Alert**: Basic alert sending scenarios
- **Multiple Alerts**: Batch alert processing scenarios
- **Error Handling**: Network error and retry scenarios
- **Rate Limiting**: Rate limit testing scenarios

### Configuration Examples
```json
{
  "configurations": [
    {
      "name": "default_config",
      "webhook_url": "https://hooks.slack.com/services/T123456/B123456/XXXXXXXXXXXXXXXXXXXXXXXX",
      "channel": "#alerts",
      "username": "Alert Engine",
      "icon_emoji": ":warning:",
      "enabled": true
    }
  ]
}
```

## Configuration Testing

### Slack Notifier Configuration Tests

```go
// Test valid configuration
notifier := notifications.NewSlackNotifier(webhookURL)
notifier.SetChannel("#alerts")
notifier.SetUsername("Alert Bot")
notifier.SetIconEmoji(":warning:")

config := notifications.NotificationConfig{
    Enabled:     true,
    MaxRetries:  3,
    RetryDelay:  5 * time.Second,
    Timeout:     30 * time.Second,
}
notifier.SetConfig(config)

// Test configuration validation
err := notifier.ValidateConfig()
assert.NoError(t, err)
```

### Invalid Configuration Tests

```go
// Test invalid webhook URL
notifier := notifications.NewSlackNotifier("invalid-url")
err := notifier.ValidateConfig()
assert.Error(t, err)

// Test invalid channel
notifier.SetChannel("invalid-channel")
err = notifier.ValidateConfig()
assert.Error(t, err)
```

## Performance Testing

### Throughput Testing
```bash
# Run performance benchmarks
go test -tags=unit,integration -bench=. ./internal/notifications/tests/...

# Run with memory profiling
go test -tags=unit,integration -bench=. -memprofile=mem.prof ./internal/notifications/tests/...

# Run with CPU profiling
go test -tags=unit,integration -bench=. -cpuprofile=cpu.prof ./internal/notifications/tests/...
```

### Load Testing
```go
// Example load testing setup
func TestHighVolumeNotifications(t *testing.T) {
    mockServer := mocks.NewMockSlackServer()
    defer mockServer.Close()
    
    mockServer.SetupSlackScenario("success")
    notifier := notifications.NewSlackNotifier(mockServer.GetWebhookURL())
    
    // Create many test alerts
    alerts := make([]models.Alert, 100)
    for i := range alerts {
        alerts[i] = createTestAlert(i)
    }
    
    // Send alerts
    start := time.Now()
    for _, alert := range alerts {
        notifier.SendAlert(alert)
    }
    duration := time.Since(start)
    
    // Verify performance
    assert.Less(t, duration, 2*time.Second)
    assert.Equal(t, 100, mockServer.GetCallCount())
}
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Notifications Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.19
      
      - name: Run Notifications Tests
        run: |
          go test -tags=unit,integration -v -race -cover ./internal/notifications/tests/...
          
      - name: Run Integration Tests
        run: |
          go test -v -tags=integration ./internal/notifications/tests/...
```

### Test Coverage
```bash
# Generate coverage report
go test -tags=unit,integration -coverprofile=coverage.out ./internal/notifications/tests/...

# View coverage report
go tool cover -html=coverage.out

# Coverage threshold check
go test -tags=unit,integration -cover ./internal/notifications/tests/... | grep -E "coverage: [0-9]+\.[0-9]+%" | awk '{print $2}' | grep -E "^[89][0-9]\.[0-9]+%|^100\.0%"
```

## Debugging and Troubleshooting

### Common Issues

#### 1. Mock Configuration Issues
```go
// Problem: Mock not behaving as expected
mockServer := mocks.NewMockSlackServer()
mockServer.SetupSlackScenario("success")

// Solution: Verify mock state
assert.Equal(t, 1, mockServer.GetCallCount())
requests := mockServer.GetRequests()
assert.Len(t, requests, 1)
```

#### 2. JSON Parsing Issues
```go
// Problem: JSON structure validation failing
lastBody := mockServer.GetLastRequestBody()

// Solution: Debug JSON structure
var payload map[string]interface{}
err := json.Unmarshal([]byte(lastBody), &payload)
if err != nil {
    t.Logf("JSON parsing error: %v", err)
    t.Logf("Raw body: %s", lastBody)
}
```

#### 3. HTTP Request Issues
```go
// Problem: HTTP requests not being captured
mockServer := mocks.NewMockSlackServer()
notifier := notifications.NewSlackNotifier(mockServer.GetWebhookURL())

// Solution: Verify URL and setup
t.Logf("Webhook URL: %s", mockServer.GetWebhookURL())
t.Logf("Server URL: %s", mockServer.GetURL())
```

#### 4. Configuration Validation Issues
```go
// Problem: Configuration validation failing unexpectedly
notifier := notifications.NewSlackNotifier(webhookURL)

// Solution: Debug configuration step by step
t.Logf("Webhook URL: %s", webhookURL)
t.Logf("Channel: %s", notifier.GetChannel())
t.Logf("Username: %s", notifier.GetUsername())

err := notifier.ValidateConfig()
if err != nil {
    t.Logf("Validation error: %v", err)
}
```

### Debug Logging
```go
// Enable debug logging in tests
func TestWithDebugLogging(t *testing.T) {
    // Create mock server with debug info
    mockServer := mocks.NewMockSlackServer()
    defer mockServer.Close()
    
    t.Logf("Mock server URL: %s", mockServer.GetURL())
    t.Logf("Mock webhook URL: %s", mockServer.GetWebhookURL())
    
    // Your test code here
    
    // Log captured requests
    requests := mockServer.GetRequests()
    for i, req := range requests {
        t.Logf("Request %d: %s %s", i, req.Method, req.URL.Path)
    }
}
```

### Test Isolation
```go
// Ensure tests don't interfere with each other
func TestIsolatedTest(t *testing.T) {
    // Create fresh mock server for each test
    mockServer := mocks.NewMockSlackServer()
    defer mockServer.Close()
    
    // Reset state if needed
    mockServer.Reset()
    
    // Your test code here
    
    // Verify clean state
    assert.Equal(t, 0, mockServer.GetCallCount())
}
```

## Best Practices

### 1. Test Organization
- Group related tests using `t.Run()` subtests
- Use descriptive test names that explain the scenario
- Keep tests focused on single functionality
- Use table-driven tests for similar scenarios

### 2. Mock Usage
- Always close mock servers after tests
- Configure mocks to match real Slack behavior
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

### 6. Integration Testing
- Use mock HTTP servers for realistic testing
- Test actual HTTP request/response cycles
- Validate message structure and content
- Test error scenarios with proper HTTP status codes

## Example Test Scenarios

### Basic Notification Test
```go
func TestBasicNotification(t *testing.T) {
    // Setup
    mockServer := mocks.NewMockSlackServer()
    defer mockServer.Close()
    
    mockServer.SetupSlackScenario("success")
    notifier := notifications.NewSlackNotifier(mockServer.GetWebhookURL())
    
    // Test
    alert := models.Alert{
        ID:       "test-001",
        RuleName: "Test Alert",
        Severity: "critical",
    }
    
    err := notifier.SendAlert(alert)
    
    // Verify
    assert.NoError(t, err)
    assert.Equal(t, 1, mockServer.GetCallCount())
    
    lastBody := mockServer.GetLastRequestBody()
    assert.Contains(t, lastBody, "Test Alert")
}
```

### Error Handling Test
```go
func TestErrorHandling(t *testing.T) {
    // Setup
    mockServer := mocks.NewMockSlackServer()
    defer mockServer.Close()
    
    mockServer.SetupSlackScenario("server_error")
    notifier := notifications.NewSlackNotifier(mockServer.GetWebhookURL())
    
    // Test
    alert := models.Alert{
        ID:       "test-002",
        RuleName: "Error Test",
        Severity: "high",
    }
    
    err := notifier.SendAlert(alert)
    
    // Verify
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "500")
    assert.Equal(t, 1, mockServer.GetCallCount())
}
```

## Known Limitations and Notes

### Integration Tests
- **Mock Server**: Tests use mock HTTP server, not real Slack webhooks
- **Network Simulation**: Tests simulate network conditions but don't test actual network issues
- **Timing**: Tests use controlled timing and may not reflect real-world latency

### Test Environment
- **HTTP Server**: Each integration test creates its own mock HTTP server
- **Port Allocation**: Tests use dynamic port allocation to avoid conflicts
- **Cleanup**: Mock servers are automatically cleaned up after tests

### Performance Considerations
- **Parallel Execution**: Unit tests run in parallel by default
- **Mock Overhead**: Mock servers add minimal overhead compared to real HTTP calls
- **Test Speed**: Integration tests are fast due to local mock servers

### Build Tags
- **Unit Tests**: Use `//go:build unit` tag - run with `-tags=unit`
- **Integration Tests**: Use `//go:build integration` tag - run with `-tags=integration`
- **Default**: Running without tags includes both unit and integration tests

### Slack Integration
- **Webhook Format**: Tests validate Slack webhook message format
- **Message Limits**: Tests respect Slack message size limits
- **Rich Formatting**: Tests use Slack-specific formatting (attachments, fields, colors)

### Troubleshooting Quick Reference
1. **Unit tests failing**: Check Go version and dependencies
2. **Integration tests timing out**: Increase timeout with `-timeout=5m`
3. **Mock server issues**: Verify server setup and URL configuration
4. **JSON validation errors**: Check message structure and required fields

---

For more information about the Notifications package implementation, see the main [README.md](../README.md) file.

For questions or issues, please refer to the project documentation or create an issue in the repository. 