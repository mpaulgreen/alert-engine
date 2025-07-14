//go:build unit

package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/log-monitoring/alert-engine/internal/alerting"
	"github.com/log-monitoring/alert-engine/internal/alerting/tests/mocks"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

func TestNewEngine(t *testing.T) {
	t.Run("creates engine successfully", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		engine := alerting.NewEngine(mockStore, mockNotifier)

		assert.NotNil(t, engine)
	})

	t.Run("loads existing rules on startup", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Add some rules to the store
		rule := models.AlertRule{
			ID:      "test-rule-1",
			Name:    "Test Rule 1",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "gt",
			},
		}
		mockStore.SaveAlertRule(rule)

		engine := alerting.NewEngine(mockStore, mockNotifier)
		rules := engine.GetRules()

		assert.Len(t, rules, 1)
		assert.Equal(t, "test-rule-1", rules[0].ID)
	})

	t.Run("handles store error during rule loading", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Make the store fail
		mockStore.SetShouldFail(true, "failed to load rules")

		// Should still create engine even if rule loading fails
		engine := alerting.NewEngine(mockStore, mockNotifier)
		assert.NotNil(t, engine)

		rules := engine.GetRules()
		assert.Len(t, rules, 0) // No rules loaded due to error
	})
}

func TestEngine_EvaluateLog(t *testing.T) {
	t.Run("evaluates log against matching rule", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Set up a rule that should match
		rule := models.AlertRule{
			ID:      "test-rule",
			Name:    "Test Rule",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Namespace:  "production",
				Service:    "user-service",
				Keywords:   []string{"failed"},
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "gt",
			},
			Actions: models.AlertActions{
				Severity: "high",
			},
		}
		mockStore.SaveAlertRule(rule)

		engine := alerting.NewEngine(mockStore, mockNotifier)

		// Create a log entry that should match
		logEntry := models.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "User authentication failed",
			Kubernetes: models.KubernetesInfo{
				Namespace: "production",
				Pod:       "user-service-abc123",
				Container: "user-service",
				Labels: map[string]string{
					"app": "user-service",
				},
			},
		}

		// Set counter to trigger alert (threshold is 1, so counter > 1 will trigger)
		// The engine will call IncrementCounter which increments from current value
		mockStore.SetCounter("test-rule", time.Minute, 1) // Will be incremented to 2

		engine.EvaluateLog(logEntry)

		// Check that an alert was sent
		assert.True(t, mockNotifier.WasCalled())
		alerts := mockNotifier.GetSentAlerts()
		assert.Len(t, alerts, 1)

		alert := alerts[0]
		assert.Equal(t, "test-rule", alert.RuleID)
		assert.Equal(t, "Test Rule", alert.RuleName)
		assert.Equal(t, "high", alert.Severity)
		assert.Equal(t, "pending", alert.Status) // Status is "pending" when sent to notifier
	})

	t.Run("skips disabled rules", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Set up a disabled rule
		rule := models.AlertRule{
			ID:      "disabled-rule",
			Name:    "Disabled Rule",
			Enabled: false, // Disabled
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "gt",
			},
		}
		mockStore.SaveAlertRule(rule)

		engine := alerting.NewEngine(mockStore, mockNotifier)

		logEntry := models.LogEntry{
			Level:   "ERROR",
			Message: "Test error message",
		}

		engine.EvaluateLog(logEntry)

		// No alerts should be sent for disabled rules
		assert.False(t, mockNotifier.WasCalled())
	})

	t.Run("does not trigger alert when conditions don't match", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		rule := models.AlertRule{
			ID:      "test-rule",
			Name:    "Test Rule",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Namespace:  "production",
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "gt",
			},
		}
		mockStore.SaveAlertRule(rule)

		engine := alerting.NewEngine(mockStore, mockNotifier)

		// Log entry with wrong level
		logEntry := models.LogEntry{
			Level:   "INFO", // Should be ERROR to match
			Message: "Test message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "production",
			},
		}

		engine.EvaluateLog(logEntry)

		assert.False(t, mockNotifier.WasCalled())
	})

	t.Run("does not trigger alert when threshold not met", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		rule := models.AlertRule{
			ID:      "test-rule",
			Name:    "Test Rule",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  5, // High threshold
				TimeWindow: time.Minute,
				Operator:   "gt",
			},
		}
		mockStore.SaveAlertRule(rule)

		engine := alerting.NewEngine(mockStore, mockNotifier)

		logEntry := models.LogEntry{
			Level:   "ERROR",
			Message: "Test error",
		}

		// Counter will be 1, which is <= 5, so no alert
		engine.EvaluateLog(logEntry)

		assert.False(t, mockNotifier.WasCalled())
	})

	t.Run("handles notifier error", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Make notifier fail
		mockNotifier.SetShouldFail(true, "notification failed")

		rule := models.AlertRule{
			ID:      "test-rule",
			Name:    "Test Rule",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "gt",
			},
		}
		mockStore.SaveAlertRule(rule)

		engine := alerting.NewEngine(mockStore, mockNotifier)

		logEntry := models.LogEntry{
			Level:   "ERROR",
			Message: "Test error",
		}

		// Set counter to trigger alert
		mockStore.SetCounter("test-rule", time.Minute, 1) // Will be incremented to 2

		engine.EvaluateLog(logEntry)

		// Notifier should have been called even though it failed
		assert.True(t, mockNotifier.WasCalled())
	})

	t.Run("handles counter increment error", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		rule := models.AlertRule{
			ID:      "test-rule",
			Name:    "Test Rule",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "gt",
			},
		}
		mockStore.SaveAlertRule(rule)

		engine := alerting.NewEngine(mockStore, mockNotifier)

		// Make store fail counter operations
		mockStore.SetShouldFail(true, "counter failed")

		logEntry := models.LogEntry{
			Level:   "ERROR",
			Message: "Test error",
		}

		engine.EvaluateLog(logEntry)

		// No alerts should be sent due to counter error
		assert.False(t, mockNotifier.WasCalled())
	})
}

