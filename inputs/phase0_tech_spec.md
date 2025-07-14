# Phase 0: Foundation & Proof of Concept - Technical Specification

## Executive Summary

This specification details the implementation of Phase 0 for the Application Log Monitoring System, delivering a minimal viable product that validates core concepts using RedHat/OpenShift native technologies. The solution provides basic log ingestion, simple rule-based alerting, and Slack notifications within a 2-3 week timeframe.

## Architecture Overview

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐    ┌─────────────┐
│   OpenShift     │    │   AMQ        │    │   Alert         │    │   Slack     │
│   Pods/Logs     │───▶│   Streams    │───▶│   Engine        │───▶│   Webhook   │
│                 │    │   (Kafka)    │    │   (Go Service)  │    │             │
└─────────────────┘    └──────────────┘    └─────────────────┘    └─────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   OpenShift     │    │   Redis      │    │   Simple Web    │
│   Logging       │    │   (State)    │    │   UI (React)    │
│   (Vector)      │    │              │    │                 │
└─────────────────┘    └──────────────┘    └─────────────────┘
```

• It's a common pattern in streaming architectures where you need both reliable event processing and fast state access.

## Technology Stack

### Core Components

| Component | Technology | Justification |
|-----------|------------|---------------|
| **Log Collection** | OpenShift Logging (Vector) | Native OpenShift integration, replaces EFK stack |
| **Message Streaming** | Red Hat AMQ Streams (Kafka) | Managed Kafka service, enterprise support |
| **Alert Engine** | Go with Gin framework | High performance, simple deployment |
| **State Management** | Redis (via OpenShift) | In-memory caching, session management |
| **Web UI** | React + TypeScript | Modern, maintainable frontend |
| **Notifications** | Slack API | Single channel for Phase 0 |
| **Container Platform** | OpenShift 4.12+ | Target deployment platform |

### Supporting Infrastructure

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Container Registry** | Red Hat Quay | Store and manage custom container images |
| **Monitoring** | OpenShift Monitoring (Prometheus) | System observability |
| **Storage** | OpenShift Container Storage | Persistent data storage |

## Detailed Component Specifications

### 1. Log Collection Layer

#### OpenShift Logging Stack Configuration

**Technology**: OpenShift Logging Operator with Vector

**Configuration**:
```yaml
apiVersion: logging.openshift.io/v1
kind: ClusterLogForwarder
metadata:
  name: log-monitoring-forwarder
  namespace: openshift-logging
spec:
  outputs:
  - name: amq-streams-output
    type: kafka
    kafka:
      brokers:
      - amq-streams-cluster-kafka-bootstrap.log-monitoring.svc.cluster.local:9092
      topic: application-logs
  pipelines:
  - name: application-logs-pipeline
    inputRefs:
    - application
    outputRefs:
    - amq-streams-output
    labels:
      log-monitoring: "enabled"
```

**What this does:**
- `inputRefs: application` = Collect from all application pods (not infrastructure)
- `type: kafka` = Send to Kafka instead of default logging backend
- `topic: application-logs` = All logs go to this Kafka topic

**Log Format Standardization**:
```json
{
  "timestamp": "2025-06-26T10:30:00Z",
  "level": "ERROR",
  "message": "Database connection failed",
  "kubernetes": {
    "namespace": "payment-service",
    "pod": "payment-api-7b8c9d",
    "container": "payment-api",
    "labels": {
      "app": "payment-service",
      "version": "v1.2.3"
    }
  },
  "host": "worker-node-1"
}
```

### 2. Message Streaming Layer

#### Red Hat AMQ Streams Configuration

**Deployment**:
```yaml
apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: log-monitoring-cluster
  namespace: log-monitoring
spec:
  kafka:
    version: 3.5.0
    replicas: 3
    listeners:
    - name: plain
      port: 9092
      type: internal
      tls: false
    - name: tls
      port: 9093
      type: internal
      tls: true
    config:
      offsets.topic.replication.factor: 3
      transaction.state.log.replication.factor: 3
      transaction.state.log.min.isr: 2
      default.replication.factor: 3
      min.insync.replicas: 2
    storage:
      type: persistent-claim
      size: 100Gi
      deleteClaim: false
  zookeeper:
    replicas: 3
    storage:
      type: persistent-claim
      size: 10Gi
      deleteClaim: false
```

**Topic Configuration**:
```yaml
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: application-logs
  namespace: log-monitoring
spec:
  partitions: 12
  replicas: 3
  config:
    retention.ms: 604800000  # 7 days
    cleanup.policy: delete
    compression.type: snappy
