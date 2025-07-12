package mocks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MockHTTPClient provides a mock implementation of http.Client for testing
type MockHTTPClient struct {
	responses       map[string]*http.Response
	requests        []*http.Request
	requestBodies   []string
	shouldFail      bool
	failureError    error
	responseDelay   time.Duration
	callCount       int
	statusCode      int
	responseBody    string
	headers         map[string]string
	simulateTimeout bool
	mu              sync.RWMutex
}

// NewMockHTTPClient creates a new mock HTTP client
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses:     make(map[string]*http.Response),
		requests:      make([]*http.Request, 0),
		requestBodies: make([]string, 0),
		statusCode:    http.StatusOK,
		responseBody:  "ok",
		headers:       make(map[string]string),
	}
}

// Do implements the http.Client interface
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	// Store the request for later inspection
	m.requests = append(m.requests, req)

	// Read and store the request body
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			m.requestBodies = append(m.requestBodies, string(bodyBytes))
			// Restore the request body for potential reuse
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	// Simulate timeout if configured
	if m.simulateTimeout {
		return nil, errors.New("context deadline exceeded")
	}

	// Simulate response delay if configured
	if m.responseDelay > 0 {
		time.Sleep(m.responseDelay)
	}

	// Return failure if configured
	if m.shouldFail {
		if m.failureError != nil {
			return nil, m.failureError
		}
		return nil, errors.New("mock HTTP client failure")
	}

	// Check if we have a specific response for this URL
	if response, exists := m.responses[req.URL.String()]; exists {
		return response, nil
	}

	// Create a default response
	response := &http.Response{
		StatusCode: m.statusCode,
		Status:     http.StatusText(m.statusCode),
		Body:       io.NopCloser(strings.NewReader(m.responseBody)),
		Header:     make(http.Header),
		Request:    req,
	}

	// Add custom headers
	for key, value := range m.headers {
		response.Header.Set(key, value)
	}

	return response, nil
}

// SetResponse sets a specific response for a URL
func (m *MockHTTPClient) SetResponse(url string, response *http.Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[url] = response
}

// SetStatusCode sets the default HTTP status code
func (m *MockHTTPClient) SetStatusCode(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCode = code
}

// SetResponseBody sets the default response body
func (m *MockHTTPClient) SetResponseBody(body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responseBody = body
}

// SetHeader sets a response header
func (m *MockHTTPClient) SetHeader(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.headers[key] = value
}

// SetShouldFail configures the client to fail requests
func (m *MockHTTPClient) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

// SetFailureError sets a specific error to return on failure
func (m *MockHTTPClient) SetFailureError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failureError = err
}

// SetResponseDelay sets a delay for responses
func (m *MockHTTPClient) SetResponseDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responseDelay = delay
}

// SetSimulateTimeout configures the client to simulate timeout errors
func (m *MockHTTPClient) SetSimulateTimeout(simulate bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateTimeout = simulate
}

// GetCallCount returns the number of times Do was called
func (m *MockHTTPClient) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetRequests returns all requests made to the client
func (m *MockHTTPClient) GetRequests() []*http.Request {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requests
}

// GetRequestBodies returns all request bodies
func (m *MockHTTPClient) GetRequestBodies() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestBodies
}

// GetLastRequest returns the last request made
func (m *MockHTTPClient) GetLastRequest() *http.Request {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.requests) > 0 {
		return m.requests[len(m.requests)-1]
	}
	return nil
}

// GetLastRequestBody returns the last request body
func (m *MockHTTPClient) GetLastRequestBody() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.requestBodies) > 0 {
		return m.requestBodies[len(m.requestBodies)-1]
	}
	return ""
}

// Reset resets the mock client to its initial state
func (m *MockHTTPClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.responses = make(map[string]*http.Response)
	m.requests = make([]*http.Request, 0)
	m.requestBodies = make([]string, 0)
	m.shouldFail = false
	m.failureError = nil
	m.responseDelay = 0
	m.callCount = 0
	m.statusCode = http.StatusOK
	m.responseBody = "ok"
	m.headers = make(map[string]string)
	m.simulateTimeout = false
}

