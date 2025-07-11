# Internal/Notifications Package

The `internal/notifications` package is a critical component of the alert-engine that handles sending notifications when alerts are triggered. It provides an extensible notification system with support for different notification channels (currently Slack, with interfaces for future channels like email, webhooks, etc.).

## Package Structure

The package consists of two main files:

1. **`interfaces.go`** - Defines interfaces and data structures for the notification system
2. **`slack.go`** - Implements Slack notifications using the `Notifier` interface

## Core Components and Examples

### 1. The `Notifier` Interface

```go
type Notifier interface {
    SendAlert(alert models.Alert) error
    TestConnection() error
    GetName() string
    IsEnabled() bool
    SetEnabled(enabled bool)
}
```

**Example Usage:**
```go
// Create a Slack notifier
slackNotifier := NewSlackNotifier("https://hooks.slack.com/services/YOUR/WEBHOOK/URL")

// Send an alert
alert := models.Alert{
    ID: "alert-001",
    RuleName: "High Error Rate",
    Severity: "critical",
    LogEntry: models.LogEntry{
        Level: "ERROR",
        Message: "Database connection failed",
        Kubernetes: models.KubernetesInfo{
            Namespace: "production",
            Pod: "api-server-abc123",
            Labels: map[string]string{"app": "api-server"},
        },
    },
    Count: 15,
}

err := slackNotifier.SendAlert(alert)
```

### 2. SlackNotifier Implementation

The `SlackNotifier` struct implements the `Notifier` interface and provides rich Slack integration.

**Key Functions with Examples:**

#### `NewSlackNotifier(webhookURL string)`
Creates a new Slack notifier with default settings.

```go
notifier := NewSlackNotifier("https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX")
// Default settings: channel=#alerts, username="Alert Engine", icon=:warning:
```

#### `SendAlert(alert models.Alert)`
Sends a formatted alert message to Slack with rich attachments.

**Example Output in Slack:**
```
🔴 Alert triggered for rule: *High Error Rate*

┌─────────────────────────────────────────────────────────────────┐
│ 🔴 High Error Rate                                              │
│ ```                                                             │
│ Database connection failed                                      │
│ ```                                                             │
│ Severity: CRITICAL          │ Namespace: production            │
│ Service: api-server         │ Pod: api-server-abc123           │
│ Log Level: ERROR            │ Count: 15                        │
└─────────────────────────────────────────────────────────────────┘
```

#### `TestConnection()`
Tests if the Slack webhook is working correctly.

```go
err := notifier.TestConnection()
if err != nil {
    log.Printf("Slack connection failed: %v", err)
}
```

**Example Test Message in Slack:**
```
Test message from Alert Engine

┌─────────────────────────────────────────────────────────────────┐
│ Connection Test                                                 │
│ If you can see this message, the Slack integration is working  │
│ correctly!                                                      │
│ Status: ✅ Connected        │ Timestamp: 2024-01-15T10:30:00Z   │
└─────────────────────────────────────────────────────────────────┘
```

#### `SendSimpleMessage(text string)`
Sends a simple text message without rich formatting.

```go
notifier.SendSimpleMessage("System maintenance starting in 5 minutes")
```

#### `CreateAlertSummary(alerts []models.Alert)`
Creates a summary of multiple alerts, grouped by severity.

```go
alerts := []models.Alert{
    {Severity: "critical", RuleName: "DB Error"},
    {Severity: "high", RuleName: "API Timeout"},
    {Severity: "critical", RuleName: "Memory Usage"},
}

summary := notifier.CreateAlertSummary(alerts)
// Results in a summary showing: 2 Critical, 1 High alerts
```

### 3. Configuration Management

The package provides comprehensive configuration management:

#### `NotificationConfig`
```go
config := DefaultNotificationConfig()
// Results in:
// {
//     Enabled: true,
//     MaxRetries: 3,
//     RetryDelay: 5s,
//     Timeout: 30s,
//     RateLimitPerMin: 60,
//     BatchSize: 10,
//     BatchDelay: 1s,
//     EnableDeduplication: true,
//     DeduplicationWindow: 5m,
// }
```

