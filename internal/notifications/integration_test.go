//go:build integration
// +build integration

package notifications

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/log-monitoring/alert-engine/internal/notifications/mocks"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

func TestSlackNotifierIntegration(t *testing.T) {
	t.Run("sends alert to mock slack server", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for success response
		mockServer.SetupSlackScenario("success")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Create test alert
		alert := models.Alert{
			ID:        "alert-001",
			RuleID:    "rule-001",
			RuleName:  "High Error Rate",
			Severity:  "critical",
			Count:     15,
			Timestamp: time.Now(),
			LogEntry: models.LogEntry{
				Level:   "ERROR",
				Message: "Database connection failed",
				Kubernetes: models.KubernetesInfo{
					Namespace: "production",
					Pod:       "api-server-abc123",
					Container: "api-server",
					Labels: map[string]string{
						"app": "api-server",
					},
				},
			},
		}

		// Send alert
		err := notifier.SendAlert(alert)

		// Verify success
		assert.NoError(t, err)

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())

		// Verify request content
		lastRequest := mockServer.GetLastRequest()
		assert.Equal(t, "POST", lastRequest.Method)
		assert.Equal(t, "application/json", lastRequest.Header.Get("Content-Type"))

		// Verify request body contains alert information
		lastBody := mockServer.GetLastRequestBody()
		assert.Contains(t, lastBody, "High Error Rate")
		assert.Contains(t, lastBody, "production")
		assert.Contains(t, lastBody, "api-server")

		// Verify JSON structure
		var slackMessage map[string]interface{}
		err = json.Unmarshal([]byte(lastBody), &slackMessage)
		assert.NoError(t, err)

		assert.Contains(t, slackMessage, "text")
		assert.Contains(t, slackMessage, "attachments")
		assert.Contains(t, slackMessage, "channel")
		assert.Contains(t, slackMessage, "username")
	})

	t.Run("handles slack server errors", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for error response
		mockServer.SetupSlackScenario("server_error")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Create test alert
		alert := models.Alert{
			ID:       "alert-002",
			RuleName: "Test Alert",
			Severity: "high",
		}

		// Send alert
		err := notifier.SendAlert(alert)

		// Verify error handling
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())
	})

	t.Run("handles slack rate limiting", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for rate limit response
		mockServer.SetRateLimitResponse(60)

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Create test alert
		alert := models.Alert{
			ID:       "alert-003",
			RuleName: "Rate Limited Alert",
			Severity: "medium",
		}

		// Send alert
		err := notifier.SendAlert(alert)

		// Verify rate limit error handling
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "429")

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())

		// Verify server returned rate limit response
		lastRequest := mockServer.GetLastRequest()
		assert.Equal(t, "POST", lastRequest.Method)
	})

	t.Run("test connection with mock server", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for success response
		mockServer.SetupSlackScenario("success")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Test connection
		err := notifier.TestConnection()

		// Verify success
		assert.NoError(t, err)

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())

		// Verify request content
		lastBody := mockServer.GetLastRequestBody()
		assert.Contains(t, lastBody, "Test message from Alert Engine")
		assert.Contains(t, lastBody, "Connection Test")
	})
}

func TestSlackNotifierIntegration_MessageFormatting(t *testing.T) {
	t.Run("formats different severity levels correctly", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for success response
		mockServer.SetupSlackScenario("success")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Test different severity levels
		severities := []string{"critical", "high", "medium", "low"}
		expectedEmojis := []string{"🔴", "🟠", "🟡", "🟢"}
		expectedColors := []string{"#ff0000", "#ff8000", "#ffff00", "#00ff00"}

		for i, severity := range severities {
			// Reset server state
			mockServer.Reset()

			// Create test alert
			alert := models.Alert{
				ID:        fmt.Sprintf("alert-%d", i),
				RuleName:  fmt.Sprintf("Test Alert %s", severity),
				Severity:  severity,
				Count:     5,
				Timestamp: time.Now(),
				LogEntry: models.LogEntry{
					Level:   "ERROR",
					Message: fmt.Sprintf("Test message for %s severity", severity),
					Kubernetes: models.KubernetesInfo{
						Namespace: "test",
						Pod:       "test-pod",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
				},
			}

			// Send alert
			err := notifier.SendAlert(alert)
			assert.NoError(t, err)

			// Verify server received the request
			assert.Equal(t, 1, mockServer.GetCallCount())

			// Verify request body contains severity-specific formatting
			lastBody := mockServer.GetLastRequestBody()
			assert.Contains(t, lastBody, expectedEmojis[i])
			assert.Contains(t, lastBody, expectedColors[i])
			assert.Contains(t, lastBody, strings.ToUpper(severity))
		}
	})

	t.Run("formats kubernetes information correctly", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for success response
		mockServer.SetupSlackScenario("success")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Create test alert with detailed Kubernetes info
		alert := models.Alert{
			ID:        "alert-k8s-001",
			RuleName:  "Kubernetes Alert",
			Severity:  "high",
			Count:     10,
			Timestamp: time.Now(),
			LogEntry: models.LogEntry{
				Level:   "WARN",
				Message: "Pod restart detected in production namespace",
				Kubernetes: models.KubernetesInfo{
					Namespace: "production",
					Pod:       "web-service-deployment-abc123",
					Container: "web-service",
					Labels: map[string]string{
						"app":         "web-service",
						"version":     "v1.2.3",
						"environment": "production",
						"team":        "backend",
					},
				},
				Host: "node-1",
			},
		}

		// Send alert
		err := notifier.SendAlert(alert)
		assert.NoError(t, err)

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())

		// Verify request body contains Kubernetes information
		lastBody := mockServer.GetLastRequestBody()
		assert.Contains(t, lastBody, "production")
		assert.Contains(t, lastBody, "web-service")
		assert.Contains(t, lastBody, "web-service-deployment-abc123")
		assert.Contains(t, lastBody, "WARN")
		assert.Contains(t, lastBody, "10")
	})
}

