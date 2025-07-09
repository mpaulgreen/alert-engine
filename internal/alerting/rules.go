package alerting

import (
    "fmt"
    "strings"
    "time"

    "github.com/log-monitoring/alert-engine/pkg/models"
)

// ValidateRule validates an alert rule for correctness
func ValidateRule(rule models.AlertRule) error {
    if rule.ID == "" {
        return fmt.Errorf("rule ID cannot be empty")
    }

    if rule.Name == "" {
        return fmt.Errorf("rule name cannot be empty")
    }

    if rule.Conditions.Threshold <= 0 {
        return fmt.Errorf("threshold must be greater than 0")
    }

    if rule.Conditions.TimeWindow <= 0 {
        return fmt.Errorf("time window must be greater than 0")
    }

    if rule.Conditions.Operator != "" {
        validOperators := []string{"gt", "gte", "lt", "lte", "eq"}
        if !contains(validOperators, rule.Conditions.Operator) {
            return fmt.Errorf("invalid operator: %s, must be one of: %s", 
                rule.Conditions.Operator, strings.Join(validOperators, ", "))
        }
    }

    if rule.Actions.Severity != "" {
        validSeverities := []string{"low", "medium", "high", "critical"}
        if !contains(validSeverities, rule.Actions.Severity) {
            return fmt.Errorf("invalid severity: %s, must be one of: %s", 
                rule.Actions.Severity, strings.Join(validSeverities, ", "))
        }
    }

    return nil
}

// CreateDefaultRules creates a set of default alert rules
func CreateDefaultRules() []models.AlertRule {
    return []models.AlertRule{
        {
            ID:          "default-error-alert",
            Name:        "Application Error Alert",
            Description: "Alert on application errors",
            Enabled:     true,
            Conditions: models.AlertConditions{
                LogLevel:   "ERROR",
                Threshold:  5,
                TimeWindow: 5 * time.Minute,
                Operator:   "gt",
            },
            Actions: models.AlertActions{
                Channel:  "#alerts",
                Severity: "high",
            },
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        },
        {
            ID:          "default-database-alert",
            Name:        "Database Connection Issues",
            Description: "Alert on database connection problems",
            Enabled:     true,
            Conditions: models.AlertConditions{
                LogLevel:   "ERROR",
                Keywords:   []string{"database", "connection", "failed"},
                Threshold:  3,
                TimeWindow: 2 * time.Minute,
                Operator:   "gt",
            },
            Actions: models.AlertActions{
                Channel:  "#infrastructure",
                Severity: "critical",
            },
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        },
        {
            ID:          "default-memory-warning",
            Name:        "High Memory Usage Warning",
            Description: "Alert on high memory usage warnings",
            Enabled:     true,
            Conditions: models.AlertConditions{
                LogLevel:   "WARN",
                Keywords:   []string{"memory", "usage", "high"},
                Threshold:  10,
                TimeWindow: 10 * time.Minute,
                Operator:   "gt",
            },
            Actions: models.AlertActions{
                Channel:  "#monitoring",
                Severity: "medium",
            },
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        },
    }
}

// GetRuleTemplate returns a template for creating new alert rules
func GetRuleTemplate() models.AlertRule {
    return models.AlertRule{
        ID:          "",
        Name:        "",
        Description: "",
        Enabled:     true,
        Conditions: models.AlertConditions{
            LogLevel:   "ERROR",
            Namespace:  "",
            Service:    "",
            Keywords:   []string{},
            Threshold:  5,
            TimeWindow: 5 * time.Minute,
            Operator:   "gt",
        },
        Actions: models.AlertActions{
            Channel:  "#alerts",
            Severity: "medium",
        },
    }
}

// RuleStats represents statistics about alert rules
type RuleStats struct {
    TotalRules    int                    `json:"total_rules"`
    EnabledRules  int                    `json:"enabled_rules"`
    DisabledRules int                    `json:"disabled_rules"`
    BySeverity    map[string]int         `json:"by_severity"`
    ByNamespace   map[string]int         `json:"by_namespace"`
    ByService     map[string]int         `json:"by_service"`
}

// GetRuleStats calculates statistics for a set of alert rules
func GetRuleStats(rules []models.AlertRule) RuleStats {
    stats := RuleStats{
        TotalRules:    len(rules),
        EnabledRules:  0,
        DisabledRules: 0,
        BySeverity:    make(map[string]int),
        ByNamespace:   make(map[string]int),
        ByService:     make(map[string]int),
    }

    for _, rule := range rules {
        if rule.Enabled {
            stats.EnabledRules++
        } else {
            stats.DisabledRules++
        }

        // Count by severity
        severity := rule.Actions.Severity
        if severity == "" {
            severity = "medium"
        }
        stats.BySeverity[severity]++

        // Count by namespace
        if rule.Conditions.Namespace != "" {
            stats.ByNamespace[rule.Conditions.Namespace]++
        }

        // Count by service
        if rule.Conditions.Service != "" {
            stats.ByService[rule.Conditions.Service]++
        }
    }

    return stats
}

// FilterRules filters alert rules based on criteria
func FilterRules(rules []models.AlertRule, filter RuleFilter) []models.AlertRule {
    var filtered []models.AlertRule

    for _, rule := range rules {
        if matchesFilter(rule, filter) {
            filtered = append(filtered, rule)
        }
    }

    return filtered
}

// RuleFilter represents filtering criteria for alert rules
type RuleFilter struct {
    Enabled   *bool  `json:"enabled,omitempty"`
    Namespace string `json:"namespace,omitempty"`
    Service   string `json:"service,omitempty"`
    Severity  string `json:"severity,omitempty"`
    LogLevel  string `json:"log_level,omitempty"`
}

// matchesFilter checks if a rule matches the filter criteria
func matchesFilter(rule models.AlertRule, filter RuleFilter) bool {
    if filter.Enabled != nil && rule.Enabled != *filter.Enabled {
        return false
    }

    if filter.Namespace != "" && rule.Conditions.Namespace != filter.Namespace {
        return false
    }

    if filter.Service != "" && rule.Conditions.Service != filter.Service {
        return false
    }

    if filter.Severity != "" && rule.Actions.Severity != filter.Severity {
        return false
    }

    if filter.LogLevel != "" && rule.Conditions.LogLevel != filter.LogLevel {
        return false
    }

    return true
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

// GenerateRuleID generates a unique rule ID based on the rule name
func GenerateRuleID(name string) string {
    // Convert to lowercase and replace spaces with hyphens
    id := strings.ToLower(name)
    id = strings.ReplaceAll(id, " ", "-")
    id = strings.ReplaceAll(id, "_", "-")
    
    // Remove invalid characters
    var result strings.Builder
    for _, char := range id {
        if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
            result.WriteRune(char)
        }
    }
    
    return result.String()
} 