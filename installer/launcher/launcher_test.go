package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseArgsDefaults(t *testing.T) {
	a, err := parseArgs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.ChatModel != defaultChatModel {
		t.Fatalf("default chat model = %q, want %q", a.ChatModel, defaultChatModel)
	}
	if a.SkipBrowser {
		t.Fatal("skip-browser should default false")
	}
	if a.OllamaURL != defaultOllamaURL {
		t.Fatalf("default ollama url = %q", a.OllamaURL)
	}
}

func TestParseArgsOverrides(t *testing.T) {
	a, err := parseArgs([]string{
		"--chat-model", "qwen2.5:14b",
		"--skip-browser",
		"--data-dir", "/tmp/lf",
		"--ollama-url", "http://127.0.0.1:9999",
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.ChatModel != "qwen2.5:14b" {
		t.Fatalf("chat model = %q", a.ChatModel)
	}
	if !a.SkipBrowser {
		t.Fatal("skip-browser not set")
	}
	if a.DataDir != "/tmp/lf" {
		t.Fatalf("data dir = %q", a.DataDir)
	}
	if a.OllamaURL != "http://127.0.0.1:9999" {
		t.Fatalf("ollama url = %q", a.OllamaURL)
	}
}

func TestParseArgsBadFlag(t *testing.T) {
	if _, err := parseArgs([]string{"--bogus"}); err == nil {
		t.Fatal("expected error on bogus flag")
	}
}

func TestFindResourcesBundleLayout(t *testing.T) {
	tmp := t.TempDir()
	contents := filepath.Join(tmp, "LocalFinance.app", "Contents")
	macOS := filepath.Join(contents, "MacOS")
	resources := filepath.Join(contents, "Resources")
	binDir := filepath.Join(resources, "bin")
	pgBin := filepath.Join(binDir, "postgres", "bin")
	for _, d := range []string{macOS, resources, binDir, pgBin, filepath.Join(resources, "iris-static")} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	exe := filepath.Join(macOS, "launcher")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"node", "iris-server.js"} {
		if err := os.WriteFile(filepath.Join(binDir, f), []byte("x"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"initdb", "pg_ctl", "postgres"} {
		if err := os.WriteFile(filepath.Join(pgBin, f), []byte("x"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	dataDir := filepath.Join(tmp, "data")
	r, err := findResources(exe, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if r.BinDir != binDir {
		t.Fatalf("BinDir = %q, want %q", r.BinDir, binDir)
	}
	if r.PgBinDir != pgBin {
		t.Fatalf("PgBinDir = %q, want %q", r.PgBinDir, pgBin)
	}
	if r.NodeBin != filepath.Join(binDir, "node") {
		t.Fatalf("NodeBin = %q", r.NodeBin)
	}
	if r.IrisServer != filepath.Join(binDir, "iris-server.js") {
		t.Fatalf("IrisServer = %q", r.IrisServer)
	}
	if r.DataDir != dataDir {
		t.Fatalf("DataDir = %q", r.DataDir)
	}
	if !strings.HasSuffix(r.LogDir, "logs") {
		t.Fatalf("LogDir = %q", r.LogDir)
	}
}

func TestFreePort(t *testing.T) {
	p, err := freePort()
	if err != nil {
		t.Fatal(err)
	}
	if p <= 0 || p > 65535 {
		t.Fatalf("port out of range: %d", p)
	}
	// Verify we can listen on it (it was free at the moment of return).
	l, err := net.Listen("tcp", "127.0.0.1:"+itoa(p))
	if err != nil {
		// Acceptable race: another process grabbed the port. Just warn.
		t.Logf("port %d no longer free: %v", p, err)
		return
	}
	l.Close()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func TestWaitForHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	if err := waitForHealthy(srv.URL, 2*time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestWaitForHealthyTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	err := waitForHealthy(srv.URL, 700*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestHasModel(t *testing.T) {
	tags := []string{"llama3.1:8b", "qwen2.5:14b"}
	if !hasModel(tags, "llama3.1:8b") {
		t.Fatal("expected match")
	}
	if hasModel(tags, "llama3.2:1b") {
		t.Fatal("unexpected match")
	}
}

func TestOllamaTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{"name": "llama3.1:8b"},
				{"name": "qwen2.5:14b"},
			},
		})
	}))
	defer srv.Close()
	tags, err := ollamaTags(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0] != "llama3.1:8b" {
		t.Fatalf("tags = %v", tags)
	}
}
