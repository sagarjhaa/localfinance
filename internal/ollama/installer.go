// Package ollama provides setup-time helpers: probes for the Ollama daemon,
// streams model-pull progress to the wizard, and detects host RAM for
// model recommendations.
package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Reachable returns true if the Ollama daemon at host accepts an HTTP
// request to /api/tags within 2 seconds.
func Reachable(host string) bool {
	if host == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(host, "/")+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// InstalledModels returns the names of every Ollama model currently pulled
// at host. Best-effort; returns nil on any error (caller should treat as
// "no models").
func InstalledModels(host string) []string {
	resp, err := http.Get(strings.TrimRight(host, "/") + "/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if json.NewDecoder(resp.Body).Decode(&body) != nil {
		return nil
	}
	out := make([]string, 0, len(body.Models))
	for _, m := range body.Models {
		out = append(out, m.Name)
	}
	return out
}

// Version queries Ollama's /api/version. Returns empty string on error.
func Version(host string) string {
	resp, err := http.Get(strings.TrimRight(host, "/") + "/api/version")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var body struct {
		Version string `json:"version"`
	}
	if json.NewDecoder(resp.Body).Decode(&body) != nil {
		return ""
	}
	return body.Version
}

// HostRAM returns the host's total physical RAM in bytes. macOS-only —
// uses sysctl hw.memsize. Returns 0 if the call fails.
func HostRAM() uint64 {
	v, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0
	}
	return v
}

// StreamPull POSTs to Ollama /api/pull and forwards each ndjson progress
// line to cb. Blocks until the pull completes or ctx is canceled.
func StreamPull(ctx context.Context, host, model string, cb func(line []byte)) error {
	body, err := json.Marshal(map[string]string{"name": model, "stream": "true"})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(host, "/")+"/api/pull",
		strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ollama pull returned %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return err
		}
		cb(raw)
	}
	return nil
}
