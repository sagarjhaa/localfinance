// Package webui serves the embedded React build.
package webui

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// Handler serves the embedded React build. SPA routing: any unknown path
// returns index.html so React Router can handle it client-side. /api/* paths
// MUST be handled by the parent router before falling through to this.
func Handler() http.Handler {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "webui dist missing — run `make webui-build`", http.StatusInternalServerError)
		})
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip leading slash to lookup against embed.FS
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			serveIndex(w, r, sub)
			return
		}
		if f, err := sub.Open(p); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA fallback
		serveIndex(w, r, sub)
	})
}

func serveIndex(w http.ResponseWriter, _ *http.Request, sub fs.FS) {
	f, err := sub.Open("index.html")
	if err != nil {
		http.Error(w, "index.html missing", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}
