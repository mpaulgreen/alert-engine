# Alert Engine - Internal API Package

## 📋 Package Overview

The `internal/api` package serves as the **RESTful API layer** for the alert engine system. It provides a comprehensive HTTP interface for managing alert rules, monitoring system health, and accessing system metrics. The package consists of two main files:

- **`handlers.go`** - Contains all HTTP handler functions
- **`routes.go`** - Defines API routing and endpoint structure

## 🏗️ Package Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │───▶│   API Package   │───▶│  Alert Engine   │
│   (Frontend/    │    │   (handlers.go) │    │   (Alerting)    │
│    CLI Tools)   │    │   (routes.go)   │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   StateStore    │
                       │   (Redis)       │
                       └─────────────────┘
```

## 🔧 Core Components

### 1. **Handlers Struct**
The main handler struct that contains dependencies:

```go
type Handlers struct {
    store       StateStore    // Redis storage for persistence
    alertEngine AlertEngine   // Alert processing engine
}
```

### 2. **Interfaces**
Two key interfaces define the contract:

- **`StateStore`** - Data persistence operations
- **`AlertEngine`** - Alert rule management operations

### 3. **Standard Response Format**
All endpoints return a consistent JSON structure:

```go
type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}
```

## 🔍 Function-by-Function Analysis

### **1. Health Check Functions**

#### `Health(c *gin.Context)`
**Purpose**: Checks if the system is running properly
**HTTP Method**: `GET /api/v1/health`

**Example Usage**:
```bash
curl http://localhost:8080/api/v1/health
```

**Response**:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:45Z"
  }
}
```

**How it works**: Calls `store.GetHealthStatus()` to check Redis connectivity and returns system health status.

---

### **2. Alert Rule Management Functions**

#### `GetRules(c *gin.Context)`
**Purpose**: Retrieves all alert rules
**HTTP Method**: `GET /api/v1/rules`

**Example Usage**:
```bash
curl http://localhost:8080/api/v1/rules
```

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "db-error-rule",
      "name": "Database Error Alert",
      "enabled": true,
      "conditions": {
        "log_level": "ERROR",
        "keywords": ["database", "connection"],
        "threshold": 5,
        "time_window": "5m"
      }
    }
  ]
}
```

---

#### `GetRule(c *gin.Context)`
**Purpose**: Retrieves a specific alert rule by ID
**HTTP Method**: `GET /api/v1/rules/{id}`

**Example Usage**:
```bash
curl http://localhost:8080/api/v1/rules/db-error-rule
```

**How it works**: Extracts rule ID from URL path parameter and calls `alertEngine.GetRule(ruleID)`.

---

#### `CreateRule(c *gin.Context)`
**Purpose**: Creates a new alert rule
**HTTP Method**: `POST /api/v1/rules`

**Example Usage**:
```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Memory Usage",
    "enabled": true,
    "conditions": {
      "log_level": "WARN",
      "keywords": ["memory", "usage"],
      "threshold": 10,
      "time_window": "10m"
    },
    "actions": {
      "channel": "#alerts",
      "severity": "medium"
    }
  }'
```

**Processing Steps**:
1. Validates JSON input
2. Calls `alerting.ValidateRule()` to ensure rule is valid
3. Generates ID if not provided
4. Adds rule to the alert engine
5. Returns created rule

---

#### `UpdateRule(c *gin.Context)`
**Purpose**: Updates an existing alert rule
**HTTP Method**: `PUT /api/v1/rules/{id}`

**Example Usage**:
```bash
curl -X PUT http://localhost:8080/api/v1/rules/db-error-rule \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Database Connection Error",
    "enabled": true,
    "conditions": {
      "log_level": "ERROR",
      "keywords": ["database", "connection", "timeout"],
      "threshold": 3,
      "time_window": "5m"
    }
  }'
