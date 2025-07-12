package mocks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MockSlackServer provides a mock Slack webhook server for integration testing
type MockSlackServer struct {
	server        *httptest.Server
	requests      []*http.Request
	requestBodies []string
	responses     map[string]MockResponse
	defaultStatus int
	defaultBody   string
	mu            sync.RWMutex
	callCount     int
}

// MockResponse represents a mock response configuration
type MockResponse struct {
	StatusCode int               `json:"status_code"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers"`
	Delay      time.Duration     `json:"delay"`
}

// NewMockSlackServer creates a new mock Slack server
func NewMockSlackServer() *MockSlackServer {
	server := &MockSlackServer{
		requests:      make([]*http.Request, 0),
		requestBodies: make([]string, 0),
		responses:     make(map[string]MockResponse),
		defaultStatus: http.StatusOK,
		defaultBody:   "ok",
	}

	// Create HTTP server with handler
	server.server = httptest.NewServer(http.HandlerFunc(server.handleRequest))

	return server
}

// NewMockSlackServerTLS creates a new mock Slack server with TLS
func NewMockSlackServerTLS() *MockSlackServer {
	server := &MockSlackServer{
		requests:      make([]*http.Request, 0),
		requestBodies: make([]string, 0),
		responses:     make(map[string]MockResponse),
		defaultStatus: http.StatusOK,
		defaultBody:   "ok",
	}

	// Create HTTPS server with handler
	server.server = httptest.NewTLSServer(http.HandlerFunc(server.handleRequest))

	return server
}

// handleRequest handles incoming HTTP requests
func (s *MockSlackServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.callCount++

	// Store request for later inspection
	s.requests = append(s.requests, r)

	// Read and store request body
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			s.requestBodies = append(s.requestBodies, string(bodyBytes))
		}
	}

	// Check if we have a specific response for this path
	if response, exists := s.responses[r.URL.Path]; exists {
		// Add delay if configured
		if response.Delay > 0 {
			time.Sleep(response.Delay)
		}

		// Set custom headers
		for key, value := range response.Headers {
			w.Header().Set(key, value)
		}

		// Write response
		w.WriteHeader(response.StatusCode)
		w.Write([]byte(response.Body))
		return
	}

	// Default response
	w.WriteHeader(s.defaultStatus)
	w.Write([]byte(s.defaultBody))
}

// GetURL returns the server's URL
func (s *MockSlackServer) GetURL() string {
	return s.server.URL
}

// GetWebhookURL returns a mock Slack webhook URL
func (s *MockSlackServer) GetWebhookURL() string {
	return s.server.URL + "/services/TEST/TEST/TESTWEBHOOKURL"
}

// GetWebhookURLWithPath returns a mock Slack webhook URL with custom path
func (s *MockSlackServer) GetWebhookURLWithPath(path string) string {
	return s.server.URL + path
}

// Close closes the mock server
func (s *MockSlackServer) Close() {
	s.server.Close()
}

// SetResponse sets a specific response for a path
func (s *MockSlackServer) SetResponse(path string, response MockResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[path] = response
}

// SetDefaultResponse sets the default response for all paths
func (s *MockSlackServer) SetDefaultResponse(statusCode int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultStatus = statusCode
	s.defaultBody = body
}

// GetCallCount returns the number of requests received
func (s *MockSlackServer) GetCallCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.callCount
}

// GetRequests returns all requests received
func (s *MockSlackServer) GetRequests() []*http.Request {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requests
}

// GetRequestBodies returns all request bodies
func (s *MockSlackServer) GetRequestBodies() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requestBodies
}

// GetLastRequest returns the last request received
func (s *MockSlackServer) GetLastRequest() *http.Request {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.requests) > 0 {
		return s.requests[len(s.requests)-1]
	}
	return nil
}

// GetLastRequestBody returns the last request body
func (s *MockSlackServer) GetLastRequestBody() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.requestBodies) > 0 {
		return s.requestBodies[len(s.requestBodies)-1]
	}
	return ""
}

// Reset resets the server state
func (s *MockSlackServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests = make([]*http.Request, 0)
	s.requestBodies = make([]string, 0)
	s.responses = make(map[string]MockResponse)
	s.callCount = 0
	s.defaultStatus = http.StatusOK
	s.defaultBody = "ok"
}

// SetupSlackScenario sets up a specific Slack testing scenario
func (s *MockSlackServer) SetupSlackScenario(scenario string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	webhookPath := "/services/TEST/TEST/TESTWEBHOOKURL"

	switch scenario {
	case "success":
		s.responses[webhookPath] = MockResponse{
			StatusCode: http.StatusOK,
			Body:       "ok",
		}

	case "rate_limit":
		s.responses[webhookPath] = MockResponse{
			StatusCode: http.StatusTooManyRequests,
			Body:       "rate_limited",
			Headers: map[string]string{
				"Retry-After": "60",
			},
		}

	case "server_error":
		s.responses[webhookPath] = MockResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "server_error",
		}

	case "bad_request":
		s.responses[webhookPath] = MockResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "invalid_payload",
		}

	case "forbidden":
		s.responses[webhookPath] = MockResponse{
			StatusCode: http.StatusForbidden,
			Body:       "invalid_token",
		}

	case "not_found":
		s.responses[webhookPath] = MockResponse{
			StatusCode: http.StatusNotFound,
			Body:       "channel_not_found",
		}

	case "slow_response":
		s.responses[webhookPath] = MockResponse{
			StatusCode: http.StatusOK,
			Body:       "ok",
			Delay:      2 * time.Second,
		}

	case "timeout":
		s.responses[webhookPath] = MockResponse{
			StatusCode: http.StatusOK,
			Body:       "ok",
			Delay:      35 * time.Second,
		}

	default:
		// Default to success
		s.responses[webhookPath] = MockResponse{
			StatusCode: http.StatusOK,
			Body:       "ok",
		}
	}
}

