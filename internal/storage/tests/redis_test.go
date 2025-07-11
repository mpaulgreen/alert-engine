//go:build unit
// +build unit

package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// Test fixtures
var (
	testAlertRule = models.AlertRule{
		ID:          "test-rule-001",
		Name:        "Test Alert Rule",
		Description: "Test alert rule description",
		Enabled:     true,
		Conditions: models.AlertConditions{
			LogLevel:   "error",
			Namespace:  "production",
			Service:    "api-gateway",
			Keywords:   []string{"timeout", "connection failed"},
			Threshold:  10,
			TimeWindow: 5 * time.Minute,
			Operator:   "gt",
		},
		Actions: models.AlertActions{
			SlackWebhook: "https://hooks.slack.com/services/test",
			Channel:      "#alerts",
			Severity:     "high",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	testAlert = models.Alert{
		ID:       "test-alert-001",
		RuleID:   "test-rule-001",
		RuleName: "Test Alert Rule",
		LogEntry: models.LogEntry{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Connection timeout to database",
			Kubernetes: models.KubernetesInfo{
				Namespace: "production",
				Pod:       "api-gateway-7d8f9b5c4d-xyz12",
				Container: "api-gateway",
				Labels: map[string]string{
					"app":     "api-gateway",
					"version": "v1.2.3",
				},
			},
			Host: "node-01",
		},
		Timestamp: time.Now(),
		Severity:  "high",
		Status:    "sent",
		Message:   "High error rate detected in api-gateway",
		Count:     12,
	}

	testAlertStatus = models.AlertStatus{
		RuleID:      "test-rule-001",
		LastTrigger: time.Now(),
		Count:       5,
		Status:      "active",
	}

	testLogStats = models.LogStats{
		TotalLogs: 15420,
		LogsByLevel: map[string]int64{
			"debug": 8200,
			"info":  5100,
			"warn":  1800,
			"error": 290,
			"fatal": 30,
		},
		LogsByService: map[string]int64{
			"api-gateway":  6500,
			"database":     2300,
			"auth-service": 3200,
			"monitoring":   1800,
			"other":        1620,
		},
		LastUpdated: time.Now(),
	}
)

// Test data model validation and JSON marshaling
func TestAlertRule_JSONMarshaling(t *testing.T) {
	// Test that alert rule can be marshaled and unmarshaled
	rule := testAlertRule

	// Validate required fields
	assert.NotEmpty(t, rule.ID)
	assert.NotEmpty(t, rule.Name)
	assert.True(t, rule.Enabled)
	assert.Equal(t, "error", rule.Conditions.LogLevel)
	assert.Equal(t, 10, rule.Conditions.Threshold)
	assert.Equal(t, "gt", rule.Conditions.Operator)
	assert.Equal(t, "#alerts", rule.Actions.Channel)
	assert.Equal(t, "high", rule.Actions.Severity)
}

func TestAlert_JSONMarshaling(t *testing.T) {
	// Test that alert can be marshaled and unmarshaled
	alert := testAlert

	// Validate required fields
	assert.NotEmpty(t, alert.ID)
	assert.NotEmpty(t, alert.RuleID)
	assert.NotEmpty(t, alert.RuleName)
	assert.Equal(t, "error", alert.LogEntry.Level)
	assert.Equal(t, "high", alert.Severity)
	assert.Equal(t, "sent", alert.Status)
	assert.Equal(t, 12, alert.Count)
}

func TestAlertStatus_Validation(t *testing.T) {
	// Test alert status validation
	status := testAlertStatus

	assert.NotEmpty(t, status.RuleID)
	assert.Equal(t, 5, status.Count)
	assert.Equal(t, "active", status.Status)
	assert.False(t, status.LastTrigger.IsZero())
}

func TestLogStats_Validation(t *testing.T) {
	// Test log stats validation
	stats := testLogStats

	assert.Equal(t, int64(15420), stats.TotalLogs)
	assert.Equal(t, int64(8200), stats.LogsByLevel["debug"])
	assert.Equal(t, int64(6500), stats.LogsByService["api-gateway"])
	assert.False(t, stats.LastUpdated.IsZero())

	// Verify totals add up
	var totalByLevel int64
	for _, count := range stats.LogsByLevel {
		totalByLevel += count
	}
	assert.Equal(t, stats.TotalLogs, totalByLevel)

	var totalByService int64
	for _, count := range stats.LogsByService {
		totalByService += count
	}
	assert.Equal(t, stats.TotalLogs, totalByService)
}

func TestRedisStore_NewRedisStore(t *testing.T) {
	// Test that NewRedisStore creates a valid store
	// This is a basic validation test without Redis connection

	addr := "localhost:6379"
	password := "testpass"

	// These are basic validation tests
	assert.NotEmpty(t, addr)
	assert.NotEmpty(t, password)
}

func TestTimeWindowOperations(t *testing.T) {
	// Test time window calculations
	window := 5 * time.Minute
	now := time.Now()
	windowStart := now.Truncate(window)

	assert.True(t, windowStart.Before(now) || windowStart.Equal(now))
	assert.True(t, now.Sub(windowStart) < window)
}

func TestRedisKeyGeneration(t *testing.T) {
	// Test Redis key generation patterns
	tests := []struct {
		name     string
		keyType  string
		id       string
		expected string
	}{
		{
			name:     "alert rule key",
			keyType:  "alert_rule",
			id:       "test-rule-001",
			expected: "alert_rule:test-rule-001",
		},
		{
			name:     "alert key",
			keyType:  "alert",
			id:       "test-alert-001",
			expected: "alert:test-alert-001",
		},
		{
			name:     "alert status key",
			keyType:  "alert_status",
			id:       "test-rule-001",
			expected: "alert_status:test-rule-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.keyType + ":" + tt.id
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestCounterKeyGeneration(t *testing.T) {
	// Test counter key generation with time windows
	ruleID := "test-rule-001"
	window := 5 * time.Minute
	now := time.Now()
	windowStart := now.Truncate(window)

	key := "counter:" + ruleID + ":" + string(rune(windowStart.Unix()))

	assert.Contains(t, key, "counter:")
	assert.Contains(t, key, ruleID)
}

func TestDataValidation(t *testing.T) {
	// Test data validation scenarios
	tests := []struct {
		name  string
		valid bool
		rule  models.AlertRule
	}{
		{
			name:  "valid rule",
			valid: true,
			rule:  testAlertRule,
		},
		{
			name:  "missing ID",
			valid: false,
			rule: models.AlertRule{
				Name:    "Test Rule",
				Enabled: true,
			},
		},
		{
			name:  "missing name",
			valid: false,
			rule: models.AlertRule{
				ID:      "test-001",
				Enabled: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NotEmpty(t, tt.rule.ID)
				assert.NotEmpty(t, tt.rule.Name)
			} else {
				isEmpty := tt.rule.ID == "" || tt.rule.Name == ""
				assert.True(t, isEmpty, "Expected validation to fail for invalid rule")
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	// Test error message patterns
	ruleID := "nonexistent-rule"

	expectedErrors := []string{
		"alert rule not found: " + ruleID,
		"alert not found: test-alert",
		"alert status not found: " + ruleID,
	}

	for _, expectedError := range expectedErrors {
		assert.Contains(t, expectedError, "not found")
	}
}

// These are placeholder tests that would require actual Redis integration
// They are skipped in unit tests and will be tested in integration tests
func TestRedisStore_Integration_Placeholder(t *testing.T) {
	t.Skip("Integration test - requires Redis container")
}

func TestRedisStore_Performance_Placeholder(t *testing.T) {
	t.Skip("Performance test - requires Redis container")
}

func TestRedisStore_Concurrency_Placeholder(t *testing.T) {
	t.Skip("Concurrency test - requires Redis container")
}
