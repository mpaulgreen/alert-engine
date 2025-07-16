package mocks

import (
	"github.com/log-monitoring/alert-engine/pkg/models"
)

// MockAlertEngine is a mock implementation of AlertEngine interface
type MockAlertEngine struct {
	EvaluateLogCalls []models.LogEntry
	SimulatePanic    bool
}

// NewMockAlertEngine creates a new mock alert engine
func NewMockAlertEngine() *MockAlertEngine {
	return &MockAlertEngine{
		EvaluateLogCalls: make([]models.LogEntry, 0),
		SimulatePanic:    false,
	}
}

// EvaluateLog mocks the EvaluateLog method
func (m *MockAlertEngine) EvaluateLog(logEntry models.LogEntry) {
	if m.SimulatePanic {
		panic("simulated alert engine panic")
	}
	m.EvaluateLogCalls = append(m.EvaluateLogCalls, logEntry)
}

// Reset clears the mock state
func (m *MockAlertEngine) Reset() {
	m.EvaluateLogCalls = make([]models.LogEntry, 0)
	m.SimulatePanic = false
}

// GetCallCount returns the number of EvaluateLog calls
func (m *MockAlertEngine) GetCallCount() int {
	return len(m.EvaluateLogCalls)
}

// GetLastCall returns the last log entry that was evaluated
func (m *MockAlertEngine) GetLastCall() *models.LogEntry {
	if len(m.EvaluateLogCalls) == 0 {
		return nil
	}
	return &m.EvaluateLogCalls[len(m.EvaluateLogCalls)-1]
}
