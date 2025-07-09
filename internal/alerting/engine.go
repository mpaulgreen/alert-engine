package alerting

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// StateStore interface for persisting alert state
type StateStore interface {
	SaveAlertRule(rule models.AlertRule) error
	GetAlertRules() ([]models.AlertRule, error)
	GetAlertRule(id string) (*models.AlertRule, error)
	DeleteAlertRule(id string) error
	IncrementCounter(ruleID string, window time.Duration) (int64, error)
	GetCounter(ruleID string, window time.Duration) (int64, error)
	SetAlertStatus(ruleID string, status models.AlertStatus) error
	GetAlertStatus(ruleID string) (*models.AlertStatus, error)
}

// Notifier interface for sending notifications
type Notifier interface {
	SendAlert(alert models.Alert) error
	TestConnection() error
}

// Engine is the main alert evaluation engine
type Engine struct {
	stateStore  StateStore
	notifier    Notifier
	rules       []models.AlertRule
	windowStore map[string]*TimeWindow
	stopChan    chan struct{}
}

// TimeWindow represents a sliding time window for counting events
type TimeWindow struct {
	RuleID    string
	StartTime time.Time
	EndTime   time.Time
	Count     int64
	Events    []time.Time
}

// NewEngine creates a new alert engine
func NewEngine(stateStore StateStore, notifier Notifier) *Engine {
	engine := &Engine{
		stateStore:  stateStore,
		notifier:    notifier,
		windowStore: make(map[string]*TimeWindow),
		stopChan:    make(chan struct{}),
	}

	// Load existing rules
	if err := engine.loadRules(); err != nil {
		log.Printf("Warning: Failed to load existing rules: %v", err)
	}

	// Start cleanup goroutine
	go engine.cleanupRoutine()

	return engine
}

// EvaluateLog evaluates a log entry against all active alert rules
func (e *Engine) EvaluateLog(logEntry models.LogEntry) {
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		if e.matchesConditions(logEntry, rule.Conditions) {
			count, err := e.updateCounter(rule.ID, logEntry.Timestamp, rule.Conditions.TimeWindow)
			if err != nil {
				log.Printf("Error updating counter for rule %s: %v", rule.ID, err)
				continue
			}

			if e.shouldTriggerAlert(rule, count) {
				alert := models.Alert{
					ID:        fmt.Sprintf("%s-%d", rule.ID, time.Now().Unix()),
					RuleID:    rule.ID,
					RuleName:  rule.Name,
					LogEntry:  logEntry,
					Timestamp: logEntry.Timestamp,
					Severity:  rule.Actions.Severity,
					Status:    "pending",
					Count:     int(count),
					Message:   e.buildAlertMessage(rule, logEntry, int(count)),
				}

				if err := e.notifier.SendAlert(alert); err != nil {
					log.Printf("Error sending alert for rule %s: %v", rule.ID, err)
					alert.Status = "failed"
				} else {
					alert.Status = "sent"
				}

				e.recordAlertSent(rule.ID, logEntry.Timestamp)
			}
		}
	}
}

// matchesConditions checks if a log entry matches the rule conditions
func (e *Engine) matchesConditions(logEntry models.LogEntry, conditions models.AlertConditions) bool {
	// Check log level
	if conditions.LogLevel != "" && logEntry.Level != conditions.LogLevel {
		return false
	}

	// Check namespace
	if conditions.Namespace != "" && logEntry.Kubernetes.Namespace != conditions.Namespace {
		return false
	}

	// Check service (from Kubernetes labels)
	if conditions.Service != "" {
		if appLabel, exists := logEntry.Kubernetes.Labels["app"]; !exists || appLabel != conditions.Service {
			return false
		}
	}

	// Check keywords
	if len(conditions.Keywords) > 0 {
		messageUpper := strings.ToUpper(logEntry.Message)
		for _, keyword := range conditions.Keywords {
			if !strings.Contains(messageUpper, strings.ToUpper(keyword)) {
				return false
			}
		}
	}

	return true
}

