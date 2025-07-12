//go:build unit
// +build unit

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/log-monitoring/alert-engine/internal/api"
	"github.com/log-monitoring/alert-engine/internal/api/tests/mocks"
	"github.com/log-monitoring/alert-engine/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() (*gin.Engine, *mocks.MockStateStore, *mocks.MockAlertEngine) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockStore := mocks.NewMockStateStore()
	mockEngine := mocks.NewMockAlertEngine()

	handlers := api.NewHandlers(mockStore, mockEngine)
	handlers.SetupRoutes(router)

	return router, mockStore, mockEngine
}

func TestHandlers_Health(t *testing.T) {
	router, mockStore, _ := setupTestRouter()

	t.Run("healthy system", func(t *testing.T) {
		mockStore.SetHealthy(true)

		req, _ := http.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Contains(t, response.Data, "status")
		assert.Contains(t, response.Data, "timestamp")
	})

	t.Run("unhealthy system", func(t *testing.T) {
		mockStore.SetShouldFailHealth(true)

		req, _ := http.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)
	})
}

func TestHandlers_GetRules(t *testing.T) {
	router, _, mockEngine := setupTestRouter()

	t.Run("get all rules success", func(t *testing.T) {
		// Populate mock engine with sample rules
		mockEngine.PopulateWithSampleRules(3)

		req, _ := http.NewRequest("GET", "/api/v1/rules", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.IsType(t, []interface{}{}, response.Data)

		// Convert to rules slice for validation
		rulesData, _ := json.Marshal(response.Data)
		var rules []models.AlertRule
		json.Unmarshal(rulesData, &rules)

		assert.Len(t, rules, 3)
		// Check that we have the expected rules (order may vary)
		ruleIDs := make([]string, len(rules))
		ruleNames := make([]string, len(rules))
		for i, rule := range rules {
			ruleIDs[i] = rule.ID
			ruleNames[i] = rule.Name
		}
		assert.Contains(t, ruleIDs, "rule-1")
		assert.Contains(t, ruleIDs, "rule-2")
		assert.Contains(t, ruleIDs, "rule-3")
		assert.Contains(t, ruleNames, "Test Rule 1")
		assert.Contains(t, ruleNames, "Test Rule 2")
		assert.Contains(t, ruleNames, "Test Rule 3")
	})

	t.Run("get rules empty", func(t *testing.T) {
		mockEngine.ClearRules()

		req, _ := http.NewRequest("GET", "/api/v1/rules", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.IsType(t, []interface{}{}, response.Data)

		// Convert to rules slice for validation
		rulesData, _ := json.Marshal(response.Data)
		var rules []models.AlertRule
		json.Unmarshal(rulesData, &rules)

		assert.Len(t, rules, 0)
	})
}

func TestHandlers_GetRule(t *testing.T) {
	router, _, mockEngine := setupTestRouter()

	t.Run("get rule success", func(t *testing.T) {
		// Add sample rule
		rule := mockEngine.CreateSampleRule("test-rule-1", "Test Rule")
		mockEngine.AddRuleDirect(rule)

		req, _ := http.NewRequest("GET", "/api/v1/rules/test-rule-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)

		// Convert to rule for validation
		ruleData, _ := json.Marshal(response.Data)
		var returnedRule models.AlertRule
		json.Unmarshal(ruleData, &returnedRule)

		assert.Equal(t, "test-rule-1", returnedRule.ID)
		assert.Equal(t, "Test Rule", returnedRule.Name)
	})

	t.Run("get rule not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/rules/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)
	})

	t.Run("get rule engine error", func(t *testing.T) {
		mockEngine.SetShouldFailOp("get", true)

		req, _ := http.NewRequest("GET", "/api/v1/rules/test-rule", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)

		// Reset mock
		mockEngine.SetShouldFailOp("get", false)
	})
}

func TestHandlers_CreateRule(t *testing.T) {
	router, _, mockEngine := setupTestRouter()

	t.Run("create rule success", func(t *testing.T) {
		rule := models.AlertRule{
			ID:          "test-rule-create",
			Name:        "Test Rule",
			Description: "Test rule description",
			Enabled:     true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Namespace:  "test",
				Service:    "test-service",
				Keywords:   []string{"error", "test"},
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
				Operator:   "gt",
			},
			Actions: models.AlertActions{
				SlackWebhook: "https://hooks.slack.com/services/test",
				Channel:      "#alerts",
				Severity:     "medium",
			},
		}

		jsonData, _ := json.Marshal(rule)
		req, _ := http.NewRequest("POST", "/api/v1/rules", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, "Rule created successfully", response.Message)
		assert.NotNil(t, response.Data)

		// Verify rule was added to engine
		assert.Equal(t, 1, mockEngine.GetRuleCount())
	})

	t.Run("create rule invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/rules", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "Invalid JSON")
	})

	t.Run("create rule engine error", func(t *testing.T) {
		mockEngine.SetShouldFailOp("add", true)

		rule := models.AlertRule{
			ID:          "test-rule-engine-error",
			Name:        "Test Rule",
			Description: "Test rule description",
			Enabled:     true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Namespace:  "test",
				Service:    "test-service",
				Keywords:   []string{"error", "test"},
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
				Operator:   "gt",
			},
			Actions: models.AlertActions{
				SlackWebhook: "https://hooks.slack.com/services/test",
				Channel:      "#alerts",
				Severity:     "medium",
			},
		}

		jsonData, _ := json.Marshal(rule)
		req, _ := http.NewRequest("POST", "/api/v1/rules", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "Failed to create rule")

		// Reset mock
		mockEngine.SetShouldFailOp("add", false)
	})
}

