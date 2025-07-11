# Internal Storage Package - Alert Engine

The `internal/storage` package provides the **persistent state management layer** for the Alert Engine application. It implements a Redis-based storage system that maintains the application's state across restarts and enables distributed operation.

## 📁 Package Structure

The package consists of a single file:
- **`redis.go`** - Complete Redis storage implementation using the `go-redis/redis/v8` client

## 🏗️ Core Components

### RedisStore Struct
```go
type RedisStore struct {
    client *redis.Client
    ctx    context.Context
}
```

The `RedisStore` serves as the main storage interface implementing all persistence operations needed by the alert engine.

### Key Storage Categories

The Redis storage organizes data into several key patterns:

| Key Pattern | Purpose | Example |
|-------------|---------|---------|
| `alert_rule:*` | Alert rule configurations | `alert_rule:db-connection-error` |
| `counter:*` | Time-windowed event counters | `counter:rule123:1642234800` |
| `alert_status:*` | Alert rule status tracking | `alert_status:rule123` |
| `alert:*` | Generated alert instances | `alert:alert-12345` |
| `log_stats` | Global log processing metrics | `log_stats` |

## 🎯 Why Storage is Essential for Alert Engine

### 1. **Alert Rule Persistence**
The alert engine needs to store **configurable alert rules** that define:
- **Conditions**: What log patterns to match (log level, keywords, namespace)
- **Thresholds**: How many occurrences trigger an alert
- **Time Windows**: Rolling time periods for counting events
- **Actions**: Where to send notifications (Slack channels, severity levels)

**Example Rule Storage:**
```json
{
  "id": "db-connection-error",
  "name": "Database Connection Error",
  "conditions": {
    "log_level": "ERROR",
    "keywords": ["database", "connection"],
    "threshold": 5,
    "time_window": "5m"
  },
  "actions": {
    "channel": "#alerts",
    "severity": "high"
  }
}
```

### 2. **Time-Windowed Counters**
The most **critical storage requirement** is maintaining **sliding time window counters**:

```go
// When a log matches a rule, increment its counter
count, err := store.IncrementCounter("rule123", 5*time.Minute)
// If count > threshold, trigger alert
```

**Why This Matters:**
- **Spam Prevention**: Only alert when ERROR logs exceed 5 occurrences in 5 minutes
- **Sliding Windows**: Counters automatically expire and reset
- **Distributed Counting**: Multiple alert engine instances can share counters
- **Precise Thresholds**: Distinguish between 1 error vs 10 errors per minute

### 3. **Alert State Management**
The storage tracks alert rule states to prevent:
- **Duplicate Alerts**: Don't re-send the same alert repeatedly
- **Alert Storms**: Rate limiting and cooldown periods
- **Status Tracking**: Know when alerts were last triggered

### 4. **Generated Alert Storage**
Store triggered alert instances for:
- **Audit Trail**: Historical record of all alerts
- **Dashboard Display**: Show recent alerts in UI
- **Metrics**: Alert frequency analysis
- **Debugging**: Troubleshoot alert logic

## 🏗️ Alert Engine Architecture Context

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   Kafka Logs    │───▶│ Alert Engine │───▶│   Slack Alerts  │
│   (Streaming)   │    │  (Stateless) │    │   (Notifications)│
└─────────────────┘    └──────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │  Redis Storage  │
                       │   (Stateful)    │
                       └─────────────────┘
```

### Data Flow with Storage:

1. **Kafka Consumer** receives log entries
2. **Alert Engine** evaluates logs against stored rules
3. **Redis Storage** increments counters for matching rules
4. **Threshold Check** compares counter against rule threshold
5. **Alert Generation** creates alert if threshold exceeded
6. **Notification** sends alert to Slack
7. **Alert Storage** saves alert instance for audit

## 📚 Redis Functions Explained with Examples

### 🏗️ Alert Rule Management (The Recipe Book)

Think of alert rules as **recipes** that tell the system "when to cook an alert." These functions manage the recipe book.

#### 1. SaveAlertRule - Add a New Recipe
```go
func (r *RedisStore) SaveAlertRule(rule models.AlertRule) error
```

**What it does:** Saves an alert rule to Redis like adding a new recipe to a cookbook.

**Simple Example:**
```go
// Create a rule: "Alert me when there are too many database errors"
rule := models.AlertRule{
    ID:   "db-error-rule",
    Name: "Database Error Alert",
    Conditions: AlertConditions{
        LogLevel:   "ERROR",
        Keywords:   ["database", "connection"],
        Threshold:  5,           // Alert after 5 errors
        TimeWindow: "5m",        // Within 5 minutes
    },
}

