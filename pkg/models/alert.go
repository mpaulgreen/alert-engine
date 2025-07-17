package models

import (
	"time"
)

// AlertRule represents a rule for triggering alerts based on log conditions
type AlertRule struct {
	ID          string          `json:"id" redis:"id"`
	Name        string          `json:"name" redis:"name"`
	Description string          `json:"description" redis:"description"`
	Enabled     bool            `json:"enabled" redis:"enabled"`
	Conditions  AlertConditions `json:"conditions" redis:"conditions"`
	Actions     AlertActions    `json:"actions" redis:"actions"`
	CreatedAt   time.Time       `json:"created_at" redis:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" redis:"updated_at"`
}

// AlertConditions defines the conditions that must be met to trigger an alert
type AlertConditions struct {
	LogLevel   string        `json:"log_level"`
	Namespace  string        `json:"namespace"`
	Service    string        `json:"service"`
	Keywords   []string      `json:"keywords"`
	Threshold  int           `json:"threshold"`
	TimeWindow time.Duration `json:"time_window"`
	Operator   string        `json:"operator"` // "gt", "lt", "eq", "contains"
}

// AlertActions defines the actions to take when an alert is triggered
type AlertActions struct {
	SlackWebhook string `json:"slack_webhook"`
	Channel      string `json:"channel"`
	Severity     string `json:"severity"` // "low", "medium", "high", "critical"
}

// Alert represents a triggered alert instance
type Alert struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	LogEntry  LogEntry  `json:"log_entry"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"` // "pending", "sent", "failed"
	Message   string    `json:"message"`
	Count     int       `json:"count"`
}

// AlertStatus represents the status of an alert
type AlertStatus struct {
	RuleID      string    `json:"rule_id"`
	LastTrigger time.Time `json:"last_trigger"`
	Count       int       `json:"count"`
	Status      string    `json:"status"`
}
