package mocks

import (
	"sync"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// MockAlertEngine is a mock implementation of the AlertEngine interface
type MockAlertEngine struct {
	mu             sync.RWMutex
	evaluatedLogs  []models.LogEntry
	callCount      int
	shouldPanic    bool
	processingTime time.Duration
}

// NewMockAlertEngine creates a new mock alert engine
func NewMockAlertEngine() *MockAlertEngine {
	return &MockAlertEngine{
		evaluatedLogs: make([]models.LogEntry, 0),
		callCount:     0,
		shouldPanic:   false,
	}
}

// EvaluateLog processes a log entry (mock implementation)
func (m *MockAlertEngine) EvaluateLog(logEntry models.LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldPanic {
		panic("mock alert engine panic")
	}

	// Simulate processing time if set
	if m.processingTime > 0 {
		time.Sleep(m.processingTime)
	}

	m.callCount++
	m.evaluatedLogs = append(m.evaluatedLogs, logEntry)
}

// GetEvaluatedLogs returns all logs that were evaluated
func (m *MockAlertEngine) GetEvaluatedLogs() []models.LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	logs := make([]models.LogEntry, len(m.evaluatedLogs))
	copy(logs, m.evaluatedLogs)
	return logs
}

// GetCallCount returns the number of times EvaluateLog was called
func (m *MockAlertEngine) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetLastEvaluatedLog returns the last log that was evaluated
func (m *MockAlertEngine) GetLastEvaluatedLog() *models.LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.evaluatedLogs) == 0 {
		return nil
	}

	log := m.evaluatedLogs[len(m.evaluatedLogs)-1]
	return &log
}

// Reset clears all evaluated logs and resets the mock
func (m *MockAlertEngine) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.evaluatedLogs = make([]models.LogEntry, 0)
	m.callCount = 0
	m.shouldPanic = false
	m.processingTime = 0
}

// SetShouldPanic configures the mock to panic on EvaluateLog
func (m *MockAlertEngine) SetShouldPanic(shouldPanic bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldPanic = shouldPanic
}

// SetProcessingTime sets a delay for processing (for testing timing)
func (m *MockAlertEngine) SetProcessingTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processingTime = duration
}

// WasCalled returns true if EvaluateLog was called at least once
func (m *MockAlertEngine) WasCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount > 0
}

// FindLogByMessage finds a log entry by message content
func (m *MockAlertEngine) FindLogByMessage(message string) *models.LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, log := range m.evaluatedLogs {
		if log.Message == message {
			return &log
		}
	}
	return nil
}

// CountLogsByLevel counts logs by level
func (m *MockAlertEngine) CountLogsByLevel(level string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, log := range m.evaluatedLogs {
		if log.Level == level {
			count++
		}
	}
	return count
}

// CountLogsByNamespace counts logs by namespace
func (m *MockAlertEngine) CountLogsByNamespace(namespace string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, log := range m.evaluatedLogs {
		if log.GetNamespace() == namespace {
			count++
		}
	}
	return count
}

// GetEvaluatedLogsInTimeRange returns logs evaluated within a time range
func (m *MockAlertEngine) GetEvaluatedLogsInTimeRange(start, end time.Time) []models.LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var logsInRange []models.LogEntry
	for _, log := range m.evaluatedLogs {
		if (log.Timestamp.After(start) || log.Timestamp.Equal(start)) &&
			(log.Timestamp.Before(end) || log.Timestamp.Equal(end)) {
			logsInRange = append(logsInRange, log)
		}
	}
	return logsInRange
}