// Save it to Redis
store.SaveAlertRule(rule)
```

**In Redis:** `alert_rule:db-error-rule` → `{"id":"db-error-rule","name":"Database Error Alert",...}`

#### 2. GetAlertRules - Get All Recipes
```go
func (r *RedisStore) GetAlertRules() ([]models.AlertRule, error)
```

**What it does:** Gets all alert rules, like reading the entire cookbook.

**Simple Example:**
```go
// Get all rules when the app starts
rules, err := store.GetAlertRules()
fmt.Printf("Found %d alert rules", len(rules))
// Output: Found 3 alert rules
```

#### 3. GetAlertRule - Get One Specific Recipe
```go
func (r *RedisStore) GetAlertRule(id string) (*models.AlertRule, error)
```

**What it does:** Gets one specific rule by ID, like finding a specific recipe.

**Simple Example:**
```go
// Get the database error rule
rule, err := store.GetAlertRule("db-error-rule")
if err != nil {
    fmt.Println("Rule not found!")
} else {
    fmt.Printf("Found rule: %s", rule.Name)
}
```

#### 4. DeleteAlertRule - Remove a Recipe
```go
func (r *RedisStore) DeleteAlertRule(id string) error
```

**What it does:** Deletes an alert rule from Redis.

**Simple Example:**
```go
// Remove the database error rule
err := store.DeleteAlertRule("db-error-rule")
if err != nil {
    fmt.Println("Failed to delete rule!")
} else {
    fmt.Println("Rule deleted successfully!")
}
```

### 🔢 Counter Management (The Scoreboard)

These functions manage **event counters** - think of them as scoreboards that count how many times something happens.

#### 5. IncrementCounter - Add One to the Score
```go
func (r *RedisStore) IncrementCounter(ruleID string, window time.Duration) (int64, error)
```

**What it does:** Counts events within a time window, like keeping score in a game.

**Simple Example:**
```go
// Every time we see a database error, increment the counter
count, err := store.IncrementCounter("db-error-rule", 5*time.Minute)
fmt.Printf("Database errors in last 5 minutes: %d", count)

// First error:  count = 1
// Second error: count = 2
// Third error:  count = 3
// ... and so on
```

**In Redis:** `counter:db-error-rule:1642234800` → `3`

#### 6. GetCounter - Check the Current Score
```go
func (r *RedisStore) GetCounter(ruleID string, window time.Duration) (int64, error)
```

**What it does:** Checks how many events happened without adding to the count.

**Simple Example:**
```go
// Check current count without incrementing
count, err := store.GetCounter("db-error-rule", 5*time.Minute)
if count >= 5 {
    fmt.Println("Too many errors! Time to send an alert!")
}
```

### 📊 Alert Status Management (The Memory)

These functions remember when alerts were last sent to avoid spam.

#### 7. SetAlertStatus - Remember When We Sent an Alert
```go
func (r *RedisStore) SetAlertStatus(ruleID string, status models.AlertStatus) error
```

**What it does:** Records when an alert was sent, like writing in a diary.

**Simple Example:**
```go
// Record that we sent a database error alert
status := models.AlertStatus{
    RuleID:      "db-error-rule",
    LastTrigger: time.Now(),
    Status:      "sent",
}

store.SetAlertStatus("db-error-rule", status)
```

**In Redis:** `alert_status:db-error-rule` → `{"rule_id":"db-error-rule","last_trigger":"2024-01-15T10:30:00Z","status":"sent"}`

#### 8. GetAlertStatus - Check When We Last Sent an Alert
```go
func (r *RedisStore) GetAlertStatus(ruleID string) (*models.AlertStatus, error)
```

**What it does:** Checks when we last sent an alert to avoid sending duplicates.

**Simple Example:**
```go
// Check if we recently sent an alert
status, err := store.GetAlertStatus("db-error-rule")
if time.Since(status.LastTrigger) < 10*time.Minute {
    fmt.Println("We sent an alert recently, don't spam!")
}
```

### 🚨 Alert Storage (The History Book)

These functions store the actual alerts that were generated.

#### 9. SaveAlert - Record an Alert in History
```go
func (r *RedisStore) SaveAlert(alert models.Alert) error
```

**What it does:** Saves a triggered alert for historical records.

**Simple Example:**
```go
// Save an alert that was just triggered
alert := models.Alert{
    ID:        "alert-12345",
    RuleID:    "db-error-rule",
    RuleName:  "Database Error Alert",
    Timestamp: time.Now(),
    Severity:  "high",
    Message:   "Database connection failed 5 times in 5 minutes",
    Count:     5,
}

