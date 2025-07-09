//go:build integration
// +build integration

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/log-monitoring/alert-engine/internal/storage"
	"github.com/log-monitoring/alert-engine/internal/storage/tests/testcontainers"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

var (
	redisContainer *testcontainers.RedisContainer
	redisStore     *storage.RedisStore
)

func TestMain(m *testing.M) {
	// Setup
	ctx := context.Background()

	var err error
	redisContainer, err = testcontainers.NewRedisContainer(ctx, &testing.T{})
	if err != nil {
		fmt.Printf("Failed to create Redis container: %v\n", err)
		os.Exit(1)
	}

	// Create Redis store
	redisStore = storage.NewRedisStore(
		redisContainer.GetConnectionString(),
		redisContainer.GetPassword(),
	)

	// Run tests
	code := m.Run()

	// Cleanup
	if redisContainer != nil {
		redisContainer.Cleanup()
	}

	os.Exit(code)
}

func setupTest(t *testing.T) {
	// Clean up Redis before each test
	err := redisContainer.FlushDB()
	require.NoError(t, err)
}

func TestRedisStore_Integration_SaveAndGetAlertRule(t *testing.T) {
	setupTest(t)

	// Create test alert rule
	rule := models.AlertRule{
		ID:          "integration-test-rule",
		Name:        "Integration Test Rule",
		Description: "Test rule for integration testing",
		Enabled:     true,
		Conditions: models.AlertConditions{
			LogLevel:   "error",
			Namespace:  "test",
			Service:    "test-service",
			Keywords:   []string{"test", "integration"},
			Threshold:  5,
			TimeWindow: 5 * time.Minute,
			Operator:   "gt",
		},
		Actions: models.AlertActions{
			SlackWebhook: "https://hooks.slack.com/test",
			Channel:      "#test",
			Severity:     "high",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test Save
	err := redisStore.SaveAlertRule(rule)
	require.NoError(t, err)

	// Test Get
	retrievedRule, err := redisStore.GetAlertRule(rule.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedRule)

	// Assert rule data
	assert.Equal(t, rule.ID, retrievedRule.ID)
	assert.Equal(t, rule.Name, retrievedRule.Name)
	assert.Equal(t, rule.Enabled, retrievedRule.Enabled)
	assert.Equal(t, rule.Conditions.LogLevel, retrievedRule.Conditions.LogLevel)
	assert.Equal(t, rule.Conditions.Threshold, retrievedRule.Conditions.Threshold)
	assert.Equal(t, rule.Actions.Channel, retrievedRule.Actions.Channel)
}

func TestRedisStore_Integration_GetAlertRules(t *testing.T) {
	setupTest(t)

	// Create multiple test rules
	rules := []models.AlertRule{
		{
			ID:      "rule-1",
			Name:    "Rule 1",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:  "error",
				Threshold: 10,
				Operator:  "gt",
			},
			Actions: models.AlertActions{
				Channel:  "#alerts",
				Severity: "high",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:      "rule-2",
			Name:    "Rule 2",
			Enabled: false,
			Conditions: models.AlertConditions{
				LogLevel:  "warn",
				Threshold: 20,
				Operator:  "gt",
			},
			Actions: models.AlertActions{
				Channel:  "#warnings",
				Severity: "medium",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Save all rules
	for _, rule := range rules {
		err := redisStore.SaveAlertRule(rule)
		require.NoError(t, err)
	}

	// Get all rules
	retrievedRules, err := redisStore.GetAlertRules()
	require.NoError(t, err)
	require.Len(t, retrievedRules, 2)

	// Verify rules are present
	ruleIDs := make(map[string]bool)
	for _, rule := range retrievedRules {
		ruleIDs[rule.ID] = true
	}
	assert.True(t, ruleIDs["rule-1"])
	assert.True(t, ruleIDs["rule-2"])
}

func TestRedisStore_Integration_DeleteAlertRule(t *testing.T) {
	setupTest(t)

	// Create and save test rule
	rule := models.AlertRule{
		ID:        "delete-test-rule",
		Name:      "Delete Test Rule",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := redisStore.SaveAlertRule(rule)
	require.NoError(t, err)

	// Verify rule exists
	_, err = redisStore.GetAlertRule(rule.ID)
	require.NoError(t, err)

	// Delete rule
	err = redisStore.DeleteAlertRule(rule.ID)
	require.NoError(t, err)

	// Verify rule is deleted
	_, err = redisStore.GetAlertRule(rule.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRedisStore_Integration_CounterOperations(t *testing.T) {
	setupTest(t)

	ruleID := "counter-test-rule"
	window := 5 * time.Minute

	// Test initial counter value
	count, err := redisStore.GetCounter(ruleID, window)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Test increment operations
	for i := 1; i <= 5; i++ {
		newCount, err := redisStore.IncrementCounter(ruleID, window)
		require.NoError(t, err)
		assert.Equal(t, int64(i), newCount)
	}

	// Test get counter after increments
	finalCount, err := redisStore.GetCounter(ruleID, window)
	require.NoError(t, err)
	assert.Equal(t, int64(5), finalCount)
}

func TestRedisStore_Integration_AlertStatusOperations(t *testing.T) {
	setupTest(t)

	ruleID := "status-test-rule"
	status := models.AlertStatus{
		RuleID:      ruleID,
		LastTrigger: time.Now(),
		Count:       3,
		Status:      "active",
	}

	// Test set status
	err := redisStore.SetAlertStatus(ruleID, status)
	require.NoError(t, err)

	// Test get status
	retrievedStatus, err := redisStore.GetAlertStatus(ruleID)
	require.NoError(t, err)
	require.NotNil(t, retrievedStatus)

	assert.Equal(t, status.RuleID, retrievedStatus.RuleID)
	assert.Equal(t, status.Count, retrievedStatus.Count)
	assert.Equal(t, status.Status, retrievedStatus.Status)
	assert.WithinDuration(t, status.LastTrigger, retrievedStatus.LastTrigger, time.Second)
}

func TestRedisStore_Integration_LogStatsOperations(t *testing.T) {
	setupTest(t)

	stats := models.LogStats{
		TotalLogs: 1000,
		LogsByLevel: map[string]int64{
			"debug": 500,
			"info":  300,
			"warn":  150,
			"error": 40,
			"fatal": 10,
		},
		LogsByService: map[string]int64{
			"api":      400,
			"database": 300,
			"auth":     200,
			"other":    100,
		},
		LastUpdated: time.Now(),
	}

	// Test save stats
	err := redisStore.SaveLogStats(stats)
	require.NoError(t, err)

	// Test get stats
	retrievedStats, err := redisStore.GetLogStats()
	require.NoError(t, err)
	require.NotNil(t, retrievedStats)

	assert.Equal(t, stats.TotalLogs, retrievedStats.TotalLogs)
	assert.Equal(t, stats.LogsByLevel["debug"], retrievedStats.LogsByLevel["debug"])
	assert.Equal(t, stats.LogsByService["api"], retrievedStats.LogsByService["api"])
	assert.WithinDuration(t, stats.LastUpdated, retrievedStats.LastUpdated, time.Second)
}

func TestRedisStore_Integration_AlertOperations(t *testing.T) {
	setupTest(t)

	alert := models.Alert{
		ID:       "test-alert",
		RuleID:   "test-rule",
		RuleName: "Test Rule",
		LogEntry: models.LogEntry{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Test error message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test",
				Pod:       "test-pod",
				Container: "test-container",
				Labels: map[string]string{
					"app": "test-app",
				},
			},
			Host: "test-host",
		},
		Timestamp: time.Now(),
		Severity:  "high",
		Status:    "sent",
		Message:   "Test alert message",
		Count:     1,
	}

	// Test save alert
	err := redisStore.SaveAlert(alert)
	require.NoError(t, err)

	// Test get alert
	retrievedAlert, err := redisStore.GetAlert(alert.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedAlert)

	assert.Equal(t, alert.ID, retrievedAlert.ID)
	assert.Equal(t, alert.RuleID, retrievedAlert.RuleID)
	assert.Equal(t, alert.Severity, retrievedAlert.Severity)
	assert.Equal(t, alert.Status, retrievedAlert.Status)
	assert.Equal(t, alert.Message, retrievedAlert.Message)
}

func TestRedisStore_Integration_GetRecentAlerts(t *testing.T) {
	setupTest(t)

	// Create multiple alerts
	alerts := []models.Alert{
		{
			ID:        "alert-1",
			RuleID:    "rule-1",
			RuleName:  "Rule 1",
			Timestamp: time.Now(),
			Severity:  "high",
			Status:    "sent",
			Message:   "Alert 1",
		},
		{
			ID:        "alert-2",
			RuleID:    "rule-2",
			RuleName:  "Rule 2",
			Timestamp: time.Now(),
			Severity:  "medium",
			Status:    "pending",
			Message:   "Alert 2",
		},
		{
			ID:        "alert-3",
			RuleID:    "rule-3",
			RuleName:  "Rule 3",
			Timestamp: time.Now(),
			Severity:  "low",
			Status:    "sent",
			Message:   "Alert 3",
		},
	}

	// Save all alerts
	for _, alert := range alerts {
		err := redisStore.SaveAlert(alert)
		require.NoError(t, err)
	}

	// Get recent alerts
	recentAlerts, err := redisStore.GetRecentAlerts(10)
	require.NoError(t, err)
	require.Len(t, recentAlerts, 3)

	// Test with limit
	limitedAlerts, err := redisStore.GetRecentAlerts(2)
	require.NoError(t, err)
	require.Len(t, limitedAlerts, 2)
}

func TestRedisStore_Integration_HealthStatus(t *testing.T) {
	setupTest(t)

	// Test health status
	healthy, err := redisStore.GetHealthStatus()
	require.NoError(t, err)
	assert.True(t, healthy)
}

func TestRedisStore_Integration_GetInfo(t *testing.T) {
	setupTest(t)

	// Test get info
	info, err := redisStore.GetInfo()
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Contains(t, info, "info")
	assert.Contains(t, info, "status")
	assert.Equal(t, "connected", info["status"])
}

func TestRedisStore_Integration_GetMetrics(t *testing.T) {
	setupTest(t)

	// Add some test data
	rule := models.AlertRule{
		ID:        "metrics-test-rule",
		Name:      "Metrics Test Rule",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := redisStore.SaveAlertRule(rule)
	require.NoError(t, err)

	alert := models.Alert{
		ID:        "metrics-test-alert",
		RuleID:    "metrics-test-rule",
		Timestamp: time.Now(),
		Severity:  "high",
		Status:    "sent",
		Message:   "Metrics test alert",
	}
	err = redisStore.SaveAlert(alert)
	require.NoError(t, err)

	// Test get metrics
	metrics, err := redisStore.GetMetrics()
	require.NoError(t, err)
	require.NotNil(t, metrics)

	assert.Contains(t, metrics, "alert_rules")
	assert.Contains(t, metrics, "alerts")
	assert.Equal(t, 1, metrics["alert_rules"])
	assert.Equal(t, 1, metrics["alerts"])
}

func TestRedisStore_Integration_BulkSaveAlertRules(t *testing.T) {
	setupTest(t)

	// Create multiple rules
	rules := []models.AlertRule{
		{
			ID:        "bulk-rule-1",
			Name:      "Bulk Rule 1",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "bulk-rule-2",
			Name:      "Bulk Rule 2",
			Enabled:   false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "bulk-rule-3",
			Name:      "Bulk Rule 3",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Test bulk save
	err := redisStore.BulkSaveAlertRules(rules)
	require.NoError(t, err)

	// Verify all rules are saved
	for _, rule := range rules {
		retrievedRule, err := redisStore.GetAlertRule(rule.ID)
		require.NoError(t, err)
		assert.Equal(t, rule.ID, retrievedRule.ID)
		assert.Equal(t, rule.Name, retrievedRule.Name)
		assert.Equal(t, rule.Enabled, retrievedRule.Enabled)
	}
}

func TestRedisStore_Integration_Search(t *testing.T) {
	setupTest(t)

	// Create test data
	rule := models.AlertRule{
		ID:        "search-test-rule",
		Name:      "Search Test Rule",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := redisStore.SaveAlertRule(rule)
	require.NoError(t, err)

	alert := models.Alert{
		ID:        "search-test-alert",
		RuleID:    "search-test-rule",
		Timestamp: time.Now(),
		Severity:  "high",
		Status:    "sent",
		Message:   "Search test alert",
	}
	err = redisStore.SaveAlert(alert)
	require.NoError(t, err)

	// Test search patterns
	tests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{
			name:     "search alert rules",
			pattern:  "alert_rule:*",
			expected: 1,
		},
		{
			name:     "search alerts",
			pattern:  "alert:*",
			expected: 1,
		},
		{
			name:     "search all",
			pattern:  "*",
			expected: 2, // rule + alert
		},
		{
			name:     "search nonexistent",
			pattern:  "nonexistent:*",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := redisStore.Search(tt.pattern)
			require.NoError(t, err)
			assert.Len(t, results, tt.expected)
		})
	}
}

func TestRedisStore_Integration_ConcurrentOperations(t *testing.T) {
	setupTest(t)

	const numGoroutines = 10
	const operationsPerGoroutine = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Test concurrent rule operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				rule := models.AlertRule{
					ID:        fmt.Sprintf("concurrent-rule-%d-%d", goroutineID, j),
					Name:      fmt.Sprintf("Concurrent Rule %d-%d", goroutineID, j),
					Enabled:   j%2 == 0,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				// Save rule
				if err := redisStore.SaveAlertRule(rule); err != nil {
					errors <- err
					return
				}

				// Get rule
				if _, err := redisStore.GetAlertRule(rule.ID); err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify all rules were created
	allRules, err := redisStore.GetAlertRules()
	require.NoError(t, err)
	assert.Len(t, allRules, numGoroutines*operationsPerGoroutine)
}

func TestRedisStore_Integration_CounterConcurrency(t *testing.T) {
	setupTest(t)

	const numGoroutines = 5
	const incrementsPerGoroutine = 10
	const ruleID = "concurrent-counter-test"
	const window = 5 * time.Minute

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*incrementsPerGoroutine)

	// Test concurrent counter increments
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < incrementsPerGoroutine; j++ {
				if _, err := redisStore.IncrementCounter(ruleID, window); err != nil {
					errors <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent counter operation failed: %v", err)
	}

	// Verify final counter value
	finalCount, err := redisStore.GetCounter(ruleID, window)
	require.NoError(t, err)
	assert.Equal(t, int64(numGoroutines*incrementsPerGoroutine), finalCount)
}

func TestRedisStore_Integration_CleanupExpiredData(t *testing.T) {
	setupTest(t)

	// This test would need to be more complex to properly test expiration
	// For now, just verify the method doesn't error
	err := redisStore.CleanupExpiredData()
	assert.NoError(t, err)
}

func TestRedisStore_Integration_TransactionOperations(t *testing.T) {
	setupTest(t)

	// Test transaction
	err := redisStore.Transaction(func(pipe redis.Pipeliner) error {
		// Create test data within transaction
		rule := models.AlertRule{
			ID:        "transaction-test-rule",
			Name:      "Transaction Test Rule",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		ruleData, err := json.Marshal(rule)
		if err != nil {
			return err
		}

		key := fmt.Sprintf("alert_rule:%s", rule.ID)
		pipe.Set(context.Background(), key, ruleData, 0)

		return nil
	})

	require.NoError(t, err)

	// Verify data was saved
	rule, err := redisStore.GetAlertRule("transaction-test-rule")
	require.NoError(t, err)
	assert.Equal(t, "transaction-test-rule", rule.ID)
	assert.Equal(t, "Transaction Test Rule", rule.Name)
}

func TestRedisStore_Integration_ErrorHandling(t *testing.T) {
	setupTest(t)

	// Test getting non-existent rule
	_, err := redisStore.GetAlertRule("nonexistent-rule")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test getting non-existent alert
	_, err = redisStore.GetAlert("nonexistent-alert")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test getting non-existent alert status
	_, err = redisStore.GetAlertStatus("nonexistent-rule")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test deleting non-existent rule
	err = redisStore.DeleteAlertRule("nonexistent-rule")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRedisStore_Integration_Performance(t *testing.T) {
	setupTest(t)

	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	const numOperations = 1000

	// Test bulk save performance
	rules := make([]models.AlertRule, numOperations)
	for i := 0; i < numOperations; i++ {
		rules[i] = models.AlertRule{
			ID:        fmt.Sprintf("perf-rule-%d", i),
			Name:      fmt.Sprintf("Performance Rule %d", i),
			Enabled:   i%2 == 0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	start := time.Now()
	err := redisStore.BulkSaveAlertRules(rules)
	duration := time.Since(start)

	require.NoError(t, err)
	t.Logf("Bulk saved %d rules in %v (%.2f ops/sec)", numOperations, duration, float64(numOperations)/duration.Seconds())

	// Test individual get performance
	start = time.Now()
	for i := 0; i < numOperations; i++ {
		_, err := redisStore.GetAlertRule(fmt.Sprintf("perf-rule-%d", i))
		require.NoError(t, err)
	}
	duration = time.Since(start)
	t.Logf("Retrieved %d rules in %v (%.2f ops/sec)", numOperations, duration, float64(numOperations)/duration.Seconds())

	// Test get all rules performance
	start = time.Now()
	allRules, err := redisStore.GetAlertRules()
	duration = time.Since(start)

	require.NoError(t, err)
	require.Len(t, allRules, numOperations)
	t.Logf("Retrieved all %d rules in %v", numOperations, duration)
}
