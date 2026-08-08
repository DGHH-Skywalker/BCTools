package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// RegisterStatic registers the embedded frontend SPA static file serving.
func RegisterStatic(r chi.Router, embeddedFiles embed.FS) {
	staticFS, err := fs.Sub(embeddedFiles, "embed/dist")
	if err != nil {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "No embedded frontend dist found", http.StatusNotFound)
		})
		return
	}

	r.Get("/", serveIndex(staticFS))
	r.HandleFunc("/*", serveFileOrIndex(staticFS))
}

func serveIndex(staticFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(staticFS, "index.html")
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(data)
	}
}

func serveFileOrIndex(staticFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}

		data, err := fs.ReadFile(staticFS, p)
		if err != nil {
			data, err = fs.ReadFile(staticFS, "index.html")
			if err != nil {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			w.Header().Set("Cache-Control", "no-cache")
		} else if p == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		ext := filepath.Ext(p)
		switch ext {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		w.Write(data)
	}
}
