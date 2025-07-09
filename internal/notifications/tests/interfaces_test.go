//go:build unit
// +build unit

package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/log-monitoring/alert-engine/internal/notifications"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

func TestNotificationChannel(t *testing.T) {
	t.Run("creates notification channel", func(t *testing.T) {
		channel := notifications.NotificationChannel{
			ID:   "slack-prod",
			Name: "Production Slack",
			Type: "slack",
			Config: map[string]string{
				"webhook_url": "https://hooks.slack.com/services/T123456/B123456/XXXXXXXXXXXXXXXXXXXXXXXX",
				"channel":     "#alerts",
				"username":    "Alert Bot",
			},
			Enabled:  true,
			Created:  time.Now(),
			Modified: time.Now(),
		}

		assert.Equal(t, "slack-prod", channel.ID)
		assert.Equal(t, "Production Slack", channel.Name)
		assert.Equal(t, "slack", channel.Type)
		assert.True(t, channel.Enabled)
		assert.Equal(t, "#alerts", channel.Config["channel"])
	})

	t.Run("marshals and unmarshals notification channel", func(t *testing.T) {
		originalChannel := notifications.NotificationChannel{
			ID:   "email-dev",
			Name: "Development Email",
			Type: "email",
			Config: map[string]string{
				"smtp_server": "smtp.example.com",
				"to":          "dev@example.com",
			},
			Enabled:  false,
			Created:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Modified: time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC),
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(originalChannel)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaledChannel notifications.NotificationChannel
		err = json.Unmarshal(jsonData, &unmarshaledChannel)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, originalChannel.ID, unmarshaledChannel.ID)
		assert.Equal(t, originalChannel.Name, unmarshaledChannel.Name)
		assert.Equal(t, originalChannel.Type, unmarshaledChannel.Type)
		assert.Equal(t, originalChannel.Config["smtp_server"], unmarshaledChannel.Config["smtp_server"])
		assert.Equal(t, originalChannel.Enabled, unmarshaledChannel.Enabled)
		assert.True(t, originalChannel.Created.Equal(unmarshaledChannel.Created))
		assert.True(t, originalChannel.Modified.Equal(unmarshaledChannel.Modified))
	})
}

func TestNotificationTemplate(t *testing.T) {
	t.Run("creates notification template", func(t *testing.T) {
		template := notifications.NotificationTemplate{
			ID:      "critical-alert",
			Name:    "Critical Alert Template",
			Type:    "slack",
			Subject: "🔴 Critical Alert: {{.RuleName}}",
			Body:    "Service {{.Service}} in {{.Namespace}} has triggered a critical alert",
			Variables: map[string]string{
				"RuleName":  "{{.RuleName}}",
				"Service":   "{{.LogEntry.Kubernetes.Labels.app}}",
				"Namespace": "{{.LogEntry.Kubernetes.Namespace}}",
			},
			Severity: "critical",
			Created:  time.Now(),
			Modified: time.Now(),
		}

		assert.Equal(t, "critical-alert", template.ID)
		assert.Equal(t, "Critical Alert Template", template.Name)
		assert.Equal(t, "slack", template.Type)
		assert.Contains(t, template.Subject, "{{.RuleName}}")
		assert.Contains(t, template.Body, "{{.Service}}")
		assert.Equal(t, "critical", template.Severity)
	})

	t.Run("marshals and unmarshals notification template", func(t *testing.T) {
		originalTemplate := notifications.NotificationTemplate{
			ID:      "warning-template",
			Name:    "Warning Template",
			Type:    "email",
			Subject: "⚠️ Warning: {{.RuleName}}",
			Body:    "Warning alert triggered",
			Variables: map[string]string{
				"RuleName": "{{.RuleName}}",
				"Count":    "{{.Count}}",
			},
			Severity: "high",
			Created:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Modified: time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC),
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(originalTemplate)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaledTemplate notifications.NotificationTemplate
		err = json.Unmarshal(jsonData, &unmarshaledTemplate)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, originalTemplate.ID, unmarshaledTemplate.ID)
		assert.Equal(t, originalTemplate.Name, unmarshaledTemplate.Name)
		assert.Equal(t, originalTemplate.Type, unmarshaledTemplate.Type)
		assert.Equal(t, originalTemplate.Subject, unmarshaledTemplate.Subject)
		assert.Equal(t, originalTemplate.Body, unmarshaledTemplate.Body)
		assert.Equal(t, originalTemplate.Variables["RuleName"], unmarshaledTemplate.Variables["RuleName"])
		assert.Equal(t, originalTemplate.Severity, unmarshaledTemplate.Severity)
		assert.True(t, originalTemplate.Created.Equal(unmarshaledTemplate.Created))
		assert.True(t, originalTemplate.Modified.Equal(unmarshaledTemplate.Modified))
	})
}

