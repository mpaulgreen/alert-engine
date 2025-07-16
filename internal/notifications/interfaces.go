package notifications

import (
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// NotificationGlobalConfig holds global notification configuration including severity mappings
type NotificationGlobalConfig struct {
	SeverityEmojis map[string]string `json:"severity_emojis"`
	SeverityColors map[string]string `json:"severity_colors"`
}

// Notifier defines the interface for sending notifications
type Notifier interface {
	SendAlert(alert models.Alert) error
	TestConnection() error
	GetName() string
	IsEnabled() bool
	SetEnabled(enabled bool)
}

// NotificationChannel represents a notification channel configuration
type NotificationChannel struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"` // "slack", "email", "webhook", etc.
	Config   map[string]string `json:"config"`
	Enabled  bool              `json:"enabled"`
	Created  time.Time         `json:"created"`
	Modified time.Time         `json:"modified"`
}

// NotificationTemplate defines message template structure
type NotificationTemplate struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Subject   string            `json:"subject,omitempty"`
	Body      string            `json:"body"`
	Variables map[string]string `json:"variables"`
	Severity  string            `json:"severity"`
	Created   time.Time         `json:"created"`
	Modified  time.Time         `json:"modified"`
}

// NotificationResult represents the result of a notification attempt
type NotificationResult struct {
	ID        string        `json:"id"`
	AlertID   string        `json:"alert_id"`
	Channel   string        `json:"channel"`
	Status    string        `json:"status"` // "success", "failed", "pending"
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
}

// NotificationHistory tracks notification history
type NotificationHistory struct {
	AlertID     string               `json:"alert_id"`
	RuleID      string               `json:"rule_id"`
	Results     []NotificationResult `json:"results"`
	TotalSent   int                  `json:"total_sent"`
	TotalFailed int                  `json:"total_failed"`
	LastSent    time.Time            `json:"last_sent"`
}

// NotificationStats provides statistics about notifications
type NotificationStats struct {
	TotalNotifications  int            `json:"total_notifications"`
	SuccessfulSent      int            `json:"successful_sent"`
	FailedSent          int            `json:"failed_sent"`
	SuccessRate         float64        `json:"success_rate"`
	ByChannel           map[string]int `json:"by_channel"`
	BySeverity          map[string]int `json:"by_severity"`
	LastNotification    time.Time      `json:"last_notification"`
	AverageResponseTime time.Duration  `json:"average_response_time"`
}

// NotificationConfig holds configuration for notification settings
type NotificationConfig struct {
	Enabled             bool          `json:"enabled"`
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`
	Timeout             time.Duration `json:"timeout"`
	RateLimitPerMin     int           `json:"rate_limit_per_min"`
	BatchSize           int           `json:"batch_size"`
	BatchDelay          time.Duration `json:"batch_delay"`
	EnableDeduplication bool          `json:"enable_deduplication"`
	DeduplicationWindow time.Duration `json:"deduplication_window"`
}

// NotificationFilter defines filtering criteria for notifications
type NotificationFilter struct {
	Severity        []string   `json:"severity,omitempty"`
	Namespace       []string   `json:"namespace,omitempty"`
	Service         []string   `json:"service,omitempty"`
	TimeRange       *TimeRange `json:"time_range,omitempty"`
	Keywords        []string   `json:"keywords,omitempty"`
	ExcludeKeywords []string   `json:"exclude_keywords,omitempty"`
}

// TimeRange represents a time range for filtering
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NotificationManager manages multiple notification channels
type NotificationManager interface {
	AddChannel(channel NotificationChannel) error
	RemoveChannel(channelID string) error
	GetChannel(channelID string) (*NotificationChannel, error)
	GetChannels() []NotificationChannel
	SendNotification(alert models.Alert, channelIDs []string) error
	GetHistory(alertID string) (*NotificationHistory, error)
	GetStats() *NotificationStats
	TestChannel(channelID string) error
}

// TemplateManager manages notification templates
type TemplateManager interface {
	CreateTemplate(template NotificationTemplate) error
	UpdateTemplate(template NotificationTemplate) error
	DeleteTemplate(templateID string) error
	GetTemplate(templateID string) (*NotificationTemplate, error)
	GetTemplates() []NotificationTemplate
	RenderTemplate(templateID string, data map[string]interface{}) (string, error)
}

// RateLimiter manages rate limiting for notifications
type RateLimiter interface {
	Allow(channelID string) bool
	GetLimit(channelID string) int
	SetLimit(channelID string, limit int)
	Reset(channelID string)
}

// NotificationDeduplicator manages notification deduplication
type NotificationDeduplicator interface {
	IsDuplicate(alert models.Alert) bool
	Add(alert models.Alert) error
	Cleanup() error
}

// NotificationQueue manages queuing of notifications
type NotificationQueue interface {
	Enqueue(alert models.Alert, channelID string) error
	Dequeue() (*QueueItem, error)
	Size() int
	IsEmpty() bool
}

// QueueItem represents an item in the notification queue
type QueueItem struct {
	Alert     models.Alert `json:"alert"`
	ChannelID string       `json:"channel_id"`
	Timestamp time.Time    `json:"timestamp"`
	Retries   int          `json:"retries"`
}

// Global notification configuration instance
var globalNotificationConfig NotificationGlobalConfig

// SetGlobalNotificationConfig sets the global notification configuration
func SetGlobalNotificationConfig(config NotificationGlobalConfig) {
	globalNotificationConfig = config
}

// GetGlobalNotificationConfig returns the global notification configuration
func GetGlobalNotificationConfig() NotificationGlobalConfig {
	return globalNotificationConfig
}

// DefaultNotificationConfig returns default notification configuration
func DefaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		Enabled:             true,
		MaxRetries:          3,
		RetryDelay:          5 * time.Second,
		Timeout:             30 * time.Second,
		RateLimitPerMin:     60,
		BatchSize:           10,
		BatchDelay:          1 * time.Second,
		EnableDeduplication: true,
		DeduplicationWindow: 5 * time.Minute,
	}
}

// DefaultGlobalNotificationConfig returns default global notification configuration
func DefaultGlobalNotificationConfig() NotificationGlobalConfig {
	return NotificationGlobalConfig{
		SeverityEmojis: map[string]string{
			"critical": "🔴",
			"high":     "🟠",
			"medium":   "🟡",
			"low":      "🟢",
			"default":  "⚪",
		},
		SeverityColors: map[string]string{
			"critical": "#ff0000",
			"high":     "#ff8000",
			"medium":   "#ffff00",
			"low":      "#00ff00",
			"default":  "#808080",
		},
	}
}

// GetSeverityEmoji returns emoji for severity levels using global configuration
func GetSeverityEmoji(severity string) string {
	config := GetGlobalNotificationConfig()
	if config.SeverityEmojis == nil {
		config = DefaultGlobalNotificationConfig()
	}

	if emoji, exists := config.SeverityEmojis[severity]; exists {
		return emoji
	}
	if defaultEmoji, exists := config.SeverityEmojis["default"]; exists {
		return defaultEmoji
	}
	return "⚪"
}

// GetSeverityColor returns color for severity levels using global configuration
func GetSeverityColor(severity string) string {
	config := GetGlobalNotificationConfig()
	if config.SeverityColors == nil {
		config = DefaultGlobalNotificationConfig()
	}

	if color, exists := config.SeverityColors[severity]; exists {
		return color
	}
	if defaultColor, exists := config.SeverityColors["default"]; exists {
		return defaultColor
	}
	return "#808080"
}