func TestSlackNotifierIntegration_MultipleMessages(t *testing.T) {
	t.Run("sends multiple alerts sequentially", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for success response
		mockServer.SetupSlackScenario("success")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Create multiple test alerts
		alerts := []models.Alert{
			{
				ID:       "alert-001",
				RuleName: "First Alert",
				Severity: "critical",
				Count:    5,
			},
			{
				ID:       "alert-002",
				RuleName: "Second Alert",
				Severity: "high",
				Count:    3,
			},
			{
				ID:       "alert-003",
				RuleName: "Third Alert",
				Severity: "medium",
				Count:    2,
			},
		}

		// Send all alerts
		for _, alert := range alerts {
			err := notifier.SendAlert(alert)
			assert.NoError(t, err)
		}

		// Verify server received all requests
		assert.Equal(t, 3, mockServer.GetCallCount())

		// Verify all request bodies
		requestBodies := mockServer.GetRequestBodies()
		assert.Len(t, requestBodies, 3)

		assert.Contains(t, requestBodies[0], "First Alert")
		assert.Contains(t, requestBodies[1], "Second Alert")
		assert.Contains(t, requestBodies[2], "Third Alert")
	})
}

func TestSlackNotifierIntegration_CustomConfiguration(t *testing.T) {
	t.Run("uses custom channel and username", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for success response
		mockServer.SetupSlackScenario("success")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())
		notifier.SetChannel("#custom-alerts")
		notifier.SetUsername("Custom Alert Bot")
		notifier.SetIconEmoji(":robot_face:")

		// Create test alert
		alert := models.Alert{
			ID:       "alert-custom-001",
			RuleName: "Custom Alert",
			Severity: "high",
			Count:    7,
		}

		// Send alert
		err := notifier.SendAlert(alert)
		assert.NoError(t, err)

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())

		// Verify request body contains custom configuration
		lastBody := mockServer.GetLastRequestBody()
		assert.Contains(t, lastBody, "#custom-alerts")
		assert.Contains(t, lastBody, "Custom Alert Bot")
		assert.Contains(t, lastBody, ":robot_face:")
	})
}

func TestSlackNotifierIntegration_ErrorScenarios(t *testing.T) {
	t.Run("handles network timeouts", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for timeout response
		mockServer.SetupSlackScenario("timeout")

		// Create notifier with mock server URL and short timeout
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())
		config := notifier.GetConfig()
		config.Timeout = 1 * time.Second
		notifier.UpdateConfig(config)

		// Create test alert
		alert := models.Alert{
			ID:       "alert-timeout-001",
			RuleName: "Timeout Alert",
			Severity: "critical",
		}

		// Send alert (should timeout)
		err := notifier.SendAlert(alert)

		// Verify timeout error handling
		assert.Error(t, err)
		// The actual error may vary based on implementation
		// but it should be related to timeout or context cancellation
	})

	t.Run("handles bad request errors", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for bad request response
		mockServer.SetupSlackScenario("bad_request")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Create test alert
		alert := models.Alert{
			ID:       "alert-bad-001",
			RuleName: "Bad Request Alert",
			Severity: "high",
		}

		// Send alert
		err := notifier.SendAlert(alert)

		// Verify bad request error handling
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())
	})

	t.Run("handles forbidden errors", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for forbidden response
		mockServer.SetupSlackScenario("forbidden")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Create test alert
		alert := models.Alert{
			ID:       "alert-forbidden-001",
			RuleName: "Forbidden Alert",
			Severity: "medium",
		}

		// Send alert
		err := notifier.SendAlert(alert)

		// Verify forbidden error handling
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "403")

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())
	})
}