#### `NotificationChannel`
```go
channel := NotificationChannel{
    ID: "slack-prod",
    Name: "Production Slack",
    Type: "slack",
    Config: map[string]string{
        "webhook_url": "https://hooks.slack.com/...",
        "channel": "#alerts",
        "username": "Alert Engine",
    },
    Enabled: true,
}
```

### 4. Utility Functions

#### `GetSeverityEmoji(severity string)` and `GetSeverityColor(severity string)`
```go
emoji := GetSeverityEmoji("critical")  // Returns "🔴"
color := GetSeverityColor("high")      // Returns "#ff8000"

// Mapping:
// critical -> 🔴 (#ff0000)
// high     -> 🟠 (#ff8000)
// medium   -> 🟡 (#ffff00)
// low      -> 🟢 (#00ff00)
// default  -> ⚪ (#808080)
```

## How It Fits Into the Overall Alert-Engine

### System Architecture Flow

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kafka Logs    │───▶│ Alerting Engine │───▶│ Notifications   │
│                 │    │                 │    │                 │
│ • Log Ingestion │    │ • Rule Matching │    │ • Slack         │
│ • Processing    │    │ • Threshold     │    │ • Email (Future)│
│ • Filtering     │    │   Checking      │    │ • Webhooks      │
└─────────────────┘    │ • Alert Creation│    │ • SMS (Future)  │
                       └─────────────────┘    └─────────────────┘
```

### Integration Points

1. **Alert Engine Integration**
   ```go
   // In internal/alerting/engine.go
   func (e *Engine) EvaluateLog(logEntry models.LogEntry) {
       // ... rule matching logic ...
       
       if e.shouldTriggerAlert(rule, count) {
           alert := models.Alert{
               ID: fmt.Sprintf("%s-%d", rule.ID, time.Now().Unix()),
               RuleID: rule.ID,
               RuleName: rule.Name,
               LogEntry: logEntry,
               Severity: rule.Actions.Severity,
               // ... other fields ...
           }
           
           // Send notification using the Notifier interface
           if err := e.notifier.SendAlert(alert); err != nil {
               log.Printf("Error sending alert: %v", err)
           }
       }
   }
   ```

2. **Configuration Integration**
   ```go
   // The engine is initialized with a notifier
   slackNotifier := notifications.NewSlackNotifier(webhookURL)
   engine := alerting.NewEngine(stateStore, slackNotifier)
   ```

3. **Data Flow Example**
   ```go
   // 1. Log entry comes from Kafka
   logEntry := models.LogEntry{
       Level: "ERROR",
       Message: "Payment processing failed",
       Kubernetes: models.KubernetesInfo{
           Namespace: "ecommerce",
           Pod: "payment-service-xyz",
           Labels: map[string]string{"app": "payment-service"},
       },
   }
   
   // 2. Engine evaluates against rules
   engine.EvaluateLog(logEntry)
   
   // 3. If rule matches and threshold exceeded, alert is created
   // 4. Notification is sent via SlackNotifier
   ```

## Advanced Features

### 1. Notification Templates
```go
type NotificationTemplate struct {
    ID: "critical-alert",
    Name: "Critical Alert Template",
    Subject: "🔴 Critical Alert: {{.RuleName}}",
    Body: "Service {{.Service}} in {{.Namespace}} has triggered a critical alert",
    Variables: map[string]string{
        "RuleName": "{{.RuleName}}",
        "Service": "{{.LogEntry.Kubernetes.Labels.app}}",
        "Namespace": "{{.LogEntry.Kubernetes.Namespace}}",
    },
}
```

### 2. Notification History and Statistics
```go
type NotificationStats struct {
    TotalNotifications: 1250,
    SuccessfulSent: 1200,
    FailedSent: 50,
    SuccessRate: 96.0,
    ByChannel: map[string]int{
        "slack": 1000,
        "email": 200,
        "webhook": 50,
    },
    BySeverity: map[string]int{
        "critical": 100,
        "high": 300,
        "medium": 500,
        "low": 350,
    },
}
```

### 3. Rate Limiting and Deduplication
The package includes interfaces for:
- Rate limiting notifications to prevent spam
- Deduplicating identical alerts within a time window
- Queuing notifications for reliable delivery

## Future Extensibility

The package is designed to easily add new notification channels:

```go
// Example: Email notifier implementation
type EmailNotifier struct {
    smtpServer string
    username   string
    password   string
    // ... other fields
}

