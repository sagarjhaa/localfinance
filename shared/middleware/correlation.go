package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	CorrelationIDHeader = "X-Correlation-ID"
	ServiceNameHeader   = "X-Service-Name"
)

// LogEntry represents a structured log entry for request/response
type LogEntry struct {
	CorrelationID string                 `json:"correlation_id"`
	ServiceName   string                 `json:"service_name"`
	Type          string                 `json:"type"` // "request" or "response"
	Method        string                 `json:"method"`
	Path          string                 `json:"path"`
	StatusCode    int                    `json:"status_code,omitempty"`
	Headers       map[string]interface{} `json:"headers,omitempty"`
	Body          interface{}            `json:"body,omitempty"`
	Duration      string                 `json:"duration,omitempty"`
	Timestamp     string                 `json:"timestamp"`
	ClientIP      string                 `json:"client_ip"`
}

// CorrelationMiddleware creates middleware for correlation ID handling and logging
func CorrelationMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Extract or generate correlation ID
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = generateCorrelationID()
		}

		// Set correlation ID in context and response header
		c.Set("correlation_id", correlationID)
		c.Header(CorrelationIDHeader, correlationID)
		c.Header(ServiceNameHeader, serviceName)

		// Log request
		logRequest(c, correlationID, serviceName, startTime)

		// Capture response
		responseWriter := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:          &bytes.Buffer{},
		}
		c.Writer = responseWriter

		// Process request
		c.Next()

		// Log response
		duration := time.Since(startTime)
		logResponse(c, responseWriter, correlationID, serviceName, duration)
	}
}

// responseBodyWriter captures the response body for logging
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// generateCorrelationID creates a new correlation ID
func generateCorrelationID() string {
	return fmt.Sprintf("lf_%s", strings.ReplaceAll(uuid.New().String(), "-", "")[:16])
}

// logRequest logs the incoming request
func logRequest(c *gin.Context, correlationID, serviceName string, timestamp time.Time) {
	// Read request body
	var requestBody interface{}
	if c.Request.Body != nil {
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		
		if len(bodyBytes) > 0 {
			// Try to parse as JSON
			var jsonBody interface{}
			if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
				requestBody = jsonBody
			} else {
				// If not JSON, store as string (truncated for large bodies)
				if len(bodyBytes) > 1000 {
					requestBody = fmt.Sprintf("%.1000s...", string(bodyBytes))
				} else {
					requestBody = string(bodyBytes)
				}
			}
		}
	}

	// Extract relevant headers (excluding sensitive ones)
	headers := make(map[string]interface{})
	for name, values := range c.Request.Header {
		if !isSensitiveHeader(name) {
			if len(values) == 1 {
				headers[name] = values[0]
			} else {
				headers[name] = values
			}
		} else {
			headers[name] = "[REDACTED]"
		}
	}

	logEntry := LogEntry{
		CorrelationID: correlationID,
		ServiceName:   serviceName,
		Type:          "request",
		Method:        c.Request.Method,
		Path:          c.Request.URL.Path,
		Headers:       headers,
		Body:          requestBody,
		Timestamp:     timestamp.Format(time.RFC3339Nano),
		ClientIP:      c.ClientIP(),
	}

	logJSON(logEntry)
}

// logResponse logs the outgoing response
func logResponse(c *gin.Context, responseWriter *responseBodyWriter, correlationID, serviceName string, duration time.Duration) {
	// Parse response body
	var responseBody interface{}
	if responseWriter.body.Len() > 0 {
		bodyBytes := responseWriter.body.Bytes()
		
		// Try to parse as JSON
		var jsonBody interface{}
		if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
			responseBody = jsonBody
		} else {
			// If not JSON, store as string (truncated for large bodies)
			if len(bodyBytes) > 1000 {
				responseBody = fmt.Sprintf("%.1000s...", string(bodyBytes))
			} else {
				responseBody = string(bodyBytes)
			}
		}
	}

	// Extract response headers
	headers := make(map[string]interface{})
	for name, values := range c.Writer.Header() {
		if len(values) == 1 {
			headers[name] = values[0]
		} else {
			headers[name] = values
		}
	}

	logEntry := LogEntry{
		CorrelationID: correlationID,
		ServiceName:   serviceName,
		Type:          "response",
		Method:        c.Request.Method,
		Path:          c.Request.URL.Path,
		StatusCode:    c.Writer.Status(),
		Headers:       headers,
		Body:          responseBody,
		Duration:      duration.String(),
		Timestamp:     time.Now().Format(time.RFC3339Nano),
		ClientIP:      c.ClientIP(),
	}

	logJSON(logEntry)
}

// isSensitiveHeader checks if a header contains sensitive information
func isSensitiveHeader(headerName string) bool {
	sensitive := []string{
		"authorization", "cookie", "x-auth-token", "x-api-key", 
		"password", "secret", "token", "key",
	}
	
	headerLower := strings.ToLower(headerName)
	for _, s := range sensitive {
		if strings.Contains(headerLower, s) {
			return true
		}
	}
	return false
}

// logJSON outputs structured JSON logs
func logJSON(entry LogEntry) {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}
	
	// Use structured logging format
	log.Printf("[%s] %s", strings.ToUpper(entry.Type), string(jsonBytes))
}

// GetCorrelationID extracts correlation ID from gin context