func TestSlackNotifierIntegration_MessageValidation(t *testing.T) {
	t.Run("validates slack message structure", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for success response
		mockServer.SetupSlackScenario("success")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Create test alert
		alert := models.Alert{
			ID:        "alert-validation-001",
			RuleName:  "Validation Alert",
			Severity:  "critical",
			Count:     20,
			Timestamp: time.Now(),
			LogEntry: models.LogEntry{
				Level:   "ERROR",
				Message: "Critical validation error occurred",
				Kubernetes: models.KubernetesInfo{
					Namespace: "validation",
					Pod:       "validation-pod",
					Labels: map[string]string{
						"app": "validation-service",
					},
				},
			},
		}

		// Send alert
		err := notifier.SendAlert(alert)
		assert.NoError(t, err)

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())

		// Validate message structure using mock server validator
		lastBody := mockServer.GetLastRequestBody()
		err = mockServer.ValidateSlackWebhookPayload(lastBody)
		assert.NoError(t, err)

		// Verify JSON structure
		var slackMessage map[string]interface{}
		err = json.Unmarshal([]byte(lastBody), &slackMessage)
		assert.NoError(t, err)

		// Verify required fields
		assert.Contains(t, slackMessage, "text")
		assert.Contains(t, slackMessage, "attachments")
		assert.Contains(t, slackMessage, "channel")
		assert.Contains(t, slackMessage, "username")
		assert.Contains(t, slackMessage, "icon_emoji")

		// Verify attachments structure
		attachments, ok := slackMessage["attachments"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, attachments, 1)

		attachment, ok := attachments[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, attachment, "color")
		assert.Contains(t, attachment, "title")
		assert.Contains(t, attachment, "text")
		assert.Contains(t, attachment, "fields")
		assert.Contains(t, attachment, "footer")
		assert.Contains(t, attachment, "ts")

		// Verify fields structure
		fields, ok := attachment["fields"].([]interface{})
		assert.True(t, ok)
		assert.Greater(t, len(fields), 0)

		// Verify first field structure
		field, ok := fields[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, field, "title")
		assert.Contains(t, field, "value")
		assert.Contains(t, field, "short")
	})
}

func TestSlackNotifierIntegration_CustomMessages(t *testing.T) {
	t.Run("sends simple message", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for success response
		mockServer.SetupSlackScenario("success")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Send simple message
		err := notifier.SendSimpleMessage("System maintenance scheduled for tonight")

		// Verify success
		assert.NoError(t, err)

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())

		// Verify request body contains simple message
		lastBody := mockServer.GetLastRequestBody()
		assert.Contains(t, lastBody, "System maintenance scheduled for tonight")

		// Verify JSON structure
		var slackMessage map[string]interface{}
		err = json.Unmarshal([]byte(lastBody), &slackMessage)
		assert.NoError(t, err)

		assert.Equal(t, "System maintenance scheduled for tonight", slackMessage["text"])
		assert.Contains(t, slackMessage, "channel")
		assert.Contains(t, slackMessage, "username")
		assert.Contains(t, slackMessage, "icon_emoji")
	})

	t.Run("creates alert summary", func(t *testing.T) {
		// Create mock Slack server
		mockServer := mocks.NewMockSlackServer()
		defer mockServer.Close()

		// Configure server for success response
		mockServer.SetupSlackScenario("success")

		// Create notifier with mock server URL
		notifier := NewSlackNotifier(mockServer.GetWebhookURL())

		// Create multiple alerts for summary
		alerts := []models.Alert{
			{
				ID:        "alert-summary-001",
				RuleName:  "Critical Alert",
				Severity:  "critical",
				Count:     10,
				Timestamp: time.Now(),
			},
			{
				ID:        "alert-summary-002",
				RuleName:  "High Alert",
				Severity:  "high",
				Count:     5,
				Timestamp: time.Now(),
			},
			{
				ID:        "alert-summary-003",
				RuleName:  "Medium Alert",
				Severity:  "medium",
				Count:     3,
				Timestamp: time.Now(),
			},
		}

		// Create alert summary
		summary := notifier.CreateAlertSummary(alerts)

		// Verify summary structure
		assert.NotNil(t, summary)
		assert.Contains(t, summary.Text, "3 alerts")
		assert.Equal(t, "#alerts", summary.Channel)
		assert.Equal(t, "Alert Engine", summary.Username)
		assert.Len(t, summary.Attachments, 1)

		// Verify attachment structure
		attachment := summary.Attachments[0]
		assert.Equal(t, "Alert Summary", attachment.Title)
		assert.NotEmpty(t, attachment.Fields)

		// Send the summary using custom message
		err := notifier.SendCustomMessage(summary)
		assert.NoError(t, err)

		// Verify server received the request
		assert.Equal(t, 1, mockServer.GetCallCount())

		// Verify request body contains summary information
		lastBody := mockServer.GetLastRequestBody()
		assert.Contains(t, lastBody, "3 alerts")
		assert.Contains(t, lastBody, "Alert Summary")
	})
}