func TestNotificationResult(t *testing.T) {
	t.Run("creates notification result", func(t *testing.T) {
		result := notifications.NotificationResult{
			ID:        "result-001",
			AlertID:   "alert-001",
			Channel:   "slack-prod",
			Status:    "success",
			Message:   "Alert sent successfully",
			Timestamp: time.Now(),
			Duration:  150 * time.Millisecond,
			Error:     "",
		}

		assert.Equal(t, "result-001", result.ID)
		assert.Equal(t, "alert-001", result.AlertID)
		assert.Equal(t, "slack-prod", result.Channel)
		assert.Equal(t, "success", result.Status)
		assert.Equal(t, "Alert sent successfully", result.Message)
		assert.Equal(t, 150*time.Millisecond, result.Duration)
		assert.Empty(t, result.Error)
	})

	t.Run("creates failed notification result", func(t *testing.T) {
		result := notifications.NotificationResult{
			ID:        "result-002",
			AlertID:   "alert-002",
			Channel:   "slack-prod",
			Status:    "failed",
			Message:   "Failed to send alert",
			Timestamp: time.Now(),
			Duration:  5 * time.Second,
			Error:     "connection timeout",
		}

		assert.Equal(t, "failed", result.Status)
		assert.Equal(t, "connection timeout", result.Error)
		assert.Equal(t, 5*time.Second, result.Duration)
	})

	t.Run("marshals and unmarshals notification result", func(t *testing.T) {
		originalResult := notifications.NotificationResult{
			ID:        "result-003",
			AlertID:   "alert-003",
			Channel:   "email-dev",
			Status:    "pending",
			Message:   "Queued for sending",
			Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Duration:  0,
			Error:     "",
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(originalResult)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaledResult notifications.NotificationResult
		err = json.Unmarshal(jsonData, &unmarshaledResult)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, originalResult.ID, unmarshaledResult.ID)
		assert.Equal(t, originalResult.AlertID, unmarshaledResult.AlertID)
		assert.Equal(t, originalResult.Channel, unmarshaledResult.Channel)
		assert.Equal(t, originalResult.Status, unmarshaledResult.Status)
		assert.Equal(t, originalResult.Message, unmarshaledResult.Message)
		assert.True(t, originalResult.Timestamp.Equal(unmarshaledResult.Timestamp))
		assert.Equal(t, originalResult.Duration, unmarshaledResult.Duration)
		assert.Equal(t, originalResult.Error, unmarshaledResult.Error)
	})
}

