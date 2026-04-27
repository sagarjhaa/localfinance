package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/ai"
	"github.com/sagarjhaa/localfinance/internal/insights"
)

// fakeFetcher implements TransactionFetcher for the handler tests.
type fakeFetcher struct {
	txs []ai.TransactionRef
	err error
}

func (f *fakeFetcher) FetchTransactionsSince(_ string, _ time.Time) ([]ai.TransactionRef, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.txs, nil
}

// noopDismiss returns no dismissals — used so the engine doesn't try to call
// Thesaurus over the network in unit tests.
type noopDismiss struct{}

func (noopDismiss) ListDismissedKeys(_ context.Context, _ string) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

func newTestHandler(fetcher TransactionFetcher, now time.Time) *InsightsHandler {
	return &InsightsHandler{
		aiService: fetcher,
		engine:    insights.NewEngine(noopDismiss{}),
		narrator:  insights.NewTemplateNarrator(),
		clock:     func() time.Time { return now },
	}
}

func tx(id string, daysAgoFromNow int, now time.Time, desc, category string, amt float64) ai.TransactionRef {
	return ai.TransactionRef{
		ID:          id,
		Date:        now.AddDate(0, 0, -daysAgoFromNow).Format(time.RFC3339),
		Description: desc,
		Category:    category,
		Amount:      amt,
		Type:        "expense",
	}
}

func TestGenerateInsights_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)

	// Build txs that should trigger detectNewRecurringMerchant: 2 charges in
	// last 30d, none earlier.
	txs := []ai.TransactionRef{
		tx("a", 28, now, "ChatGPT Plus", "Software", 20),
		tx("b", 1, now, "ChatGPT Plus", "Software", 20),
	}

	h := newTestHandler(&fakeFetcher{txs: txs}, now)

	router := gin.New()
	router.POST("/insights", h.GenerateInsights)

	body, _ := json.Marshal(ai.InsightsRequest{UserID: "u1", Period: "the last 90 days"})
	req := httptest.NewRequest("POST", "/insights", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var got struct {
		UserID    string                    `json:"user_id"`
		Insights  []ai.FinancialInsight `json:"insights"`
		Narrative struct {
			Overall    string            `json:"overall"`
			PerInsight map[string]string `json:"per_insight"`
			Source     string            `json:"source"`
		} `json:"narrative"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got.UserID != "u1" {
		t.Errorf("user_id mismatch: %s", got.UserID)
	}
	if len(got.Insights) == 0 {
		t.Fatalf("expected at least one insight, got 0; body=%s", w.Body.String())
	}
	foundRecurring := false
	for _, ins := range got.Insights {
		if ins.Type == insights.RuleNewRecurringMerchant {
			foundRecurring = true
			if !strings.Contains(ins.Description, "ChatGPT Plus") {
				t.Errorf("expected merchant in description, got %q", ins.Description)
			}
		}
	}
	if !foundRecurring {
		t.Errorf("expected new_recurring_merchant insight; got types=%v", insightTypes(got.Insights))
	}
	if got.Narrative.Source != "template" {
		t.Errorf("expected template narrator, got %q", got.Narrative.Source)
	}
	if got.Narrative.Overall == "" {
		t.Errorf("expected non-empty Overall narrative")
	}
}

func insightTypes(xs []ai.FinancialInsight) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, x.Type)
	}
	return out
}

func TestGenerateInsights_FetcherError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	h := newTestHandler(&fakeFetcher{err: errors.New("thesaurus down")}, now)

	router := gin.New()
	router.POST("/insights", h.GenerateInsights)

	body, _ := json.Marshal(ai.InsightsRequest{UserID: "u1"})
	req := httptest.NewRequest("POST", "/insights", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestGenerateInsights_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&fakeFetcher{}, time.Now())

	router := gin.New()
	router.POST("/insights", h.GenerateInsights)

	req := httptest.NewRequest("POST", "/insights", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestGenerateInsights_EmptyTransactions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(&fakeFetcher{txs: nil}, time.Now())

	router := gin.New()
	router.POST("/insights", h.GenerateInsights)

	body, _ := json.Marshal(ai.InsightsRequest{UserID: "u1"})
	req := httptest.NewRequest("POST", "/insights", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 even with no data, got %d body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["insights"] == nil {
		t.Error("expected insights field present (even if empty)")
	}
}
