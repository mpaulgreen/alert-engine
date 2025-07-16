# Notifications Package

The notifications package provides interfaces and implementations for sending alert notifications through various channels (currently Slack). It includes comprehensive message formatting, configuration management, and robust testing infrastructure.

## Overview

This package consists of:
- **Core interfaces** (`interfaces.go`) - Define notification contracts and data structures
- **Slack implementation** (`slack.go`) - Full-featured Slack notifier with templating support
- **Test infrastructure** - Unit tests, integration tests, and mocks for thorough testing
- **Fixtures and mocks** - Test data and mock HTTP clients/servers

## Key Features

### Slack Notifier
- **Rich message formatting** with attachments, fields, and colors
- **Configurable templates** for alert messages and fields
- **Severity-based styling** with emojis and colors
- **Enable/disable functionality** with state management
- **Comprehensive validation** of webhook URLs and configuration
- **Multiple message types**: simple text, rich attachments, custom messages
- **Alert summarization** for multiple alerts

### Configuration Management
- **Global notification settings** with severity mappings
- **Per-notifier configuration** with full customization
- **Default configurations** for quick setup
- **Validation and error handling**

## Quick Start

### Basic Slack Notifier Setup

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/log-monitoring/alert-engine/internal/notifications"
    "github.com/log-monitoring/alert-engine/pkg/models"
)

func main() {
    // Create a Slack notifier
    webhookURL := "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
    notifier := notifications.NewSlackNotifier(webhookURL)
    
    // Create an alert
    alert := models.Alert{
        ID:        "alert-001",
        RuleName:  "High Error Rate",
        Severity:  "critical",
        Count:     15,
        Timestamp: time.Now(),
        LogEntry: models.LogEntry{
            Level:   "ERROR",
            Message: "Database connection failed",
            Kubernetes: models.KubernetesInfo{
                Namespace: "production",
                Pod:       "api-server-abc123",
                Container: "api-server",
            },
        },
    }
    
    // Send the alert
    if err := notifier.SendAlert(alert); err != nil {
        fmt.Printf("Failed to send alert: %v\n", err)
    }
}
```

### Custom Configuration

```go
// Create custom Slack configuration
config := notifications.SlackConfig{
    WebhookURL:     "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK",
    Channel:        "#critical-alerts",
    Username:       "Alert Bot",
    IconEmoji:      ":fire:",
    Timeout:        60 * time.Second,
    Enabled:        true,
    Templates:      notifications.DefaultTemplateConfig(),
    SeverityEmojis: notifications.DefaultSeverityEmojis(),
    SeverityColors: notifications.DefaultSeverityColors(),
}

notifier := notifications.NewSlackNotifierWithConfig(config)
```

## Testing

The package includes comprehensive test coverage (85.5%) with unit tests, integration tests, and mocks.

### Running Tests

#### Unit Tests Only
```bash
# Run unit tests with coverage
go test ./internal/notifications/... -v -tags=unit -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out
```

#### Integration Tests Only
```bash
# Run integration tests
go test ./internal/notifications/... -v -tags=integration
```

#### All Tests
```bash
# Run all tests with coverage
go test ./internal/notifications/... -v -tags=unit,integration -coverprofile=coverage.out

# Generate coverage report
go tool cover -func=coverage.out
```

#### Test Coverage Summary
Current test coverage: **85.5%**

Coverage includes:
- ✅ Core notification interfaces and data structures
- ✅ Slack notifier functionality (send alerts, configuration, validation)
- ✅ Message formatting and templating
- ✅ Enable/disable state management
- ✅ Error handling and edge cases
- ✅ Global configuration management
- ✅ Alert summarization
- ✅ Integration tests with mock Slack server

### Mock Testing

The package includes sophisticated mocks for testing:

```go
// Mock Slack server for integration tests
mockServer := mocks.NewMockSlackServer()
defer mockServer.Close()

// Configure mock responses
mockServer.SetupSlackScenario("success")

// Create notifier with mock server
notifier := notifications.NewSlackNotifier(mockServer.GetWebhookURL())
```

## Package Structure

```
internal/notifications/
├── interfaces.go           # Core interfaces and data structures
├── slack.go               # Slack notifier implementation
├── interfaces_test.go     # Unit tests for interfaces
├── slack_test.go          # Unit tests for Slack notifier
├── integration_test.go    # Integration tests
├── mocks/
│   ├── mock_http_client.go   # Mock HTTP client for unit tests
│   └── mock_http_server.go   # Mock HTTP server for integration tests
└── fixtures/
    └── test_alerts.json      # Test data for alerts