```

### 3. Alert Engine Implementation

#### Go Service Architecture

**Project Structure**:
```
alert-engine/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── alerting/
│   │   ├── engine.go
│   │   ├── rules.go
│   │   └── evaluator.go
│   ├── kafka/
│   │   ├── consumer.go
│   │   └── processor.go
│   ├── notifications/
│   │   ├── slack.go
│   │   └── interfaces.go
│   ├── storage/
│   │   └── redis.go
│   └── api/
│       ├── handlers.go
│       └── routes.go
├── pkg/
│   └── models/
│       ├── alert.go
│       └── log.go
├── configs/
│   └── config.yaml
├── deployments/
│   └── openshift/
│       ├── deployment.yaml
│       └── service.yaml
└── go.mod
```

**Core Alert Rule Structure**:
```go
package models

import (
    "time"
)

type AlertRule struct {
    ID          string            `json:"id" redis:"id"`
    Name        string            `json:"name" redis:"name"`
    Description string            `json:"description" redis:"description"`
    Enabled     bool              `json:"enabled" redis:"enabled"`
    Conditions  AlertConditions   `json:"conditions" redis:"conditions"`
    Actions     AlertActions      `json:"actions" redis:"actions"`
    CreatedAt   time.Time         `json:"created_at" redis:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at" redis:"updated_at"`
}

type AlertConditions struct {
    LogLevel    string            `json:"log_level"`
    Namespace   string            `json:"namespace"`
    Service     string            `json:"service"`
    Keywords    []string          `json:"keywords"`
    Threshold   int               `json:"threshold"`
    TimeWindow  time.Duration     `json:"time_window"`
    Operator    string            `json:"operator"` // "gt", "lt", "eq", "contains"
}

type AlertActions struct {
    SlackWebhook string           `json:"slack_webhook"`
    Channel      string           `json:"channel"`
    Severity     string           `json:"severity"` // "low", "medium", "high", "critical"
}
```
#### Real-World Alert Rule Examples

**Database Connection Errors:**
```json
{
  "id": "db-connection-alert",
  "name": "Database Connection Issues",
  "description": "Alert when services can't connect to database",
  "conditions": {
    "logLevel": "ERROR",
    "keywords": ["database", "connection", "failed"],
    "timeWindow": "2m",
    "threshold": 3
  },
  "actions": {
    "notifications": [
      {
        "type": "slack",
        "channel": "#infrastructure",
        "message": "🔴 Database connection issues detected in {{namespace}}/{{service}}"
      }
    ]
  }
}
```

**High Memory Usage Warning:**
```json
{
  "id": "memory-warning-alert",
  "name": "High Memory Usage Warning",
  "description": "Alert when applications report high memory usage",
  "conditions": {
    "logLevel": "WARN",
    "keywords": ["memory", "usage", "high"],
    "timeWindow": "10m",
    "threshold": 10
  },
  "actions": {
    "notifications": [
      {
        "type": "email",
        "recipients": ["platform-team@company.com"],
        "subject": "Memory Warning: {{service}} in {{namespace}}"
      }
    ]
  }
}
```

**Specific Service Alert:**
```json
{
  "id": "payment-timeout-alert",
  "name": "Payment Processing Timeouts",
  "description": "Alert on payment processing timeouts",
  "conditions": {
    "logLevel": "ERROR",
    "namespace": "payment-service",
    "service": "payment-processor",
    "keywords": ["timeout", "payment"],
    "timeWindow": "5m",
    "threshold": 2
  },
  "actions": {
    "notifications": [
      {
        "type": "slack",
        "channel": "#payments",
        "message": "⚠️ Payment timeouts: {{count}} timeouts in {{timeWindow}} for {{service}}"
      },
      {
        "type": "email",
        "recipients": ["payments-oncall@company.com"],
        "subject": "URGENT: Payment Processing Issues"
      }
    ]
  }
}
```
**Kafka Consumer Implementation**:
```go
package kafka

import (
    "context"
    "encoding/json"
    "log"
    
    "github.com/segmentio/kafka-go"
    "github.com/your-org/log-monitoring/pkg/models"
)

type LogProcessor struct {
    reader   *kafka.Reader
    alertEngine AlertEngine
}

func NewLogProcessor(brokers []string, topic string) *LogProcessor {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  brokers,
        Topic:    topic,
        GroupID:  "log-monitoring-group",
        MinBytes: 10e3, // 10KB
        MaxBytes: 10e6, // 10MB
    })
    
    return &LogProcessor{
        reader: reader,
    }
}

func (p *LogProcessor) ProcessLogs(ctx context.Context) error {
    for {
        msg, err := p.reader.ReadMessage(ctx)
        if err != nil {
            log.Printf("Error reading message: %v", err)
            continue
        }
        
        var logEntry models.LogEntry
        if err := json.Unmarshal(msg.Value, &logEntry); err != nil {
            log.Printf("Error unmarshaling log: %v", err)
            continue
        }
        
        // Process log entry against all active rules
        p.alertEngine.EvaluateLog(logEntry)
    }
}
```

**Alert Evaluation Engine**:
```go
package alerting

