package insights

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestThesaurusDismissalFetcher_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/internal/users/u1/dismissed-insights" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	f := &ThesaurusDismissalFetcher{BaseURL: srv.URL, Client: srv.Client()}
	got, err := f.ListDismissedKeys(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty set, got %d keys", len(got))
	}
}

func TestThesaurusDismissalFetcher_PopulatedList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"insight_key":"abc","created_at":"2026-04-22T00:00:00Z"},
			{"insight_key":"def"},
			{"insight_key":""}
		]`))
	}))
	defer srv.Close()

	f := &ThesaurusDismissalFetcher{BaseURL: srv.URL, Client: srv.Client()}
	got, err := f.ListDismissedKeys(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["abc"]; !ok {
		t.Errorf("missing key abc")
	}
	if _, ok := got["def"]; !ok {
		t.Errorf("missing key def")
	}
	if _, ok := got[""]; ok {
		t.Errorf("empty insight_key should be filtered out")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 keys, got %d", len(got))
	}
}

func TestThesaurusDismissalFetcher_NetworkError(t *testing.T) {
	// Closed server immediately produces connection refused.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	f := &ThesaurusDismissalFetcher{BaseURL: srv.URL, Client: &http.Client{Timeout: 100 * time.Millisecond}}
	_, err := f.ListDismissedKeys(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestThesaurusDismissalFetcher_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := &ThesaurusDismissalFetcher{BaseURL: srv.URL, Client: srv.Client()}
	_, err := f.ListDismissedKeys(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected server error, got nil")
	}
}

func TestThesaurusDismissalFetcher_EmptyBaseURL(t *testing.T) {
	f := &ThesaurusDismissalFetcher{}
	_, err := f.ListDismissedKeys(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected error for empty base URL")
	}
}

func TestThesaurusDismissalFetcher_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f := &ThesaurusDismissalFetcher{BaseURL: srv.URL, Client: srv.Client()}
	_, err := f.ListDismissedKeys(ctx, "u1")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	// Be lenient: some Go versions wrap; just ensure it's an error.
	if !errors.Is(err, context.Canceled) {
		// httptest may surface as "Get ... context canceled" — accept either form.
		t.Logf("error (acceptable): %v", err)
	}
}