func TestHandlers_UpdateRule(t *testing.T) {
	router, _, mockEngine := setupTestRouter()

	t.Run("update rule success", func(t *testing.T) {
		// Add initial rule
		rule := mockEngine.CreateSampleRule("test-rule-1", "Original Rule")
		mockEngine.AddRuleDirect(rule)

		// Update rule data
		updatedRule := models.AlertRule{
			Name:        "Updated Rule",
			Description: "Updated description",
			Enabled:     false,
			Conditions: models.AlertConditions{
				LogLevel:   "WARN",
				Namespace:  "updated",
				Service:    "updated-service",
				Keywords:   []string{"updated"},
				Threshold:  10,
				TimeWindow: 10 * time.Minute,
				Operator:   "gte",
			},
			Actions: models.AlertActions{
				SlackWebhook: "https://hooks.slack.com/services/updated",
				Channel:      "#updated-alerts",
				Severity:     "high",
			},
		}

		jsonData, _ := json.Marshal(updatedRule)
		req, _ := http.NewRequest("PUT", "/api/v1/rules/test-rule-1", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, "Rule updated successfully", response.Message)
	})

	t.Run("update rule invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/rules/test-rule-1", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "Invalid JSON")
	})

	t.Run("update rule engine error", func(t *testing.T) {
		mockEngine.SetShouldFailOp("update", true)

		rule := models.AlertRule{
			Name:        "Updated Rule",
			Description: "Updated description",
			Enabled:     true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
				Operator:   "gt",
			},
			Actions: models.AlertActions{
				SlackWebhook: "https://hooks.slack.com/services/test",
				Channel:      "#alerts",
				Severity:     "medium",
			},
		}

		jsonData, _ := json.Marshal(rule)
		req, _ := http.NewRequest("PUT", "/api/v1/rules/test-rule-1", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "Failed to update rule")

		// Reset mock
		mockEngine.SetShouldFailOp("update", false)
	})
}

func TestHandlers_DeleteRule(t *testing.T) {
	router, _, mockEngine := setupTestRouter()

	t.Run("delete rule success", func(t *testing.T) {
		// Add rule to delete
		rule := mockEngine.CreateSampleRule("test-rule-1", "Test Rule")
		mockEngine.AddRuleDirect(rule)

		req, _ := http.NewRequest("DELETE", "/api/v1/rules/test-rule-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, "Rule deleted successfully", response.Message)

		// Verify rule was removed
		assert.Equal(t, 0, mockEngine.GetRuleCount())
	})

	t.Run("delete rule not found", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/rules/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)
	})

	t.Run("delete rule engine error", func(t *testing.T) {
		mockEngine.SetShouldFailOp("delete", true)

		req, _ := http.NewRequest("DELETE", "/api/v1/rules/test-rule", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)

		// Reset mock
		mockEngine.SetShouldFailOp("delete", false)
	})
}

func TestHandlers_GetRecentAlerts(t *testing.T) {
	router, mockStore, _ := setupTestRouter()

	t.Run("get recent alerts success", func(t *testing.T) {
		// Add sample alerts
		alert := models.Alert{
			ID:        "alert-1",
			RuleID:    "rule-1",
			RuleName:  "Test Rule",
			Severity:  "high",
			Message:   "Test alert message",
			Timestamp: time.Now(),
		}
		mockStore.AddAlert(alert)

		req, _ := http.NewRequest("GET", "/api/v1/alerts/recent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)
	})

	t.Run("get recent alerts with limit", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/alerts/recent?limit=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
	})

	t.Run("get recent alerts store error", func(t *testing.T) {
		mockStore.SetShouldFailOps(true)

		req, _ := http.NewRequest("GET", "/api/v1/alerts/recent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)

		// Reset mock
		mockStore.SetShouldFailOps(false)
	})
}

func TestHandlers_GetLogStats(t *testing.T) {
	router, mockStore, _ := setupTestRouter()

	t.Run("get log stats success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/system/logs/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)
	})

	t.Run("get log stats error", func(t *testing.T) {
		mockStore.SetShouldFailOps(true)

		req, _ := http.NewRequest("GET", "/api/v1/system/logs/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)

		// Reset mock
		mockStore.SetShouldFailOps(false)
	})
}

func TestHandlers_GetMetrics(t *testing.T) {
	router, mockStore, _ := setupTestRouter()

	t.Run("get metrics success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/system/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)
	})

	t.Run("get metrics error", func(t *testing.T) {
		mockStore.SetShouldFailOps(true)

		req, _ := http.NewRequest("GET", "/api/v1/system/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response api.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)

		// Reset mock
		mockStore.SetShouldFailOps(false)
	})
}

func TestHandlers_CORS(t *testing.T) {
	router, _, _ := setupTestRouter()

	t.Run("CORS headers present", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/rules", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/api/v1/rules", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}