import (
    "context"
    "time"
    
    "github.com/your-org/log-monitoring/pkg/models"
)

type Engine struct {
    rules       []models.AlertRule
    stateStore  StateStore
    notifier    Notifier
    windowStore map[string]*TimeWindow
}

func (e *Engine) EvaluateLog(log models.LogEntry) {
    for _, rule := range e.rules {
        if !rule.Enabled {
            continue
        }
        
        if e.matchesConditions(log, rule.Conditions) {
            e.updateCounter(rule.ID, log.Timestamp)
            
            if e.shouldTriggerAlert(rule) {
                alert := models.Alert{
                    RuleID:    rule.ID,
                    RuleName:  rule.Name,
                    LogEntry:  log,
                    Timestamp: log.Timestamp,
                    Severity:  rule.Actions.Severity,
                }
                
                e.notifier.SendAlert(alert)
                e.recordAlertSent(rule.ID, log.Timestamp)
            }
        }
    }
}

func (e *Engine) matchesConditions(log models.LogEntry, conditions models.AlertConditions) bool {
    // Check log level - STANDARDIZED FIELD
    if conditions.LogLevel != "" && log.Level != conditions.LogLevel {
        return false
    }
    
    // Check namespace - KUBERNETES METADATA ADDED BY VECTOR
    if conditions.Namespace != "" && log.Kubernetes.Namespace != conditions.Namespace {
        return false
    }
    
    // Check service - KUBERNETES LABELS ADDED BY VECTOR
    if conditions.Service != "" && log.Kubernetes.Labels["app"] != conditions.Service {
        return false
    }
    
    // Check keywords
    for _, keyword := range conditions.Keywords {
        if !strings.Contains(log.Message, keyword) {
            return false
        }
    }
    
    return true
}
```

### 4. Notification System

#### Slack Integration

**Slack Webhook Configuration**:
```go
package notifications

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "github.com/your-org/log-monitoring/pkg/models"
)

type SlackNotifier struct {
    webhookURL string
    client     *http.Client
}

type SlackMessage struct {
    Channel     string            `json:"channel"`
    Username    string            `json:"username"`
    Text        string            `json:"text"`
    Attachments []SlackAttachment `json:"attachments"`
}

type SlackAttachment struct {
    Color      string       `json:"color"`
    Title      string       `json:"title"`
    Text       string       `json:"text"`
    Fields     []SlackField `json:"fields"`
    Timestamp  int64        `json:"ts"`
}

type SlackField struct {
    Title string `json:"title"`
    Value string `json:"value"`
    Short bool   `json:"short"`
}

func (s *SlackNotifier) SendAlert(alert models.Alert) error {
    color := s.getSeverityColor(alert.Severity)
    
    message := SlackMessage{
        Channel:  "#alerts",
        Username: "Log Monitor",
        Text:     fmt.Sprintf("🚨 Alert: %s", alert.RuleName),
        Attachments: []SlackAttachment{
            {
                Color: color,
                Title: alert.RuleName,
                Text:  fmt.Sprintf("Alert triggered for %s", alert.LogEntry.Kubernetes.Namespace),
                Fields: []SlackField{
                    {
                        Title: "Namespace",
                        Value: alert.LogEntry.Kubernetes.Namespace,
                        Short: true,
                    },
                    {
                        Title: "Service",
                        Value: alert.LogEntry.Kubernetes.Labels["app"],
                        Short: true,
                    },
                    {
                        Title: "Log Level",
                        Value: alert.LogEntry.Level,
                        Short: true,
                    },
                    {
                        Title: "Pod",
                        Value: alert.LogEntry.Kubernetes.Pod,
                        Short: true,
                    },
                    {
                        Title: "Message",
                        Value: alert.LogEntry.Message,
                        Short: false,
                    },
                },
                Timestamp: alert.Timestamp.Unix(),
            },
        },
    }
    
    return s.sendSlackMessage(message)
}

func (s *SlackNotifier) getSeverityColor(severity string) string {
    switch severity {
    case "critical":
        return "danger"
    case "high":
        return "warning"
    case "medium":
        return "good"
    default:
        return "#439FE0"
    }
}
```

### 5. State Management

#### Redis Configuration

**OpenShift Redis Deployment**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: log-monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: registry.redhat.io/rhel8/redis-6:latest
        ports:
        - containerPort: 6379
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: password
        volumeMounts:
        - name: redis-data
          mountPath: /var/lib/redis/data
      volumes:
      - name: redis-data
        persistentVolumeClaim:
          claimName: redis-pvc
```

