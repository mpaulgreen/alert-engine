//go:build integration
// +build integration

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/stretchr/testify/suite"
)

// APIIntegrationTestSuite is the integration test suite for the API package
type APIIntegrationTestSuite struct {
	suite.Suite
	server     *httptest.Server
	client     *http.Client
	router     *gin.Engine
	mockStore  *mocks.MockStateStore
	mockEngine *mocks.MockAlertEngine
}

// SetupSuite sets up the test suite
func (suite *APIIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	suite.router = gin.New()
	suite.mockStore = mocks.NewMockStateStore()
	suite.mockEngine = mocks.NewMockAlertEngine()

	handlers := api.NewHandlers(suite.mockStore, suite.mockEngine)
	handlers.SetupRoutes(suite.router)

	suite.server = httptest.NewServer(suite.router)
	suite.client = &http.Client{
		Timeout: 30 * time.Second,
	}
}

// TearDownSuite tears down the test suite
func (suite *APIIntegrationTestSuite) TearDownSuite() {
	suite.server.Close()
}

// SetupTest sets up each test
func (suite *APIIntegrationTestSuite) SetupTest() {
	suite.mockStore.ClearRules()
	suite.mockStore.ClearAlerts()
	suite.mockEngine.ClearRules()
	suite.mockStore.SetHealthy(true)
	suite.mockStore.SetShouldFailHealth(false)
	suite.mockStore.SetShouldFailOps(false)
	suite.mockEngine.SetShouldFail(false)
}

// TestAPIIntegrationSuite runs the integration test suite
func TestAPIIntegrationSuite(t *testing.T) {
	suite.Run(t, new(APIIntegrationTestSuite))
}

// Helper method to make HTTP requests
func (suite *APIIntegrationTestSuite) makeRequest(method, path string, body interface{}) (*http.Response, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", suite.server.URL, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := suite.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}

	return resp, respBody, nil
}

// Test Health endpoint
func (suite *APIIntegrationTestSuite) TestHealthEndpoint() {
	suite.T().Run("health check success", func(t *testing.T) {
		resp, body, err := suite.makeRequest("GET", "/api/v1/health", nil)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.APIResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Contains(t, response.Data, "status")
		assert.Contains(t, response.Data, "timestamp")
	})

	suite.T().Run("health check failure", func(t *testing.T) {
		suite.mockStore.SetShouldFailHealth(true)

		resp, body, err := suite.makeRequest("GET", "/api/v1/health", nil)
		require.NoError(t, err)

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

		var response api.APIResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)
	})
}

