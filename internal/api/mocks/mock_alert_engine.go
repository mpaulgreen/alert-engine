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

// MockAlertEngine is a mock implementation of the AlertEngine interface
type MockAlertEngine struct {
	mu           sync.RWMutex
	rules        map[string]models.AlertRule
	shouldFail   bool
	shouldFailOp map[string]bool
	ruleStats    map[string]interface{}
	reloadError  error
}

// NewMockAlertEngine creates a new mock alert engine
func NewMockAlertEngine() *MockAlertEngine {
	return &MockAlertEngine{
		rules:        make(map[string]models.AlertRule),
		shouldFailOp: make(map[string]bool),
		ruleStats: map[string]interface{}{
			"total_rules":    10,
			"enabled_rules":  8,
			"disabled_rules": 2,
			"rules_by_severity": map[string]int{
				"critical": 2,
				"high":     3,
				"medium":   4,
				"low":      1,
			},
			"rules_by_namespace": map[string]int{
				"production":  6,
				"staging":     3,
				"development": 1,
			},
		},
	}
}

// AddRule adds a new alert rule
func (m *MockAlertEngine) AddRule(rule models.AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail || m.shouldFailOp["add"] {
		return errors.New("mock add operation failed")
	}

	if rule.ID == "" {
		return errors.New("rule ID is required")
	}

	if rule.Name == "" {
		return errors.New("rule name is required")
	}

	if _, exists := m.rules[rule.ID]; exists {
		return errors.New("rule already exists")
	}

	// Set timestamps
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	m.rules[rule.ID] = rule
	return nil
}

// UpdateRule updates an existing alert rule
func (m *MockAlertEngine) UpdateRule(rule models.AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail || m.shouldFailOp["update"] {
		return errors.New("mock update operation failed")
	}

	if rule.ID == "" {
		return errors.New("rule ID is required")
	}

	if rule.Name == "" {
		return errors.New("rule name is required")
	}

	existingRule, exists := m.rules[rule.ID]
	if !exists {
		return errors.New("rule not found")
	}

	// Preserve created timestamp
	rule.CreatedAt = existingRule.CreatedAt
	rule.UpdatedAt = time.Now()

	m.rules[rule.ID] = rule
	return nil
}

// DeleteRule deletes an alert rule
func (m *MockAlertEngine) DeleteRule(ruleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail || m.shouldFailOp["delete"] {
		return errors.New("mock delete operation failed")
	}

	if _, exists := m.rules[ruleID]; !exists {
		return errors.New("rule not found")
	}

	delete(m.rules, ruleID)
	return nil
}

// GetRules returns all alert rules
func (m *MockAlertEngine) GetRules() []models.AlertRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]models.AlertRule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}
	return rules
}

// GetRule returns a specific alert rule
func (m *MockAlertEngine) GetRule(ruleID string) (*models.AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail || m.shouldFailOp["get"] {
		return nil, errors.New("mock get operation failed")
	}

	rule, exists := m.rules[ruleID]
	if !exists {
		return nil, errors.New("rule not found")
	}
	return &rule, nil
}

// ReloadRules reloads all alert rules
func (m *MockAlertEngine) ReloadRules() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.reloadError != nil {
		return m.reloadError
	}

	if m.shouldFail || m.shouldFailOp["reload"] {
		return errors.New("mock reload operation failed")
	}

	// Simulate reloading rules
	return nil
}

// Mock control methods

// SetShouldFail sets whether all operations should fail
func (m *MockAlertEngine) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

// SetShouldFailOp sets whether specific operations should fail
func (m *MockAlertEngine) SetShouldFailOp(operation string, shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailOp[operation] = shouldFail
}

// AddRule adds a rule to the mock engine (direct access for testing)
func (m *MockAlertEngine) AddRuleDirect(rule models.AlertRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules[rule.ID] = rule
}

// RemoveRule removes a rule from the mock engine (direct access for testing)
func (m *MockAlertEngine) RemoveRule(ruleID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rules, ruleID)
}