**State Store Implementation**:
```go
package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/go-redis/redis/v8"
    "github.com/your-org/log-monitoring/pkg/models"
)

type RedisStore struct {
    client *redis.Client
}

func NewRedisStore(addr, password string) *RedisStore {
    rdb := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       0,
    })
    
    return &RedisStore{client: rdb}
}

func (r *RedisStore) SaveAlertRule(rule models.AlertRule) error {
    ctx := context.Background()
    
    data, err := json.Marshal(rule)
    if err != nil {
        return err
    }
    
    key := fmt.Sprintf("alert_rule:%s", rule.ID)
    return r.client.Set(ctx, key, data, 0).Err()
}

func (r *RedisStore) GetAlertRules() ([]models.AlertRule, error) {
    ctx := context.Background()
    
    keys, err := r.client.Keys(ctx, "alert_rule:*").Result()
    if err != nil {
        return nil, err
    }
    
    var rules []models.AlertRule
    for _, key := range keys {
        val, err := r.client.Get(ctx, key).Result()
        if err != nil {
            continue
        }
        
        var rule models.AlertRule
        if err := json.Unmarshal([]byte(val), &rule); err != nil {
            continue
        }
        
        rules = append(rules, rule)
    }
    
    return rules, nil
}

func (r *RedisStore) IncrementCounter(ruleID string, window time.Duration) (int64, error) {
    ctx := context.Background()
    key := fmt.Sprintf("counter:%s:%d", ruleID, time.Now().Unix()/int64(window.Seconds()))
    
    pipe := r.client.Pipeline()
    incr := pipe.Incr(ctx, key)
    pipe.Expire(ctx, key, window*2) // Keep data for 2x window duration
    
    _, err := pipe.Exec(ctx)
    if err != nil {
        return 0, err
    }
    
    return incr.Val(), nil
}
```

### 6. Web UI Implementation

#### React Application Structure

**Component Architecture**:
```
src/
├── components/
│   ├── AlertCreator/
│   │   ├── AlertCreator.tsx
│   │   ├── ConditionBuilder.tsx
│   │   └── PreviewPanel.tsx
│   ├── AlertList/
│   │   ├── AlertList.tsx
│   │   ├── AlertCard.tsx
│   │   └── AlertStatus.tsx
│   └── Common/
│       ├── Layout.tsx
│       └── Navigation.tsx
├── services/
│   ├── api.ts
│   └── websocket.ts
├── types/
│   └── alert.ts
└── utils/
    └── formatting.ts
```

**Alert Creator Component**:
```typescript
import React, { useState } from 'react';
import { AlertRule, AlertConditions, AlertActions } from '../types/alert';
import { createAlert, testAlert } from '../services/api';

interface AlertCreatorProps {
  onAlertCreated: (alert: AlertRule) => void;
}

export const AlertCreator: React.FC<AlertCreatorProps> = ({ onAlertCreated }) => {
  const [rule, setRule] = useState<Partial<AlertRule>>({
    name: '',
    description: '',
    enabled: true,
    conditions: {
      logLevel: 'ERROR',
      namespace: '',
      service: '',
      keywords: [],
      threshold: 10,
      timeWindow: 300, // 5 minutes in seconds
      operator: 'gt'
    },
    actions: {
      slackWebhook: '',
      channel: '#alerts',
      severity: 'medium'
    }
  });

  const [testResults, setTestResults] = useState<any>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleTestAlert = async () => {
    setIsLoading(true);
    try {
      const results = await testAlert(rule);
      setTestResults(results);
    } catch (error) {
      console.error('Error testing alert:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreateAlert = async () => {
    setIsLoading(true);
    try {
      const createdAlert = await createAlert(rule);
      onAlertCreated(createdAlert);
    } catch (error) {
      console.error('Error creating alert:', error);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="alert-creator">
      <h2>Create New Alert</h2>
      
      <div className="form-section">
        <label>Alert Name</label>
        <input
          type="text"
          value={rule.name}
          onChange={(e) => setRule({...rule, name: e.target.value})}
          placeholder="e.g., Payment Service Errors"
        />
      </div>

      <div className="form-section">
        <label>Log Level</label>
        <select
          value={rule.conditions?.logLevel}
          onChange={(e) => setRule({
            ...rule,
            conditions: {...rule.conditions, logLevel: e.target.value}
          })}
        >
          <option value="ERROR">ERROR</option>
          <option value="WARN">WARN</option>
          <option value="INFO">INFO</option>
          <option value="DEBUG">DEBUG</option>
        </select>
      </div>

      <div className="form-section">
        <label>Namespace</label>
        <input
          type="text"
          value={rule.conditions?.namespace}
          onChange={(e) => setRule({
            ...rule,
            conditions: {...rule.conditions, namespace: e.target.value}
          })}
          placeholder="e.g., payment-service"
        />
      </div>

      <div className="form-section">
        <label>Threshold</label>
        <input
          type="number"
          value={rule.conditions?.threshold}
          onChange={(e) => setRule({
            ...rule,
            conditions: {...rule.conditions, threshold: parseInt(e.target.value)}
          })}
          placeholder="10"
        />
        <span> logs in </span>
        <input
          type="number"
          value={rule.conditions?.timeWindow ? rule.conditions.timeWindow / 60 : 5}
          onChange={(e) => setRule({
            ...rule,
            conditions: {...rule.conditions, timeWindow: parseInt(e.target.value) * 60}
          })}
          placeholder="5"
        />
        <span> minutes</span>
      </div>

      <div className="form-section">
        <label>Slack Channel</label>
        <input
          type="text"
          value={rule.actions?.channel}
          onChange={(e) => setRule({
            ...rule,
            actions: {...rule.actions, channel: e.target.value}
          })}
          placeholder="#alerts"
        />
      </div>

      <div className="actions">
        <button onClick={handleTestAlert} disabled={isLoading}>
          Test Alert
        </button>
        <button onClick={handleCreateAlert} disabled={isLoading}>
          Create Alert
        </button>
      </div>

      {testResults && (
        <div className="test-results">
          <h3>Test Results</h3>
          <pre>{JSON.stringify(testResults, null, 2)}</pre>
        </div>
      )}
    </div>
  );
};
```