func TestEngine_AddRule(t *testing.T) {
	t.Run("adds rule successfully", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		engine := alerting.NewEngine(mockStore, mockNotifier)

		rule := models.AlertRule{
			ID:      "new-rule",
			Name:    "New Rule",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "gt",
			},
		}

		err := engine.AddRule(rule)
		require.NoError(t, err)

		// Check that the rule was added
		rules := engine.GetRules()
		assert.Len(t, rules, 1)
		assert.Equal(t, "new-rule", rules[0].ID)
		assert.Equal(t, "New Rule", rules[0].Name)

		// Check that timestamps were set
		assert.False(t, rules[0].CreatedAt.IsZero())
		assert.False(t, rules[0].UpdatedAt.IsZero())
	})

	t.Run("handles store error", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Make store fail
		mockStore.SetShouldFail(true, "save failed")

		engine := alerting.NewEngine(mockStore, mockNotifier)

		rule := models.AlertRule{
			ID:   "test-rule",
			Name: "Test Rule",
		}

		err := engine.AddRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save failed")
	})
}

func TestEngine_UpdateRule(t *testing.T) {
	t.Run("updates rule successfully", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Add initial rule
		originalRule := models.AlertRule{
			ID:        "update-rule",
			Name:      "Original Name",
			Enabled:   true,
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Hour),
		}
		mockStore.SaveAlertRule(originalRule)

		engine := alerting.NewEngine(mockStore, mockNotifier)

		// Update the rule
		updatedRule := originalRule
		updatedRule.Name = "Updated Name"
		updatedRule.Enabled = false

		err := engine.UpdateRule(updatedRule)
		require.NoError(t, err)

		// Check that the rule was updated
		rules := engine.GetRules()
		assert.Len(t, rules, 1)
		assert.Equal(t, "Updated Name", rules[0].Name)
		assert.False(t, rules[0].Enabled)

		// Check that UpdatedAt was modified
		assert.True(t, rules[0].UpdatedAt.After(originalRule.UpdatedAt))
	})

	t.Run("handles store error", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Make store fail
		mockStore.SetShouldFail(true, "update failed")

		engine := alerting.NewEngine(mockStore, mockNotifier)

		rule := models.AlertRule{
			ID:   "test-rule",
			Name: "Test Rule",
		}

		err := engine.UpdateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})
}

