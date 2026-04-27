// Package setup serves the first-run wizard endpoints. No auth — these run
// before login and gate access to the rest of the app.
package setup

import (
	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/ai"
	"github.com/sagarjhaa/localfinance/internal/ollama"
)

type Handlers struct {
	OllamaHost string
	HostRAM    func() uint64 // injectable for tests
}

func New(ollamaHost string) *Handlers {
	if ollamaHost == "" {
		ollamaHost = "http://127.0.0.1:11434"
	}
	return &Handlers{
		OllamaHost: ollamaHost,
		HostRAM:    ollama.HostRAM,
	}
}

// State reports which wizard step the user is currently on.
func (h *Handlers) State(c *gin.Context) {
	if !ollama.Reachable(h.OllamaHost) {
		c.JSON(200, gin.H{"step": "install_ollama"})
		return
	}
	models := ollama.InstalledModels(h.OllamaHost)
	if !ai.HasSweetSpotModel(models) {
		c.JSON(200, gin.H{"step": "pull_model", "installed_models": models})
		return
	}
	c.JSON(200, gin.H{"step": "ready"})
}

func (h *Handlers) OllamaStatus(c *gin.Context) {
	c.JSON(200, gin.H{
		"installed":   ollama.Reachable(h.OllamaHost),
		"version":     ollama.Version(h.OllamaHost),
		"host_ram_gb": h.HostRAM() / (1 << 30),
	})
}

func (h *Handlers) Recommended(c *gin.Context) {
	gb := h.HostRAM() / (1 << 30)
	model, size, reason := ai.RecommendByRAM(gb)
	c.JSON(200, gin.H{
		"model":   model,
		"size_gb": size,
		"reason":  reason,
	})
}

// PullModel SSE-streams Ollama /api/pull progress to the browser.
func (h *Handlers) PullModel(c *gin.Context) {
	var req struct {
		Model string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Model == "" {
		c.JSON(400, gin.H{"error": "model is required"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	err := ollama.StreamPull(c.Request.Context(), h.OllamaHost, req.Model, func(line []byte) {
		_, _ = c.Writer.Write([]byte("data: "))
		_, _ = c.Writer.Write(line)
		_, _ = c.Writer.Write([]byte("\n\n"))
		c.Writer.Flush()
	})
	if err != nil {
		_, _ = c.Writer.Write([]byte("event: error\ndata: " + err.Error() + "\n\n"))
		c.Writer.Flush()
	}
}