## Deployment Configuration

### OpenShift Deployment Manifests

#### Alert Engine Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alert-engine
  namespace: log-monitoring
  labels:
    app: alert-engine
spec:
  replicas: 2
  selector:
    matchLabels:
      app: alert-engine
  template:
    metadata:
      labels:
        app: alert-engine
    spec:
      containers:
      - name: alert-engine
        image: quay.io/your-org/log-monitoring/alert-engine:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: KAFKA_BROKERS
          value: "amq-streams-cluster-kafka-bootstrap:9092"
        - name: REDIS_HOST
          value: "redis:6379"
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: password
        - name: SLACK_WEBHOOK_URL
          valueFrom:
            secretKeyRef:
              name: slack-webhook
              key: url
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: alert-engine
  namespace: log-monitoring
spec:
  selector:
    app: alert-engine
  ports:
  - name: http
    port: 80
    targetPort: 8080
  type: ClusterIP
```

#### Web UI Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: log-monitoring-ui
  namespace: log-monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: log-monitoring-ui
  template:
    metadata:
      labels:
        app: log-monitoring-ui
    spec:
      containers:
      - name: ui
        image: quay.io/your-org/log-monitoring/ui:latest
        ports:
        - containerPort: 3000
        env:
        - name: REACT_APP_API_URL
          value: "http://alert-engine/api"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
---
apiVersion: v1
kind: Service
metadata:
  name: log-monitoring-ui
  namespace: log-monitoring
spec:
  selector:
    app: log-monitoring-ui
  ports:
  - name: http
    port: 80
    targetPort: 3000
---
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: log-monitoring-ui
  namespace: log-monitoring
spec:
  to:
    kind: Service
    name: log-monitoring-ui
  port:
    targetPort: http
  tls:
    termination: edge
```

## Security Configuration

### RBAC Setup

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: log-monitoring-service-account
  namespace: log-monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: log-monitoring-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "namespaces"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: log-monitoring-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: log-monitoring-role
subjects:
- kind: ServiceAccount
  name: log-monitoring-service-account
  namespace: log-monitoring
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: log-monitoring-network-policy
  namespace: log-monitoring
spec:
  podSelector:
    matchLabels:
      app: alert-engine
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: log-monitoring
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: log-monitoring
    ports:
    - protocol: TCP
      port: 9092  # Kafka
    - protocol: TCP
      port: 6379  # Redis
  - to: []
    ports:
    - protocol: TCP
      port: 443   # HTTPS for Slack webhook
```

## Testing Strategy

### Unit Testing

**Go Testing Framework**:
```go
package alerting

import (
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/your-org/log-monitoring/pkg/models"
)

