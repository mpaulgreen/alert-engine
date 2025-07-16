//go:build unit

package alerting_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/log-monitoring/alert-engine/internal/alerting/mocks"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

func TestMockNotifier(t *testing.T) {
	t.Run("sends alerts successfully", func(t *testing.T) {
		notifier := mocks.NewMockNotifier()

		alert := models.Alert{
			ID:        "test-alert",
			RuleID:    "test-rule",
			Message:   "Test message",
			Severity:  "high",
			Timestamp: time.Now(),
		}

		err := notifier.SendAlert(alert)
		assert.NoError(t, err)
		assert.True(t, notifier.WasCalled())
		assert.Equal(t, 1, notifier.GetCallCount())
	})

	t.Run("handles send alert failure", func(t *testing.T) {
		notifier := mocks.NewMockNotifier()
		notifier.SetShouldFail(true, "notification failed")

		alert := models.Alert{
			ID:      "test-alert",
			RuleID:  "test-rule",
			Message: "Test message",
		}

		err := notifier.SendAlert(alert)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock error: notification failed")
	})

	t.Run("tests connection successfully", func(t *testing.T) {
		notifier := mocks.NewMockNotifier()

		err := notifier.TestConnection()
		assert.NoError(t, err)
	})

	t.Run("handles connection test failure", func(t *testing.T) {
		notifier := mocks.NewMockNotifier()
		notifier.SetShouldFail(true, "connection test failed")

		err := notifier.TestConnection()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock error: connection test failed")
	})

	t.Run("tracks sent alerts correctly", func(t *testing.T) {
		notifier := mocks.NewMockNotifier()

		alert1 := models.Alert{ID: "alert1", RuleID: "rule1", Message: "Message 1"}
		alert2 := models.Alert{ID: "alert2", RuleID: "rule1", Message: "Message 2"}
		alert3 := models.Alert{ID: "alert3", RuleID: "rule2", Message: "Message 3"}

		notifier.SendAlert(alert1)
		notifier.SendAlert(alert2)
		notifier.SendAlert(alert3)

		// Test GetSentAlerts
		sentAlerts := notifier.GetSentAlerts()
		assert.Len(t, sentAlerts, 3)

		// Test GetSentAlertsCount
		assert.Equal(t, 3, notifier.GetSentAlertsCount())

		// Test GetLastSentAlert
		lastAlert := notifier.GetLastSentAlert()
		assert.NotNil(t, lastAlert)
		assert.Equal(t, "alert3", lastAlert.ID)

		// Test FindAlertByRuleID
		foundAlert := notifier.FindAlertByRuleID("rule1")
		assert.NotNil(t, foundAlert)
		assert.Equal(t, "rule1", foundAlert.RuleID)

		// Test CountAlertsByRuleID
		count := notifier.CountAlertsByRuleID("rule1")
		assert.Equal(t, 2, count)

		// Test CountAlertsBySeverity (assuming default severity is empty)
		severityCount := notifier.CountAlertsBySeverity("")
		assert.Equal(t, 3, severityCount)
	})

	t.Run("resets state correctly", func(t *testing.T) {
		notifier := mocks.NewMockNotifier()

		// Send some alerts
		alert := models.Alert{ID: "test", RuleID: "rule", Message: "Test"}
		notifier.SendAlert(alert)

		assert.Equal(t, 1, notifier.GetSentAlertsCount())
		assert.True(t, notifier.WasCalled())

		// Reset
		notifier.Reset()

		assert.Equal(t, 0, notifier.GetSentAlertsCount())
		assert.Equal(t, 0, notifier.GetCallCount())
		assert.False(t, notifier.WasCalled())

		lastAlert := notifier.GetLastSentAlert()
		assert.Nil(t, lastAlert)
	})
}

