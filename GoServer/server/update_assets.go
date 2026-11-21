package server

import (
	"bytes"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"broadcast-tool/internal/updates"
)

const webPrefix = "/_ui/"

func WithUpdateAssets(next http.Handler, manager *updates.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		switch r.URL.Path {
		case "/", "/index.html", "/um-react", "/um-react/", "/um-react/index.html":
			revision, files := manager.Web()
			if files != nil {
				url := webPrefix + revision + "/"
				if strings.HasPrefix(r.URL.Path, "/um-react") {
					url += "um-react/"
				}
				w.Header().Set("Cache-Control", "no-store")
				http.Redirect(w, r, url, http.StatusTemporaryRedirect)
				return
			}
		}
		if revisionPath, ok := strings.CutPrefix(r.URL.Path, webPrefix); ok {
			revision, asset, found := strings.Cut(revisionPath, "/")
			files := manager.WebVersion(revision)
			if !found || files == nil {
				w.Header().Set("Cache-Control", "no-store")
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return
			}
			serveWebRevision(w, r, files, revision, asset)
			return
		}
		if route, ok := strings.CutPrefix(r.URL.Path, "/um-react/"); ok && path.Ext(route) == "" && fs.ValidPath(strings.TrimSuffix(route, "/")) {
			if revision, files := manager.Web(); files != nil {
				target := webPrefix + revision + "/um-react/" + strings.TrimPrefix(r.URL.EscapedPath(), "/um-react/")
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				w.Header().Set("Cache-Control", "no-store")
				http.Redirect(w, r, target, http.StatusTemporaryRedirect)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func serveWebRevision(w http.ResponseWriter, r *http.Request, files fs.FS, revision, asset string) {
	_, um := strings.CutPrefix(asset, "um-react/")
	spaRoute := um && path.Ext(strings.TrimSuffix(asset, "/")) == ""
	if !um {
		asset = "main/" + asset
	}
	if strings.HasSuffix(asset, "/") {
		asset += "index.html"
	}
	if !fs.ValidPath(asset) || path.Base(asset) == "sw.js" || path.Base(asset) == "registerSW.js" {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(files, asset)
	if err != nil && spaRoute {
		asset = "um-react/index.html"
		data, err = fs.ReadFile(files, asset)
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if asset == "um-react/index.html" {
		base := webPrefix + revision + "/um-react/"
		data = bytes.ReplaceAll(data, []byte("window.BASE_URL = '/um-react/'"), []byte("window.BASE_URL = '"+base+"'"))
		data = umReactDocumentBase(data, base)
	}
	if contentType := mime.TypeByExtension(path.Ext(asset)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		setUMReactContentType(w, path.Ext(asset))
	}
	if strings.HasSuffix(asset, ".html") {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}