func TestAlertEngine_EvaluateLog(t *testing.T) {
    tests := []struct {
        name        string
        rule        models.AlertRule
        logEntry    models.LogEntry
        shouldAlert bool
    }{
        {
            name: "Should trigger alert for ERROR log exceeding threshold",
            rule: models.AlertRule{
                ID: "test-rule-1",
                Conditions: models.AlertConditions{
                    LogLevel:   "ERROR",
                    Namespace:  "payment-service",
                    Threshold:  5,
                    TimeWindow: 5 * time.Minute,
                },
            },
            logEntry: models.LogEntry{
                Level:     "ERROR",
                Message:   "Database connection failed",
                Timestamp: time.Now(),
                Kubernetes: models.KubernetesInfo{
                    Namespace: "payment-service",
                    Pod:       "payment-api-123",
                },
            },
            shouldAlert: true,
        },
        {
            name: "Should not trigger alert for different namespace",
            rule: models.AlertRule{
                ID: "test-rule-2",
                Conditions: models.AlertConditions{
                    LogLevel:   "ERROR",
                    Namespace:  "user-service",
                    Threshold:  5,
                    TimeWindow: 5 * time.Minute,
                },
            },
            logEntry: models.LogEntry{
                Level:     "ERROR",
                Message:   "Database connection failed",
                Timestamp: time.Now(),
                Kubernetes: models.KubernetesInfo{
                    Namespace: "payment-service",
                    Pod:       "payment-api-123",
                },
            },
            shouldAlert: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := NewEngine()
            engine.AddRule(tt.rule)
            
            // Simulate multiple log entries to exceed threshold
            for i := 0; i < 10; i++ {
                engine.EvaluateLog(tt.logEntry)
            }
            
            alerts := engine.GetTriggeredAlerts()
            if tt.shouldAlert {
                assert.NotEmpty(t, alerts)
            } else {
                assert.Empty(t, alerts)
            }
        })
    }
}
```

### Integration Testing

**Kafka Integration Test**:
```go
package integration

import (
    "context"
    "testing"
    "time"
    
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/kafka"
    "github.com/stretchr/testify/assert"
    "github.com/your-org/log-monitoring/internal/kafka"
    "github.com/your-org/log-monitoring/pkg/models"
)

func TestKafkaLogProcessing(t *testing.T) {
    ctx := context.Background()
    
    // Start Kafka container
    kafkaContainer, err := kafka.RunContainer(ctx,
        kafka.WithClusterID("test-cluster"),
        testcontainers.WithImage("confluentinc/cp-kafka:7.4.0"),
    )
    assert.NoError(t, err)
    defer kafkaContainer.Terminate(ctx)
    
    brokers, err := kafkaContainer.Brokers(ctx)
    assert.NoError(t, err)
    
    // Create log processor
    processor := kafka.NewLogProcessor(brokers, "test-logs")
    
    // Test log processing
    testLog := models.LogEntry{
        Level:     "ERROR",
        Message:   "Test error message",
        Timestamp: time.Now(),
        Kubernetes: models.KubernetesInfo{
            Namespace: "test-namespace",
            Pod:       "test-pod",
        },
    }
    
    // Send test log and verify processing
    err = processor.SendLog(testLog)
    assert.NoError(t, err)
    
    // Verify log was received and processed
    receivedLogs := processor.GetProcessedLogs()
    assert.Len(t, receivedLogs, 1)
    assert.Equal(t, testLog.Message, receivedLogs[0].Message)
}
```

### End-to-End Testing

**Alert Flow Test**:
```go
package e2e

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/your-org/log-monitoring/internal/api"
    "github.com/your-org/log-monitoring/pkg/models"
)

func TestCompleteAlertFlow(t *testing.T) {
    // Setup test server
    server := httptest.NewServer(api.SetupRoutes())
    defer server.Close()
    
    // Step 1: Create alert rule
    rule := models.AlertRule{
        Name: "Test Error Alert",
        Conditions: models.AlertConditions{
            LogLevel:   "ERROR",
            Namespace:  "test-service",
            Threshold:  3,
            TimeWindow: 1 * time.Minute,
        },
        Actions: models.AlertActions{
            SlackWebhook: "http://test-webhook",
            Channel:      "#test-alerts",
            Severity:     "high",
        },
    }
    
    ruleJSON, _ := json.Marshal(rule)
    resp, err := http.Post(server.URL+"/api/alerts", "application/json", strings.NewReader(string(ruleJSON)))
    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
    
    // Step 2: Simulate log entries that should trigger alert
    for i := 0; i < 5; i++ {
        logEntry := models.LogEntry{
            Level:     "ERROR",
            Message:   "Test error message",
            Timestamp: time.Now(),
            Kubernetes: models.KubernetesInfo{
                Namespace: "test-service",
                Pod:       "test-pod-123",
            },
        }
        
        logJSON, _ := json.Marshal(logEntry)
        http.Post(server.URL+"/api/logs", "application/json", strings.NewReader(string(logJSON)))
    }
    
    // Step 3: Verify alert was triggered
    time.Sleep(2 * time.Second) // Allow processing time
    
    resp, err = http.Get(server.URL + "/api/alerts/triggered")
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // Verify alert response contains our triggered alert
    var alerts []models.Alert
    json.NewDecoder(resp.Body).Decode(&alerts)
    assert.NotEmpty(t, alerts)
    assert.Equal(t, "Test Error Alert", alerts[0].RuleName)
}
```

## Performance Specifications

### Throughput Requirements

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| **Log Processing Rate** | 10,000 logs/second | Kafka consumer lag monitoring |
| **Alert Evaluation Latency** | <500ms per log | Application metrics |
| **Notification Delivery** | <30 seconds | End-to-end timing |
| **Memory Usage** | <512MB per service | Container resource monitoring |
| **CPU Usage** | <500m per service | Container resource monitoring |

### Load Testing Configuration

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: log-load-test
  namespace: log-monitoring
spec:
  template:
    spec:
      containers:
      - name: load-generator
        image: curlimages/curl:latest
        command: ["/bin/sh"]
        args:
        - -c
        - |
          for i in $(seq 1 10000); do
            curl -X POST http://alert-engine/api/logs \
              -H "Content-Type: application/json" \
              -d "{
                \"level\": \"ERROR\",
                \"message\": \"Load test error $i\",
                \"timestamp\": \"$(date -Iseconds)\",
                \"kubernetes\": {
                  \"namespace\": \"load-test\",
                  \"pod\": \"load-pod-$i\"
                }
              }"
            sleep 0.1
          done
      restartPolicy: Never
```

