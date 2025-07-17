package notifications

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// SlackConfig holds configuration for Slack notifications
type SlackConfig struct {
	WebhookURL     string            `json:"webhook_url"`
	Channel        string            `json:"channel"`
	Username       string            `json:"username"`
	IconEmoji      string            `json:"icon_emoji"`
	Timeout        time.Duration     `json:"timeout"`
	Enabled        bool              `json:"enabled"`
	Templates      TemplateConfig    `json:"templates"`
	SeverityEmojis map[string]string `json:"severity_emojis"`
	SeverityColors map[string]string `json:"severity_colors"`
}

// TemplateConfig holds message template configuration
type TemplateConfig struct {
	AlertMessage     string               `json:"alert_message"`
	SlackAlertTitle  string               `json:"slack_alert_title"`
	SlackAlertFields []SlackTemplateField `json:"slack_alert_fields"`
}

// SlackTemplateField represents a field template for Slack attachments
type SlackTemplateField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackNotifier implements the Notifier interface for Slack
type SlackNotifier struct {
	config               SlackConfig
	client               *http.Client
	alertMessageTemplate *template.Template
	titleTemplate        *template.Template
	fieldTemplates       []SlackFieldTemplate
}

// SlackFieldTemplate holds compiled field templates
type SlackFieldTemplate struct {
	Title         string
	ValueTemplate *template.Template
	Short         bool
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

// NewSlackNotifier creates a new Slack notifier with configuration
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return NewSlackNotifierWithConfig(SlackConfig{
		WebhookURL:     webhookURL,
		Channel:        "#alerts",
		Username:       "Alert Engine",
		IconEmoji:      ":warning:",
		Timeout:        30 * time.Second,
		Enabled:        true,
		Templates:      DefaultTemplateConfig(),
		SeverityEmojis: DefaultSeverityEmojis(),
		SeverityColors: DefaultSeverityColors(),
	})
}

// NewSlackNotifierWithConfig creates a new Slack notifier with full configuration
func NewSlackNotifierWithConfig(config SlackConfig) *SlackNotifier {
	// Create HTTP client with custom transport for TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip TLS verification for testing
	}

	notifier := &SlackNotifier{
		config: config,
		client: &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		},
	}

	// Compile templates
	notifier.compileTemplates()

	return notifier
}

// compileTemplates compiles all message templates
func (s *SlackNotifier) compileTemplates() {
	var err error

	// Compile alert message template
	s.alertMessageTemplate, err = template.New("alert_message").Parse(s.config.Templates.AlertMessage)
	if err != nil {
		// Fallback to default template
		s.alertMessageTemplate, _ = template.New("alert_message").Parse(DefaultTemplateConfig().AlertMessage)
	}

	// Compile title template
	s.titleTemplate, err = template.New("title").Parse(s.config.Templates.SlackAlertTitle)
	if err != nil {
		// Fallback to default template
		s.titleTemplate, _ = template.New("title").Parse(DefaultTemplateConfig().SlackAlertTitle)
	}

	// Compile field templates
	s.fieldTemplates = make([]SlackFieldTemplate, len(s.config.Templates.SlackAlertFields))
	for i, fieldConfig := range s.config.Templates.SlackAlertFields {
		tmpl, err := template.New(fmt.Sprintf("field_%d", i)).Parse(fieldConfig.Value)
		if err != nil {
			// Use raw value if template compilation fails
			tmpl, _ = template.New(fmt.Sprintf("field_%d", i)).Parse("{{.Value}}")
		}

		s.fieldTemplates[i] = SlackFieldTemplate{
			Title:         fieldConfig.Title,
			ValueTemplate: tmpl,
			Short:         fieldConfig.Short,
		}
	}
}

// SendAlert sends an alert to Slack using configured templates
func (s *SlackNotifier) SendAlert(alert models.Alert) error {
	if !s.config.Enabled {
		return fmt.Errorf("slack notifier is disabled")
	}

	if s.config.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	message := s.buildSlackMessage(alert)
	return s.sendMessage(message)
}