store.SaveAlert(alert)
```

**In Redis:** `alert:alert-12345` → `{"id":"alert-12345","rule_id":"db-error-rule",...}`

#### 10. GetAlert - Get a Specific Alert
```go
func (r *RedisStore) GetAlert(alertID string) (*models.Alert, error)
```

**What it does:** Retrieves a specific alert by ID.

**Simple Example:**
```go
// Get a specific alert
alert, err := store.GetAlert("alert-12345")
if err != nil {
    fmt.Println("Alert not found!")
} else {
    fmt.Printf("Alert: %s", alert.Message)
}
```

#### 11. GetRecentAlerts - Show Recent History
```go
func (r *RedisStore) GetRecentAlerts(limit int) ([]models.Alert, error)
```

**What it does:** Gets recent alerts, like checking recent notifications on your phone.

**Simple Example:**
```go
// Get the last 10 alerts for a dashboard
alerts, err := store.GetRecentAlerts(10)
fmt.Printf("Recent alerts:")
for _, alert := range alerts {
    fmt.Printf("- %s: %s", alert.RuleName, alert.Message)
}
```

### 📈 Statistics and Health (The Report Card)

These functions track how the system is performing.

#### 12. SaveLogStats - Update Statistics
```go
func (r *RedisStore) SaveLogStats(stats models.LogStats) error
```

**What it does:** Updates statistics about log processing, like a report card.

**Simple Example:**
```go
// Update statistics every minute
stats := models.LogStats{
    TotalLogs:     1000,
    LogsByLevel:   map[string]int64{"ERROR": 50, "INFO": 950},
    LogsByService: map[string]int64{"user-service": 500, "api-service": 500},
    LastUpdated:   time.Now(),
}

store.SaveLogStats(stats)
```

#### 13. GetLogStats - Get Current Statistics
```go
func (r *RedisStore) GetLogStats() (*models.LogStats, error)
```

**What it does:** Retrieves current log processing statistics.

**Simple Example:**
```go
// Get statistics for dashboard
stats, err := store.GetLogStats()
if err == nil {
    fmt.Printf("Total logs processed: %d", stats.TotalLogs)
    fmt.Printf("Error logs: %d", stats.LogsByLevel["ERROR"])
}
```

#### 14. GetHealthStatus - Check if Redis is Working
```go
func (r *RedisStore) GetHealthStatus() (bool, error)
```

**What it does:** Checks if Redis is still working, like checking your internet connection.

**Simple Example:**
```go
// Check if Redis is healthy
healthy, err := store.GetHealthStatus()
if healthy {
    fmt.Println("✅ Redis is working fine!")
} else {
    fmt.Println("❌ Redis is having problems!")
}
```

#### 15. GetInfo - Get Redis Server Information
```go
func (r *RedisStore) GetInfo() (map[string]string, error)
```

**What it does:** Gets detailed information about the Redis server.

**Simple Example:**
```go
// Get Redis server info
info, err := store.GetInfo()
if err == nil {
    fmt.Printf("Redis status: %s", info["status"])
}
```

#### 16. GetMetrics - Get Storage Metrics
```go
func (r *RedisStore) GetMetrics() (map[string]interface{}, error)
```

**What it does:** Gets detailed metrics about stored data.

**Simple Example:**
```go
// Get metrics for monitoring
metrics, err := store.GetMetrics()
if err == nil {
    fmt.Printf("Alert rules: %d", metrics["alert_rules"])
    fmt.Printf("Active counters: %d", metrics["counters"])
}
```

### 🔧 Utility Functions (The Helper Tools)

#### 17. CleanupExpiredData - Take Out the Trash
```go
func (r *RedisStore) CleanupExpiredData() error
```

**What it does:** Removes old data that's no longer needed, like emptying your trash.

**Simple Example:**
```go
// Clean up old counters every hour
err := store.CleanupExpiredData()
if err == nil {
    fmt.Println("Cleaned up old counter data")
}
```

#### 18. Transaction - Do Multiple Things at Once
```go
func (r *RedisStore) Transaction(fn func(pipe redis.Pipeliner) error) error
```

**What it does:** Performs multiple operations together, like paying for multiple items at once.

**Simple Example:**
```go
// Save multiple rules at the same time
err := store.Transaction(func(pipe redis.Pipeliner) error {
    // Save rule 1
    data1, _ := json.Marshal(rule1)
    pipe.Set(ctx, "alert_rule:rule1", data1, 0)
    // Save rule 2
    data2, _ := json.Marshal(rule2)
    pipe.Set(ctx, "alert_rule:rule2", data2, 0)
    return nil
})
```

#### 19. BulkSaveAlertRules - Save Multiple Rules Efficiently
```go
func (r *RedisStore) BulkSaveAlertRules(rules []models.AlertRule) error
```

**What it does:** Saves multiple alert rules in a single operation for better performance.

**Simple Example:**
```go
// Save multiple rules at once
rules := []models.AlertRule{rule1, rule2, rule3}
err := store.BulkSaveAlertRules(rules)
if err == nil {
    fmt.Printf("Saved %d rules in bulk", len(rules))
}
```

#### 20. Search - Find Keys by Pattern
```go
func (r *RedisStore) Search(pattern string) ([]string, error)
```

**What it does:** Searches for keys matching a pattern.

**Simple Example:**
```go
// Find all alert rules
keys, err := store.Search("alert_rule:*")
if err == nil {
    fmt.Printf("Found %d alert rules", len(keys))
}
```

#### 21. Close - Cleanup Connection
```go
func (r *RedisStore) Close() error
```

**What it does:** Closes the Redis connection when shutting down.

**Simple Example:**
```go
// Close connection when app shuts down
defer store.Close()
```

## 🎯 Real-World Example: How It All Works Together

Let's say a database error occurs:

1. **Log arrives:** `"ERROR: Database connection failed"`

2. **Engine checks rule:** Gets rule using `GetAlertRule("db-error-rule")`

3. **Increment counter:** Uses `IncrementCounter("db-error-rule", 5*time.Minute)` → Returns `5`

4. **Check threshold:** Rule says "alert after 5 errors" → 5 >= 5, so trigger alert!

5. **Check recent alerts:** Uses `GetAlertStatus("db-error-rule")` to avoid spam

6. **Save alert:** Uses `SaveAlert(alert)` to record the alert

7. **Update status:** Uses `SetAlertStatus("db-error-rule", status)` to remember we sent it

8. **Send to Slack:** Alert goes to `#alerts` channel

