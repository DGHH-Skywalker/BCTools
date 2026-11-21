package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestUpdatedUMReactDirectNavigation(t *testing.T) {
	files := fstest.MapFS{
		"um-react/index.html": {Data: []byte(`<head><script>window.BASE_URL = '/um-react/'</script><script src="./assets/index.js"></script></head>`)},
		"um-react/sw.js":      {Data: []byte(`unexpected worker`)},
	}
	for _, path := range []string{"um-react/", "um-react/settings", "um-react/settings/general/"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			serveWebRevision(w, httptest.NewRequest(http.MethodGet, "/", nil), files, "revision", path)
			if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" ||
				!strings.Contains(w.Body.String(), `<base href="/_ui/revision/um-react/">`) ||
				!strings.Contains(w.Body.String(), `window.BASE_URL = '/_ui/revision/um-react/'`) {
				t.Fatalf("GET %s: %d %s", path, w.Code, w.Body)
			}
		})
	}
	for _, path := range []string{"um-react/assets/missing.js", "um-react/sw.js"} {
		w := httptest.NewRecorder()
		serveWebRevision(w, httptest.NewRequest(http.MethodGet, "/", nil), files, "revision", path)
		if w.Code != http.StatusNotFound {
			t.Fatalf("GET %s: %d %s", path, w.Code, w.Body)
		}
	}
}
