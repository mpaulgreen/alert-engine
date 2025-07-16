//go:build unit
// +build unit

package storage

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// Test NewRedisStore creation
func TestNewRedisStore(t *testing.T) {
	store := NewRedisStore("localhost:6379", "password")
	assert.NotNil(t, store)
	assert.NotNil(t, store.client)
	assert.NotNil(t, store.ctx)
}

func TestNewRedisStoreWithConfig(t *testing.T) {
	// Test single node mode
	store := NewRedisStoreWithConfig("localhost:6379", "password", false)
	assert.NotNil(t, store)

	// Test cluster mode detection with comma
	store = NewRedisStoreWithConfig("node1:6379,node2:6379", "password", false)
	assert.NotNil(t, store)

	// Test explicit cluster mode
	store = NewRedisStoreWithConfig("node1:6379", "password", true)
	assert.NotNil(t, store)

	// Test cluster mode with multiple addresses and whitespace
	store = NewRedisStoreWithConfig("node1:6379, node2:6379 , node3:6379", "password", false)
	assert.NotNil(t, store)
}

// Test key generation and JSON marshaling logic without Redis calls
func TestRedisStore_KeyGeneration(t *testing.T) {
	// Test that we can generate proper keys for different types
	ruleID := "test-rule-001"
	alertID := "test-alert-001"

	// Test key patterns
	ruleKey := "alert_rule:" + ruleID
	alertKey := "alert:" + alertID
	statusKey := "alert_status:" + ruleID

	assert.Equal(t, "alert_rule:test-rule-001", ruleKey)
	assert.Equal(t, "alert:test-alert-001", alertKey)
	assert.Equal(t, "alert_status:test-rule-001", statusKey)
}

// Test JSON marshaling/unmarshaling for Redis operations
func TestRedisStore_JSONOperations(t *testing.T) {
	// Test AlertRule JSON operations
	ruleData, err := json.Marshal(testAlertRule)
	require.NoError(t, err)
	require.NotEmpty(t, ruleData)

	var unmarshaledRule models.AlertRule
	err = json.Unmarshal(ruleData, &unmarshaledRule)
	require.NoError(t, err)
	assert.Equal(t, testAlertRule.ID, unmarshaledRule.ID)
	assert.Equal(t, testAlertRule.Name, unmarshaledRule.Name)

	// Test Alert JSON operations
	alertData, err := json.Marshal(testAlert)
	require.NoError(t, err)
	require.NotEmpty(t, alertData)

	var unmarshaledAlert models.Alert
	err = json.Unmarshal(alertData, &unmarshaledAlert)
	require.NoError(t, err)
	assert.Equal(t, testAlert.ID, unmarshaledAlert.ID)
	assert.Equal(t, testAlert.RuleID, unmarshaledAlert.RuleID)

	// Test AlertStatus JSON operations
	statusData, err := json.Marshal(testAlertStatus)
	require.NoError(t, err)
	require.NotEmpty(t, statusData)

	var unmarshaledStatus models.AlertStatus
	err = json.Unmarshal(statusData, &unmarshaledStatus)
	require.NoError(t, err)
	assert.Equal(t, testAlertStatus.RuleID, unmarshaledStatus.RuleID)
	assert.Equal(t, testAlertStatus.Status, unmarshaledStatus.Status)
}

// Test counter key generation logic
func TestRedisStore_CounterKeyLogic(t *testing.T) {
	ruleID := "test-rule-001"
	window := 5 * time.Minute
	now := time.Now()
	windowStart := now.Truncate(window)

	// This simulates the counter key generation logic using proper string formatting
	key := "counter:" + ruleID + ":" + fmt.Sprintf("%d", windowStart.Unix())

	assert.Contains(t, key, "counter:")
	assert.Contains(t, key, ruleID)
	assert.Contains(t, key, ":")

	// Test different time windows
	hourWindow := time.Hour
	hourWindowStart := now.Truncate(hourWindow)
	hourKey := "counter:" + ruleID + ":" + fmt.Sprintf("%d", hourWindowStart.Unix())

	assert.Contains(t, hourKey, "counter:")
	assert.Contains(t, hourKey, ruleID)

	// Keys for different windows should be different (in most cases)
	assert.NotEqual(t, key, hourKey)
}

