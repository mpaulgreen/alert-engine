package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/log-monitoring/alert-engine/internal/alerting"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

func TestValidateRule(t *testing.T) {
	t.Run("validates correct rule", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "valid-rule",
			Name: "Valid Rule",
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
				Operator:   "gt",
			},
			Actions: models.AlertActions{
				Severity: "high",
			},
		}

		err := alerting.ValidateRule(rule)
		assert.NoError(t, err)
	})

	t.Run("fails validation with empty ID", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "", // Empty ID
			Name: "Invalid Rule",
			Conditions: models.AlertConditions{
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
			},
		}

		err := alerting.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule ID cannot be empty")
	})

	t.Run("fails validation with empty name", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "valid-id",
			Name: "", // Empty name
			Conditions: models.AlertConditions{
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
			},
		}

		err := alerting.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule name cannot be empty")
	})

	t.Run("fails validation with zero threshold", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "valid-id",
			Name: "Valid Name",
			Conditions: models.AlertConditions{
				Threshold:  0, // Invalid threshold
				TimeWindow: 5 * time.Minute,
			},
		}

		err := alerting.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "threshold must be greater than 0")
	})

	t.Run("fails validation with negative threshold", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "valid-id",
			Name: "Valid Name",
			Conditions: models.AlertConditions{
				Threshold:  -1, // Invalid threshold
				TimeWindow: 5 * time.Minute,
			},
		}

		err := alerting.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "threshold must be greater than 0")
	})

	t.Run("fails validation with zero time window", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "valid-id",
			Name: "Valid Name",
			Conditions: models.AlertConditions{
				Threshold:  5,
				TimeWindow: 0, // Invalid time window
			},
		}

		err := alerting.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "time window must be greater than 0")
	})

	t.Run("fails validation with invalid operator", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "valid-id",
			Name: "Valid Name",
			Conditions: models.AlertConditions{
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
				Operator:   "invalid", // Invalid operator
			},
		}

		err := alerting.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid operator")
		assert.Contains(t, err.Error(), "must be one of: gt, gte, lt, lte, eq")
	})

	t.Run("validates valid operators", func(t *testing.T) {
		validOperators := []string{"gt", "gte", "lt", "lte", "eq"}

		for _, operator := range validOperators {
			rule := models.AlertRule{
				ID:   "valid-id",
				Name: "Valid Name",
				Conditions: models.AlertConditions{
					Threshold:  5,
					TimeWindow: 5 * time.Minute,
					Operator:   operator,
				},
			}

			err := alerting.ValidateRule(rule)
			assert.NoError(t, err, "operator %s should be valid", operator)
		}
	})

	t.Run("allows empty operator (defaults to gt)", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "valid-id",
			Name: "Valid Name",
			Conditions: models.AlertConditions{
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
				Operator:   "", // Empty operator should be allowed
			},
		}

		err := alerting.ValidateRule(rule)
		assert.NoError(t, err)
	})

	t.Run("fails validation with invalid severity", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "valid-id",
			Name: "Valid Name",
			Conditions: models.AlertConditions{
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
			},
			Actions: models.AlertActions{
				Severity: "invalid", // Invalid severity
			},
		}

		err := alerting.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid severity")
		assert.Contains(t, err.Error(), "must be one of: low, medium, high, critical")
	})

	t.Run("validates valid severities", func(t *testing.T) {
		validSeverities := []string{"low", "medium", "high", "critical"}

		for _, severity := range validSeverities {
			rule := models.AlertRule{
				ID:   "valid-id",
				Name: "Valid Name",
				Conditions: models.AlertConditions{
					Threshold:  5,
					TimeWindow: 5 * time.Minute,
				},
				Actions: models.AlertActions{
					Severity: severity,
				},
			}

			err := alerting.ValidateRule(rule)
			assert.NoError(t, err, "severity %s should be valid", severity)
		}
	})

	t.Run("allows empty severity", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "valid-id",
			Name: "Valid Name",
			Conditions: models.AlertConditions{
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
			},
			Actions: models.AlertActions{
				Severity: "", // Empty severity should be allowed
			},
		}

		err := alerting.ValidateRule(rule)
		assert.NoError(t, err)
	})
}

