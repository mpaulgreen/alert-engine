//go:build unit || integration
// +build unit integration

package mocks

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// MockStateStore is a mock implementation of the StateStore interface
type MockStateStore struct {
	mu               sync.RWMutex
	rules            map[string]models.AlertRule
	counters         map[string]int64
	alertStatuses    map[string]models.AlertStatus
	recentAlerts     []models.Alert
	healthy          bool
	shouldFailHealth bool
	shouldFailOps    bool
	metrics          map[string]interface{}
	logStats         *models.LogStats
}

// NewMockStateStore creates a new mock state store
func NewMockStateStore() *MockStateStore {
	return &MockStateStore{
		rules:         make(map[string]models.AlertRule),
		counters:      make(map[string]int64),
		alertStatuses: make(map[string]models.AlertStatus),
		recentAlerts:  make([]models.Alert, 0),
		healthy:       true,
		metrics: map[string]interface{}{
			"system_metrics": map[string]interface{}{
				"cpu_usage":    45.5,
				"memory_usage": 67.2,
				"disk_usage":   23.8,
			},
			"alert_metrics": map[string]interface{}{
				"rules_evaluated":  1000,
				"alerts_triggered": 25,
				"alerts_sent":      23,
			},
		},
		logStats: &models.LogStats{
			TotalLogs: 1000000,
			LogsByLevel: map[string]int64{
				"ERROR": 5000,
				"WARN":  15000,
				"INFO":  800000,
				"DEBUG": 180000,
			},
			LogsByService: map[string]int64{
				"api-service":    500000,
				"worker-service": 300000,
				"cache-service":  200000,
			},
		},
	}
}

// SaveAlertRule saves an alert rule
func (m *MockStateStore) SaveAlertRule(rule models.AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOps {
		return errors.New("mock operation failed")
	}

	if rule.ID == "" {
		return errors.New("rule ID is required")
	}

	m.rules[rule.ID] = rule
	return nil
}

// GetAlertRules returns all alert rules
func (m *MockStateStore) GetAlertRules() ([]models.AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailOps {
		return nil, errors.New("mock operation failed")
	}

	rules := make([]models.AlertRule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}
	return rules, nil
}

// GetAlertRule returns a specific alert rule
func (m *MockStateStore) GetAlertRule(id string) (*models.AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailOps {
		return nil, errors.New("mock operation failed")
	}

	rule, exists := m.rules[id]
	if !exists {
		return nil, errors.New("rule not found")
	}
	return &rule, nil
}

// DeleteAlertRule deletes an alert rule
func (m *MockStateStore) DeleteAlertRule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOps {
		return errors.New("mock operation failed")
	}

	if _, exists := m.rules[id]; !exists {
		return errors.New("rule not found")
	}

	delete(m.rules, id)
	return nil
}

// IncrementCounter increments a counter for a rule
func (m *MockStateStore) IncrementCounter(ruleID string, window time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOps {
		return 0, errors.New("mock operation failed")
	}

	key := fmt.Sprintf("%s-%s", ruleID, window.String())
	m.counters[key]++
	return m.counters[key], nil
}

// GetCounter returns the current counter value for a rule
func (m *MockStateStore) GetCounter(ruleID string, window time.Duration) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailOps {
		return 0, errors.New("mock operation failed")
	}

	key := fmt.Sprintf("%s-%s", ruleID, window.String())
	return m.counters[key], nil
}

// SetAlertStatus sets the alert status for a rule
func (m *MockStateStore) SetAlertStatus(ruleID string, status models.AlertStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailOps {
		return errors.New("mock operation failed")
	}

	m.alertStatuses[ruleID] = status
	return nil
}

// GetAlertStatus returns the alert status for a rule
func (m *MockStateStore) GetAlertStatus(ruleID string) (*models.AlertStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailOps {
		return nil, errors.New("mock operation failed")
	}

	status, exists := m.alertStatuses[ruleID]
	if !exists {
		return nil, errors.New("alert status not found")
	}
	return &status, nil
}

// GetRecentAlerts returns recent alerts
func (m *MockStateStore) GetRecentAlerts(limit int) ([]models.Alert, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailOps {
		return nil, errors.New("mock operation failed")
	}

	if limit > len(m.recentAlerts) {
		return m.recentAlerts, nil
	}
	return m.recentAlerts[:limit], nil
}

// GetLogStats returns log processing statistics
func (m *MockStateStore) GetLogStats() (*models.LogStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailOps {
		return nil, errors.New("mock operation failed")
	}

	return m.logStats, nil
}

// GetHealthStatus returns the health status
func (m *MockStateStore) GetHealthStatus() (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailHealth {
		return false, errors.New("health check failed")
	}

	return m.healthy, nil
}

// GetMetrics returns system metrics
func (m *MockStateStore) GetMetrics() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailOps {
		return nil, errors.New("mock operation failed")
	}

	return m.metrics, nil
}

// Mock control methods

// SetHealthy sets the health status
func (m *MockStateStore) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

// SetShouldFailHealth sets whether health checks should fail
func (m *MockStateStore) SetShouldFailHealth(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailHealth = shouldFail
}

// SetShouldFailOps sets whether operations should fail
func (m *MockStateStore) SetShouldFailOps(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailOps = shouldFail
}

// AddRule adds a rule to the mock store
func (m *MockStateStore) AddRule(rule models.AlertRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules[rule.ID] = rule
}

// AddAlert adds an alert to the recent alerts list
func (m *MockStateStore) AddAlert(alert models.Alert) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recentAlerts = append(m.recentAlerts, alert)
}

// ClearRules clears all rules
func (m *MockStateStore) ClearRules() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = make(map[string]models.AlertRule)
}

// ClearAlerts clears all alerts
func (m *MockStateStore) ClearAlerts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recentAlerts = make([]models.Alert, 0)
}

// GetRuleCount returns the number of rules
func (m *MockStateStore) GetRuleCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rules)
}

// GetAlertCount returns the number of alerts
func (m *MockStateStore) GetAlertCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.recentAlerts)
}

// SetLogStats sets the log statistics
func (m *MockStateStore) SetLogStats(stats *models.LogStats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logStats = stats
}

// SetMetrics sets the system metrics
func (m *MockStateStore) SetMetrics(metrics map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = metrics
}
