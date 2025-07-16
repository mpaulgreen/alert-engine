package mocks

import (
	"fmt"
	"sync"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// MockNotifier is a mock implementation of the Notifier interface
type MockNotifier struct {
	mu         sync.RWMutex
	sentAlerts []models.Alert
	shouldFail bool
	failureMsg string
	callCount  int
}

// NewMockNotifier creates a new mock notifier
func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		sentAlerts: make([]models.Alert, 0),
		shouldFail: false,
	}
}

// SetShouldFail configures the mock to fail operations
func (m *MockNotifier) SetShouldFail(shouldFail bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
	m.failureMsg = message
}

// SendAlert sends an alert (mock implementation)
func (m *MockNotifier) SendAlert(alert models.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.shouldFail {
		return fmt.Errorf("mock error: %s", m.failureMsg)
	}

	m.sentAlerts = append(m.sentAlerts, alert)
	return nil
}

// TestConnection tests the connection (mock implementation)
func (m *MockNotifier) TestConnection() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail {
		return fmt.Errorf("mock error: %s", m.failureMsg)
	}

	return nil
}

// GetSentAlerts returns all alerts that were sent
func (m *MockNotifier) GetSentAlerts() []models.Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alerts := make([]models.Alert, len(m.sentAlerts))
	copy(alerts, m.sentAlerts)
	return alerts
}

// GetSentAlertsCount returns the number of alerts that were sent
func (m *MockNotifier) GetSentAlertsCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sentAlerts)
}

// GetCallCount returns the number of times SendAlert was called
func (m *MockNotifier) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetLastSentAlert returns the last alert that was sent
func (m *MockNotifier) GetLastSentAlert() *models.Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.sentAlerts) == 0 {
		return nil
	}

	alert := m.sentAlerts[len(m.sentAlerts)-1]
	return &alert
}

// Reset clears all sent alerts and resets the mock
func (m *MockNotifier) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sentAlerts = make([]models.Alert, 0)
	m.shouldFail = false
	m.failureMsg = ""
	m.callCount = 0
}

// WasCalled returns true if SendAlert was called at least once
func (m *MockNotifier) WasCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount > 0
}

// FindAlertByRuleID finds an alert by rule ID
func (m *MockNotifier) FindAlertByRuleID(ruleID string) *models.Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, alert := range m.sentAlerts {
		if alert.RuleID == ruleID {
			return &alert
		}
	}
	return nil
}

// CountAlertsByRuleID counts alerts sent for a specific rule
func (m *MockNotifier) CountAlertsByRuleID(ruleID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, alert := range m.sentAlerts {
		if alert.RuleID == ruleID {
			count++
		}
	}
	return count
}

// CountAlertsBySeverity counts alerts by severity level
func (m *MockNotifier) CountAlertsBySeverity(severity string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, alert := range m.sentAlerts {
		if alert.Severity == severity {
			count++
		}
	}
	return count
}
