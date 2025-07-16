//go:build unit

package models

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlertRule_JSONMarshaling(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		original := AlertRule{
			ID:          "test-rule-1",
			Name:        "Test Rule",
			Description: "A test alert rule",
			Enabled:     true,
			Conditions: AlertConditions{
				LogLevel:   "ERROR",
				Namespace:  "production",
				Service:    "user-service",
				Keywords:   []string{"error", "failed"},
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
				Operator:   "gt",
			},
			Actions: AlertActions{
				SlackWebhook: "https://hooks.slack.com/services/test",
				Channel:      "#alerts",
				Severity:     "high",
			},
			CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC),
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal from JSON
		var restored AlertRule
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		// Verify all fields are correctly preserved
		assert.Equal(t, original.ID, restored.ID)
		assert.Equal(t, original.Name, restored.Name)
		assert.Equal(t, original.Description, restored.Description)
		assert.Equal(t, original.Enabled, restored.Enabled)
		assert.Equal(t, original.Conditions.LogLevel, restored.Conditions.LogLevel)
		assert.Equal(t, original.Conditions.Namespace, restored.Conditions.Namespace)
		assert.Equal(t, original.Conditions.Service, restored.Conditions.Service)
		assert.Equal(t, original.Conditions.Keywords, restored.Conditions.Keywords)
		assert.Equal(t, original.Conditions.Threshold, restored.Conditions.Threshold)
		assert.Equal(t, original.Conditions.TimeWindow, restored.Conditions.TimeWindow)
		assert.Equal(t, original.Conditions.Operator, restored.Conditions.Operator)
		assert.Equal(t, original.Actions.SlackWebhook, restored.Actions.SlackWebhook)
		assert.Equal(t, original.Actions.Channel, restored.Actions.Channel)
		assert.Equal(t, original.Actions.Severity, restored.Actions.Severity)
		assert.True(t, original.CreatedAt.Equal(restored.CreatedAt))
		assert.True(t, original.UpdatedAt.Equal(restored.UpdatedAt))
	})

	t.Run("unmarshal with missing optional fields", func(t *testing.T) {
		jsonData := `{
			"id": "test-rule-2",
			"name": "Minimal Rule",
			"enabled": true,
			"conditions": {
				"log_level": "ERROR",
				"threshold": 1,
				"time_window": 60000000000,
				"operator": "gt"
			},
			"actions": {
				"severity": "medium"
			}
		}`

		var rule AlertRule
		err := json.Unmarshal([]byte(jsonData), &rule)
		require.NoError(t, err)

		assert.Equal(t, "test-rule-2", rule.ID)
		assert.Equal(t, "Minimal Rule", rule.Name)
		assert.Equal(t, "", rule.Description)
		assert.True(t, rule.Enabled)
		assert.Equal(t, "ERROR", rule.Conditions.LogLevel)
		assert.Equal(t, "", rule.Conditions.Namespace)
		assert.Equal(t, "", rule.Conditions.Service)
		assert.Empty(t, rule.Conditions.Keywords)
		assert.Equal(t, 1, rule.Conditions.Threshold)
		assert.Equal(t, time.Minute, rule.Conditions.TimeWindow)
		assert.Equal(t, "gt", rule.Conditions.Operator)
		assert.Equal(t, "", rule.Actions.SlackWebhook)
		assert.Equal(t, "", rule.Actions.Channel)
		assert.Equal(t, "medium", rule.Actions.Severity)
	})

	t.Run("unmarshal with invalid JSON", func(t *testing.T) {
		invalidJSON := `{
			"id": "test-rule-3",
			"name": "Invalid Rule",
			"enabled": "not-a-boolean"
		}`

		var rule AlertRule
		err := json.Unmarshal([]byte(invalidJSON), &rule)
		assert.Error(t, err)
	})
}