// buildSlackMessage builds a Slack message from an alert using templates
func (s *SlackNotifier) buildSlackMessage(alert models.Alert) SlackMessage {
	// Prepare template data
	templateData := s.prepareTemplateData(alert)

	// Build title
	title := s.renderTemplate(s.titleTemplate, templateData)

	// Build alert message text
	text := s.renderTemplate(s.alertMessageTemplate, templateData)

	// Build fields
	var fields []SlackField
	for _, fieldTemplate := range s.fieldTemplates {
		value := s.renderTemplate(fieldTemplate.ValueTemplate, templateData)
		fields = append(fields, SlackField{
			Title: fieldTemplate.Title,
			Value: value,
			Short: fieldTemplate.Short,
		})
	}

	// Get severity color
	severityColor := s.getSeverityColor(alert.Severity)

	// Build attachment
	attachment := SlackAttachment{
		Color:      severityColor,
		Title:      title,
		Text:       s.formatLogMessage(alert.LogEntry.Message),
		Timestamp:  alert.Timestamp.Unix(),
		Footer:     "Alert Engine",
		FooterIcon: ":warning:",
		Fields:     fields,
	}

	return SlackMessage{
		Channel:     s.config.Channel,
		Username:    s.config.Username,
		IconEmoji:   s.config.IconEmoji,
		Text:        text,
		Attachments: []SlackAttachment{attachment},
	}
}

// prepareTemplateData prepares data for template rendering
func (s *SlackNotifier) prepareTemplateData(alert models.Alert) map[string]interface{} {
	return map[string]interface{}{
		"RuleName":      alert.RuleName,
		"Service":       s.getServiceName(alert.LogEntry),
		"Namespace":     alert.LogEntry.GetNamespace(),
		"Level":         alert.LogEntry.Level,
		"Count":         alert.Count,
		"TimeWindow":    "N/A", // TODO: Extract from rule if available
		"Message":       alert.LogEntry.Message,
		"Severity":      strings.ToUpper(alert.Severity),
		"SeverityEmoji": s.getSeverityEmoji(alert.Severity),
		"Pod":           alert.LogEntry.GetPodName(),
	}
}

// renderTemplate renders a template with the given data
func (s *SlackNotifier) renderTemplate(tmpl *template.Template, data map[string]interface{}) string {
	if tmpl == nil {
		return ""
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Template error: %v", err)
	}

	return buf.String()
}

// getSeverityEmoji returns emoji for severity levels using configuration
func (s *SlackNotifier) getSeverityEmoji(severity string) string {
	if emoji, exists := s.config.SeverityEmojis[severity]; exists {
		return emoji
	}
	if defaultEmoji, exists := s.config.SeverityEmojis["default"]; exists {
		return defaultEmoji
	}
	return "⚪"
}