// Test Rules CRUD operations
func (suite *APIIntegrationTestSuite) TestRulesCRUD() {
	suite.T().Run("complete CRUD workflow", func(t *testing.T) {
		// Create a rule
		rule := models.AlertRule{
			ID:          "integration-test-rule",
			Name:        "Integration Test Rule",
			Description: "Test rule for integration testing",
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
				Channel:      "#test-alerts",
				Severity:     "high",
			},
		}

		// 1. Create rule
		resp, body, err := suite.makeRequest("POST", "/api/v1/rules", rule)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createResponse api.APIResponse
		err = json.Unmarshal(body, &createResponse)
		require.NoError(t, err)
		assert.True(t, createResponse.Success)

		// Extract created rule ID
		createdRuleData, _ := json.Marshal(createResponse.Data)
		var createdRule models.AlertRule
		json.Unmarshal(createdRuleData, &createdRule)
		ruleID := createdRule.ID

		// 2. Get all rules
		resp, body, err = suite.makeRequest("GET", "/api/v1/rules", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var getAllResponse api.APIResponse
		err = json.Unmarshal(body, &getAllResponse)
		require.NoError(t, err)
		assert.True(t, getAllResponse.Success)

		// 3. Get specific rule
		resp, body, err = suite.makeRequest("GET", fmt.Sprintf("/api/v1/rules/%s", ruleID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var getResponse api.APIResponse
		err = json.Unmarshal(body, &getResponse)
		require.NoError(t, err)
		assert.True(t, getResponse.Success)

		// 4. Update rule
		rule.Name = "Updated Integration Test Rule"
		rule.Description = "Updated description"
		rule.Enabled = false

		resp, body, err = suite.makeRequest("PUT", fmt.Sprintf("/api/v1/rules/%s", ruleID), rule)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updateResponse api.APIResponse
		err = json.Unmarshal(body, &updateResponse)
		require.NoError(t, err)
		assert.True(t, updateResponse.Success)

		// 5. Delete rule
		resp, body, err = suite.makeRequest("DELETE", fmt.Sprintf("/api/v1/rules/%s", ruleID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var deleteResponse api.APIResponse
		err = json.Unmarshal(body, &deleteResponse)
		require.NoError(t, err)
		assert.True(t, deleteResponse.Success)

		// 6. Verify rule is deleted
		resp, body, err = suite.makeRequest("GET", fmt.Sprintf("/api/v1/rules/%s", ruleID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// Test Alerts endpoint
func (suite *APIIntegrationTestSuite) TestAlertsEndpoint() {
	suite.T().Run("get recent alerts", func(t *testing.T) {
		// Add test alerts
		for i := 0; i < 5; i++ {
			alert := models.Alert{
				ID:        fmt.Sprintf("alert-%d", i+1),
				RuleID:    fmt.Sprintf("rule-%d", i+1),
				RuleName:  fmt.Sprintf("Test Rule %d", i+1),
				Severity:  "medium",
				Message:   fmt.Sprintf("Test alert message %d", i+1),
				Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			}
			suite.mockStore.AddAlert(alert)
		}

		resp, body, err := suite.makeRequest("GET", "/api/v1/alerts/recent", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.APIResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		assert.True(t, response.Success)

		// Verify alerts are returned
		alertsData, _ := json.Marshal(response.Data)
		var alerts []models.Alert
		json.Unmarshal(alertsData, &alerts)
		assert.Len(t, alerts, 5)
	})

	suite.T().Run("get recent alerts with limit", func(t *testing.T) {
		resp, body, err := suite.makeRequest("GET", "/api/v1/alerts/recent?limit=3", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.APIResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		assert.True(t, response.Success)

		// Verify limited alerts are returned
		alertsData, _ := json.Marshal(response.Data)
		var alerts []models.Alert
		json.Unmarshal(alertsData, &alerts)
		assert.Len(t, alerts, 3)
	})
}

// Test System endpoints
func (suite *APIIntegrationTestSuite) TestSystemEndpoints() {
	suite.T().Run("get system metrics", func(t *testing.T) {
		resp, body, err := suite.makeRequest("GET", "/api/v1/system/metrics", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.APIResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)
	})

	suite.T().Run("get log statistics", func(t *testing.T) {
		resp, body, err := suite.makeRequest("GET", "/api/v1/system/logs/stats", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.APIResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)
	})
}

// Test Error scenarios
func (suite *APIIntegrationTestSuite) TestErrorScenarios() {
	suite.T().Run("invalid JSON request", func(t *testing.T) {
		req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/rules", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := suite.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response api.APIResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "Invalid JSON")
	})

	suite.T().Run("not found endpoints", func(t *testing.T) {
		resp, _, err := suite.makeRequest("GET", "/api/v1/rules/nonexistent", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	suite.T().Run("method not allowed", func(t *testing.T) {
		resp, _, err := suite.makeRequest("PATCH", "/api/v1/rules/test-rule", nil)
		require.NoError(t, err)
		// Gin returns 404 for unmatched routes, which is acceptable behavior
		assert.True(t, resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound)
	})
}

// Test CORS headers
func (suite *APIIntegrationTestSuite) TestCORSHeaders() {
	suite.T().Run("CORS headers present", func(t *testing.T) {
		resp, _, err := suite.makeRequest("GET", "/api/v1/rules", nil)
		require.NoError(t, err)

		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", resp.Header.Get("Access-Control-Allow-Headers"))
	})

	suite.T().Run("OPTIONS preflight request", func(t *testing.T) {
		resp, _, err := suite.makeRequest("OPTIONS", "/api/v1/rules", nil)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	})
}

// Test API Documentation endpoints
func (suite *APIIntegrationTestSuite) TestDocumentationEndpoints() {
	suite.T().Run("root endpoint", func(t *testing.T) {
		resp, body, err := suite.makeRequest("GET", "/", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Contains(t, response, "message")
		assert.Contains(t, response, "version")
		assert.Contains(t, response, "endpoints")
	})

	suite.T().Run("docs endpoint", func(t *testing.T) {
		resp, body, err := suite.makeRequest("GET", "/docs", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Contains(t, response, "title")
		assert.Contains(t, response, "version")
		assert.Contains(t, response, "description")
		assert.Contains(t, response, "endpoints")
		assert.Contains(t, response, "models")
	})
}

// Test concurrent requests
func (suite *APIIntegrationTestSuite) TestConcurrentRequests() {
	suite.T().Run("concurrent rule creation", func(t *testing.T) {
		const numRequests = 10
		done := make(chan bool, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(index int) {
				rule := models.AlertRule{
					ID:          fmt.Sprintf("concurrent-test-rule-%d", index),
					Name:        fmt.Sprintf("Concurrent Test Rule %d", index),
					Description: fmt.Sprintf("Test rule %d for concurrent testing", index),
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
						Channel:      "#test-alerts",
						Severity:     "medium",
					},
				}

				resp, body, err := suite.makeRequest("POST", "/api/v1/rules", rule)
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

				var response api.APIResponse
				err = json.Unmarshal(body, &response)
				assert.NoError(suite.T(), err)
				assert.True(suite.T(), response.Success)

				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			<-done
		}

		// Verify all rules were created
		assert.Equal(t, numRequests, suite.mockEngine.GetRuleCount())
	})
}

// Test request timeout scenarios
func (suite *APIIntegrationTestSuite) TestRequestTimeouts() {
	suite.T().Run("request with timeout", func(t *testing.T) {
		// Create a client with very short timeout
		shortTimeoutClient := &http.Client{
			Timeout: 1 * time.Millisecond,
		}

		rule := models.AlertRule{
			ID:          "timeout-test-rule",
			Name:        "Timeout Test Rule",
			Description: "Test rule for timeout testing",
			Enabled:     true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Namespace:  "test",
				Service:    "test-service",
				Keywords:   []string{"timeout", "test"},
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
				Operator:   "gt",
			},
			Actions: models.AlertActions{
				SlackWebhook: "https://hooks.slack.com/services/test",
				Channel:      "#test-alerts",
				Severity:     "medium",
			},
		}

		jsonData, _ := json.Marshal(rule)
		req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/rules", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		// This should timeout or succeed very quickly
		_, err := shortTimeoutClient.Do(req)
		if err != nil {
			// If there's an error, it should be a timeout
			assert.Contains(t, err.Error(), "timeout")
		}
		// If no error, the request completed faster than the timeout, which is also valid
	})
}

// Test large payload handling
func (suite *APIIntegrationTestSuite) TestLargePayloads() {
	suite.T().Run("large rule description", func(t *testing.T) {
		// Create a rule with very large description
		largeDescription := make([]byte, 10000)
		for i := range largeDescription {
			largeDescription[i] = 'a'
		}

		rule := models.AlertRule{
			ID:          "large-payload-test-rule",
			Name:        "Large Payload Test Rule",
			Description: string(largeDescription),
			Enabled:     true,
			Conditions: models.AlertConditions{
				LogLevel:   "ERROR",
				Namespace:  "test",
				Service:    "test-service",
				Keywords:   []string{"large", "test"},
				Threshold:  5,
				TimeWindow: 5 * time.Minute,
				Operator:   "gt",
			},
			Actions: models.AlertActions{
				SlackWebhook: "https://hooks.slack.com/services/test",
				Channel:      "#test-alerts",
				Severity:     "medium",
			},
		}

		resp, body, err := suite.makeRequest("POST", "/api/v1/rules", rule)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response api.APIResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
	})
}

// Test rate limiting simulation
func (suite *APIIntegrationTestSuite) TestRateLimitingSimulation() {
	suite.T().Run("rapid consecutive requests", func(t *testing.T) {
		const numRequests = 100

		// Make rapid consecutive requests
		for i := 0; i < numRequests; i++ {
			resp, _, err := suite.makeRequest("GET", "/api/v1/health", nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		// All requests should succeed since we don't have rate limiting implemented
		// This test verifies the server can handle rapid requests without crashing
	})
}

// Benchmark tests
func (suite *APIIntegrationTestSuite) TestBenchmarkEndpoints() {
	suite.T().Run("benchmark health endpoint", func(t *testing.T) {
		startTime := time.Now()
		const numRequests = 1000

		for i := 0; i < numRequests; i++ {
			resp, _, err := suite.makeRequest("GET", "/api/v1/health", nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		duration := time.Since(startTime)
		requestsPerSecond := float64(numRequests) / duration.Seconds()

		// Should be able to handle at least 100 requests per second
		assert.Greater(t, requestsPerSecond, 100.0, "API should handle at least 100 requests per second")

		t.Logf("Processed %d requests in %v (%.2f requests/second)", numRequests, duration, requestsPerSecond)
	})
}

// Test cleanup and resource management
func (suite *APIIntegrationTestSuite) TestResourceManagement() {
	suite.T().Run("server resource cleanup", func(t *testing.T) {
		// Create many rules and then clean them up
		const numRules = 50
		var ruleIDs []string

		for i := 0; i < numRules; i++ {
			rule := models.AlertRule{
				ID:          fmt.Sprintf("cleanup-test-rule-%d", i),
				Name:        fmt.Sprintf("Cleanup Test Rule %d", i),
				Description: fmt.Sprintf("Test rule %d for cleanup testing", i),
				Enabled:     true,
				Conditions: models.AlertConditions{
					LogLevel:   "ERROR",
					Namespace:  "test",
					Service:    "test-service",
					Keywords:   []string{"cleanup", "test"},
					Threshold:  5,
					TimeWindow: 5 * time.Minute,
					Operator:   "gt",
				},
				Actions: models.AlertActions{
					SlackWebhook: "https://hooks.slack.com/services/test",
					Channel:      "#test-alerts",
					Severity:     "medium",
				},
			}

			resp, body, err := suite.makeRequest("POST", "/api/v1/rules", rule)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response api.APIResponse
			err = json.Unmarshal(body, &response)
			require.NoError(t, err)

			createdRuleData, _ := json.Marshal(response.Data)
			var createdRule models.AlertRule
			json.Unmarshal(createdRuleData, &createdRule)
			ruleIDs = append(ruleIDs, createdRule.ID)
		}

		// Clean up all rules
		for _, ruleID := range ruleIDs {
			resp, _, err := suite.makeRequest("DELETE", fmt.Sprintf("/api/v1/rules/%s", ruleID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		// Verify all rules are deleted
		assert.Equal(t, 0, suite.mockEngine.GetRuleCount())
	})
}
