package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
)

// RegisterUMReact registers the embedded um-react (Unlock Music) SPA at /um-react/.
// It must be registered before the catch-all static handler (RegisterStatic) so
// that /um-react/* takes precedence over the /* fallback.
func RegisterUMReact(r chi.Router, files embed.FS) {
	staticFS, err := fs.Sub(files, "embed/um-react")
	if err != nil {
		notEmbedded := func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "um-react not embedded", http.StatusNotFound)
		}
		r.Get("/um-react", notEmbedded)
		r.HandleFunc("/um-react/*", notEmbedded)
		return
	}

	r.Get("/um-react", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/um-react/", http.StatusFound)
	})
	r.HandleFunc("/um-react/*", serveUMReact(staticFS))
}

// serveUMReact serves um-react static files with SPA fallback to index.html
// (so client-side routes like /um-react/settings survive a refresh).
func serveUMReact(staticFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/um-react/")
		if p == "" {
			p = "index.html"
		}
		p = path.Clean(p)

		data, err := fs.ReadFile(staticFS, p)
		if err != nil {
			// SPA fallback for unknown paths
			data, err = fs.ReadFile(staticFS, "index.html")
			if err != nil {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			w.Header().Set("Cache-Control", "no-cache")
			setUMReactContentType(w, ".html")
			w.Write(data)
			return
		}

		if p == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		setUMReactContentType(w, path.Ext(p))
		w.Write(data)
	}
}

// setUMReactContentType sets the Content-Type for um-react assets. The .wasm
// MIME is required for WebAssembly instantiation (e.g. @unlock-music/crypto).
func setUMReactContentType(w http.ResponseWriter, ext string) {
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".js", ".mjs":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".wasm":
		w.Header().Set("Content-Type", "application/wasm")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
}