```

## API Reference

### Core Interfaces

#### Notifier Interface
```go
type Notifier interface {
    SendAlert(alert models.Alert) error
    TestConnection() error
    GetName() string
    IsEnabled() bool
    SetEnabled(enabled bool)
}
```

### Slack Notifier

#### Creating Notifiers
- `NewSlackNotifier(webhookURL string) *SlackNotifier`
- `NewSlackNotifierWithConfig(config SlackConfig) *SlackNotifier`

#### Configuration Methods
- `GetConfig() SlackConfig`
- `UpdateConfig(config SlackConfig)`
- `SetChannel(channel string)`
- `SetUsername(username string)`
- `SetIconEmoji(iconEmoji string)`
- `SetWebhookURL(webhookURL string)`
- `ValidateConfig() error`

#### Messaging Methods
- `SendAlert(alert models.Alert) error`
- `SendSimpleMessage(text string) error`
- `SendRichMessage(text string, attachments []SlackAttachment) error`
- `SendCustomMessage(message SlackMessage) error`
- `CreateAlertSummary(alerts []models.Alert) SlackMessage`
- `TestConnection() error`

#### State Management
- `IsEnabled() bool`
- `SetEnabled(enabled bool)`
- `GetName() string`

### Configuration Types

#### SlackConfig
```go
type SlackConfig struct {
    WebhookURL     string
    Channel        string
    Username       string
    IconEmoji      string
    Timeout        time.Duration
    Enabled        bool
    Templates      TemplateConfig
    SeverityEmojis map[string]string
    SeverityColors map[string]string
}
```

#### Global Configuration
- `SetGlobalNotificationConfig(config NotificationGlobalConfig)`
- `GetGlobalNotificationConfig() NotificationGlobalConfig`
- `DefaultGlobalNotificationConfig() NotificationGlobalConfig`

## Error Handling

The package provides comprehensive error handling for:
- Invalid webhook URLs
- Network timeouts and failures
- Slack API errors (rate limiting, forbidden, etc.)
- Configuration validation errors
- Template parsing errors

Example error handling:
```go
if err := notifier.SendAlert(alert); err != nil {
    switch {
    case strings.Contains(err.Error(), "disabled"):
        log.Warn("Notifier is disabled")
    case strings.Contains(err.Error(), "webhook URL"):
        log.Error("Invalid webhook configuration")
    case strings.Contains(err.Error(), "timeout"):
        log.Error("Network timeout")
    default:
        log.Error("Unexpected error: %v", err)
    }
}
```

## Configuration Examples

### Custom Severity Styling
```go
customEmojis := map[string]string{
    "critical": "🔥",
    "high":     "⚠️",
    "medium":   "ℹ️",
    "low":      "✅",
}

customColors := map[string]string{
    "critical": "#FF0000",
    "high":     "#FFA500",
    "medium":   "#FFFF00",
    "low":      "#00FF00",
}

config := notifier.GetConfig()
config.SeverityEmojis = customEmojis
config.SeverityColors = customColors
notifier.UpdateConfig(config)
```

### Message Templates
```go
customTemplate := notifications.TemplateConfig{
    AlertMessage: "🚨 Alert: {{.RuleName}} - {{.Severity}} severity",
    SlackAlertTitle: "{{.RuleName}} ({{.Count}} occurrences)",
    SlackAlertFields: []notifications.SlackTemplateField{
        {Title: "Severity", Value: "{{.Severity}}", Short: true},
        {Title: "Count", Value: "{{.Count}}", Short: true},
        {Title: "Namespace", Value: "{{.LogEntry.Kubernetes.Namespace}}", Short: true},
        {Title: "Pod", Value: "{{.LogEntry.Kubernetes.Pod}}", Short: true},
    },
}

config := notifier.GetConfig()
config.Templates = customTemplate
notifier.UpdateConfig(config)
```

## Best Practices

1. **Always validate configuration** before using notifiers
2. **Handle errors appropriately** based on error type
3. **Use mock servers** for integration testing
4. **Test with different severity levels** to verify styling
5. **Monitor webhook rate limits** in production
6. **Use meaningful channel names** for different alert types
7. **Keep templates simple** to avoid parsing errors
8. **Test alert summarization** for bulk notifications

## Production Considerations

- **Rate Limiting**: Slack webhooks have rate limits; implement appropriate backoff
- **Error Monitoring**: Monitor notification failures in production
- **Configuration Management**: Store webhook URLs securely
- **Testing**: Regularly test webhook connectivity
- **Fallbacks**: Consider implementing fallback notification channels
- **Alerting on Alerting**: Monitor the notification system itself

---

This package provides a robust foundation for notification delivery with comprehensive testing and flexible configuration options. 