// updateCounter updates the event counter for a rule within its time window
func (e *Engine) updateCounter(ruleID string, timestamp time.Time, window time.Duration) (int64, error) {
	// Use Redis for distributed counting
	return e.stateStore.IncrementCounter(ruleID, window)
}

// shouldTriggerAlert determines if an alert should be triggered based on threshold
func (e *Engine) shouldTriggerAlert(rule models.AlertRule, count int64) bool {
	switch rule.Conditions.Operator {
	case "gt", "":
		return count > int64(rule.Conditions.Threshold)
	case "gte":
		return count >= int64(rule.Conditions.Threshold)
	case "lt":
		return count < int64(rule.Conditions.Threshold)
	case "lte":
		return count <= int64(rule.Conditions.Threshold)
	case "eq":
		return count == int64(rule.Conditions.Threshold)
	default:
		return count > int64(rule.Conditions.Threshold)
	}
}

// buildAlertMessage creates a formatted alert message
func (e *Engine) buildAlertMessage(rule models.AlertRule, logEntry models.LogEntry, count int) string {
	return fmt.Sprintf(
		"🚨 Alert: %s\n"+
			"Service: %s\n"+
			"Namespace: %s\n"+
			"Level: %s\n"+
			"Count: %d in %s\n"+
			"Message: %s",
		rule.Name,
		logEntry.Kubernetes.Labels["app"],
		logEntry.Kubernetes.Namespace,
		logEntry.Level,
		count,
		rule.Conditions.TimeWindow.String(),
		logEntry.Message,
	)
}

// recordAlertSent records that an alert was sent for a rule
func (e *Engine) recordAlertSent(ruleID string, timestamp time.Time) {
	status := models.AlertStatus{
		RuleID:      ruleID,
		LastTrigger: timestamp,
		Status:      "sent",
	}

	if err := e.stateStore.SetAlertStatus(ruleID, status); err != nil {
		log.Printf("Error recording alert status for rule %s: %v", ruleID, err)
	}
}

// loadRules loads alert rules from the state store
func (e *Engine) loadRules() error {
	rules, err := e.stateStore.GetAlertRules()
	if err != nil {
		return err
	}

	e.rules = rules
	log.Printf("Loaded %d alert rules", len(rules))
	return nil
}

// ReloadRules reloads alert rules from the state store
func (e *Engine) ReloadRules() error {
	return e.loadRules()
}

// AddRule adds a new alert rule
func (e *Engine) AddRule(rule models.AlertRule) error {
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	if err := e.stateStore.SaveAlertRule(rule); err != nil {
		return err
	}

	return e.loadRules()
}

// UpdateRule updates an existing alert rule
func (e *Engine) UpdateRule(rule models.AlertRule) error {
	rule.UpdatedAt = time.Now()

	if err := e.stateStore.SaveAlertRule(rule); err != nil {
		return err
	}

	return e.loadRules()
}

// DeleteRule deletes an alert rule
func (e *Engine) DeleteRule(ruleID string) error {
	if err := e.stateStore.DeleteAlertRule(ruleID); err != nil {
		return err
	}

	return e.loadRules()
}

// GetRules returns all alert rules
func (e *Engine) GetRules() []models.AlertRule {
	return e.rules
}

// GetRule returns a specific alert rule
func (e *Engine) GetRule(ruleID string) (*models.AlertRule, error) {
	return e.stateStore.GetAlertRule(ruleID)
}

// cleanupRoutine periodically cleans up old time windows
func (e *Engine) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.cleanupOldWindows()
		case <-e.stopChan:
			return
		}
	}
}

// cleanupOldWindows removes old time windows from memory
func (e *Engine) cleanupOldWindows() {
	now := time.Now()
	for ruleID, window := range e.windowStore {
		if now.Sub(window.EndTime) > time.Hour {
			delete(e.windowStore, ruleID)
		}
	}
}

// Stop stops the alert engine
func (e *Engine) Stop() {
	close(e.stopChan)
}