// CreateSlackSuccessResponse creates a successful Slack response
func (m *MockHTTPClient) CreateSlackSuccessResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
	}
}

// CreateSlackErrorResponse creates an error response with specific status and body
func (m *MockHTTPClient) CreateSlackErrorResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// CreateSlackRateLimitResponse creates a rate limit response
func (m *MockHTTPClient) CreateSlackRateLimitResponse(retryAfter string) *http.Response {
	response := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Status:     "429 Too Many Requests",
		Body:       io.NopCloser(strings.NewReader("rate_limited")),
		Header:     make(http.Header),
	}
	response.Header.Set("Retry-After", retryAfter)
	return response
}

// SetupSlackScenario sets up a specific Slack testing scenario
func (m *MockHTTPClient) SetupSlackScenario(scenario string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch scenario {
	case "success":
		m.statusCode = http.StatusOK
		m.responseBody = "ok"
		m.shouldFail = false

	case "rate_limit":
		m.statusCode = http.StatusTooManyRequests
		m.responseBody = "rate_limited"
		m.headers["Retry-After"] = "60"
		m.shouldFail = false

	case "server_error":
		m.statusCode = http.StatusInternalServerError
		m.responseBody = "server_error"
		m.shouldFail = false

	case "bad_request":
		m.statusCode = http.StatusBadRequest
		m.responseBody = "invalid_payload"
		m.shouldFail = false

	case "forbidden":
		m.statusCode = http.StatusForbidden
		m.responseBody = "invalid_token"
		m.shouldFail = false

	case "not_found":
		m.statusCode = http.StatusNotFound
		m.responseBody = "channel_not_found"
		m.shouldFail = false

	case "timeout":
		m.simulateTimeout = true
		m.shouldFail = false

	case "network_error":
		m.shouldFail = true
		m.failureError = errors.New("network error")

	default:
		// Default to success
		m.statusCode = http.StatusOK
		m.responseBody = "ok"
		m.shouldFail = false
	}
}

// ValidateSlackWebhookRequest validates that the request is a proper Slack webhook request
func (m *MockHTTPClient) ValidateSlackWebhookRequest(req *http.Request) error {
	if req.Method != "POST" {
		return fmt.Errorf("expected POST request, got %s", req.Method)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}

	if !strings.HasPrefix(req.URL.String(), "https://hooks.slack.com/services/") {
		return fmt.Errorf("invalid Slack webhook URL: %s", req.URL.String())
	}

	return nil
}

// GetRequestCount returns the number of requests made to a specific URL
func (m *MockHTTPClient) GetRequestCount(url string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, req := range m.requests {
		if req.URL.String() == url {
			count++
		}
	}
	return count
}

// GetRequestsByURL returns all requests made to a specific URL
func (m *MockHTTPClient) GetRequestsByURL(url string) []*http.Request {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var requests []*http.Request
	for _, req := range m.requests {
		if req.URL.String() == url {
			requests = append(requests, req)
		}
	}
	return requests
}

// GetUniqueURLs returns all unique URLs that received requests
func (m *MockHTTPClient) GetUniqueURLs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	urlSet := make(map[string]bool)
	for _, req := range m.requests {
		urlSet[req.URL.String()] = true
	}

	var urls []string
	for url := range urlSet {
		urls = append(urls, url)
	}
	return urls
}

// SimulateSlackWebhookSequence simulates a sequence of Slack webhook responses
func (m *MockHTTPClient) SimulateSlackWebhookSequence(responses []int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a counter for tracking which response to use
	counter := 0

	// Note: This is a simplified approach. In a real implementation, you'd need
	// to handle this more carefully, possibly using a queue or callback system
	for _, statusCode := range responses {
		url := fmt.Sprintf("https://hooks.slack.com/services/TEST/TEST/TEST%d", counter)
		response := m.CreateSlackErrorResponse(statusCode, fmt.Sprintf("response_%d", counter))
		m.SetResponse(url, response)
		counter++
	}
}
