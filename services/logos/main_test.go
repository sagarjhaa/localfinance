package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sagarjhaa/localfinance/services/logos/models"
)

func TestInferPeriod_FirstNonZero(t *testing.T) {
	txns := []models.Transaction{
		{}, // zero date
		{Date: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	}
	if got := inferPeriod(txns); got != "2026-04" {
		t.Fatalf("got %q", got)
	}
}

func TestInferPeriod_FallsBackToNow(t *testing.T) {
	got := inferPeriod(nil)
	now := time.Now().UTC()
	want := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	wantStr := want.Format("2006-01")
	if got != wantStr {
		t.Fatalf("got %q want %q", got, wantStr)
	}
}

func TestTriggerMonthReview_PostsToSophia(t *testing.T) {
	type body struct {
		UserID string `json:"user_id"`
		Period string `json:"period"`
	}
	got := make(chan body, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/internal/month-review/generate" {
			http.NotFound(w, r)
			return
		}
		var b body
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &b)
		got <- b
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	t.Setenv("SOPHIA_URL", srv.URL)

	triggerMonthReview("11111111-1111-1111-1111-111111111111", "2026-04", "doc-1")

	select {
	case b := <-got:
		if b.Period != "2026-04" {
			t.Fatalf("period: %v", b)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}

func TestTriggerMonthReview_NopForZeroUser(t *testing.T) {
	// Should not panic or hit network; just exercise the early return.
	t.Setenv("SOPHIA_URL", "http://127.0.0.1:1") // would fail if invoked
	triggerMonthReview("00000000-0000-0000-0000-000000000000", "2026-04", "doc-1")
	triggerMonthReview("", "2026-04", "doc-1")
}