func (e *EmailNotifier) SendAlert(alert models.Alert) error {
    // Implementation for email sending
}

// Example: Webhook notifier
type WebhookNotifier struct {
    webhookURL string
    headers    map[string]string
}

func (w *WebhookNotifier) SendAlert(alert models.Alert) error {
    // Implementation for webhook posting
}
```

## Data Structures

### Core Interfaces

- **`Notifier`** - Main interface for all notification implementations
- **`NotificationManager`** - Manages multiple notification channels
- **`TemplateManager`** - Manages notification templates
- **`RateLimiter`** - Controls notification rate limiting
- **`NotificationDeduplicator`** - Prevents duplicate notifications
- **`NotificationQueue`** - Queues notifications for reliable delivery

### Key Data Types

- **`NotificationChannel`** - Configuration for a notification channel
- **`NotificationTemplate`** - Message template structure
- **`NotificationResult`** - Result of a notification attempt
- **`NotificationHistory`** - Historical tracking of notifications
- **`NotificationStats`** - Statistical information about notifications
- **`NotificationConfig`** - Configuration settings for notifications

## Usage Examples

### Basic Setup
```go
// Create and configure a Slack notifier
slackNotifier := NewSlackNotifier("https://hooks.slack.com/services/YOUR/WEBHOOK/URL")
slackNotifier.SetChannel("#critical-alerts")
slackNotifier.SetUsername("Production Alert Bot")
slackNotifier.SetIconEmoji(":fire:")

// Test the connection
if err := slackNotifier.TestConnection(); err != nil {
    log.Fatal("Failed to connect to Slack:", err)
}
```

### Sending Different Types of Notifications
```go
// Send an alert (rich formatting)
alert := models.Alert{
    RuleName: "High CPU Usage",
    Severity: "high",
    LogEntry: logEntry,
    Count: 25,
}
slackNotifier.SendAlert(alert)

// Send a simple message
slackNotifier.SendSimpleMessage("Deployment completed successfully")

// Send a custom formatted message
customMessage := SlackMessage{
    Text: "Custom notification",
    Attachments: []SlackAttachment{
        {
            Color: "#ff0000",
            Title: "Custom Alert",
            Text: "This is a custom formatted alert",
        },
    },
}
slackNotifier.SendCustomMessage(customMessage)
```

### Configuration Management
```go
// Create custom configuration
config := NotificationConfig{
    Enabled: true,
    MaxRetries: 5,
    RetryDelay: 10 * time.Second,
    Timeout: 60 * time.Second,
    RateLimitPerMin: 30,
    EnableDeduplication: true,
    DeduplicationWindow: 10 * time.Minute,
}

slackNotifier.SetConfig(config)
```

## Summary

The `internal/notifications` package serves as the **output layer** of the alert-engine, transforming structured alert data into user-friendly notifications across various channels. It provides:

- **Abstraction**: Clean interfaces for different notification types
- **Rich Formatting**: Context-aware message formatting with severity indicators
- **Reliability**: Error handling, retries, and status tracking
- **Extensibility**: Easy addition of new notification channels
- **Configuration**: Flexible configuration management for different environments

This design allows the alert-engine to focus on log processing and rule evaluation while delegating the complexity of notification delivery to this specialized package. 