func TestEngine_DeleteRule(t *testing.T) {
	t.Run("deletes rule successfully", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Add a rule first
		rule := models.AlertRule{
			ID:   "delete-me",
			Name: "Delete Me",
		}
		mockStore.SaveAlertRule(rule)

		engine := alerting.NewEngine(mockStore, mockNotifier)

		// Verify rule exists
		rules := engine.GetRules()
		assert.Len(t, rules, 1)

		// Delete the rule
		err := engine.DeleteRule("delete-me")
		require.NoError(t, err)

		// Verify rule is gone
		rules = engine.GetRules()
		assert.Len(t, rules, 0)
	})

	t.Run("handles non-existent rule", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		engine := alerting.NewEngine(mockStore, mockNotifier)

		err := engine.DeleteRule("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule not found")
	})

	t.Run("handles store error", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Make store fail
		mockStore.SetShouldFail(true, "delete failed")

		engine := alerting.NewEngine(mockStore, mockNotifier)

		err := engine.DeleteRule("test-rule")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})
}

func TestEngine_ReloadRules(t *testing.T) {
	t.Run("reloads rules successfully", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		engine := alerting.NewEngine(mockStore, mockNotifier)

		// Initially no rules
		rules := engine.GetRules()
		assert.Len(t, rules, 0)

		// Add a rule directly to store (simulating external addition)
		rule := models.AlertRule{
			ID:   "external-rule",
			Name: "External Rule",
		}
		mockStore.SaveAlertRule(rule)

		// Reload rules
		err := engine.ReloadRules()
		require.NoError(t, err)

		// Check that the new rule is loaded
		rules = engine.GetRules()
		assert.Len(t, rules, 1)
		assert.Equal(t, "external-rule", rules[0].ID)
	})

	t.Run("handles store error during reload", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		engine := alerting.NewEngine(mockStore, mockNotifier)

		// Make store fail
		mockStore.SetShouldFail(true, "reload failed")

		err := engine.ReloadRules()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reload failed")
	})
}

func TestEngine_GetRules(t *testing.T) {
	t.Run("returns all rules", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		// Add multiple rules
		rules := []models.AlertRule{
			{ID: "rule-1", Name: "Rule 1"},
			{ID: "rule-2", Name: "Rule 2"},
			{ID: "rule-3", Name: "Rule 3"},
		}

		for _, rule := range rules {
			mockStore.SaveAlertRule(rule)
		}

		engine := alerting.NewEngine(mockStore, mockNotifier)

		retrievedRules := engine.GetRules()
		assert.Len(t, retrievedRules, 3)

		// Check that all rules are present (order may vary)
		ruleIDs := make(map[string]bool)
		for _, rule := range retrievedRules {
			ruleIDs[rule.ID] = true
		}

		assert.True(t, ruleIDs["rule-1"])
		assert.True(t, ruleIDs["rule-2"])
		assert.True(t, ruleIDs["rule-3"])
	})

	t.Run("returns empty slice when no rules", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		engine := alerting.NewEngine(mockStore, mockNotifier)

		rules := engine.GetRules()
		assert.NotNil(t, rules)
		assert.Len(t, rules, 0)
	})
}

func TestEngine_Stop(t *testing.T) {
	t.Run("stops engine gracefully", func(t *testing.T) {
		mockStore := mocks.NewMockStateStore()
		mockNotifier := mocks.NewMockNotifier()

		engine := alerting.NewEngine(mockStore, mockNotifier)

		// Stop should not hang or panic
		engine.Stop()
	})
}
