// Package main is the entry point for the LocalFinance single-binary server.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sagarjhaa/localfinance/internal/ai"
	"github.com/sagarjhaa/localfinance/internal/api"
	"github.com/sagarjhaa/localfinance/internal/data/database"
	"github.com/sagarjhaa/localfinance/internal/postgres"
	sophiaconfig "github.com/sagarjhaa/localfinance/internal/ai/config"
	"github.com/sagarjhaa/localfinance/internal/data/config"
)

// initLogging tees slog output to a file under the data dir AND stderr so the
// .app launched via Finder/`open` (which swallows stderr) still leaves a
// post-mortem log behind. Honors LOCALFINANCE_LOG_FILE for override.
func initLogging() {
	logPath := os.Getenv("LOCALFINANCE_LOG_FILE")
	if logPath == "" {
		logDir := filepath.Join(dataDir(), "logs")
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			// Can't write logs; stderr will have to do.
			return
		}
		logPath = filepath.Join(logDir, "server.log")
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	w := io.MultiWriter(os.Stderr, f)
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})))
}

// ensureBrewInPath prepends Homebrew's bin dirs to PATH so child processes
// (pdftoppm, pdftotext, etc.) can be found when the .app is launched via
// Finder. macOS launchd starts processes with a minimal PATH that omits
// /opt/homebrew/bin and /usr/local/bin, which is where brew puts binaries.
func ensureBrewInPath() {
	current := os.Getenv("PATH")
	for _, p := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		if !strings.Contains(current, p) {
			current = p + string(os.PathListSeparator) + current
		}
	}
	_ = os.Setenv("PATH", current)
}

func main() {
	ensureBrewInPath()
	initLogging()
	// If DB_HOST is unset, run an embedded Postgres for the .app build path.
	// If DB_HOST is set, connect to a host Postgres (dev/Docker).
	if os.Getenv("DB_HOST") == "" {
		dataRoot := dataDir()
		mgr, err := postgres.New(filepath.Join(dataRoot, "postgres"), 0)
		if err != nil {
			slog.Error("postgres init failed", "err", err)
			os.Exit(1)
		}
		if err := mgr.Start(context.Background()); err != nil {
			slog.Error("postgres start failed", "err", err)
			os.Exit(1)
		}
		defer mgr.Stop(context.Background())
		// Set env vars so config.Load() picks up the embedded instance.
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", fmt.Sprintf("%d", mgr.Port()))
		os.Setenv("DB_USER", "postgres")
		os.Setenv("DB_PASSWORD", "postgres")
		os.Setenv("DB_NAME", "localfinance")
		os.Setenv("DB_SSLMODE", "disable")
		slog.Info("embedded postgres up", "port", mgr.Port(), "data", dataRoot)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	db, err := database.Initialize(cfg.Database)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	if err := database.Migrate(db); err != nil {
		slog.Error("db migrate failed", "err", err)
		os.Exit(1)
	}
	if err := database.SeedDefaultUser(db); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}

	// AI service: load Sophia-style config, auto-pick model if needed, probe Ollama.
	aiCfg, err := sophiaconfig.Load()
	if err != nil {
		slog.Error("ai config load failed", "err", err)
		os.Exit(1)
	}
	// First-run setup-mode: if MODEL_NAME=auto and no model is installed yet,
	// or Ollama itself isn't reachable, we DON'T fail-fast. The /api/setup/*
	// wizard is a public route group and will guide the user through Ollama
	// install + model pull. The binary still boots; AI features just won't
	// work until a model is present.
	setupMode := false
	if aiCfg.AI.ModelName == "" || strings.EqualFold(aiCfg.AI.ModelName, "auto") {
		selectCtx, selectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		selectClient := &http.Client{Timeout: 10 * time.Second}
		picked, err := ai.SelectBestModel(selectCtx, selectClient, aiCfg.AI.OllamaHost)
		selectCancel()
		if err != nil {
			slog.Warn("auto model selection deferred — entering setup mode", "err", err)
			setupMode = true
			aiCfg.AI.ModelName = ""
		} else {
			slog.Info("auto-selected model", "model", picked)
			aiCfg.AI.ModelName = picked
		}
	}
	aiSvc, err := ai.NewService(aiCfg.AI)
	if err != nil {
		slog.Error("ai service init failed", "err", err)
		os.Exit(1)
	}
	// In the consolidated binary, AI handlers that fetch transactions through
	// THESAURUS_URL are self-talking — Thesaurus IS this same process. Default
	// the URL to localhost:<PORT> so the in-process HTTP loop hits ourselves.
	// (A future cleanup should switch these to direct GORM calls; tracked.)
	thesaurusURL := os.Getenv("THESAURUS_URL")
	if thesaurusURL == "" {
		listenPort := os.Getenv("PORT")
		if listenPort == "" {
			listenPort = "3001"
		}
		thesaurusURL = "http://localhost:" + listenPort
	}
	aiSvc.SetThesaurusURL(thesaurusURL)

	if os.Getenv("SKIP_OLLAMA_PROBE") == "1" {
		slog.Info("ollama probe skipped (SKIP_OLLAMA_PROBE=1)")
	} else if setupMode {
		slog.Info("running in setup mode — wizard at /setup will guide first-run install")
	} else {
		probeCtx, probeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		probeClient := &http.Client{Timeout: 10 * time.Second}
		err := ai.Probe(probeCtx, probeClient, aiCfg.AI.OllamaHost, aiCfg.AI.ModelName)
		probeCancel()
		if err != nil {
			// Non-fatal: enter setup mode so the wizard can guide the user.
			slog.Warn("ollama probe failed — entering setup mode", "err", err)
			setupMode = true
		} else {
			slog.Info("ollama probe ok", "model", aiCfg.AI.ModelName, "host", aiCfg.AI.OllamaHost)
		}
	}

	router := api.NewGinRouterWithAI(db, aiSvc, aiCfg.AI.ModelName)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	slog.Info("localfinance starting", "port", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}

// dataDir returns the on-disk root for embedded-Postgres data files. Honors
// LOCALFINANCE_DATA_DIR for override; otherwise defaults to the macOS
// Application Support directory.
func dataDir() string {
	if d := os.Getenv("LOCALFINANCE_DATA_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "LocalFinance")
}
