package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestServeUMReact 验证 um-react 静态托管的关键行为：根路径返回 index.html、
// 未知路径 SPA 回退到 index.html、wasm 资源带正确的 Content-Type。
//
// 依赖 GoServer/embed/um-react 下由 npm 构建流程生成的真实产物。
func TestServeUMReact(t *testing.T) {
	dir := filepath.Join("..", "embed", "um-react")
	assetsDir := filepath.Join(dir, "assets")
	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		t.Skipf("um-react 未构建（%s 不存在），跳过：请运行 npm run build:um-react", assetsDir)
	}
	var wasmName string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".wasm") {
			wasmName = e.Name()
			break
		}
	}
	if wasmName == "" {
		t.Skip("未在 assets/ 找到 wasm 文件，跳过")
	}

	handler := serveUMReact(os.DirFS(dir))

	cases := []struct {
		name     string
		path     string
		wantCT   string
		wantBody string
	}{
		{"根路径 index", "/um-react/", "text/html", "window.BASE_URL = '/um-react/'"},
		{"SPA 回退", "/um-react/settings", "text/html", "window.BASE_URL = '/um-react/'"},
		{"wasm 资源", "/um-react/assets/" + wasmName, "application/wasm", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, c.path, nil)
			rec := httptest.NewRecorder()
			handler(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want 200", res.StatusCode)
			}
			if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, c.wantCT) {
				t.Errorf("Content-Type = %q, want prefix %q", ct, c.wantCT)
			}
			if c.wantBody != "" {
				body, _ := io.ReadAll(res.Body)
				if !strings.Contains(string(body), c.wantBody) {
					t.Errorf("响应体未包含 %q", c.wantBody)
				}
			}
		})
	}
}