## Monitoring and Observability

### Prometheus Metrics

**Custom Metrics Definition**:
```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    LogsProcessedTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "log_monitoring_logs_processed_total",
            Help: "Total number of logs processed",
        },
        []string{"namespace", "level"},
    )
    
    AlertsTriggeredTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "log_monitoring_alerts_triggered_total",
            Help: "Total number of alerts triggered",
        },
        []string{"rule_name", "severity"},
    )
    
    AlertEvaluationDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "log_monitoring_alert_evaluation_duration_seconds",
            Help: "Time spent evaluating alerts",
            Buckets: prometheus.DefBuckets,
        },
        []string{"rule_id"},
    )
    
    NotificationDeliveryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "log_monitoring_notification_delivery_duration_seconds",
            Help: "Time spent delivering notifications",
            Buckets: prometheus.DefBuckets,
        },
        []string{"channel", "severity"},
    )
    
    ActiveAlertRules = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "log_monitoring_active_alert_rules",
            Help: "Number of active alert rules",
        },
    )
)
```

### ServiceMonitor Configuration

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: log-monitoring-metrics
  namespace: log-monitoring
spec:
  selector:
    matchLabels:
      app: alert-engine
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
---
apiVersion: v1
kind: Service
metadata:
  name: alert-engine-metrics
  namespace: log-monitoring
  labels:
    app: alert-engine
spec:
  selector:
    app: alert-engine
  ports:
  - name: metrics
    port: 8081
    targetPort: 8081
```

### Grafana Dashboard

**Dashboard JSON Configuration**:
```json
{
  "dashboard": {
    "title": "Log Monitoring System - Phase 0",
    "tags": ["log-monitoring", "alerts"],
    "panels": [
      {
        "title": "Log Processing Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(log_monitoring_logs_processed_total[5m])",
            "legendFormat": "{{namespace}} - {{level}}"
          }
        ]
      },
      {
        "title": "Alert Trigger Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(log_monitoring_alerts_triggered_total[5m])",
            "legendFormat": "{{rule_name}} - {{severity}}"
          }
        ]
      },
      {
        "title": "Alert Evaluation Latency",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(log_monitoring_alert_evaluation_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(log_monitoring_alert_evaluation_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ]
      },
      {
        "title": "Active Alert Rules",
        "type": "singlestat",
        "targets": [
          {
            "expr": "log_monitoring_active_alert_rules"
          }
        ]
      }
    ]
  }
}
```

## Deployment and CI/CD Pipeline

### GitOps Configuration

**Kustomization Structure**:
```
deploy/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── alert-engine/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── configmap.yaml
│   ├── ui/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── route.yaml
│   ├── kafka/
│   │   ├── kafka-cluster.yaml
│   │   └── kafka-topics.yaml
│   └── redis/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── pvc.yaml
├── overlays/
│   ├── development/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   ├── staging/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   └── production/
│       ├── kustomization.yaml
│       └── patches/
```

**Base Kustomization**:
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: log-monitoring

resources:
- namespace.yaml
- alert-engine/deployment.yaml
- alert-engine/service.yaml
- alert-engine/configmap.yaml
- ui/deployment.yaml
- ui/service.yaml
- ui/route.yaml
- kafka/kafka-cluster.yaml
- kafka/kafka-topics.yaml
- redis/deployment.yaml
- redis/service.yaml
- redis/pvc.yaml

commonLabels:
  app.kubernetes.io/name: log-monitoring
  app.kubernetes.io/part-of: log-monitoring-system
  app.kubernetes.io/version: v0.1.0
```

### Tekton Pipeline