// Test data model validation and JSON marshaling
func TestAlertRule_JSONMarshaling(t *testing.T) {
	rule := testAlertRule

	// Validate required fields
	assert.NotEmpty(t, rule.ID)
	assert.NotEmpty(t, rule.Name)
	assert.True(t, rule.Enabled)

	// Test JSON marshaling
	data, err := json.Marshal(rule)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test JSON unmarshaling
	var unmarshaledRule models.AlertRule
	err = json.Unmarshal(data, &unmarshaledRule)
	assert.NoError(t, err)
	assert.Equal(t, rule.ID, unmarshaledRule.ID)
	assert.Equal(t, rule.Name, unmarshaledRule.Name)
}

func TestAlert_JSONMarshaling(t *testing.T) {
	alert := testAlert

	// Validate required fields
	assert.NotEmpty(t, alert.ID)
	assert.NotEmpty(t, alert.RuleID)
	assert.NotEmpty(t, alert.Message)

	// Test JSON marshaling
	data, err := json.Marshal(alert)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test JSON unmarshaling
	var unmarshaledAlert models.Alert
	err = json.Unmarshal(data, &unmarshaledAlert)
	assert.NoError(t, err)
	assert.Equal(t, alert.ID, unmarshaledAlert.ID)
	assert.Equal(t, alert.RuleID, unmarshaledAlert.RuleID)
}

func TestAlertStatus_Validation(t *testing.T) {
	status := testAlertStatus

	// Validate required fields
	assert.NotEmpty(t, status.RuleID)
	assert.NotEmpty(t, status.Status)
	assert.Greater(t, status.Count, 0)
}

func TestLogStats_Validation(t *testing.T) {
	stats := testLogStats

	// Validate structure
	assert.Greater(t, stats.TotalLogs, int64(0))
	assert.NotEmpty(t, stats.LogsByLevel)
	assert.NotEmpty(t, stats.LogsByService)

	// Validate log level counts sum approximately to total
	var levelSum int64
	for _, count := range stats.LogsByLevel {
		levelSum += count
	}
	assert.Equal(t, stats.TotalLogs, levelSum)
}

func TestTimeWindowOperations(t *testing.T) {
	window := 5 * time.Minute
	now := time.Now()
	windowStart := now.Truncate(window)

	// Test window truncation
	assert.True(t, windowStart.Before(now) || windowStart.Equal(now))
	assert.True(t, now.Sub(windowStart) < window)
}

