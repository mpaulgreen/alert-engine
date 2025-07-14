package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// SlackNotifier implements the Notifier interface for Slack
type SlackNotifier struct {
	webhookURL string
	channel    string
	username   string
	iconEmoji  string
	enabled    bool
	config     NotificationConfig
	client     *http.Client
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    "#alerts",
		username:   "Alert Engine",
		iconEmoji:  ":warning:",
		enabled:    true,
		config:     DefaultNotificationConfig(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendAlert sends an alert to Slack
func (s *SlackNotifier) SendAlert(alert models.Alert) error {
	if !s.enabled {
		return fmt.Errorf("slack notifier is disabled")
	}

	message := s.buildSlackMessage(alert)

	return s.sendMessage(message)
}

// buildSlackMessage builds a Slack message from an alert
func (s *SlackNotifier) buildSlackMessage(alert models.Alert) SlackMessage {
	severityEmoji := GetSeverityEmoji(alert.Severity)
	severityColor := GetSeverityColor(alert.Severity)

	title := fmt.Sprintf("%s %s", severityEmoji, alert.RuleName)

	// Build main text
	text := fmt.Sprintf("Alert triggered for rule: *%s*", alert.RuleName)

	// Build attachment with details
	attachment := SlackAttachment{
		Color:      severityColor,
		Title:      title,
		Text:       s.formatLogMessage(alert.LogEntry.Message),
		Timestamp:  alert.Timestamp.Unix(),
		Footer:     "Alert Engine",
		FooterIcon: ":warning:",
		Fields: []SlackField{
			{
				Title: "Severity",
				Value: strings.ToUpper(alert.Severity),
				Short: true,
			},
			{
				Title: "Namespace",
				Value: alert.LogEntry.GetNamespace(),
				Short: true,
			},
			{
				Title: "Service",
				Value: s.getServiceName(alert.LogEntry),
				Short: true,
			},
			{
				Title: "Pod",
				Value: alert.LogEntry.Kubernetes.Pod,
				Short: true,
			},
			{
				Title: "Log Level",
				Value: alert.LogEntry.Level,
				Short: true,
			},
			{
				Title: "Count",
				Value: fmt.Sprintf("%d", alert.Count),
				Short: true,
			},
		},
	}

	return SlackMessage{
		Channel:     s.channel,
		Username:    s.username,
		IconEmoji:   s.iconEmoji,
		Text:        text,
		Attachments: []SlackAttachment{attachment},
	}
}

// formatLogMessage formats the log message for Slack
func (s *SlackNotifier) formatLogMessage(message string) string {
	// Limit message length to avoid overly long Slack messages
	if len(message) > 500 {
		return message[:500] + "..."
	}

	// Escape special characters for Slack
	message = strings.ReplaceAll(message, "&", "&amp;")
	message = strings.ReplaceAll(message, "<", "&lt;")
	message = strings.ReplaceAll(message, ">", "&gt;")

	return fmt.Sprintf("```%s```", message)
}

// getServiceName extracts service name from log entry
func (s *SlackNotifier) getServiceName(logEntry models.LogEntry) string {
	if appLabel, exists := logEntry.Kubernetes.Labels["app"]; exists {
		return appLabel
	}
	if serviceLabel, exists := logEntry.Kubernetes.Labels["service"]; exists {
		return serviceLabel
	}
	return "unknown"
}

// sendMessage sends a message to Slack
func (s *SlackNotifier) sendMessage(message SlackMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	req, err := http.NewRequest("POST", s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message to Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	return nil
}

// TestConnection tests the Slack connection
func (s *SlackNotifier) TestConnection() error {
	testMessage := SlackMessage{
		Channel:   s.channel,
		Username:  s.username,
		IconEmoji: s.iconEmoji,
		Text:      "Test message from Alert Engine",
		Attachments: []SlackAttachment{
			{
				Color: "#36a64f",
				Title: "Connection Test",
				Text:  "If you can see this message, the Slack integration is working correctly!",
				Fields: []SlackField{
					{
						Title: "Status",
						Value: "✅ Connected",
						Short: true,
					},
					{
						Title: "Timestamp",
						Value: time.Now().Format(time.RFC3339),
						Short: true,
					},
				},
			},
		},
	}

	return s.sendMessage(testMessage)
}

// GetName returns the notifier name
func (s *SlackNotifier) GetName() string {
	return "slack"
}

// IsEnabled returns whether the notifier is enabled
func (s *SlackNotifier) IsEnabled() bool {
	return s.enabled
}

// SetEnabled enables or disables the notifier
func (s *SlackNotifier) SetEnabled(enabled bool) {
	s.enabled = enabled
}

// SetChannel sets the Slack channel
func (s *SlackNotifier) SetChannel(channel string) {
	s.channel = channel
}

// SetUsername sets the Slack username
func (s *SlackNotifier) SetUsername(username string) {
	s.username = username
}

// SetIconEmoji sets the Slack icon emoji
func (s *SlackNotifier) SetIconEmoji(iconEmoji string) {
	s.iconEmoji = iconEmoji
}

// SetWebhookURL sets the Slack webhook URL
func (s *SlackNotifier) SetWebhookURL(webhookURL string) {
	s.webhookURL = webhookURL
}

// GetConfig returns the notification configuration
func (s *SlackNotifier) GetConfig() NotificationConfig {
	return s.config
}

// SetConfig sets the notification configuration
func (s *SlackNotifier) SetConfig(config NotificationConfig) {
	s.config = config
	s.client.Timeout = config.Timeout
}

// SendSimpleMessage sends a simple text message to Slack
func (s *SlackNotifier) SendSimpleMessage(text string) error {
	message := SlackMessage{
		Channel:   s.channel,
		Username:  s.username,
		IconEmoji: s.iconEmoji,
		Text:      text,
	}

	return s.sendMessage(message)
}

// SendRichMessage sends a rich message with attachments to Slack
func (s *SlackNotifier) SendRichMessage(text string, attachments []SlackAttachment) error {
	message := SlackMessage{
		Channel:     s.channel,
		Username:    s.username,
		IconEmoji:   s.iconEmoji,
		Text:        text,
		Attachments: attachments,
	}

	return s.sendMessage(message)
}

// SendCustomMessage sends a custom message to Slack
func (s *SlackNotifier) SendCustomMessage(message SlackMessage) error {
	// Override with notifier defaults if not specified
	if message.Channel == "" {
		message.Channel = s.channel
	}
	if message.Username == "" {
		message.Username = s.username
	}
	if message.IconEmoji == "" {
		message.IconEmoji = s.iconEmoji
	}

	return s.sendMessage(message)
}

// CreateAlertSummary creates a summary of multiple alerts
func (s *SlackNotifier) CreateAlertSummary(alerts []models.Alert) SlackMessage {
	if len(alerts) == 0 {
		return SlackMessage{}
	}

	// Group alerts by severity
	severityCounts := make(map[string]int)
	for _, alert := range alerts {
		severityCounts[alert.Severity]++
	}

	// Build summary text
	text := fmt.Sprintf("📊 *Alert Summary* - %d alerts triggered", len(alerts))

	// Build fields for severity breakdown
	var fields []SlackField
	for severity, count := range severityCounts {
		emoji := GetSeverityEmoji(severity)
		fields = append(fields, SlackField{
			Title: fmt.Sprintf("%s %s", emoji, strings.ToTitle(severity)),
			Value: fmt.Sprintf("%d", count),
			Short: true,
		})
	}

	// Add time range
	if len(alerts) > 1 {
		firstAlert := alerts[0]
		lastAlert := alerts[len(alerts)-1]
		fields = append(fields, SlackField{
			Title: "Time Range",
			Value: fmt.Sprintf("%s - %s",
				firstAlert.Timestamp.Format("15:04:05"),
				lastAlert.Timestamp.Format("15:04:05")),
			Short: true,
		})
	}

	attachment := SlackAttachment{
		Color:      "#ffcc00",
		Title:      "Alert Summary",
		Fields:     fields,
		Footer:     "Alert Engine",
		FooterIcon: ":warning:",
		Timestamp:  time.Now().Unix(),
	}

	return SlackMessage{
		Channel:     s.channel,
		Username:    s.username,
		IconEmoji:   s.iconEmoji,
		Text:        text,
		Attachments: []SlackAttachment{attachment},
	}
}

// ValidateConfig validates the Slack notifier configuration
func (s *SlackNotifier) ValidateConfig() error {
	if s.webhookURL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if !strings.HasPrefix(s.webhookURL, "https://hooks.slack.com/") {
		return fmt.Errorf("invalid Slack webhook URL format")
	}

	if s.channel == "" {
		return fmt.Errorf("channel is required")
	}

	if !strings.HasPrefix(s.channel, "#") && !strings.HasPrefix(s.channel, "@") {
		return fmt.Errorf("channel must start with # or @")
	}

	return nil
}