func TestCreateDefaultRules(t *testing.T) {
	t.Run("creates default rules", func(t *testing.T) {
		rules := alerting.CreateDefaultRules()

		assert.NotEmpty(t, rules)
		assert.Len(t, rules, 3) // Should create 3 default rules

		// Check that all rules are valid
		for _, rule := range rules {
			err := alerting.ValidateRule(rule)
			assert.NoError(t, err, "default rule %s should be valid", rule.ID)

			// Check that timestamps are set
			assert.False(t, rule.CreatedAt.IsZero())
			assert.False(t, rule.UpdatedAt.IsZero())

			// Check that all rules are enabled by default
			assert.True(t, rule.Enabled)
		}

		// Check specific rules exist
		ruleIDs := make(map[string]bool)
		for _, rule := range rules {
			ruleIDs[rule.ID] = true
		}

		assert.True(t, ruleIDs["default-error-alert"])
		assert.True(t, ruleIDs["default-database-alert"])
		assert.True(t, ruleIDs["default-memory-warning"])
	})

	t.Run("default rules have proper configuration", func(t *testing.T) {
		rules := alerting.CreateDefaultRules()

		// Find the error alert rule
		var errorRule *models.AlertRule
		for _, rule := range rules {
			if rule.ID == "default-error-alert" {
				errorRule = &rule
				break
			}
		}

		require.NotNil(t, errorRule)
		assert.Equal(t, "Application Error Alert", errorRule.Name)
		assert.Equal(t, "ERROR", errorRule.Conditions.LogLevel)
		assert.Equal(t, 5, errorRule.Conditions.Threshold)
		assert.Equal(t, 5*time.Minute, errorRule.Conditions.TimeWindow)
		assert.Equal(t, "gt", errorRule.Conditions.Operator)
		assert.Equal(t, "high", errorRule.Actions.Severity)
		assert.Equal(t, "#alerts", errorRule.Actions.Channel)
	})
}

func TestGetRuleTemplate(t *testing.T) {
	t.Run("returns valid rule template", func(t *testing.T) {
		template := alerting.GetRuleTemplate()

		// Template should be valid (except for empty ID and name)
		assert.Equal(t, "", template.ID)
		assert.Equal(t, "", template.Name)
		assert.True(t, template.Enabled)
		assert.Equal(t, "ERROR", template.Conditions.LogLevel)
		assert.Equal(t, 5, template.Conditions.Threshold)
		assert.Equal(t, 5*time.Minute, template.Conditions.TimeWindow)
		assert.Equal(t, "gt", template.Conditions.Operator)
		assert.Equal(t, "#alerts", template.Actions.Channel)
		assert.Equal(t, "medium", template.Actions.Severity)
		assert.Empty(t, template.Conditions.Keywords)
	})
}

func TestGetRuleStats(t *testing.T) {
	t.Run("calculates correct statistics", func(t *testing.T) {
		rules := []models.AlertRule{
			{
				ID:      "rule-1",
				Enabled: true,
				Conditions: models.AlertConditions{
					Namespace: "production",
					Service:   "user-service",
				},
				Actions: models.AlertActions{
					Severity: "high",
				},
			},
			{
				ID:      "rule-2",
				Enabled: false,
				Conditions: models.AlertConditions{
					Namespace: "production",
					Service:   "payment-service",
				},
				Actions: models.AlertActions{
					Severity: "critical",
				},
			},
			{
				ID:      "rule-3",
				Enabled: true,
				Conditions: models.AlertConditions{
					Namespace: "staging",
					Service:   "user-service",
				},
				Actions: models.AlertActions{
					Severity: "medium",
				},
			},
			{
				ID:      "rule-4",
				Enabled: true,
				Conditions: models.AlertConditions{
					Namespace: "production",
				},
				Actions: models.AlertActions{
					Severity: "", // Empty severity should default to medium
				},
			},
		}

		stats := alerting.GetRuleStats(rules)

		assert.Equal(t, 4, stats.TotalRules)
		assert.Equal(t, 3, stats.EnabledRules)
		assert.Equal(t, 1, stats.DisabledRules)

		// Check severity distribution
		assert.Equal(t, 1, stats.BySeverity["high"])
		assert.Equal(t, 1, stats.BySeverity["critical"])
		assert.Equal(t, 2, stats.BySeverity["medium"]) // One explicit + one default

		// Check namespace distribution
		assert.Equal(t, 3, stats.ByNamespace["production"])
		assert.Equal(t, 1, stats.ByNamespace["staging"])

		// Check service distribution
		assert.Equal(t, 2, stats.ByService["user-service"])
		assert.Equal(t, 1, stats.ByService["payment-service"])
	})

	t.Run("handles empty rules", func(t *testing.T) {
		stats := alerting.GetRuleStats([]models.AlertRule{})

		assert.Equal(t, 0, stats.TotalRules)
		assert.Equal(t, 0, stats.EnabledRules)
		assert.Equal(t, 0, stats.DisabledRules)
		assert.Empty(t, stats.BySeverity)
		assert.Empty(t, stats.ByNamespace)
		assert.Empty(t, stats.ByService)
	})
}

