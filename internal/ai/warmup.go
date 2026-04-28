package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// WarmupKeepAlive is the keep-alive duration we ask Ollama to hold the model
// in VRAM after a request. 30 minutes covers a friend-and-family use pattern
// where the user uploads a few statements, walks away for coffee, comes back,
// and uploads more — without paying the 10-15s reload cost each time.
const WarmupKeepAlive = "30m"

// WarmupModel preloads a model into Ollama's VRAM cache. Sending an empty
// prompt with keep_alive set causes Ollama to load the weights but generate
// zero tokens, so the call returns in roughly the model-load time (5-15s on
// a fresh start, ~0s if already cached).
//
// Best-effort: callers should ignore errors. A failed warmup just means the
// first real parse pays the cold-start cost itself.
func WarmupModel(ctx context.Context, client *http.Client, ollamaHost, model string) error {
	if model == "" {
		return fmt.Errorf("warmup: empty model name")
	}
	body, err := json.Marshal(map[string]interface{}{
		"model":      model,
		"prompt":     "",
		"stream":     false,
		"keep_alive": WarmupKeepAlive,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(ollamaHost, "/")+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("warmup: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("warmup: status %d", resp.StatusCode)
	}
	return nil
}

// WarmupAsync fires WarmupModel in a goroutine with a 60s budget. Use this
// from startup paths that don't want to block on the model load.
func WarmupAsync(ollamaHost, model string) {
	if model == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		client := &http.Client{Timeout: 60 * time.Second}
		_ = WarmupModel(ctx, client, ollamaHost, model)
	}()
}