func TestAlert_JSONMarshaling(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		logEntry := LogEntry{
			Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 123000000, time.UTC),
			Level:     "ERROR",
			Message:   "Database connection failed",
			Kubernetes: KubernetesInfo{
				Namespace: "production",
				Pod:       "user-service-abc123",
				Container: "user-service",
				Labels: map[string]string{
					"app":     "user-service",
					"version": "1.2.3",
				},
			},
			Host: "worker-node-01",
		}

		original := Alert{
			ID:        "alert-12345",
			RuleID:    "db-connection-rule",
			RuleName:  "Database Connection Error",
			LogEntry:  logEntry,
			Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 456000000, time.UTC),
			Severity:  "high",
			Status:    "pending",
			Message:   "Database connection error detected",
			Count:     1,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal from JSON
		var restored Alert
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		// Verify all fields are correctly preserved
		assert.Equal(t, original.ID, restored.ID)
		assert.Equal(t, original.RuleID, restored.RuleID)
		assert.Equal(t, original.RuleName, restored.RuleName)
		assert.Equal(t, original.LogEntry.Timestamp, restored.LogEntry.Timestamp)
		assert.Equal(t, original.LogEntry.Level, restored.LogEntry.Level)
		assert.Equal(t, original.LogEntry.Message, restored.LogEntry.Message)
		assert.Equal(t, original.LogEntry.Kubernetes.Namespace, restored.LogEntry.Kubernetes.Namespace)
		assert.Equal(t, original.LogEntry.Kubernetes.Pod, restored.LogEntry.Kubernetes.Pod)
		assert.Equal(t, original.LogEntry.Kubernetes.Container, restored.LogEntry.Kubernetes.Container)
		assert.Equal(t, original.LogEntry.Kubernetes.Labels, restored.LogEntry.Kubernetes.Labels)
		assert.Equal(t, original.LogEntry.Host, restored.LogEntry.Host)
		assert.True(t, original.Timestamp.Equal(restored.Timestamp))
		assert.Equal(t, original.Severity, restored.Severity)
		assert.Equal(t, original.Status, restored.Status)
		assert.Equal(t, original.Message, restored.Message)
		assert.Equal(t, original.Count, restored.Count)
	})

	t.Run("unmarshal with empty nested structures", func(t *testing.T) {
		jsonData := `{
			"id": "alert-empty",
			"rule_id": "rule-empty",
			"rule_name": "Empty Rule",
			"log_entry": {
				"timestamp": "2024-01-15T10:30:45.123Z",
				"level": "INFO",
				"message": "Test message",
				"kubernetes": {
					"namespace": "",
					"pod": "",
					"container": "",
					"labels": {}
				},
				"host": ""
			},
			"timestamp": "2024-01-15T10:30:45.456Z",
			"severity": "low",
			"status": "sent",
			"message": "Test alert",
			"count": 0
		}`

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		assert.Equal(t, "alert-empty", alert.ID)
		assert.Equal(t, "", alert.LogEntry.Kubernetes.Namespace)
		assert.Equal(t, "", alert.LogEntry.Kubernetes.Pod)
		assert.Equal(t, "", alert.LogEntry.Kubernetes.Container)
		assert.Empty(t, alert.LogEntry.Kubernetes.Labels)
		assert.Equal(t, "", alert.LogEntry.Host)
		assert.Equal(t, 0, alert.Count)
	})
}

func TestAlertStatus_JSONMarshaling(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		original := AlertStatus{
			RuleID:      "rule-status-1",
			LastTrigger: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Count:       5,
			Status:      "active",
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal from JSON
		var restored AlertStatus
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, original.RuleID, restored.RuleID)
		assert.True(t, original.LastTrigger.Equal(restored.LastTrigger))
		assert.Equal(t, original.Count, restored.Count)
		assert.Equal(t, original.Status, restored.Status)
	})
}

func TestAlertConditions_Validation(t *testing.T) {
	t.Run("valid log levels", func(t *testing.T) {
		validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
		for _, level := range validLevels {
			conditions := AlertConditions{
				LogLevel:   level,
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "gt",
			}

			// Test that it can be marshaled/unmarshaled without error
			data, err := json.Marshal(conditions)
			require.NoError(t, err)

			var restored AlertConditions
			err = json.Unmarshal(data, &restored)
			require.NoError(t, err)
			assert.Equal(t, level, restored.LogLevel)
		}
	})

	t.Run("valid operators", func(t *testing.T) {
		validOperators := []string{"gt", "lt", "eq", "contains"}
		for _, operator := range validOperators {
			conditions := AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   operator,
			}

			// Test that it can be marshaled/unmarshaled without error
			data, err := json.Marshal(conditions)
			require.NoError(t, err)

			var restored AlertConditions
			err = json.Unmarshal(data, &restored)
			require.NoError(t, err)
			assert.Equal(t, operator, restored.Operator)
		}
	})

	t.Run("time window parsing", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    int64
			expected time.Duration
		}{
			{"1 minute", 60000000000, time.Minute},
			{"5 minutes", 300000000000, 5 * time.Minute},
			{"1 hour", 3600000000000, time.Hour},
			{"30 seconds", 30000000000, 30 * time.Second},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				jsonData := fmt.Sprintf(`{
					"log_level": "ERROR",
					"threshold": 1,
					"time_window": %d,
					"operator": "gt"
				}`, tc.input)

				var conditions AlertConditions
				err := json.Unmarshal([]byte(jsonData), &conditions)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, conditions.TimeWindow)
			})
		}
	})
}

