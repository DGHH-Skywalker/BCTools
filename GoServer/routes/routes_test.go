package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"broadcast-tool/handlers"
	"broadcast-tool/internal/requestscope"

	"github.com/go-chi/chi/v5"
)

func TestVersionedRouteContractCoverage(t *testing.T) {
	b, err := os.ReadFile("../internal/contracts/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(b, &spec); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	RegisterRoutes(router, HandlerSet{Lifecycle: handlers.NewLifecycleHandler(nil), Extensions: &handlers.ExtensionsHandler{}})
	actual := map[string]bool{}
	if err := chi.Walk(router, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/api/v1/") || route == "/api/v1/openapi.json" {
			return nil
		}
		route = strings.TrimSuffix(route, "/")
		actual[method+" "+route] = true
		if spec.Paths[route][strings.ToLower(method)] == nil {
			t.Errorf("undocumented: %s %s", method, route)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for route, methods := range spec.Paths {
		for method := range methods {
			if !actual[strings.ToUpper(method)+" "+route] {
				t.Errorf("unregistered: %s %s", method, route)
			}
		}
	}
}

func TestNewAdministrativeRoutesRejectLAN(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, HandlerSet{Extensions: &handlers.ExtensionsHandler{}})
	for _, tc := range []struct {
		addr, origin string
		desktop      bool
	}{
		{"192.0.2.55:1234", "http://localhost:1743", false},
		{"127.0.0.1:5432", "https://unrelated.example", false},
		{"192.0.2.1:1234", "https://unrelated.example", true},
	} {
		request := httptest.NewRequest("POST", "http://localhost:1743/api/v1/plugins/example", nil)
		request.RemoteAddr = tc.addr
		request.Header.Set("X-BCTools-Local", "1")
		request.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		var handler http.Handler = router
		if tc.desktop {
			handler = requestscope.InDesktop(handler)
		}
		handler.ServeHTTP(w, request)
		if w.Code != http.StatusForbidden {
			t.Fatalf("untrusted administrative action accepted: %d", w.Code)
		}
	}
}