// ValidateSlackWebhookPayload validates that a request body is a valid Slack webhook payload
func (s *MockSlackServer) ValidateSlackWebhookPayload(body string) error {
	var payload map[string]interface{}

	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}

	// Check for required Slack webhook fields
	if text, exists := payload["text"]; !exists || text == "" {
		return fmt.Errorf("missing or empty 'text' field")
	}

	// Validate attachments if present
	if attachments, exists := payload["attachments"]; exists {
		if attachmentList, ok := attachments.([]interface{}); ok {
			for _, attachment := range attachmentList {
				if attachmentMap, ok := attachment.(map[string]interface{}); ok {
					// Basic validation of attachment structure
					if color, exists := attachmentMap["color"]; exists {
						if colorStr, ok := color.(string); ok {
							if !strings.HasPrefix(colorStr, "#") && !isValidColorName(colorStr) {
								return fmt.Errorf("invalid color format: %s", colorStr)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// isValidColorName checks if a color name is valid for Slack
func isValidColorName(color string) bool {
	validColors := map[string]bool{
		"good":    true,
		"warning": true,
		"danger":  true,
	}
	return validColors[color]
}

// GetRequestsByPath returns all requests for a specific path
func (s *MockSlackServer) GetRequestsByPath(path string) []*http.Request {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var requests []*http.Request
	for _, req := range s.requests {
		if req.URL.Path == path {
			requests = append(requests, req)
		}
	}
	return requests
}

// GetRequestCountByPath returns the number of requests for a specific path
func (s *MockSlackServer) GetRequestCountByPath(path string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, req := range s.requests {
		if req.URL.Path == path {
			count++
		}
	}
	return count
}

// GetUniqueRequestPaths returns all unique paths that received requests
func (s *MockSlackServer) GetUniqueRequestPaths() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pathSet := make(map[string]bool)
	for _, req := range s.requests {
		pathSet[req.URL.Path] = true
	}

	var paths []string
	for path := range pathSet {
		paths = append(paths, path)
	}
	return paths
}

// SimulateSlackWebhookSequence simulates a sequence of responses for the webhook endpoint
func (s *MockSlackServer) SimulateSlackWebhookSequence(responses []MockResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	webhookPath := "/services/TEST/TEST/TESTWEBHOOKURL"

	// For simplicity, we'll just use the first response for all requests
	// In a more sophisticated implementation, you'd cycle through responses
	if len(responses) > 0 {
		s.responses[webhookPath] = responses[0]
	}
}

// SetRateLimitResponse sets up a rate limit response with specific retry-after
func (s *MockSlackServer) SetRateLimitResponse(retryAfter int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	webhookPath := "/services/TEST/TEST/TESTWEBHOOKURL"
	s.responses[webhookPath] = MockResponse{
		StatusCode: http.StatusTooManyRequests,
		Body:       "rate_limited",
		Headers: map[string]string{
			"Retry-After": strconv.Itoa(retryAfter),
		},
	}
}

// WaitForRequest waits for a request to be received (useful for async testing)
func (s *MockSlackServer) WaitForRequest(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		s.mu.RLock()
		count := len(s.requests)
		s.mu.RUnlock()

		if count > 0 {
			return true
		}

		time.Sleep(10 * time.Millisecond)
	}

	return false
}

// WaitForRequestCount waits for a specific number of requests to be received
func (s *MockSlackServer) WaitForRequestCount(expectedCount int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		s.mu.RLock()
		count := len(s.requests)
		s.mu.RUnlock()

		if count >= expectedCount {
			return true
		}

		time.Sleep(10 * time.Millisecond)
	}

	return false
}

// CreateSlackWebhookTestScenario creates a specific test scenario with multiple endpoints
func (s *MockSlackServer) CreateSlackWebhookTestScenario(scenario string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch scenario {
	case "multi_webhook":
		// Setup multiple webhook endpoints
		s.responses["/services/TEST1/TEST1/WEBHOOK1"] = MockResponse{
			StatusCode: http.StatusOK,
			Body:       "ok",
		}
		s.responses["/services/TEST2/TEST2/WEBHOOK2"] = MockResponse{
			StatusCode: http.StatusOK,
			Body:       "ok",
		}
		s.responses["/services/TEST3/TEST3/WEBHOOK3"] = MockResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "server_error",
		}

	case "progressive_failure":
		// First request succeeds, second fails
		s.responses["/services/TEST/TEST/TESTWEBHOOKURL"] = MockResponse{
			StatusCode: http.StatusOK,
			Body:       "ok",
		}
		// Note: This is a simplified version. A real implementation would
		// need to track request counts and change responses accordingly

	case "intermittent_errors":
		// Simulate intermittent errors
		s.responses["/services/TEST/TEST/TESTWEBHOOKURL"] = MockResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "server_error",
		}

	default:
		s.responses["/services/TEST/TEST/TESTWEBHOOKURL"] = MockResponse{
			StatusCode: http.StatusOK,
			Body:       "ok",
		}
	}
}