// ClearRules clears all rules from the mock engine
func (m *MockAlertEngine) ClearRules() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = make(map[string]models.AlertRule)
}

// GetRuleCount returns the number of rules
func (m *MockAlertEngine) GetRuleCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rules)
}

// SetRuleStats sets the rule statistics
func (m *MockAlertEngine) SetRuleStats(stats map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ruleStats = stats
}

// SetReloadError sets the error to return when ReloadRules is called
func (m *MockAlertEngine) SetReloadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadError = err
}

// SetRules sets the rules in the mock engine
func (m *MockAlertEngine) SetRules(rules []models.AlertRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = make(map[string]models.AlertRule)
	for _, rule := range rules {
		m.rules[rule.ID] = rule
	}
}

// GetRuleStats returns rule statistics
func (m *MockAlertEngine) GetRuleStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ruleStats
}

// Helper methods for creating test data

// CreateSampleRule creates a sample alert rule for testing
func (m *MockAlertEngine) CreateSampleRule(id, name string) models.AlertRule {
	return models.AlertRule{
		ID:          id,
		Name:        name,
		Description: fmt.Sprintf("Test rule: %s", name),
		Enabled:     true,
		Conditions: models.AlertConditions{
			LogLevel:   "ERROR",
			Namespace:  "test",
			Service:    "test-service",
			Keywords:   []string{"error", "test"},
			Threshold:  5,
			TimeWindow: 5 * time.Minute,
			Operator:   "gt",
		},
		Actions: models.AlertActions{
			SlackWebhook: "https://hooks.slack.com/services/test",
			Channel:      "#alerts",
			Severity:     "medium",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateSampleRules creates multiple sample alert rules for testing
func (m *MockAlertEngine) CreateSampleRules(count int) []models.AlertRule {
	rules := make([]models.AlertRule, count)
	for i := 0; i < count; i++ {
		rules[i] = m.CreateSampleRule(
			fmt.Sprintf("rule-%d", i+1),
			fmt.Sprintf("Test Rule %d", i+1),
		)
	}
	return rules
}

// PopulateWithSampleRules populates the mock engine with sample rules
func (m *MockAlertEngine) PopulateWithSampleRules(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := 0; i < count; i++ {
		rule := m.CreateSampleRule(
			fmt.Sprintf("rule-%d", i+1),
			fmt.Sprintf("Test Rule %d", i+1),
		)
		m.rules[rule.ID] = rule
	}
}

// FilterRules filters rules based on criteria (for testing filter functionality)
func (m *MockAlertEngine) FilterRules(criteria map[string]interface{}) []models.AlertRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []models.AlertRule

	for _, rule := range m.rules {
		match := true

		if enabled, ok := criteria["enabled"].(bool); ok && rule.Enabled != enabled {
			match = false
		}

		if namespace, ok := criteria["namespace"].(string); ok && rule.Conditions.Namespace != namespace {
			match = false
		}

		if service, ok := criteria["service"].(string); ok && rule.Conditions.Service != service {
			match = false
		}

		if severity, ok := criteria["severity"].(string); ok && rule.Actions.Severity != severity {
			match = false
		}

		if match {
			filtered = append(filtered, rule)
		}
	}

	return filtered
}

// TestRule tests a rule against sample logs (for testing rule validation)
func (m *MockAlertEngine) TestRule(rule models.AlertRule, sampleLogs []models.LogEntry) (bool, []map[string]interface{}) {
	results := make([]map[string]interface{}, 0)
	matches := 0

	for _, log := range sampleLogs {
		matched := false
		reason := ""

		// Simple matching logic for testing
		if rule.Conditions.LogLevel == log.Level {
			matched = true
			reason = "Log level matched"
		}

		if matched {
			matches++
		}

		results = append(results, map[string]interface{}{
			"log_entry":    log,
			"matched":      matched,
			"match_reason": reason,
		})
	}

	wouldTrigger := matches >= rule.Conditions.Threshold

	return wouldTrigger, results
}
