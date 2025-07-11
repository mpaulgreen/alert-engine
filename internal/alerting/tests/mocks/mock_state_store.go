package mocks

import (
	"fmt"
	"sync"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// MockStateStore is a mock implementation of the StateStore interface
type MockStateStore struct {
	mu          sync.RWMutex
	rules       map[string]models.AlertRule
	counters    map[string]int64
	alertStatus map[string]models.AlertStatus
	shouldFail  bool
	failureMsg  string
}

// NewMockStateStore creates a new mock state store
func NewMockStateStore() *MockStateStore {
	return &MockStateStore{
		rules:       make(map[string]models.AlertRule),
		counters:    make(map[string]int64),
		alertStatus: make(map[string]models.AlertStatus),
		shouldFail:  false,
	}
}

// SetShouldFail configures the mock to fail operations
func (m *MockStateStore) SetShouldFail(shouldFail bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
	m.failureMsg = message
}

// SaveAlertRule saves an alert rule
func (m *MockStateStore) SaveAlertRule(rule models.AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return fmt.Errorf("mock error: %s", m.failureMsg)
	}

	m.rules[rule.ID] = rule
	return nil
}

// GetAlertRules returns all alert rules
func (m *MockStateStore) GetAlertRules() ([]models.AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail {
		return nil, fmt.Errorf("mock error: %s", m.failureMsg)
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

	if m.shouldFail {
		return nil, fmt.Errorf("mock error: %s", m.failureMsg)
	}

	rule, exists := m.rules[id]
	if !exists {
		return nil, fmt.Errorf("rule not found: %s", id)
	}
	return &rule, nil
}

// DeleteAlertRule deletes an alert rule
func (m *MockStateStore) DeleteAlertRule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return fmt.Errorf("mock error: %s", m.failureMsg)
	}

	if _, exists := m.rules[id]; !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	delete(m.rules, id)
	return nil
}

// IncrementCounter increments and returns the counter for a rule
func (m *MockStateStore) IncrementCounter(ruleID string, window time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return 0, fmt.Errorf("mock error: %s", m.failureMsg)
	}

	key := fmt.Sprintf("%s:%s", ruleID, window.String())
	m.counters[key]++
	return m.counters[key], nil
}

// GetCounter returns the current counter value for a rule
func (m *MockStateStore) GetCounter(ruleID string, window time.Duration) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail {
		return 0, fmt.Errorf("mock error: %s", m.failureMsg)
	}

	key := fmt.Sprintf("%s:%s", ruleID, window.String())
	return m.counters[key], nil
}

// SetAlertStatus sets the alert status for a rule
func (m *MockStateStore) SetAlertStatus(ruleID string, status models.AlertStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return fmt.Errorf("mock error: %s", m.failureMsg)
	}

	m.alertStatus[ruleID] = status
	return nil
}

// GetAlertStatus returns the alert status for a rule
func (m *MockStateStore) GetAlertStatus(ruleID string) (*models.AlertStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail {
		return nil, fmt.Errorf("mock error: %s", m.failureMsg)
	}

	status, exists := m.alertStatus[ruleID]
	if !exists {
		return nil, fmt.Errorf("alert status not found: %s", ruleID)
	}
	return &status, nil
}

// Reset clears all stored data and resets the mock
func (m *MockStateStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rules = make(map[string]models.AlertRule)
	m.counters = make(map[string]int64)
	m.alertStatus = make(map[string]models.AlertStatus)
	m.shouldFail = false
	m.failureMsg = ""
}

// SetCounter manually sets a counter value (for testing)
func (m *MockStateStore) SetCounter(ruleID string, window time.Duration, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", ruleID, window.String())
	m.counters[key] = value
}

// GetRulesCount returns the number of stored rules (for testing)
func (m *MockStateStore) GetRulesCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rules)
}

// GetCountersCount returns the number of stored counters (for testing)
func (m *MockStateStore) GetCountersCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.counters)
}
