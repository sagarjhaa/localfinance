package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterHealth(t *testing.T) {
	r := NewRouter()
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", resp.StatusCode)
	}
}