```

---

#### `DeleteRule(c *gin.Context)`
**Purpose**: Deletes an alert rule
**HTTP Method**: `DELETE /api/v1/rules/{id}`

**Example Usage**:
```bash
curl -X DELETE http://localhost:8080/api/v1/rules/db-error-rule
```

---

### **3. System Monitoring Functions**

#### `GetRuleStats(c *gin.Context)`
**Purpose**: Returns statistics about alert rules
**HTTP Method**: `GET /api/v1/rules/stats`

**Example Usage**:
```bash
curl http://localhost:8080/api/v1/rules/stats
```

**Response**:
```json
{
  "success": true,
  "data": {
    "total_rules": 15,
    "enabled_rules": 12,
    "disabled_rules": 3,
    "rules_by_severity": {
      "high": 5,
      "medium": 7,
      "low": 3
    }
  }
}
```

---

#### `GetRecentAlerts(c *gin.Context)`
**Purpose**: Returns recent alert instances
**HTTP Method**: `GET /api/v1/alerts/recent?limit=50`

**Example Usage**:
```bash
curl http://localhost:8080/api/v1/alerts/recent?limit=10
```

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "alert-123",
      "rule_id": "db-error-rule",
      "rule_name": "Database Error Alert",
      "timestamp": "2024-01-15T10:30:45Z",
      "severity": "high",
      "message": "Database connection failed",
      "count": 7
    }
  ]
}
```

---

#### `GetLogStats(c *gin.Context)`
**Purpose**: Returns log processing statistics
**HTTP Method**: `GET /api/v1/system/logs/stats`

**Example Usage**:
```bash
curl http://localhost:8080/api/v1/system/logs/stats
```

**Response**:
```json
{
  "success": true,
  "data": {
    "total_logs_processed": 1234567,
    "logs_per_second": 150,
    "error_rate": 0.02,
    "last_processed": "2024-01-15T10:30:45Z"
  }
}
```

---

#### `GetMetrics(c *gin.Context)`
**Purpose**: Returns system performance metrics
**HTTP Method**: `GET /api/v1/system/metrics`

**Example Usage**:
```bash
curl http://localhost:8080/api/v1/system/metrics
```

---

### **4. Advanced Rule Management Functions**

#### `TestRule(c *gin.Context)`
**Purpose**: Tests an alert rule against sample log data
**HTTP Method**: `POST /api/v1/rules/test`

**Example Usage**:
```bash
curl -X POST http://localhost:8080/api/v1/rules/test \
  -H "Content-Type: application/json" \
  -d '{
    "rule": {
      "name": "Test Rule",
      "conditions": {
        "log_level": "ERROR",
        "keywords": ["database"]
      }
    },
    "sample_logs": [
      {
        "level": "ERROR",
        "message": "Database connection failed",
        "timestamp": "2024-01-15T10:30:45Z"
      }
    ]
  }'
```

**How it works**: Creates an evaluator and tests the rule against sample logs without actually triggering alerts.

---

#### `GetRuleTemplate(c *gin.Context)`
**Purpose**: Returns a template for creating new rules
**HTTP Method**: `GET /api/v1/rules/template`

**Example Usage**:
```bash
curl http://localhost:8080/api/v1/rules/template
```

---

#### `GetDefaultRules(c *gin.Context)`
**Purpose**: Returns pre-configured default alert rules
**HTTP Method**: `GET /api/v1/rules/defaults`

**Example Usage**:
```bash
curl http://localhost:8080/api/v1/rules/defaults
```

---

#### `BulkCreateRules(c *gin.Context)`
**Purpose**: Creates multiple alert rules in one operation
**HTTP Method**: `POST /api/v1/rules/bulk`

**Example Usage**:
```bash
curl -X POST http://localhost:8080/api/v1/rules/bulk \
  -H "Content-Type: application/json" \
  -d '[
    {
      "name": "Database Error Rule",
      "conditions": {...}
    },
    {
      "name": "Memory Warning Rule", 
      "conditions": {...}
    }
  ]'
```

**Processing**: Validates each rule individually and returns success/error counts.

---

#### `ReloadRules(c *gin.Context)`
**Purpose**: Reloads all alert rules from storage
**HTTP Method**: `POST /api/v1/rules/reload`

**Example Usage**:
```bash
curl -X POST http://localhost:8080/api/v1/rules/reload
```

**Use Case**: Useful after manual database changes or system restarts.

---

#### `FilterRules(c *gin.Context)`
**Purpose**: Filters alert rules based on criteria
**HTTP Method**: `POST /api/v1/rules/filter`

**Example Usage**:
```bash
curl -X POST http://localhost:8080/api/v1/rules/filter \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "severity": "high",
    "namespace": "production"
  }'
```

---

