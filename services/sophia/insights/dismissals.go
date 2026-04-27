package insights

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ThesaurusDismissalFetcher implements DismissalFetcher by calling Thesaurus's
// internal endpoint:
//
//	GET ${BaseURL}/internal/users/${userID}/dismissed-insights
//
// The endpoint returns a JSON array of objects with at least an "insight_key"
// field. Other fields (created_at, etc.) are ignored — the engine only cares
// about the set of keys.
type ThesaurusDismissalFetcher struct {
	BaseURL string
	Client  *http.Client
}

// NewThesaurusDismissalFetcher returns a fetcher with the default http.Client
// (which has no timeout). For production wiring you should pass a Client with a
// reasonable timeout via the struct literal.
func NewThesaurusDismissalFetcher(baseURL string) *ThesaurusDismissalFetcher {
	return &ThesaurusDismissalFetcher{BaseURL: baseURL, Client: http.DefaultClient}
}

// ListDismissedKeys fetches the user's dismissed insight keys and returns them
// as a set. An empty list is a valid empty set (not an error). Any non-200
// response or transport error is propagated.
func (f *ThesaurusDismissalFetcher) ListDismissedKeys(ctx context.Context, userID string) (map[string]struct{}, error) {
	if f.BaseURL == "" {
		return nil, fmt.Errorf("ThesaurusDismissalFetcher: empty BaseURL")
	}
	client := f.Client
	if client == nil {
		client = http.DefaultClient
	}

	url := fmt.Sprintf("%s/internal/users/%s/dismissed-insights", f.BaseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build dismissals request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch dismissals: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dismissals returned %d: %s", resp.StatusCode, string(body))
	}

	var rows []struct {
		InsightKey string `json:"insight_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode dismissals: %w", err)
	}

	out := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		if r.InsightKey == "" {
			continue
		}
		out[r.InsightKey] = struct{}{}
	}
	return out, nil
}
