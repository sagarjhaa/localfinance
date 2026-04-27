// Package main is the entry point for the LocalFinance single-binary server.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/sagarjhaa/localfinance/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	slog.Info("localfinance starting", "port", port)
	if err := http.ListenAndServe(":"+port, api.NewRouter()); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
