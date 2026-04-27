package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/sophia/ai"
	"github.com/sagarjhaa/localfinance/services/sophia/api"
	"github.com/sagarjhaa/localfinance/services/sophia/config"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Auto-pick the best installed model if MODEL_NAME is "auto" or empty.
	// Heuristic: largest parameter count, family-preference tie-break
	// (qwen2.5 > llama3.1 > llama3.2 > others). Useful when the user has a
	// few models pulled and doesn't want to pin one.
	if cfg.AI.ModelName == "" || strings.EqualFold(cfg.AI.ModelName, "auto") {
		selectCtx, selectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		selectClient := &http.Client{Timeout: 10 * time.Second}
		picked, err := ai.SelectBestModel(selectCtx, selectClient, cfg.AI.OllamaHost)
		selectCancel()
		if err != nil {
			log.Fatalf("Auto model selection failed: %v", err)
		}
		log.Printf("Auto-selected model: %s", picked)
		cfg.AI.ModelName = picked
	}

	// Initialize AI service
	aiService, err := ai.NewService(cfg.AI)
	if err != nil {
		log.Fatalf("Failed to initialize AI service: %v", err)
	}

	// Startup probe: verify Ollama is reachable AND the configured model is
	// installed. Fail fast — better than discovering it on the first user
	// request. Can be skipped by setting SKIP_OLLAMA_PROBE=1 (useful for tests
	// or environments where Ollama lives behind a slow cold-start tunnel).
	if os.Getenv("SKIP_OLLAMA_PROBE") != "1" {
		probeCtx, probeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		probeClient := &http.Client{Timeout: 10 * time.Second}
		if err := ai.Probe(probeCtx, probeClient, cfg.AI.OllamaHost, cfg.AI.ModelName); err != nil {
			probeCancel()
			log.Fatalf("Ollama startup probe failed: %v", err)
		}
		probeCancel()
		log.Printf("Ollama probe OK: model %q reachable at %s", cfg.AI.ModelName, cfg.AI.OllamaHost)
	} else {
		log.Printf("Ollama startup probe skipped (SKIP_OLLAMA_PROBE=1)")
	}

	router := gin.New()
	router.Use(middleware.CorrelationMiddleware("sophia"))
	router.Use(gin.Recovery())

	// Setup full routes
	api.SetupRoutes(router, aiService, cfg.Thesaurus, cfg.AI.ModelName)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8002"
	}

	log.Printf("Sophia service starting on port %s", port)
	log.Printf("Correlation ID tracking enabled")
	log.Printf("AI model: %s", cfg.AI.ModelName)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Sophia service: %v", err)
	}
}