// getSeverityColor returns color for severity levels using configuration
func (s *SlackNotifier) getSeverityColor(severity string) string {
	if color, exists := s.config.SeverityColors[severity]; exists {
		return color
	}
	if defaultColor, exists := s.config.SeverityColors["default"]; exists {
		return defaultColor
	}
	return "#808080"
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

	req, err := http.NewRequest("POST", s.config.WebhookURL, bytes.NewBuffer(payload))
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
		Channel:   s.config.Channel,
		Username:  s.config.Username,
		IconEmoji: s.config.IconEmoji,
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

// Configuration getter/setter methods for backward compatibility
func (s *SlackNotifier) GetName() string { return "slack" }
func (s *SlackNotifier) IsEnabled() bool { return s.config.Enabled && s.config.WebhookURL != "" }
func (s *SlackNotifier) SetEnabled(enabled bool) {
	s.config.Enabled = enabled
}
func (s *SlackNotifier) SetChannel(channel string)       { s.config.Channel = channel }
func (s *SlackNotifier) SetUsername(username string)     { s.config.Username = username }
func (s *SlackNotifier) SetIconEmoji(iconEmoji string)   { s.config.IconEmoji = iconEmoji }
func (s *SlackNotifier) SetWebhookURL(webhookURL string) { s.config.WebhookURL = webhookURL }

// UpdateConfig updates the entire configuration and recompiles templates
func (s *SlackNotifier) UpdateConfig(config SlackConfig) {
	s.config = config
	s.client.Timeout = config.Timeout
	s.compileTemplates()
}

// GetConfig returns the current configuration
func (s *SlackNotifier) GetConfig() SlackConfig {
	return s.config
}

// Default configuration functions
func DefaultTemplateConfig() TemplateConfig {
	return TemplateConfig{
		AlertMessage: `🚨 Alert: {{.RuleName}}
Service: {{.Service}}
Namespace: {{.Namespace}}
Level: {{.Level}}
Count: {{.Count}} in {{.TimeWindow}}
Message: {{.Message}}`,
		SlackAlertTitle: "{{.SeverityEmoji}} {{.RuleName}}",
		SlackAlertFields: []SlackTemplateField{
			{Title: "Severity", Value: "{{.Severity}}", Short: true},
			{Title: "Namespace", Value: "{{.Namespace}}", Short: true},
			{Title: "Service", Value: "{{.Service}}", Short: true},
			{Title: "Pod", Value: "{{.Pod}}", Short: true},
			{Title: "Log Level", Value: "{{.Level}}", Short: true},
			{Title: "Count", Value: "{{.Count}}", Short: true},
		},
	}
}

func DefaultSeverityEmojis() map[string]string {
	return map[string]string{
		"critical": "🔴",
		"high":     "🟠",
		"medium":   "🟡",
		"low":      "🟢",
		"default":  "⚪",
	}
}

func DefaultSeverityColors() map[string]string {
	return map[string]string{
		"critical": "#ff0000",
		"high":     "#ff8000",
		"medium":   "#ffff00",
		"low":      "#00ff00",
		"default":  "#808080",
	}
}

// Legacy methods for backward compatibility
func (s *SlackNotifier) SendSimpleMessage(text string) error {
	message := SlackMessage{
		Channel:   s.config.Channel,
		Username:  s.config.Username,
		IconEmoji: s.config.IconEmoji,
		Text:      text,
	}
	return s.sendMessage(message)
}

func (s *SlackNotifier) SendRichMessage(text string, attachments []SlackAttachment) error {
	message := SlackMessage{
		Channel:     s.config.Channel,
		Username:    s.config.Username,
		IconEmoji:   s.config.IconEmoji,
		Text:        text,
		Attachments: attachments,
	}
	return s.sendMessage(message)
}

func (s *SlackNotifier) SendCustomMessage(message SlackMessage) error {
	if message.Channel == "" {
		message.Channel = s.config.Channel
	}
	if message.Username == "" {
		message.Username = s.config.Username
	}
	if message.IconEmoji == "" {
		message.IconEmoji = s.config.IconEmoji
	}
	return s.sendMessage(message)
}

func (s *SlackNotifier) ValidateConfig() error {
	if s.config.WebhookURL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if !strings.HasPrefix(s.config.WebhookURL, "https://hooks.slack.com/") {
		return fmt.Errorf("invalid Slack webhook URL format")
	}

	if s.config.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if !strings.HasPrefix(s.config.Channel, "#") && !strings.HasPrefix(s.config.Channel, "@") {
		return fmt.Errorf("channel must start with # or @")
	}

	return nil
}

// CreateAlertSummary creates a summary message for multiple alerts
func (s *SlackNotifier) CreateAlertSummary(alerts []models.Alert) SlackMessage {
	if len(alerts) == 0 {
		return SlackMessage{}
	}

	// Count alerts by severity
	severityCount := make(map[string]int)
	for _, alert := range alerts {
		severityCount[alert.Severity]++
	}

	// Create summary text
	text := fmt.Sprintf("%d alerts triggered", len(alerts))

	// Create attachment with summary details
	var fields []SlackField

	// Add severity breakdown
	for severity, count := range severityCount {
		emoji := s.getSeverityEmoji(severity)
		fields = append(fields, SlackField{
			Title: fmt.Sprintf("%s %s", emoji, strings.Title(severity)),
			Value: fmt.Sprintf("%d alert(s)", count),
			Short: true,
		})
	}

	// Add time range
	if len(alerts) > 0 {
		fields = append(fields, SlackField{
			Title: "Time Range",
			Value: fmt.Sprintf("%s - %s",
				alerts[len(alerts)-1].Timestamp.Format("15:04:05"),
				alerts[0].Timestamp.Format("15:04:05")),
			Short: true,
		})
	}

	attachment := SlackAttachment{
		Color:      "#808080", // Default gray color for summaries
		Title:      "Alert Summary",
		Fields:     fields,
		Footer:     "Alert Engine",
		FooterIcon: ":warning:",
		Timestamp:  time.Now().Unix(),
	}

	return SlackMessage{
		Channel:     s.config.Channel,
		Username:    s.config.Username,
		IconEmoji:   s.config.IconEmoji,
		Text:        text,
		Attachments: []SlackAttachment{attachment},
	}
}