func TestNotificationHistory(t *testing.T) {
	t.Run("creates notification history", func(t *testing.T) {
		history := notifications.NotificationHistory{
			AlertID: "alert-001",
			RuleID:  "rule-001",
			Results: []notifications.NotificationResult{
				{
					ID:      "result-001",
					AlertID: "alert-001",
					Channel: "slack-prod",
					Status:  "success",
				},
				{
					ID:      "result-002",
					AlertID: "alert-001",
					Channel: "email-dev",
					Status:  "failed",
				},
			},
			TotalSent:   2,
			TotalFailed: 1,
			LastSent:    time.Now(),
		}

		assert.Equal(t, "alert-001", history.AlertID)
		assert.Equal(t, "rule-001", history.RuleID)
		assert.Len(t, history.Results, 2)
		assert.Equal(t, 2, history.TotalSent)
		assert.Equal(t, 1, history.TotalFailed)
		assert.Equal(t, "success", history.Results[0].Status)
		assert.Equal(t, "failed", history.Results[1].Status)
	})

	t.Run("marshals and unmarshals notification history", func(t *testing.T) {
		originalHistory := notifications.NotificationHistory{
			AlertID: "alert-002",
			RuleID:  "rule-002",
			Results: []notifications.NotificationResult{
				{
					ID:      "result-003",
					AlertID: "alert-002",
					Channel: "slack-prod",
					Status:  "success",
				},
			},
			TotalSent:   1,
			TotalFailed: 0,
			LastSent:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(originalHistory)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaledHistory notifications.NotificationHistory
		err = json.Unmarshal(jsonData, &unmarshaledHistory)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, originalHistory.AlertID, unmarshaledHistory.AlertID)
		assert.Equal(t, originalHistory.RuleID, unmarshaledHistory.RuleID)
		assert.Len(t, unmarshaledHistory.Results, 1)
		assert.Equal(t, originalHistory.TotalSent, unmarshaledHistory.TotalSent)
		assert.Equal(t, originalHistory.TotalFailed, unmarshaledHistory.TotalFailed)
		assert.True(t, originalHistory.LastSent.Equal(unmarshaledHistory.LastSent))
	})
}

func TestNotificationStats(t *testing.T) {
	t.Run("creates notification stats", func(t *testing.T) {
		stats := notifications.NotificationStats{
			TotalNotifications: 1250,
			SuccessfulSent:     1200,
			FailedSent:         50,
			SuccessRate:        96.0,
			ByChannel: map[string]int{
				"slack":   1000,
				"email":   200,
				"webhook": 50,
			},
			BySeverity: map[string]int{
				"critical": 100,
				"high":     300,
				"medium":   500,
				"low":      350,
			},
			LastNotification:    time.Now(),
			AverageResponseTime: 250 * time.Millisecond,
		}

		assert.Equal(t, 1250, stats.TotalNotifications)
		assert.Equal(t, 1200, stats.SuccessfulSent)
		assert.Equal(t, 50, stats.FailedSent)
		assert.Equal(t, 96.0, stats.SuccessRate)
		assert.Equal(t, 1000, stats.ByChannel["slack"])
		assert.Equal(t, 100, stats.BySeverity["critical"])
		assert.Equal(t, 250*time.Millisecond, stats.AverageResponseTime)
	})

	t.Run("marshals and unmarshals notification stats", func(t *testing.T) {
		originalStats := notifications.NotificationStats{
			TotalNotifications: 500,
			SuccessfulSent:     450,
			FailedSent:         50,
			SuccessRate:        90.0,
			ByChannel: map[string]int{
				"slack": 400,
				"email": 100,
			},
			BySeverity: map[string]int{
				"critical": 50,
				"high":     150,
				"medium":   200,
				"low":      100,
			},
			LastNotification:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			AverageResponseTime: 300 * time.Millisecond,
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(originalStats)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaledStats notifications.NotificationStats
		err = json.Unmarshal(jsonData, &unmarshaledStats)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, originalStats.TotalNotifications, unmarshaledStats.TotalNotifications)
		assert.Equal(t, originalStats.SuccessfulSent, unmarshaledStats.SuccessfulSent)
		assert.Equal(t, originalStats.FailedSent, unmarshaledStats.FailedSent)
		assert.Equal(t, originalStats.SuccessRate, unmarshaledStats.SuccessRate)
		assert.Equal(t, originalStats.ByChannel["slack"], unmarshaledStats.ByChannel["slack"])
		assert.Equal(t, originalStats.BySeverity["critical"], unmarshaledStats.BySeverity["critical"])
		assert.True(t, originalStats.LastNotification.Equal(unmarshaledStats.LastNotification))
		assert.Equal(t, originalStats.AverageResponseTime, unmarshaledStats.AverageResponseTime)
	})
}