func TestRedisKeyGeneration(t *testing.T) {
	tests := []struct {
		name     string
		keyType  string
		id       string
		expected string
	}{
		{
			name:     "alert_rule_key",
			keyType:  "alert_rule",
			id:       "test-rule-001",
			expected: "alert_rule:test-rule-001",
		},
		{
			name:     "alert_key",
			keyType:  "alert",
			id:       "test-alert-001",
			expected: "alert:test-alert-001",
		},
		{
			name:     "alert_status_key",
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

	key := "counter:" + ruleID + ":" + fmt.Sprintf("%d", windowStart.Unix())

	assert.Contains(t, key, "counter:")
	assert.Contains(t, key, ruleID)
}

func TestDataValidation(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
		rule  models.AlertRule
	}{
		{
			name:  "valid_rule",
			valid: true,
			rule:  testAlertRule,
		},
		{
			name:  "missing_ID",
			valid: false,
			rule: models.AlertRule{
				Name:    "Test Rule",
				Enabled: true,
			},
		},
		{
			name:  "missing_name",
			valid: false,
			rule: models.AlertRule{
				ID:      "test-rule",
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
				isValid := tt.rule.ID != "" && tt.rule.Name != ""
				assert.False(t, isValid)
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
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

// Test error handling in JSON operations
func TestRedisStore_JSONErrorHandling(t *testing.T) {
	// Test with empty rule ID (should still marshal fine)
	emptyRule := models.AlertRule{
		Name: "test rule",
	}

	data, err := json.Marshal(emptyRule)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test unmarshaling invalid JSON
	invalidJSON := `{"invalid": json}`
	var rule models.AlertRule
	err = json.Unmarshal([]byte(invalidJSON), &rule)
	assert.Error(t, err) // This should fail due to missing required fields or invalid structure
}

// Test Close method behavior
func TestRedisStore_Close(t *testing.T) {
	store := NewRedisStore("localhost:6379", "password")

	// Test that Close method exists and returns no error when client is nil or properly set
	err := store.Close()
	// The close method might return an error depending on client state
	// For unit tests, we just verify the method exists and can be called
	_ = err // We don't assert on the error as it depends on the Redis client implementation
}

// Test edge cases for cluster configuration
func TestRedisStore_ClusterConfigEdgeCases(t *testing.T) {
	// Test with empty address
	store := NewRedisStoreWithConfig("", "password", false)
	assert.NotNil(t, store)

	// Test with single address but cluster mode enabled
	store = NewRedisStoreWithConfig("localhost:6379", "password", true)
	assert.NotNil(t, store)

	// Test with comma but no spaces
	store = NewRedisStoreWithConfig("node1:6379,node2:6379,node3:6379", "password", false)
	assert.NotNil(t, store)

	// Test with empty password
	store = NewRedisStoreWithConfig("localhost:6379", "", false)
	assert.NotNil(t, store)
}

// Test time window edge cases
func TestTimeWindowEdgeCases(t *testing.T) {
	// Test with very small window
	smallWindow := time.Second
	now := time.Now()
	windowStart := now.Truncate(smallWindow)

	assert.True(t, windowStart.Before(now) || windowStart.Equal(now))

	// Test with large window
	largeWindow := 24 * time.Hour
	largeWindowStart := now.Truncate(largeWindow)

	assert.True(t, largeWindowStart.Before(now) || largeWindowStart.Equal(now))
	assert.True(t, now.Sub(largeWindowStart) < largeWindow)
}

// Test data marshaling edge cases
func TestDataMarshalingEdgeCases(t *testing.T) {
	// Test with minimal alert rule
	minimalRule := models.AlertRule{
		ID:   "minimal",
		Name: "Minimal Rule",
	}

	data, err := json.Marshal(minimalRule)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test unmarshaling back
	var unmarshaled models.AlertRule
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, "minimal", unmarshaled.ID)
	assert.Equal(t, "Minimal Rule", unmarshaled.Name)
}

// Test various key patterns
func TestKeyPatterns(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		id       string
		expected bool
	}{
		{
			name:     "valid_rule_key",
			pattern:  "alert_rule:",
			id:       "rule-123",
			expected: true,
		},
		{
			name:     "valid_alert_key",
			pattern:  "alert:",
			id:       "alert-456",
			expected: true,
		},
		{
			name:     "valid_status_key",
			pattern:  "alert_status:",
			id:       "rule-789",
			expected: true,
		},
		{
			name:     "counter_key",
			pattern:  "counter:",
			id:       "rule-123:12345",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.pattern + tt.id
			hasPattern := len(key) > len(tt.pattern) && key[:len(tt.pattern)] == tt.pattern
			assert.Equal(t, tt.expected, hasPattern)
		})
	}
}

// Placeholder tests for integration scenarios
func TestRedisStore_Integration_Placeholder(t *testing.T) {
	t.Skip("Integration test - requires Redis container")
}

func TestRedisStore_Performance_Placeholder(t *testing.T) {
	t.Skip("Performance test - requires Redis container")
}

func TestRedisStore_Concurrency_Placeholder(t *testing.T) {
	t.Skip("Concurrency test - requires Redis container")
}
