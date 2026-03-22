package shared

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	CorrelationID string                 `json:"correlation_id"`
	ServiceName   string                 `json:"service_name"`
	Type          string                 `json:"type"` // request, response, business_logic, database_error, etc.
	Method        string                 `json:"method,omitempty"`
	Path          string                 `json:"path,omitempty"`
	StatusCode    int                    `json:"status_code,omitempty"`
	Message       string                 `json:"message,omitempty"`
	Data          map[string]interface{} `json:"data,omitempty"`
	Duration      string                 `json:"duration,omitempty"`
	Timestamp     string                 `json:"timestamp"`
	ClientIP      string                 `json:"client_ip,omitempty"`
}

// Log outputs the log entry as structured JSON
func (entry LogEntry) Log(message string) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339Nano)
	}
	if message != "" {
		entry.Message = message
	}
	
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}
	
	log.Printf("[%s] %s", strings.ToUpper(entry.Type), string(jsonBytes))
}

// Business logic logging helpers
func LogBusinessLogic(correlationID, serviceName, message string) {
	LogEntry{
		CorrelationID: correlationID,
		ServiceName:   serviceName,
		Type:          "business_logic",
		Message:       message,
		Timestamp:     time.Now().Format(time.RFC3339Nano),
	}.Log("")
}

func LogDatabaseError(correlationID, serviceName, operation string, err error) {
	LogEntry{
		CorrelationID: correlationID,
		ServiceName:   serviceName,
		Type:          "database_error",
		Message:       fmt.Sprintf("Database %s failed: %v", operation, err),
		Timestamp:     time.Now().Format(time.RFC3339Nano),
	}.Log("")
}

func LogServiceCall(correlationID, fromService, toService, endpoint string) {
	LogEntry{
		CorrelationID: correlationID,
		ServiceName:   fromService,
		Type:          "service_call",
		Message:       fmt.Sprintf("Calling %s service: %s", toService, endpoint),
		Data: map[string]interface{}{
			"target_service": toService,
			"endpoint":       endpoint,
		},
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}.Log("")
}

func LogServiceError(correlationID, fromService, toService string, err error) {
	LogEntry{
		CorrelationID: correlationID,
		ServiceName:   fromService,
		Type:          "service_error",
		Message:       fmt.Sprintf("Service call to %s failed: %v", toService, err),
		Data: map[string]interface{}{
			"target_service": toService,
			"error":          err.Error(),
		},
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}.Log("")
}