func TestAlertActions_Validation(t *testing.T) {
	t.Run("valid severity levels", func(t *testing.T) {
		validSeverities := []string{"low", "medium", "high", "critical"}
		for _, severity := range validSeverities {
			actions := AlertActions{
				SlackWebhook: "https://hooks.slack.com/services/test",
				Channel:      "#alerts",
				Severity:     severity,
			}

			// Test that it can be marshaled/unmarshaled without error
			data, err := json.Marshal(actions)
			require.NoError(t, err)

			var restored AlertActions
			err = json.Unmarshal(data, &restored)
			require.NoError(t, err)
			assert.Equal(t, severity, restored.Severity)
		}
	})

	t.Run("slack webhook URL format", func(t *testing.T) {
		testCases := []struct {
			name    string
			webhook string
			valid   bool
		}{
			{"valid slack webhook", "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX", true},
			{"valid slack webhook with path", "https://hooks.slack.com/services/test/webhook", true},
			{"empty webhook", "", true},        // Empty is valid (optional field)
			{"invalid URL", "not-a-url", true}, // We don't validate URL format in marshaling
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				actions := AlertActions{
					SlackWebhook: tc.webhook,
					Channel:      "#alerts",
					Severity:     "medium",
				}

				// Test that it can be marshaled/unmarshaled without error
				data, err := json.Marshal(actions)
				require.NoError(t, err)

				var restored AlertActions
				err = json.Unmarshal(data, &restored)
				require.NoError(t, err)
				assert.Equal(t, tc.webhook, restored.SlackWebhook)
			})
		}
	})
}

func TestAlert_StatusValidation(t *testing.T) {
	t.Run("valid alert statuses", func(t *testing.T) {
		validStatuses := []string{"pending", "sent", "failed"}
		for _, status := range validStatuses {
			alert := Alert{
				ID:        "test-alert",
				RuleID:    "test-rule",
				RuleName:  "Test Rule",
				LogEntry:  LogEntry{},
				Timestamp: time.Now(),
				Severity:  "medium",
				Status:    status,
				Message:   "Test message",
				Count:     1,
			}

			// Test that it can be marshaled/unmarshaled without error
			data, err := json.Marshal(alert)
			require.NoError(t, err)

			var restored Alert
			err = json.Unmarshal(data, &restored)
			require.NoError(t, err)
			assert.Equal(t, status, restored.Status)
		}
	})
}

func TestAlertEdgeCases(t *testing.T) {
	t.Run("empty alert rule", func(t *testing.T) {
		empty := AlertRule{}

		data, err := json.Marshal(empty)
		require.NoError(t, err)

		var restored AlertRule
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, "", restored.ID)
		assert.Equal(t, "", restored.Name)
		assert.Equal(t, false, restored.Enabled)
		assert.Equal(t, 0, restored.Conditions.Threshold)
		assert.Equal(t, time.Duration(0), restored.Conditions.TimeWindow)
		assert.Empty(t, restored.Conditions.Keywords)
	})

	t.Run("nil labels map", func(t *testing.T) {
		kubernetesInfo := KubernetesInfo{
			Namespace: "test",
			Pod:       "test-pod",
			Container: "test-container",
			Labels:    nil,
		}

		data, err := json.Marshal(kubernetesInfo)
		require.NoError(t, err)

		var restored KubernetesInfo
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, "test", restored.Namespace)
		assert.Equal(t, "test-pod", restored.Pod)
		assert.Equal(t, "test-container", restored.Container)
		assert.Nil(t, restored.Labels)
	})

	t.Run("zero timestamp", func(t *testing.T) {
		alert := Alert{
			ID:        "test-alert",
			RuleID:    "test-rule",
			RuleName:  "Test Rule",
			LogEntry:  LogEntry{},
			Timestamp: time.Time{}, // Zero timestamp
			Severity:  "medium",
			Status:    "pending",
			Message:   "Test message",
			Count:     1,
		}

		data, err := json.Marshal(alert)
		require.NoError(t, err)

		var restored Alert
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.True(t, restored.Timestamp.IsZero())
	})
}