9. **Next error:** Uses `GetAlertStatus("db-error-rule")` to check if we recently sent an alert → Don't spam!

## 🔑 Key Concepts Summary

| Function Type | Purpose | Real-World Analogy |
|---------------|---------|-------------------|
| **Rule Management** | Store alert configurations | Recipe book |
| **Counter Management** | Count events in time windows | Scoreboard |
| **Status Management** | Track alert history | Diary/Memory |
| **Alert Storage** | Store triggered alerts | History book |
| **Health/Stats** | Monitor system performance | Report card |
| **Utilities** | Maintenance and cleanup | Helper tools |

## 🚀 Why Redis for Storage?

1. **High Performance**: Sub-millisecond operations for real-time log processing
2. **Atomic Operations**: `INCR` operations for precise counter management
3. **TTL Support**: Automatic expiration for time-windowed counters
4. **Distributed**: Multiple alert engine instances can share state
5. **Persistence**: Data survives application restarts
6. **Scalability**: Handles thousands of rules and millions of counters

## 🎯 Key Benefits

Without the storage layer, the alert engine would:
- ❌ **Lose rules on restart** (no persistence)
- ❌ **Generate duplicate alerts** (no state tracking)
- ❌ **Fail threshold logic** (no counter persistence)
- ❌ **Cannot scale horizontally** (no shared state)

With the storage layer, the alert engine achieves:
- ✅ **Persistent Configuration** (rules survive restarts)
- ✅ **Accurate Threshold Detection** (precise time-windowed counting)
- ✅ **Spam Prevention** (alert status tracking)
- ✅ **Horizontal Scalability** (shared state across instances)
- ✅ **Operational Insights** (metrics and audit trail)

The storage package is **essential** for transforming the alert engine from a simple log processor into a **robust, production-ready alerting system** capable of handling enterprise-scale log monitoring with precise threshold detection and intelligent alert management.

## 🔧 Configuration

The Redis storage is configured through environment variables:

```bash
export REDIS_ADDRESS="localhost:6379"
export REDIS_PASSWORD=""
export REDIS_DATABASE=0
```

Or through the configuration file:

```yaml
redis:
  address: "localhost:6379"
  password: ""
  database: 0
  max_retries: 3
  pool_size: 10
```

## 🏁 Getting Started

```go
// Initialize Redis store
store := storage.NewRedisStore("localhost:6379", "")

// Create an alert rule
rule := models.AlertRule{
    ID:   "my-first-rule",
    Name: "My First Alert Rule",
    // ... other fields
}

// Save the rule
err := store.SaveAlertRule(rule)
if err != nil {
    log.Fatal("Failed to save rule:", err)
}

// The rule is now stored in Redis and ready to use!
```

The storage functions work together to create a **smart alerting system** that remembers rules, counts events accurately, prevents spam, and keeps track of everything for monitoring and debugging purposes. 