func TestNotificationConfig(t *testing.T) {
	t.Run("creates default notification config", func(t *testing.T) {
		config := notifications.DefaultNotificationConfig()

		assert.True(t, config.Enabled)
		assert.Equal(t, 3, config.MaxRetries)
		assert.Equal(t, 5*time.Second, config.RetryDelay)
		assert.Equal(t, 30*time.Second, config.Timeout)
		assert.Equal(t, 60, config.RateLimitPerMin)
		assert.Equal(t, 10, config.BatchSize)
		assert.Equal(t, 1*time.Second, config.BatchDelay)
		assert.True(t, config.EnableDeduplication)
		assert.Equal(t, 5*time.Minute, config.DeduplicationWindow)
	})

	t.Run("creates custom notification config", func(t *testing.T) {
		config := notifications.NotificationConfig{
			Enabled:             false,
			MaxRetries:          5,
			RetryDelay:          10 * time.Second,
			Timeout:             60 * time.Second,
			RateLimitPerMin:     30,
			BatchSize:           20,
			BatchDelay:          2 * time.Second,
			EnableDeduplication: false,
			DeduplicationWindow: 10 * time.Minute,
		}

		assert.False(t, config.Enabled)
		assert.Equal(t, 5, config.MaxRetries)
		assert.Equal(t, 10*time.Second, config.RetryDelay)
		assert.Equal(t, 60*time.Second, config.Timeout)
		assert.Equal(t, 30, config.RateLimitPerMin)
		assert.Equal(t, 20, config.BatchSize)
		assert.Equal(t, 2*time.Second, config.BatchDelay)
		assert.False(t, config.EnableDeduplication)
		assert.Equal(t, 10*time.Minute, config.DeduplicationWindow)
	})

	t.Run("marshals and unmarshals notification config", func(t *testing.T) {
		originalConfig := notifications.NotificationConfig{
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

		// Marshal to JSON
		jsonData, err := json.Marshal(originalConfig)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaledConfig notifications.NotificationConfig
		err = json.Unmarshal(jsonData, &unmarshaledConfig)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, originalConfig.Enabled, unmarshaledConfig.Enabled)
		assert.Equal(t, originalConfig.MaxRetries, unmarshaledConfig.MaxRetries)
		assert.Equal(t, originalConfig.RetryDelay, unmarshaledConfig.RetryDelay)
		assert.Equal(t, originalConfig.Timeout, unmarshaledConfig.Timeout)
		assert.Equal(t, originalConfig.RateLimitPerMin, unmarshaledConfig.RateLimitPerMin)
		assert.Equal(t, originalConfig.BatchSize, unmarshaledConfig.BatchSize)
		assert.Equal(t, originalConfig.BatchDelay, unmarshaledConfig.BatchDelay)
		assert.Equal(t, originalConfig.EnableDeduplication, unmarshaledConfig.EnableDeduplication)
		assert.Equal(t, originalConfig.DeduplicationWindow, unmarshaledConfig.DeduplicationWindow)
	})
}

