package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/ai"
)

type ChatHandler struct {
	aiService *ai.Service
}

func NewChatHandler(aiService *ai.Service) *ChatHandler {
	return &ChatHandler{aiService: aiService}
}

func (h *ChatHandler) HandleFinancialQuery(c *gin.Context) {
	var query ai.FinancialQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.aiService.AnswerFinancialQuery(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process financial query",
			"details": err.Error(),
		})
		return
	}

	response.GeneratedAt = time.Now()
	c.JSON(http.StatusOK, response)
}

// HandleFinancialQueryStream serves the chat over Server-Sent Events so
// the frontend can show live status ("Asking AI to plan queries…",
// "Running 2 queries…", "Writing the answer…") while the multi-step
// flow runs.
//
// Wire format: each event is one line of JSON, prefixed with the SSE
// fields. Two event types:
//
//	event: status
//	data:  {"phase": "...", "detail": "..."}
//
//	event: final
//	data:  <full AIResponse JSON>
//
// On error we emit `event: error` with `{"error": "..."}` and close.
//
// The handler accepts the same JSON body shape as HandleFinancialQuery
// so the frontend can switch endpoints without changing payloads.
func (h *ChatHandler) HandleFinancialQueryStream(c *gin.Context) {
	var query ai.FinancialQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // disable nginx-style buffering if reverse-proxied

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Channel buffers progress events between the worker goroutine and
	// the writer loop here. Buffer is generous so a slow client can't
	// block the chat flow.
	type evt struct {
		Name string
		Data any
	}
	events := make(chan evt, 32)

	progress := func(phase, detail string) {
		select {
		case events <- evt{Name: "status", Data: gin.H{"phase": phase, "detail": detail}}:
		default:
			// Drop on full — status updates are best-effort.
		}
	}

	go func() {
		defer close(events)
		resp, err := h.aiService.AnswerFinancialQueryStream(c.Request.Context(), query, progress)
		if err != nil {
			events <- evt{Name: "error", Data: gin.H{"error": err.Error()}}
			return
		}
		resp.GeneratedAt = time.Now()
		events <- evt{Name: "final", Data: resp}
	}()

	for ev := range events {
		body, _ := json.Marshal(ev.Data)
		// SSE wire format. Each event is `event:` line + `data:` line + blank.
		_, _ = c.Writer.Write([]byte("event: " + ev.Name + "\n"))
		_, _ = c.Writer.Write([]byte("data: " + string(body) + "\n\n"))
		flusher.Flush()
	}
}
