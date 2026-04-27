package setup

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newRouter(h *Handlers) *gin.Engine {
	r := gin.New()
	g := r.Group("/api/setup")
	g.GET("/state", h.State)
	g.GET("/ollama-status", h.OllamaStatus)
	g.GET("/recommended", h.Recommended)
	g.POST("/pull-model", h.PullModel)
	return r
}

func TestState_InstallOllama(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", 500)
	}))
	defer srv.Close()

	h := &Handlers{OllamaHost: srv.URL, HostRAM: func() uint64 { return 16 << 30 }}
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/api/setup/state", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["step"] != "install_ollama" {
		t.Errorf("expected install_ollama, got %v", body["step"])
	}
}

func TestState_PullModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:1b"}]}`))
	}))
	defer srv.Close()

	h := &Handlers{OllamaHost: srv.URL, HostRAM: func() uint64 { return 16 << 30 }}
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/api/setup/state", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["step"] != "pull_model" {
		t.Errorf("expected pull_model, got %v", body["step"])
	}
}

func TestState_Ready(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen2.5:7b"}]}`))
	}))
	defer srv.Close()

	h := &Handlers{OllamaHost: srv.URL, HostRAM: func() uint64 { return 16 << 30 }}
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/api/setup/state", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["step"] != "ready" {
		t.Errorf("expected ready, got %v", body["step"])
	}
}

func TestRecommended_Tiers(t *testing.T) {
	cases := []struct {
		name    string
		ramGB   uint64
		wantMdl string
	}{
		{"low-ram", 8, "llama3.2:3b"},
		{"mid-ram", 16, "gemma3:4b"},
		{"high-ram", 32, "qwen2.5:7b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &Handlers{OllamaHost: "http://x", HostRAM: func() uint64 { return tc.ramGB << 30 }}
			r := newRouter(h)
			req := httptest.NewRequest("GET", "/api/setup/recommended", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			var body map[string]any
			_ = json.Unmarshal(w.Body.Bytes(), &body)
			if body["model"] != tc.wantMdl {
				t.Errorf("ram=%d: want model %q got %q", tc.ramGB, tc.wantMdl, body["model"])
			}
		})
	}
}

func TestPullModel_MissingModel(t *testing.T) {
	h := &Handlers{OllamaHost: "http://x", HostRAM: func() uint64 { return 16 << 30 }}
	r := newRouter(h)
	req := httptest.NewRequest("POST", "/api/setup/pull-model", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "model") {
		t.Errorf("expected model error, got %s", w.Body.String())
	}
}