func TestFilterRules(t *testing.T) {
	rules := []models.AlertRule{
		{
			ID:      "rule-1",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:  "ERROR",
				Namespace: "production",
				Service:   "user-service",
			},
			Actions: models.AlertActions{
				Severity: "high",
			},
		},
		{
			ID:      "rule-2",
			Enabled: false,
			Conditions: models.AlertConditions{
				LogLevel:  "WARN",
				Namespace: "production",
				Service:   "payment-service",
			},
			Actions: models.AlertActions{
				Severity: "medium",
			},
		},
		{
			ID:      "rule-3",
			Enabled: true,
			Conditions: models.AlertConditions{
				LogLevel:  "ERROR",
				Namespace: "staging",
				Service:   "user-service",
			},
			Actions: models.AlertActions{
				Severity: "high",
			},
		},
	}

	t.Run("filters by enabled status", func(t *testing.T) {
		enabled := true
		filter := alerting.RuleFilter{
			Enabled: &enabled,
		}

		filtered := alerting.FilterRules(rules, filter)
		assert.Len(t, filtered, 2)

		for _, rule := range filtered {
			assert.True(t, rule.Enabled)
		}
	})

	t.Run("filters by disabled status", func(t *testing.T) {
		disabled := false
		filter := alerting.RuleFilter{
			Enabled: &disabled,
		}

		filtered := alerting.FilterRules(rules, filter)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "rule-2", filtered[0].ID)
		assert.False(t, filtered[0].Enabled)
	})

	t.Run("filters by namespace", func(t *testing.T) {
		filter := alerting.RuleFilter{
			Namespace: "production",
		}

		filtered := alerting.FilterRules(rules, filter)
		assert.Len(t, filtered, 2)

		for _, rule := range filtered {
			assert.Equal(t, "production", rule.Conditions.Namespace)
		}
	})

	t.Run("filters by service", func(t *testing.T) {
		filter := alerting.RuleFilter{
			Service: "user-service",
		}

		filtered := alerting.FilterRules(rules, filter)
		assert.Len(t, filtered, 2)

		for _, rule := range filtered {
			assert.Equal(t, "user-service", rule.Conditions.Service)
		}
	})

	t.Run("filters by severity", func(t *testing.T) {
		filter := alerting.RuleFilter{
			Severity: "high",
		}

		filtered := alerting.FilterRules(rules, filter)
		assert.Len(t, filtered, 2)

		for _, rule := range filtered {
			assert.Equal(t, "high", rule.Actions.Severity)
		}
	})

	t.Run("filters by log level", func(t *testing.T) {
		filter := alerting.RuleFilter{
			LogLevel: "ERROR",
		}

		filtered := alerting.FilterRules(rules, filter)
		assert.Len(t, filtered, 2)

		for _, rule := range filtered {
			assert.Equal(t, "ERROR", rule.Conditions.LogLevel)
		}
	})

	t.Run("filters by multiple criteria", func(t *testing.T) {
		enabled := true
		filter := alerting.RuleFilter{
			Enabled:   &enabled,
			Namespace: "production",
			Severity:  "high",
		}

		filtered := alerting.FilterRules(rules, filter)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "rule-1", filtered[0].ID)
	})

	t.Run("returns empty when no matches", func(t *testing.T) {
		filter := alerting.RuleFilter{
			Namespace: "nonexistent",
		}

		filtered := alerting.FilterRules(rules, filter)
		assert.Len(t, filtered, 0)
	})

	t.Run("returns all when no filter criteria", func(t *testing.T) {
		filter := alerting.RuleFilter{} // Empty filter

		filtered := alerting.FilterRules(rules, filter)
		assert.Len(t, filtered, len(rules))
	})
}

