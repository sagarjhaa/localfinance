// Package api wires the unified HTTP surface for the localfinance binary.
// Replaces the per-service Gin routers from the old 5-service stack.
package api

import (
	"net/http"
)

// NewRouter returns the top-level mux. As phases land, more routes get
// registered here. For now: just /health so cmd/localfinance can boot.
func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"localfinance"}`))
	})
	return mux
}