## 🚀 How API Fits into Overall Alert Engine Architecture

### **1. Data Flow Through API**
```
External Client → API Handler → Alert Engine → StateStore
                              ↓
                        Processing Logic
                              ↓
                        Response to Client
```

### **2. Key Integration Points**

#### **With Alert Engine**:
- **Rule Management**: API handlers call `alertEngine.AddRule()`, `UpdateRule()`, etc.
- **Rule Retrieval**: Handlers use `alertEngine.GetRules()` for current active rules
- **Rule Testing**: API provides safe testing via `evaluator.TestRule()`

#### **With StateStore (Redis)**:
- **Persistence**: All rule changes are persisted via `store.SaveAlertRule()`
- **Statistics**: Metrics come from `store.GetMetrics()`, `store.GetLogStats()`
- **Health Checks**: System health via `store.GetHealthStatus()`

#### **With Kafka Consumer**:
- **Indirect Integration**: API manages rules that the Kafka consumer uses for log evaluation
- **Configuration**: API allows dynamic rule changes without restarting consumers

### **3. Real-World Usage Scenarios**

#### **DevOps Dashboard**:
```javascript
// Frontend dashboard fetching rules
fetch('/api/v1/rules')
  .then(response => response.json())
  .then(data => displayRules(data.data));

// Creating new rule from UI
fetch('/api/v1/rules', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify(newRule)
});
```

#### **CLI Tool**:
```bash
# DevOps engineer creating rules
./alertctl create-rule --name "High CPU" --threshold 80 --severity high

# Checking system health
./alertctl health
```

#### **Infrastructure as Code**:
```yaml
# Terraform/Ansible deploying rules
resource "http" "alert_rules" {
  url = "http://alert-engine:8080/api/v1/rules/bulk"
  method = "POST"
  request_body = jsonencode(var.alert_rules)
}
```

### **4. Key Benefits of API Design**

1. **RESTful Design**: Standard HTTP methods and status codes
2. **Consistent Responses**: All endpoints use the same response format
3. **Comprehensive Coverage**: Full CRUD operations for rules
4. **Validation**: Server-side validation prevents invalid rules
5. **Batch Operations**: Bulk operations for efficiency
6. **Testing Support**: Safe rule testing without triggering alerts
7. **Monitoring**: Built-in health checks and metrics

### **5. Security & Error Handling**

- **Input Validation**: JSON schema validation for all inputs
- **Error Responses**: Structured error messages with HTTP status codes
- **CORS Support**: Cross-origin requests supported
- **Health Checks**: Liveness and readiness probes for Kubernetes

## 📊 Complete API Endpoint Reference

### **Health & System**
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | System health check |
| GET | `/api/v1/system/metrics` | System performance metrics |
| GET | `/api/v1/system/logs/stats` | Log processing statistics |

### **Alert Rules**
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/rules` | Get all alert rules |
| POST | `/api/v1/rules` | Create new alert rule |
| GET | `/api/v1/rules/{id}` | Get specific alert rule |
| PUT | `/api/v1/rules/{id}` | Update alert rule |
| DELETE | `/api/v1/rules/{id}` | Delete alert rule |
| GET | `/api/v1/rules/stats` | Get rule statistics |
| GET | `/api/v1/rules/template` | Get rule template |
| GET | `/api/v1/rules/defaults` | Get default rules |
| POST | `/api/v1/rules/bulk` | Create multiple rules |
| POST | `/api/v1/rules/reload` | Reload rules from storage |
| POST | `/api/v1/rules/filter` | Filter rules by criteria |
| POST | `/api/v1/rules/test` | Test rule against sample logs |

### **Alerts**
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/alerts/recent` | Get recent alert instances |

### **Documentation**
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | API overview |
| GET | `/docs` | Interactive API documentation |

## 🔧 Usage in Alert Engine Flow

The API package serves as the **control interface** for the entire alert engine system:

1. **Configuration Phase**: DevOps teams use the API to set up alert rules
2. **Runtime Phase**: The alert engine uses these rules to process Kafka logs
3. **Monitoring Phase**: Teams use the API to check system health and recent alerts
4. **Maintenance Phase**: Rules are updated, tested, and managed via the API

The API provides a user-friendly way to configure, monitor, and manage the real-time log alerting system without requiring direct database access or system restarts. 