func TestNotificationFilter(t *testing.T) {
	t.Run("creates notification filter", func(t *testing.T) {
		filter := notifications.NotificationFilter{
			Severity:  []string{"critical", "high"},
			Namespace: []string{"production", "staging"},
			Service:   []string{"api-server", "web-service"},
			TimeRange: &notifications.TimeRange{
				Start: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			},
			Keywords:        []string{"error", "timeout"},
			ExcludeKeywords: []string{"test", "debug"},
		}

		assert.Contains(t, filter.Severity, "critical")
		assert.Contains(t, filter.Namespace, "production")
		assert.Contains(t, filter.Service, "api-server")
		assert.NotNil(t, filter.TimeRange)
		assert.Contains(t, filter.Keywords, "error")
		assert.Contains(t, filter.ExcludeKeywords, "test")
	})

	t.Run("marshals and unmarshals notification filter", func(t *testing.T) {
		originalFilter := notifications.NotificationFilter{
			Severity:  []string{"medium", "low"},
			Namespace: []string{"development"},
			Service:   []string{"test-service"},
			TimeRange: &notifications.TimeRange{
				Start: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			Keywords:        []string{"warning"},
			ExcludeKeywords: []string{"ignore"},
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(originalFilter)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaledFilter notifications.NotificationFilter
		err = json.Unmarshal(jsonData, &unmarshaledFilter)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, originalFilter.Severity, unmarshaledFilter.Severity)
		assert.Equal(t, originalFilter.Namespace, unmarshaledFilter.Namespace)
		assert.Equal(t, originalFilter.Service, unmarshaledFilter.Service)
		assert.NotNil(t, unmarshaledFilter.TimeRange)
		assert.True(t, originalFilter.TimeRange.Start.Equal(unmarshaledFilter.TimeRange.Start))
		assert.True(t, originalFilter.TimeRange.End.Equal(unmarshaledFilter.TimeRange.End))
		assert.Equal(t, originalFilter.Keywords, unmarshaledFilter.Keywords)
		assert.Equal(t, originalFilter.ExcludeKeywords, unmarshaledFilter.ExcludeKeywords)
	})
}

func TestQueueItem(t *testing.T) {
	t.Run("creates queue item", func(t *testing.T) {
		alert := models.Alert{
			ID:       "alert-001",
			RuleName: "Test Alert",
			Severity: "critical",
		}

		item := notifications.QueueItem{
			Alert:     alert,
			ChannelID: "slack-prod",
			Timestamp: time.Now(),
			Retries:   0,
		}

		assert.Equal(t, "alert-001", item.Alert.ID)
		assert.Equal(t, "slack-prod", item.ChannelID)
		assert.Equal(t, 0, item.Retries)
	})

	t.Run("marshals and unmarshals queue item", func(t *testing.T) {
		originalAlert := models.Alert{
			ID:       "alert-002",
			RuleName: "Queue Test Alert",
			Severity: "high",
		}

		originalItem := notifications.QueueItem{
			Alert:     originalAlert,
			ChannelID: "email-dev",
			Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Retries:   2,
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(originalItem)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaledItem notifications.QueueItem
		err = json.Unmarshal(jsonData, &unmarshaledItem)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, originalItem.Alert.ID, unmarshaledItem.Alert.ID)
		assert.Equal(t, originalItem.Alert.RuleName, unmarshaledItem.Alert.RuleName)
		assert.Equal(t, originalItem.Alert.Severity, unmarshaledItem.Alert.Severity)
		assert.Equal(t, originalItem.ChannelID, unmarshaledItem.ChannelID)
		assert.True(t, originalItem.Timestamp.Equal(unmarshaledItem.Timestamp))
		assert.Equal(t, originalItem.Retries, unmarshaledItem.Retries)
	})
}

func TestTimeRange(t *testing.T) {
	t.Run("creates time range", func(t *testing.T) {
		timeRange := notifications.TimeRange{
			Start: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
		}

		assert.True(t, timeRange.Start.Before(timeRange.End))
		assert.Equal(t, time.Hour, timeRange.End.Sub(timeRange.Start))
	})

	t.Run("marshals and unmarshals time range", func(t *testing.T) {
		originalTimeRange := notifications.TimeRange{
			Start: time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
			End:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(originalTimeRange)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaledTimeRange notifications.TimeRange
		err = json.Unmarshal(jsonData, &unmarshaledTimeRange)
		require.NoError(t, err)

		// Verify
		assert.True(t, originalTimeRange.Start.Equal(unmarshaledTimeRange.Start))
		assert.True(t, originalTimeRange.End.Equal(unmarshaledTimeRange.End))
	})
}

// Test that SlackNotifier implements the Notifier interface
func TestSlackNotifier_ImplementsNotifierInterface(t *testing.T) {
	t.Run("slack notifier implements notifier interface", func(t *testing.T) {
		webhookURL := "https://hooks.slack.com/services/T123456/B123456/XXXXXXXXXXXXXXXXXXXXXXXX"
		notifier := notifications.NewSlackNotifier(webhookURL)

		// Verify that SlackNotifier implements the Notifier interface
		var _ notifications.Notifier = notifier

		// Test interface methods
		assert.Equal(t, "slack", notifier.GetName())
		assert.True(t, notifier.IsEnabled())

		notifier.SetEnabled(false)
		assert.False(t, notifier.IsEnabled())

		notifier.SetEnabled(true)
		assert.True(t, notifier.IsEnabled())
	})
}
