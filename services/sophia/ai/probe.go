package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
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

// paramCountRE matches the parameter-count token in an Ollama tag like
// "llama3.1:8b" or "qwen2.5:14b" — captures the numeric part before "b".
var paramCountRE = regexp.MustCompile(`(?i):(\d+(?:\.\d+)?)b\b`)

// familyPreference assigns a tie-breaker score to known-good model families.
// Higher is better. Used only when two models have the same parameter count.
func familyPreference(name string) int {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "qwen2.5"):
		return 4
	case strings.HasPrefix(lower, "qwen"):
		return 3
	case strings.HasPrefix(lower, "llama3.1"):
		return 3
	case strings.HasPrefix(lower, "llama3.2"):
		return 2
	case strings.HasPrefix(lower, "llama"):
		return 1
	}
	return 0
}

// extractParamCount returns the parameter count in billions for a tag like
// "llama3.1:8b" → 8.0, or 0 if it can't be parsed.
func extractParamCount(name string) float64 {
	m := paramCountRE.FindStringSubmatch(name)
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	return v
}

// SelectBestModel queries Ollama at ollamaHost for installed models and
// returns the "best" one by parameter count, breaking ties with the
// known-good family preference (qwen2.5 > qwen ~ llama3.1 > llama3.2 > others).
// Returns an error if Ollama is unreachable OR if no models are installed.
//
// Used by main.go when MODEL_NAME is "auto" or empty, so the user doesn't have
// to pick a model — Sophia picks the largest one already pulled.
func SelectBestModel(ctx context.Context, client HTTPDoer, ollamaHost string) (string, error) {
	if ollamaHost == "" {
		return "", fmt.Errorf("ollama host not configured")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	url := strings.TrimRight(ollamaHost, "/") + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("select best model: build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("select best model: %s unreachable: %w", ollamaHost, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("select best model: %s returned status %d", url, resp.StatusCode)
	}

	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("select best model: decode tags: %w", err)
	}
	if len(body.Models) == 0 {
		return "", fmt.Errorf("select best model: no models installed at %s — run `ollama pull <model>`", ollamaHost)
	}

	type scored struct {
		name   string
		params float64
		family int
	}
	candidates := make([]scored, 0, len(body.Models))
	for _, m := range body.Models {
		candidates = append(candidates, scored{
			name:   m.Name,
			params: extractParamCount(m.Name),
			family: familyPreference(m.Name),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].params != candidates[j].params {
			return candidates[i].params > candidates[j].params
		}
		if candidates[i].family != candidates[j].family {
			return candidates[i].family > candidates[j].family
		}
		return candidates[i].name < candidates[j].name
	})
	return candidates[0].name, nil
}
