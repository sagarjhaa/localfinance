package shared

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sagarjhaa/localfinance/shared/middleware"
)

// HTTPClient wraps http.Client with correlation ID support and structured logging
type HTTPClient struct {
	client      *http.Client
	serviceName string
	baseURL     string
}

// NewHTTPClient creates a new HTTP client with correlation ID support
func NewHTTPClient(serviceName, baseURL string) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &loggingRoundTripper{
				next:        http.DefaultTransport,
				serviceName: serviceName,
			},
		},
		serviceName: serviceName,
		baseURL:     baseURL,
	}
}

// RequestOptions contains options for HTTP requests
type RequestOptions struct {
	CorrelationID string
	Headers       map[string]string
	Timeout       time.Duration
}

// Get makes a GET request with correlation ID
func (c *HTTPClient) Get(ctx context.Context, path string, opts *RequestOptions) (*http.Response, error) {
	return c.makeRequest(ctx, "GET", path, nil, opts)
}

// Post makes a POST request with correlation ID
func (c *HTTPClient) Post(ctx context.Context, path string, body interface{}, opts *RequestOptions) (*http.Response, error) {
	return c.makeRequest(ctx, "POST", path, body, opts)
}

// Put makes a PUT request with correlation ID
func (c *HTTPClient) Put(ctx context.Context, path string, body interface{}, opts *RequestOptions) (*http.Response, error) {
	return c.makeRequest(ctx, "PUT", path, body, opts)
}

// Delete makes a DELETE request with correlation ID
func (c *HTTPClient) Delete(ctx context.Context, path string, opts *RequestOptions) (*http.Response, error) {
	return c.makeRequest(ctx, "DELETE", path, nil, opts)
}

// makeRequest is the core method that handles all HTTP requests
func (c *HTTPClient) makeRequest(ctx context.Context, method, path string, body interface{}, opts *RequestOptions) (*http.Response, error) {
	if opts == nil {
		opts = &RequestOptions{}
	}

	// Build URL
	url := c.baseURL + path

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set correlation ID
	if opts.CorrelationID != "" {
		req.Header.Set(middleware.CorrelationIDHeader, opts.CorrelationID)
	}

	// Set service name
	req.Header.Set("X-Source-Service", c.serviceName)

	// Set content type for POST/PUT requests with body
	if body != nil && (method == "POST" || method == "PUT") {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	if opts.Headers != nil {
		for key, value := range opts.Headers {
			req.Header.Set(key, value)
		}
	}

	// Apply timeout if specified
	if opts.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// Make request
	return c.client.Do(req)
}

// PostJSON is a convenience method for JSON POST requests
func (c *HTTPClient) PostJSON(ctx context.Context, path string, request, response interface{}, opts *RequestOptions) error {
	resp, err := c.Post(ctx, path, request, opts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return fmt.Errorf("HTTP %d: %v", resp.StatusCode, errorResp)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if response != nil {
		return json.NewDecoder(resp.Body).Decode(response)
	}

	return nil
}

// GetJSON is a convenience method for JSON GET requests
func (c *HTTPClient) GetJSON(ctx context.Context, path string, response interface{}, opts *RequestOptions) error {
	resp, err := c.Get(ctx, path, opts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return fmt.Errorf("HTTP %d: %v", resp.StatusCode, errorResp)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if response != nil {
		return json.NewDecoder(resp.Body).Decode(response)
	}

	return nil
}

// loggingRoundTripper implements http.RoundTripper with correlation ID logging
type loggingRoundTripper struct {
	next        http.RoundTripper
	serviceName string
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()
	correlationID := req.Header.Get(middleware.CorrelationIDHeader)

	// Log outgoing request
	l.logRequest(req, correlationID)

	// Make request
	resp, err := l.next.RoundTrip(req)
	duration := time.Since(startTime)

	// Log response
	if err != nil {
		l.logError(req, correlationID, err, duration)
	} else {
		l.logResponse(req, resp, correlationID, duration)
	}

	return resp, err
}

func (l *loggingRoundTripper) logRequest(req *http.Request, correlationID string) {
	var body interface{}
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		if len(bodyBytes) > 0 {
			var jsonBody interface{}
			if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
				body = jsonBody
			} else {
				body = truncateString(string(bodyBytes), 500)
			}
		}
	}

	logEntry := map[string]interface{}{
		"correlation_id": correlationID,
		"service_name":   l.serviceName,
		"type":           "outgoing_request",
		"method":         req.Method,
		"url":            req.URL.String(),
		"headers":        sanitizeHeaders(req.Header),
		"body":           body,
		"timestamp":      time.Now().Format(time.RFC3339Nano),
	}

	logJSON("OUTGOING", logEntry)
}

func (l *loggingRoundTripper) logResponse(req *http.Request, resp *http.Response, correlationID string, duration time.Duration) {
	var body interface{}
	if resp.Body != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		if len(bodyBytes) > 0 {
			var jsonBody interface{}
			if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
				body = jsonBody
			} else {
				body = truncateString(string(bodyBytes), 500)
			}
		}
	}

	logEntry := map[string]interface{}{
		"correlation_id": correlationID,
		"service_name":   l.serviceName,
		"type":           "incoming_response",
		"method":         req.Method,
		"url":            req.URL.String(),
		"status_code":    resp.StatusCode,
		"headers":        sanitizeHeaders(resp.Header),
		"body":           body,
		"duration":       duration.String(),
		"timestamp":      time.Now().Format(time.RFC3339Nano),
	}

	logJSON("INCOMING", logEntry)
}

func (l *loggingRoundTripper) logError(req *http.Request, correlationID string, err error, duration time.Duration) {
	logEntry := map[string]interface{}{
		"correlation_id": correlationID,
		"service_name":   l.serviceName,
		"type":           "request_error",
		"method":         req.Method,
		"url":            req.URL.String(),
		"error":          err.Error(),
		"duration":       duration.String(),
		"timestamp":      time.Now().Format(time.RFC3339Nano),
	}

	logJSON("ERROR", logEntry)
}

// Helper functions
func sanitizeHeaders(headers http.Header) map[string]interface{} {
	result := make(map[string]interface{})
	for name, values := range headers {
		if isSensitiveHeader(name) {
			result[name] = "[REDACTED]"
		} else if len(values) == 1 {
			result[name] = values[0]
		} else {
			result[name] = values
		}
	}
	return result
}

func isSensitiveHeader(headerName string) bool {
	sensitive := []string{"authorization", "cookie", "x-auth-token", "x-api-key", "password", "secret", "token", "key"}
	headerLower := strings.ToLower(headerName)
	for _, s := range sensitive {
		if strings.Contains(headerLower, s) {
			return true
		}
	}
	return false
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func logJSON(level string, entry map[string]interface{}) {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}
	log.Printf("[%s] %s", level, string(jsonBytes))
}