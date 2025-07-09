# Alert Engine - Internal Alerting Package

The `internal/alerting` package is the core of the alert engine system that monitors logs and triggers alerts based on configurable rules. This package processes incoming log entries from Kubernetes applications and generates alerts when specified conditions are met.

## 📋 Table of Contents

- [Package Overview](#package-overview)
- [Core Components](#core-components)
- [Data Flow](#data-flow)
- [Function Examples](#function-examples)
- [Alert Rule Structure](#alert-rule-structure)
- [Usage Examples](#usage-examples)
- [Configuration](#configuration)
- [Performance Features](#performance-features)

## 🔍 Package Overview

The package consists of three main files:

- **`engine.go`** - Main alert engine that orchestrates the alerting system
- **`evaluator.go`** - Handles rule evaluation logic and performance tracking
- **`rules.go`** - Provides rule management, validation, and utility functions

## 🏗️ Core Components

### 1. Alert Engine (`engine.go`)

The `Engine` struct is the heart of the system:

```go
type Engine struct {
    stateStore  StateStore
    notifier    Notifier
    rules       []models.AlertRule
    windowStore map[string]*TimeWindow
    stopChan    chan struct{}
}
```

**Key Features:**
- **State Management**: Uses a `StateStore` interface for persisting alert rules and counters
- **Notifications**: Uses a `Notifier` interface for sending alerts (e.g., Slack)
- **Time Windows**: Tracks sliding time windows for counting events
- **Rule Management**: Loads, adds, updates, and deletes alert rules
- **Log Evaluation**: Processes incoming logs against all active rules

### 2. Evaluator (`evaluator.go`)

The `Evaluator` provides granular evaluation capabilities:

```go
type Evaluator struct {
    stateStore StateStore
}
```

**Key Features:**
- **Condition Evaluation**: Evaluates individual conditions against log entries
- **Threshold Checking**: Handles different comparison operators (gt, gte, lt, lte, eq)
- **Rule Testing**: Provides functionality to test rules against sample logs
- **Performance Tracking**: Monitors rule evaluation performance
- **Batch Processing**: Handles batch evaluation of multiple log entries

### 3. Rule Management (`rules.go`)

Provides comprehensive rule management functionality:

**Key Features:**
- **Rule Validation**: Ensures rules are properly configured
- **Default Rules**: Provides pre-configured alert rules
- **Rule Statistics**: Calculates metrics about rule usage
- **Rule Filtering**: Filters rules based on various criteria
- **Rule Templates**: Provides templates for creating new rules

## 🔄 Data Flow

```
Log Entry → Rule Evaluation → Condition Matching → Counter Updates → Threshold Check → Alert Generation → Notification
```

1. **Log Ingestion**: Logs arrive as `LogEntry` structs with Kubernetes metadata
2. **Rule Evaluation**: Each log is evaluated against all active `AlertRule`s
3. **Condition Matching**: System checks log level, namespace, service, and keywords
4. **Counter Updates**: Matching logs increment counters in time windows
5. **Threshold Checking**: System determines if alert thresholds are met
6. **Alert Generation**: Creates `Alert` structs with formatted messages
7. **Notification**: Sends alerts via configured notifiers (Slack, etc.)

## 💡 Function Examples

### 1. EvaluateLog Function

The main orchestration function that processes each incoming log entry.

**Example Input:**
```go
logEntry := models.LogEntry{
    Timestamp: time.Now(),
    Level:     "ERROR",
    Message:   "Database connection failed: timeout after 30s",
    Kubernetes: KubernetesInfo{
        Namespace: "production",
        Pod:       "user-service-7d4b9c8f-x2k9l",
        Container: "user-service",
        Labels: map[string]string{
            "app": "user-service",
            "version": "1.2.3",
        },
    },
    Host: "worker-node-1",
}
```

**Processing Steps:**
1. Filter enabled rules
2. Check condition matching
3. Update counters for matching logs
4. Check threshold conditions
5. Generate and send alerts if thresholds are met

### 2. matchesConditions Function

Checks if a log entry matches all rule conditions.

**Example 1: Matching Log**
```go
logEntry := models.LogEntry{
    Level:     "ERROR",
    Message:   "Database connection failed: timeout after 30s",
    Kubernetes: KubernetesInfo{
        Namespace: "production",
        Labels: map[string]string{
            "app": "user-service",
        },
    },
}

conditions := models.AlertConditions{
    LogLevel:   "ERROR",                           // ✅ Matches log.Level
    Namespace:  "production",                      // ✅ Matches log.Kubernetes.Namespace
    Service:    "user-service",                    // ✅ Matches log.Kubernetes.Labels["app"]
    Keywords:   []string{"database", "connection"}, // ✅ Both found in message
}

// Result: true (all conditions match)
```

**Example 2: Non-matching Log**
```go
logEntry := models.LogEntry{
    Level:     "INFO",  // ❌ Doesn't match "ERROR"
    Message:   "User login successful",
    Kubernetes: KubernetesInfo{
        Namespace: "production",
        Labels: map[string]string{
            "app": "user-service",
        },
    },
}

// Result: false (log level doesn't match)
```

### 3. shouldTriggerAlert Function

Determines if the current count meets the threshold criteria.

**Example Scenarios:**
```go
rule := models.AlertRule{
    Conditions: models.AlertConditions{
        Threshold: 5,
        Operator:  "gt", // greater than
    },
}

// Scenario 1: Should trigger
count := int64(6)
result := shouldTriggerAlert(rule, count)
// Result: true (6 > 5)

// Scenario 2: Should not trigger
count = int64(3)
result = shouldTriggerAlert(rule, count)
// Result: false (3 is not > 5)
```

**Supported Operators:**
- `gt` (greater than): count > threshold
- `gte` (greater than or equal): count >= threshold
- `lt` (less than): count < threshold
- `lte` (less than or equal): count <= threshold
- `eq` (equal): count == threshold

### 4. buildAlertMessage Function

Creates formatted alert messages with all relevant information.

**Example:**
```go
rule := models.AlertRule{
    Name: "Database Error Alert",
    Conditions: models.AlertConditions{
        TimeWindow: 5 * time.Minute,
    },
}

logEntry := models.LogEntry{
    Level:     "ERROR",
    Message:   "Database connection failed: timeout after 30s",
    Kubernetes: KubernetesInfo{
        Namespace: "production",
        Labels: map[string]string{
            "app": "user-service",
        },
    },
}

count := 6
```

**Generated Message:**
```
🚨 Alert: Database Error Alert
Service: user-service
Namespace: production
Level: ERROR
Count: 6 in 5m0s
Message: Database connection failed: timeout after 30s
```

## 📊 Alert Rule Structure

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

### Alert Conditions

```go
type AlertConditions struct {
    LogLevel    string        `json:"log_level"`    // ERROR, WARN, INFO, DEBUG
    Namespace   string        `json:"namespace"`    // Kubernetes namespace
    Service     string        `json:"service"`      // Service name (from app label)
    Keywords    []string      `json:"keywords"`     // Keywords to match in message
    Threshold   int           `json:"threshold"`    // Number of occurrences
    TimeWindow  time.Duration `json:"time_window"`  // Time window for counting
    Operator    string        `json:"operator"`     // gt, gte, lt, lte, eq
}
```

### Alert Actions

```go
type AlertActions struct {
    SlackWebhook string `json:"slack_webhook"`
    Channel      string `json:"channel"`      // #alerts, #infrastructure
    Severity     string `json:"severity"`     // low, medium, high, critical
}
```

## 🚀 Usage Examples

### Basic Engine Setup

```go
// Create engine with state store and notifier
engine := NewEngine(stateStore, notifier)

// Add a custom rule
rule := models.AlertRule{
    ID:      "custom-error-rule",
    Name:    "Application Error Alert",
    Enabled: true,
    Conditions: models.AlertConditions{
        LogLevel:   "ERROR",
        Namespace:  "production",
        Service:    "user-service",
        Threshold:  3,
        TimeWindow: 5 * time.Minute,
        Operator:   "gt",
    },
    Actions: models.AlertActions{
        Channel:  "#alerts",
        Severity: "high",
    },
}

engine.AddRule(rule)

// Process incoming logs
engine.EvaluateLog(logEntry)
```

### Testing Rules

```go
evaluator := NewEvaluator(stateStore)

// Test rule against sample logs
sampleLogs := []models.LogEntry{
    {Level: "ERROR", Message: "Database timeout"},
    {Level: "ERROR", Message: "Connection failed"},
    {Level: "INFO", Message: "Request processed"},
}

result, err := evaluator.TestRule(rule, sampleLogs)
if err != nil {
    log.Printf("Error testing rule: %v", err)
}

fmt.Printf("Rule matched %d out of %d logs\n", 
    result.Summary.MatchedLogs, result.Summary.TotalLogs)
```

### Default Rules

The package provides three default rules:

1. **Application Error Alert**: 5+ ERROR logs in 5 minutes
2. **Database Connection Issues**: 3+ database-related errors in 2 minutes
3. **High Memory Usage Warning**: 10+ memory warnings in 10 minutes

```go
// Load default rules
defaultRules := CreateDefaultRules()
for _, rule := range defaultRules {
    engine.AddRule(rule)
}
```

## ⚙️ Configuration

### Rule Validation

All rules are validated before being added:

```go
err := ValidateRule(rule)
if err != nil {
    log.Printf("Invalid rule: %v", err)
    return
}
```

### Rule Filtering

Filter rules based on various criteria:

```go
filter := RuleFilter{
    Enabled:   &[]bool{true}[0],
    Namespace: "production",
    Severity:  "high",
}

filteredRules := FilterRules(allRules, filter)
```

### Rule Statistics

Get insights about your rules:

```go
stats := GetRuleStats(engine.GetRules())
fmt.Printf("Total rules: %d\n", stats.TotalRules)
fmt.Printf("Enabled rules: %d\n", stats.EnabledRules)
fmt.Printf("Rules by severity: %+v\n", stats.BySeverity)
```

## 🚄 Performance Features

- **Sliding Time Windows**: Efficient event counting using time-based windows
- **Batch Processing**: Handle high-volume log streams with batch evaluation
- **Performance Metrics**: Track rule evaluation performance and bottlenecks
- **Memory Management**: Automatic cleanup of old time windows and counters
- **Distributed Counters**: Use Redis for scalable, distributed event counting
- **Early Filtering**: Skip expensive operations for non-matching logs

## 🔌 Key Interfaces

### StateStore Interface

```go
type StateStore interface {
    SaveAlertRule(rule models.AlertRule) error
    GetAlertRules() ([]models.AlertRule, error)
    GetAlertRule(id string) (*models.AlertRule, error)
    DeleteAlertRule(id string) error
    IncrementCounter(ruleID string, window time.Duration) (int64, error)
    GetCounter(ruleID string, window time.Duration) (int64, error)
    SetAlertStatus(ruleID string, status models.AlertStatus) error
    GetAlertStatus(ruleID string) (*models.AlertStatus, error)
}
```

### Notifier Interface

```go
type Notifier interface {
    SendAlert(alert models.Alert) error
    TestConnection() error
}
```

These interfaces allow for easy swapping of storage backends and notification channels, making the system highly extensible and testable.

---

## 🎯 Key Takeaways

1. **EvaluateLog** is called for every incoming log entry
2. **matchesConditions** acts as a filter - only matching logs proceed
3. **shouldTriggerAlert** prevents spam by checking thresholds
4. **buildAlertMessage** formats the final alert for human consumption

The system is designed to be both flexible (configurable rules) and efficient (early filtering, time windows) while providing rich context in alerts. 