func TestMockStateStore(t *testing.T) {
	t.Run("handles alert rules correctly", func(t *testing.T) {
		store := mocks.NewMockStateStore()

		rule := models.AlertRule{
			ID:   "test-rule",
			Name: "Test Rule",
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
			},
		}

		// Save rule
		err := store.SaveAlertRule(rule)
		assert.NoError(t, err)

		// Get all rules
		rules, err := store.GetAlertRules()
		assert.NoError(t, err)
		assert.Len(t, rules, 1)
		assert.Equal(t, "test-rule", rules[0].ID)

		// Get specific rule
		retrievedRule, err := store.GetAlertRule("test-rule")
		assert.NoError(t, err)
		assert.Equal(t, "test-rule", retrievedRule.ID)

		// Delete rule
		err = store.DeleteAlertRule("test-rule")
		assert.NoError(t, err)

		// Verify deletion
		rules, err = store.GetAlertRules()
		assert.NoError(t, err)
		assert.Len(t, rules, 0)
	})

	t.Run("handles store failures", func(t *testing.T) {
		store := mocks.NewMockStateStore()
		store.SetShouldFail(true, "store operation failed")

		rule := models.AlertRule{ID: "test", Name: "Test"}

		// All operations should fail
		err := store.SaveAlertRule(rule)
		assert.Error(t, err)

		_, err = store.GetAlertRules()
		assert.Error(t, err)

		_, err = store.GetAlertRule("test")
		assert.Error(t, err)

		err = store.DeleteAlertRule("test")
		assert.Error(t, err)
	})

	t.Run("handles counters correctly", func(t *testing.T) {
		store := mocks.NewMockStateStore()

		// Increment counter
		count, err := store.IncrementCounter("test-rule", 5*time.Minute)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Set counter
		store.SetCounter("test-rule", 5*time.Minute, 5)

		// Get counter
		count, err = store.GetCounter("test-rule", 5*time.Minute)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("handles counter failures", func(t *testing.T) {
		store := mocks.NewMockStateStore()
		store.SetShouldFail(true, "counter operation failed")

		_, err := store.IncrementCounter("test-rule", 5*time.Minute)
		assert.Error(t, err)

		_, err = store.GetCounter("test-rule", 5*time.Minute)
		assert.Error(t, err)
	})

	t.Run("handles alert status correctly", func(t *testing.T) {
		store := mocks.NewMockStateStore()

		// Set alert status
		status := models.AlertStatus{
			RuleID:      "test-rule",
			LastTrigger: time.Now(),
			Count:       1,
			Status:      "triggered",
		}
		err := store.SetAlertStatus("test-rule", status)
		assert.NoError(t, err)

		// Get alert status
		retrievedStatus, err := store.GetAlertStatus("test-rule")
		assert.NoError(t, err)
		assert.Equal(t, "triggered", retrievedStatus.Status)
		assert.True(t, retrievedStatus.LastTrigger.After(time.Now().Add(-time.Minute)))
	})

	t.Run("handles alert status failures", func(t *testing.T) {
		store := mocks.NewMockStateStore()
		store.SetShouldFail(true, "alert status operation failed")

		status := models.AlertStatus{
			RuleID: "test-rule",
			Status: "triggered",
		}
		err := store.SetAlertStatus("test-rule", status)
		assert.Error(t, err)

		_, err = store.GetAlertStatus("test-rule")
		assert.Error(t, err)
	})

	t.Run("saves alerts correctly", func(t *testing.T) {
		store := mocks.NewMockStateStore()

		alert := models.Alert{
			ID:      "test-alert",
			RuleID:  "test-rule",
			Message: "Test message",
		}

		err := store.SaveAlert(alert)
		assert.NoError(t, err)
	})

	t.Run("handles save alert failure", func(t *testing.T) {
		store := mocks.NewMockStateStore()
		store.SetShouldFail(true, "save alert failed")

		alert := models.Alert{ID: "test", RuleID: "rule", Message: "Test"}

		err := store.SaveAlert(alert)
		assert.Error(t, err)
	})

	t.Run("provides state statistics", func(t *testing.T) {
		store := mocks.NewMockStateStore()

		// Add some rules and counters
		rule := models.AlertRule{ID: "rule1", Name: "Rule 1"}
		store.SaveAlertRule(rule)
		store.SetCounter("rule1", 5*time.Minute, 10)

		rulesCount := store.GetRulesCount()
		assert.Equal(t, 1, rulesCount)

		countersCount := store.GetCountersCount()
		assert.Equal(t, 1, countersCount)
	})

	t.Run("resets state correctly", func(t *testing.T) {
		store := mocks.NewMockStateStore()

		// Add some data
		rule := models.AlertRule{ID: "test", Name: "Test"}
		store.SaveAlertRule(rule)
		store.SetCounter("test", 5*time.Minute, 5)

		// Verify data exists
		rules, _ := store.GetAlertRules()
		assert.Len(t, rules, 1)
		assert.Equal(t, 1, store.GetRulesCount())

		// Reset
		store.Reset()

		// Verify reset
		rules, _ = store.GetAlertRules()
		assert.Len(t, rules, 0)
		assert.Equal(t, 0, store.GetRulesCount())
		assert.Equal(t, 0, store.GetCountersCount())
	})
}
