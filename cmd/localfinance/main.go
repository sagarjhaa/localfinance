// Package main is the entry point for the LocalFinance single-binary server.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/sagarjhaa/localfinance/internal/api"
	"github.com/sagarjhaa/localfinance/internal/data/database"
	"github.com/sagarjhaa/localfinance/services/thesaurus/config"
)

func main() {
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

	router := api.NewGinRouter(db)

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