**Build Pipeline**:
```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: log-monitoring-build
  namespace: log-monitoring
spec:
  params:
  - name: git-url
    type: string
    default: https://github.com/your-org/log-monitoring.git
  - name: git-revision
    type: string
    default: main
  - name: image-tag
    type: string
    default: latest
  
  workspaces:
  - name: shared-workspace
  
  tasks:
  - name: fetch-source
    taskRef:
      name: git-clone
    workspaces:
    - name: output
      workspace: shared-workspace
    params:
    - name: url
      value: $(params.git-url)
    - name: revision
      value: $(params.git-revision)
  
  - name: build-alert-engine
    taskRef:
      name: buildah
    runAfter:
    - fetch-source
    workspaces:
    - name: source
      workspace: shared-workspace
    params:
    - name: IMAGE
      value: quay.io/your-org/log-monitoring/alert-engine:$(params.image-tag)
    - name: DOCKERFILE
      value: ./cmd/alert-engine/Dockerfile
  
  - name: build-ui
    taskRef:
      name: buildah
    runAfter:
    - fetch-source
    workspaces:
    - name: source
      workspace: shared-workspace
    params:
    - name: IMAGE
      value: quay.io/your-org/log-monitoring/ui:$(params.image-tag)
    - name: DOCKERFILE
      value: ./ui/Dockerfile
  
  - name: run-tests
    taskRef:
      name: golang-test
    runAfter:
    - fetch-source
    workspaces:
    - name: source
      workspace: shared-workspace
  
  - name: deploy-dev
    taskRef:
      name: openshift-client
    runAfter:
    - build-alert-engine
    - build-ui
    - run-tests
    params:
    - name: SCRIPT
      value: |
        kubectl apply -k deploy/overlays/development/
        kubectl rollout status deployment/alert-engine -n log-monitoring
        kubectl rollout status deployment/log-monitoring-ui -n log-monitoring
```

### Dockerfile Configurations

**Alert Engine Dockerfile**:
```dockerfile
# Build stage
FROM registry.redhat.io/ubi8/go-toolset:latest AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o alert-engine ./cmd/server/main.go

# Runtime stage
FROM registry.redhat.io/ubi8/ubi-minimal:latest

RUN microdnf update && microdnf install -y ca-certificates && microdnf clean all

WORKDIR /app
COPY --from=builder /app/alert-engine .
COPY --from=builder /app/configs/config.yaml ./configs/

EXPOSE 8080 8081

USER 1001

CMD ["./alert-engine"]
```

**UI Dockerfile**:
```dockerfile
# Build stage
FROM registry.redhat.io/ubi8/nodejs-16:latest AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# Runtime stage
FROM registry.redhat.io/ubi8/nginx-120:latest

COPY --from=builder /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 3000

USER 1001

CMD ["nginx", "-g", "daemon off;"]
```

## Implementation Timeline

### Week 1: Infrastructure Setup
- **Day 1-2**: OpenShift project setup, namespace creation, RBAC configuration
- **Day 3-4**: AMQ Streams (Kafka) cluster deployment and topic configuration
- **Day 5**: Redis deployment and OpenShift Logging configuration

### Week 2: Core Development
- **Day 1-3**: Go alert engine development (basic rule evaluation)
- **Day 4-5**: Kafka consumer implementation and log processing
- **Day 6-7**: Slack notification integration and testing

### Week 3: UI and Integration
- **Day 1-3**: React UI development (alert creation form)
- **Day 4-5**: API integration and end-to-end testing
- **Day 6-7**: Performance testing and documentation

## Success Criteria Validation

### Functional Validation
1. **Log Pipeline**: Successfully ingest logs from OpenShift pods to Kafka
2. **Alert Processing**: Evaluate simple threshold-based rules within 30 seconds
3. **Notifications**: Deliver Slack alerts with proper formatting and context
4. **UI Functionality**: Create and manage alert rules through web interface

### Performance Validation
1. **Throughput**: Process 1,000 logs/second without message loss
2. **Latency**: Alert delivery within 30 seconds of threshold breach
3. **Resource Usage**: Stay within allocated CPU and memory limits
4. **Reliability**: 99% uptime during testing period

### Integration Validation
1. **OpenShift Integration**: Proper metadata enrichment from Kubernetes API
2. **AMQ Streams**: Reliable message processing with partition balancing
3. **Redis State**: Consistent alert state management and counter tracking
4. **Slack Integration**: Rich message formatting with actionable buttons

## Risk Mitigation

### Technical Risks
1. **Message Loss**: Implement Kafka acknowledgments and offset management
2. **Memory Leaks**: Use proper Go garbage collection and resource cleanup
3. **Alert Storms**: Implement cooldown periods and alert grouping
4. **Network Failures**: Add retry logic and circuit breakers

### Operational Risks
1. **Resource Exhaustion**: Set resource limits and monitoring alerts
2. **Configuration Errors**: Validate configurations before applying
3. **Security Vulnerabilities**: Use RedHat verified images and security scanning
4. **Data Loss**: Implement proper backup strategies for Redis state

This technical specification provides a comprehensive foundation for implementing Phase 0 of the log monitoring system using RedHat/OpenShift ecosystem products, ensuring production-readiness while maintaining simplicity for the proof of concept phase.