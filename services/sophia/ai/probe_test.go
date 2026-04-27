package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProbe_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.1:8b"},{"name":"llama3.2:1b"}]}`))
	}))
	defer srv.Close()

	if err := Probe(context.Background(), srv.Client(), srv.URL, "llama3.2:1b"); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestProbe_ModelMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.1:8b"}]}`))
	}))
	defer srv.Close()

	err := Probe(context.Background(), srv.Client(), srv.URL, "qwen2.5:14b")
	if err == nil {
		t.Fatal("expected missing-model error")
	}
	if !strings.Contains(err.Error(), "ollama pull qwen2.5:14b") {
		t.Errorf("expected pull hint in error, got %v", err)
	}
}

func TestProbe_OllamaDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // refuse connections

	err := Probe(context.Background(), srv.Client(), srv.URL, "llama3.2:1b")
	if err == nil {
		t.Fatal("expected unreachable error")
	}
	if !strings.Contains(err.Error(), "unreachable") {
		t.Errorf("expected unreachable message, got %v", err)
	}
}

func TestProbe_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := Probe(context.Background(), srv.Client(), srv.URL, "llama3.2:1b")
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected status 500 error, got %v", err)
	}
}

func TestProbe_EmptyConfig(t *testing.T) {
	if err := Probe(context.Background(), nil, "", "x"); err == nil {
		t.Error("expected error for empty host")
	}
	if err := Probe(context.Background(), nil, "http://x", ""); err == nil {
		t.Error("expected error for empty model")
	}
}
