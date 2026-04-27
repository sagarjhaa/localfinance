// Package main is the entry point for the LocalFinance single-binary server.
package main

import (
	"context"
	"fmt"
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
	sophiaconfig "github.com/sagarjhaa/localfinance/services/sophia/config"
	"github.com/sagarjhaa/localfinance/services/thesaurus/config"
)

func main() {
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
	if aiCfg.AI.ModelName == "" || strings.EqualFold(aiCfg.AI.ModelName, "auto") {
		selectCtx, selectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		selectClient := &http.Client{Timeout: 10 * time.Second}
		picked, err := ai.SelectBestModel(selectCtx, selectClient, aiCfg.AI.OllamaHost)
		selectCancel()
		if err != nil {
			slog.Error("auto model selection failed", "err", err)
			os.Exit(1)
		}
		slog.Info("auto-selected model", "model", picked)
		aiCfg.AI.ModelName = picked
	}
	aiSvc, err := ai.NewService(aiCfg.AI)
	if err != nil {
		slog.Error("ai service init failed", "err", err)
		os.Exit(1)
	}
	thesaurusURL := os.Getenv("THESAURUS_URL")
	if thesaurusURL == "" {
		thesaurusURL = aiCfg.Thesaurus.BaseURL
	}
	aiSvc.SetThesaurusURL(thesaurusURL)

	if os.Getenv("SKIP_OLLAMA_PROBE") != "1" {
		probeCtx, probeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		probeClient := &http.Client{Timeout: 10 * time.Second}
		if err := ai.Probe(probeCtx, probeClient, aiCfg.AI.OllamaHost, aiCfg.AI.ModelName); err != nil {
			probeCancel()
			slog.Error("ollama probe failed", "err", err)
			os.Exit(1)
		}
		probeCancel()
		slog.Info("ollama probe ok", "model", aiCfg.AI.ModelName, "host", aiCfg.AI.OllamaHost)
	} else {
		slog.Info("ollama probe skipped (SKIP_OLLAMA_PROBE=1)")
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
