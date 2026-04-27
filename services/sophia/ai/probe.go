package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPDoer is the minimal HTTP interface needed by Probe — just *http.Client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Probe verifies that Ollama is reachable at ollamaHost AND that the named
// model is in the installed list. It returns:
//   - a "transport" error (Ollama unreachable) wrapping the underlying cause
//   - a "missing model" error pointing at `ollama pull <model>` if the host
//     is up but the model is not installed
//   - nil on success
//
// Callers (typically Sophia's main.go at startup) should treat any non-nil
// return as fatal so we fail fast rather than discovering it on first request.
func Probe(ctx context.Context, client HTTPDoer, ollamaHost, model string) error {
	if ollamaHost == "" {
		return fmt.Errorf("ollama host not configured")
	}
	if model == "" {
		return fmt.Errorf("ollama model name not configured")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	url := strings.TrimRight(ollamaHost, "/") + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("ollama probe: build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama probe: %s unreachable: %w", ollamaHost, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama probe: %s returned status %d", url, resp.StatusCode)
	}

	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("ollama probe: decode tags: %w", err)
	}

	for _, m := range body.Models {
		// Match either exact name (e.g. "llama3.1:8b") or the bare base
		// (e.g. "llama3.1") so config'd default tags work.
		if m.Name == model || strings.HasPrefix(m.Name, model+":") || strings.HasPrefix(model, m.Name+":") {
			return nil
		}
	}

	available := make([]string, 0, len(body.Models))
	for _, m := range body.Models {
		available = append(available, m.Name)
	}
	return fmt.Errorf(
		"ollama probe: model %q not installed at %s (available: %v). Run: ollama pull %s",
		model, ollamaHost, available, model,
	)
}
