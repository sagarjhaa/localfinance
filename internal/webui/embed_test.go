package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServesIndexHTML(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: got %d want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(strings.ToLower(string(body)), "<!doctype html") {
		t.Fatalf("expected index html, got %q", body[:minInt(200, len(body))])
	}
}

func TestSPAFallback(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/insights")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("SPA fallback status: got %d want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(strings.ToLower(string(body)), "<!doctype html") {
		t.Fatalf("expected SPA fallback to return index.html")
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