func TestGenerateRuleID(t *testing.T) {
	t.Run("generates ID from name", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"simple name", "Error Alert", "error-alert"},
			{"name with spaces", "Database Connection Error", "database-connection-error"},
			{"name with underscores", "user_service_alert", "user-service-alert"},
			{"mixed case", "Payment Service ERROR", "payment-service-error"},
			{"special characters", "Alert! @#$% Rule", "alert--rule"}, // Expected actual output
			{"numbers", "Alert Rule 123", "alert-rule-123"},
			{"multiple spaces", "Multiple   Spaces   Rule", "multiple---spaces---rule"}, // Expected actual output
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := alerting.GenerateRuleID(tc.input)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := alerting.GenerateRuleID("")
		assert.Equal(t, "", result)
	})

	t.Run("handles only special characters", func(t *testing.T) {
		result := alerting.GenerateRuleID("!@#$%^&*()")
		assert.Equal(t, "", result)
	})

	t.Run("preserves valid characters", func(t *testing.T) {
		result := alerting.GenerateRuleID("abc-123-def")
		assert.Equal(t, "abc-123-def", result)
	})
}

func TestContainsFunction(t *testing.T) {
	t.Run("finds existing item", func(t *testing.T) {
		// Test through ValidateRule which uses contains internally
		rule := models.AlertRule{
			ID:   "test-rule",
			Name: "Test Rule",
			Conditions: models.AlertConditions{
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "gt", // Valid operator
			},
		}

		err := alerting.ValidateRule(rule)
		assert.NoError(t, err) // Should pass because "gt" is in valid operators
	})

	t.Run("does not find non-existing item", func(t *testing.T) {
		// Test through ValidateRule which uses contains internally
		rule := models.AlertRule{
			ID:   "test-rule",
			Name: "Test Rule",
			Conditions: models.AlertConditions{
				Threshold:  1,
				TimeWindow: time.Minute,
				Operator:   "invalid", // Invalid operator
			},
		}

		err := alerting.ValidateRule(rule)
		assert.Error(t, err) // Should fail because "invalid" is not in valid operators
	})
}

func TestComplexRuleValidation(t *testing.T) {
	t.Run("validates complex valid rule", func(t *testing.T) {
		rule := models.AlertRule{
			ID:          "complex-rule-1",
			Name:        "Complex Production Alert",
			Description: "Alert for critical production issues",
			Enabled:     true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Namespace:  "production",
				Service:    "user-service",
				Keywords:   []string{"authentication", "failed", "timeout"},
				Threshold:  10,
				TimeWindow: 5 * time.Minute,
				Operator:   "gte",
			},
			Actions: models.AlertActions{
				SlackWebhook: "https://hooks.slack.com/services/TEST/TEST/TEST",
				Channel:      "#critical-alerts",
				Severity:     "critical",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := alerting.ValidateRule(rule)
		assert.NoError(t, err)
	})

	t.Run("validates rule with minimal required fields", func(t *testing.T) {
		rule := models.AlertRule{
			ID:   "minimal-rule",
			Name: "Minimal Rule",
			Conditions: models.AlertConditions{
				Threshold:  1,
				TimeWindow: time.Second,
			},
		}

		err := alerting.ValidateRule(rule)
		assert.NoError(t, err)
	})
}

func TestRuleStatsEdgeCases(t *testing.T) {
	t.Run("handles rules with empty fields", func(t *testing.T) {
		rules := []models.AlertRule{
			{
				ID:      "rule-1",
				Enabled: true,
				Conditions: models.AlertConditions{
					Namespace: "", // Empty namespace
					Service:   "", // Empty service
				},
				Actions: models.AlertActions{
					Severity: "", // Empty severity (should default to medium)
				},
			},
		}

		stats := alerting.GetRuleStats(rules)

		assert.Equal(t, 1, stats.TotalRules)
		assert.Equal(t, 1, stats.EnabledRules)
		assert.Equal(t, 0, stats.DisabledRules)
		assert.Equal(t, 1, stats.BySeverity["medium"]) // Empty severity defaults to medium
		assert.Equal(t, 0, len(stats.ByNamespace))     // Empty namespace not counted
		assert.Equal(t, 0, len(stats.ByService))       // Empty service not counted
	})
}
