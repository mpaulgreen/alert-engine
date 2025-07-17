package api

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up all API routes
func (h *Handlers) SetupRoutes(router *gin.Engine) {
	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API version 1
	v1 := router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", h.Health)

		// Alert rules endpoints
		rules := v1.Group("/rules")
		{
			rules.GET("", h.GetRules)                 // GET /api/v1/rules
			rules.POST("", h.CreateRule)              // POST /api/v1/rules
			rules.GET("/stats", h.GetRuleStats)       // GET /api/v1/rules/stats
			rules.GET("/template", h.GetRuleTemplate) // GET /api/v1/rules/template
			rules.GET("/defaults", h.GetDefaultRules) // GET /api/v1/rules/defaults
			rules.POST("/bulk", h.BulkCreateRules)    // POST /api/v1/rules/bulk
			rules.POST("/reload", h.ReloadRules)      // POST /api/v1/rules/reload
			rules.POST("/filter", h.FilterRules)      // POST /api/v1/rules/filter
			rules.POST("/test", h.TestRule)           // POST /api/v1/rules/test

			rules.GET("/:id", h.GetRule)       // GET /api/v1/rules/{id}
			rules.PUT("/:id", h.UpdateRule)    // PUT /api/v1/rules/{id}
			rules.DELETE("/:id", h.DeleteRule) // DELETE /api/v1/rules/{id}
		}

		// Alerts endpoints
		alerts := v1.Group("/alerts")
		{
			alerts.GET("/recent", h.GetRecentAlerts) // GET /api/v1/alerts/recent
		}

		// System monitoring endpoints
		system := v1.Group("/system")
		{
			system.GET("/metrics", h.GetMetrics)     // GET /api/v1/system/metrics
			system.GET("/logs/stats", h.GetLogStats) // GET /api/v1/system/logs/stats
		}
	}

	// Root endpoints
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Alert Engine API",
			"version": "1.0.0",
			"endpoints": map[string]string{
				"health": "/api/v1/health",
				"rules":  "/api/v1/rules",
				"alerts": "/api/v1/alerts",
				"system": "/api/v1/system",
				"docs":   "/docs",
			},
		})
	})

	// API documentation endpoint
	router.GET("/docs", func(c *gin.Context) {
		docs := map[string]interface{}{
			"title":       "Alert Engine API Documentation",
			"version":     "1.0.0",
			"description": "REST API for managing alert rules and monitoring system status",
			"endpoints": map[string]interface{}{
				"GET /api/v1/health": map[string]string{
					"description": "Check system health status",
					"response":    "Health status and timestamp",
				},
				"GET /api/v1/rules": map[string]string{
					"description": "Get all alert rules",
					"response":    "Array of alert rules",
				},
				"POST /api/v1/rules": map[string]string{
					"description": "Create a new alert rule",
					"body":        "AlertRule JSON object",
					"response":    "Created alert rule",
				},
				"GET /api/v1/rules/{id}": map[string]string{
					"description": "Get specific alert rule by ID",
					"response":    "Alert rule object",
				},
				"PUT /api/v1/rules/{id}": map[string]string{
					"description": "Update existing alert rule",
					"body":        "AlertRule JSON object",
					"response":    "Updated alert rule",
				},
				"DELETE /api/v1/rules/{id}": map[string]string{
					"description": "Delete alert rule by ID",
					"response":    "Success message",
				},
				"GET /api/v1/rules/stats": map[string]string{
					"description": "Get alert rules statistics",
					"response":    "Statistics object",
				},
				"GET /api/v1/rules/template": map[string]string{
					"description": "Get alert rule template",
					"response":    "Template alert rule object",
				},
				"GET /api/v1/rules/defaults": map[string]string{
					"description": "Get default alert rules",
					"response":    "Array of default alert rules",
				},
				"POST /api/v1/rules/bulk": map[string]string{
					"description": "Create multiple alert rules",
					"body":        "Array of AlertRule objects",
					"response":    "Bulk creation results",
				},
				"POST /api/v1/rules/reload": map[string]string{
					"description": "Reload all alert rules from storage",
					"response":    "Success message",
				},
				"POST /api/v1/rules/filter": map[string]string{
					"description": "Filter alert rules by criteria",
					"body":        "RuleFilter object",
					"response":    "Filtered alert rules",
				},
				"POST /api/v1/rules/test": map[string]string{
					"description": "Test alert rule against sample logs",
					"body":        "Rule and sample logs",
					"response":    "Test results",
				},
				"GET /api/v1/alerts/recent": map[string]string{
					"description": "Get recent alerts",
					"query":       "limit (optional, default: 50)",
					"response":    "Array of recent alerts",
				},
				"GET /api/v1/system/metrics": map[string]string{
					"description": "Get system metrics",
					"response":    "System metrics object",
				},
				"GET /api/v1/system/logs/stats": map[string]string{
					"description": "Get log processing statistics",
					"response":    "Log statistics object",
				},
			},
			"models": map[string]interface{}{
				"AlertRule": map[string]string{
					"id":          "string - Unique identifier",
					"name":        "string - Rule name",
					"description": "string - Rule description",
					"enabled":     "boolean - Whether rule is enabled",
					"conditions":  "AlertConditions - Trigger conditions",
					"actions":     "AlertActions - Actions to take",
					"created_at":  "timestamp - Creation time",
					"updated_at":  "timestamp - Last update time",
				},
				"AlertConditions": map[string]string{
					"log_level":   "string - Log level to match (ERROR, WARN, INFO, DEBUG)",
					"namespace":   "string - Kubernetes namespace to match",
					"service":     "string - Service name to match",
					"keywords":    "array - Keywords to search for in log message",
					"threshold":   "integer - Number of matches required to trigger",
					"time_window": "duration - Time window for counting matches",
					"operator":    "string - Comparison operator (gt, gte, lt, lte, eq)",
				},
				"AlertActions": map[string]string{
					"slack_webhook": "string - Slack webhook URL",
					"channel":       "string - Slack channel",
					"severity":      "string - Alert severity (low, medium, high, critical)",
				},
			},
		}

		c.JSON(200, docs)
	})
}
