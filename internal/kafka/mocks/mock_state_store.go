package mocks

import (
	"sync"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// MockStateStore is a mock implementation of the StateStore interface for kafka tests
type MockStateStore struct {
	mu       sync.RWMutex
	logStats *models.LogStats
}

// NewMockStateStore creates a new mock state store
func NewMockStateStore() *MockStateStore {
	return &MockStateStore{
		logStats: &models.LogStats{
			TotalLogs: 0,
			LogsByLevel: map[string]int64{
				"ERROR": 0,
				"WARN":  0,
				"INFO":  0,
				"DEBUG": 0,
			},
			LogsByService: map[string]int64{},
		},
	}
}

// SaveLogStats saves log statistics
func (m *MockStateStore) SaveLogStats(stats models.LogStats) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logStats = &stats
	return nil
}

// GetLogStats returns log statistics
func (m *MockStateStore) GetLogStats() (*models.LogStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.logStats, nil
}
