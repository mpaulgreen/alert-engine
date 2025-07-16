package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/log-monitoring/alert-engine/internal/alerting"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

// StateStore interface for data operations
type StateStore interface {
	SaveAlertRule(rule models.AlertRule) error
	GetAlertRules() ([]models.AlertRule, error)
	GetAlertRule(id string) (*models.AlertRule, error)
	DeleteAlertRule(id string) error
	IncrementCounter(ruleID string, window time.Duration) (int64, error)
	GetCounter(ruleID string, window time.Duration) (int64, error)
	SetAlertStatus(ruleID string, status models.AlertStatus) error
	GetAlertStatus(ruleID string) (*models.AlertStatus, error)
	SaveAlert(alert models.Alert) error
	GetRecentAlerts(limit int) ([]models.Alert, error)
	GetLogStats() (*models.LogStats, error)
	GetHealthStatus() (bool, error)
	GetMetrics() (map[string]interface{}, error)
}

// AlertEngine interface for alert operations
type AlertEngine interface {
	AddRule(rule models.AlertRule) error
	UpdateRule(rule models.AlertRule) error
	DeleteRule(ruleID string) error
	GetRules() []models.AlertRule
	GetRule(ruleID string) (*models.AlertRule, error)
	ReloadRules() error
}

// Handlers contains the HTTP handlers
type Handlers struct {
	store       StateStore
	alertEngine AlertEngine
}

// NewHandlers creates a new handlers instance
func NewHandlers(store StateStore, alertEngine AlertEngine) *Handlers {
	return &Handlers{
		store:       store,
		alertEngine: alertEngine,
	}
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Health check endpoint
func (h *Handlers) Health(c *gin.Context) {
	healthy, err := h.store.GetHealthStatus()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    status,
			"timestamp": time.Now(),
		},
	})
}

// GetRules returns all alert rules
func (h *Handlers) GetRules(c *gin.Context) {
	rules := h.alertEngine.GetRules()

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    rules,
	})
}

// GetRule returns a specific alert rule
func (h *Handlers) GetRule(c *gin.Context) {
	ruleID := c.Param("id")

	rule, err := h.alertEngine.GetRule(ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    rule,
	})
}

// CreateRule creates a new alert rule
func (h *Handlers) CreateRule(c *gin.Context) {
	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON: " + err.Error(),
		})
		return
	}

	// Validate rule
	if err := alerting.ValidateRule(rule); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Validation failed: " + err.Error(),
		})
		return
	}

	// Generate ID if not provided
	if rule.ID == "" {
		rule.ID = alerting.GenerateRuleID(rule.Name)
	}

	// Add rule to engine
	if err := h.alertEngine.AddRule(rule); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to create rule: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: "Rule created successfully",
		Data:    rule,
	})
}

// UpdateRule updates an existing alert rule
func (h *Handlers) UpdateRule(c *gin.Context) {
	ruleID := c.Param("id")

	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON: " + err.Error(),
		})
		return
	}

	// Ensure ID matches
	rule.ID = ruleID

	// Validate rule
	if err := alerting.ValidateRule(rule); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Validation failed: " + err.Error(),
		})
		return
	}

	// Update rule in engine
	if err := h.alertEngine.UpdateRule(rule); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to update rule: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Rule updated successfully",
		Data:    rule,
	})
}

// DeleteRule deletes an alert rule
func (h *Handlers) DeleteRule(c *gin.Context) {
	ruleID := c.Param("id")

	if err := h.alertEngine.DeleteRule(ruleID); err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Rule deleted successfully",
	})
}

// GetRuleStats returns statistics about alert rules
func (h *Handlers) GetRuleStats(c *gin.Context) {
	rules := h.alertEngine.GetRules()
	stats := alerting.GetRuleStats(rules)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// GetRecentAlerts returns recent alerts
func (h *Handlers) GetRecentAlerts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	alerts, err := h.store.GetRecentAlerts(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    alerts,
	})
}

// GetLogStats returns log processing statistics
func (h *Handlers) GetLogStats(c *gin.Context) {
	stats, err := h.store.GetLogStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// GetMetrics returns system metrics
func (h *Handlers) GetMetrics(c *gin.Context) {
	metrics, err := h.store.GetMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    metrics,
	})
}

// TestRule tests an alert rule against sample data
func (h *Handlers) TestRule(c *gin.Context) {
	var request struct {
		Rule       models.AlertRule  `json:"rule"`
		SampleLogs []models.LogEntry `json:"sample_logs"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON: " + err.Error(),
		})
		return
	}

	// Validate rule
	if err := alerting.ValidateRule(request.Rule); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Validation failed: " + err.Error(),
		})
		return
	}

	// Test rule against sample logs
	evaluator := alerting.NewEvaluator(h.store)
	result, err := evaluator.TestRule(request.Rule, request.SampleLogs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

// GetRuleTemplate returns a template for creating new rules
func (h *Handlers) GetRuleTemplate(c *gin.Context) {
	template := alerting.GetRuleTemplate()

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    template,
	})
}

// GetDefaultRules returns default alert rules
func (h *Handlers) GetDefaultRules(c *gin.Context) {
	defaultRules := alerting.CreateDefaultRules()

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    defaultRules,
	})
}

// BulkCreateRules creates multiple alert rules
func (h *Handlers) BulkCreateRules(c *gin.Context) {
	var rules []models.AlertRule
	if err := c.ShouldBindJSON(&rules); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON: " + err.Error(),
		})
		return
	}

	var created []models.AlertRule
	var errors []string

	for _, rule := range rules {
		// Validate rule
		if err := alerting.ValidateRule(rule); err != nil {
			errors = append(errors, fmt.Sprintf("Rule %s: %s", rule.ID, err.Error()))
			continue
		}

		// Generate ID if not provided
		if rule.ID == "" {
			rule.ID = alerting.GenerateRuleID(rule.Name)
		}

		// Add rule to engine
		if err := h.alertEngine.AddRule(rule); err != nil {
			errors = append(errors, fmt.Sprintf("Rule %s: %s", rule.ID, err.Error()))
			continue
		}

		created = append(created, rule)
	}

	response := map[string]interface{}{
		"created": created,
		"errors":  errors,
	}

	statusCode := http.StatusCreated
	if len(errors) > 0 {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, APIResponse{
		Success: len(created) > 0,
		Message: fmt.Sprintf("Created %d rules, %d errors", len(created), len(errors)),
		Data:    response,
	})
}

// ReloadRules reloads all alert rules
func (h *Handlers) ReloadRules(c *gin.Context) {
	if err := h.alertEngine.ReloadRules(); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Rules reloaded successfully",
	})
}

// FilterRules filters alert rules based on criteria
func (h *Handlers) FilterRules(c *gin.Context) {
	var filter alerting.RuleFilter
	if err := c.ShouldBindJSON(&filter); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON: " + err.Error(),
		})
		return
	}

	rules := h.alertEngine.GetRules()
	filteredRules := alerting.FilterRules(rules, filter)

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    filteredRules,
	})
}
