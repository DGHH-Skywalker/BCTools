package securelocal

import (
	"net/http"
	"net/url"
	"strings"

	"broadcast-tool/internal/requestscope"
	"broadcast-tool/response"
)

// Require protects new administrative endpoints in the browser and desktop.
func Require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		if origin != "" {
			if parsed, err := url.Parse(origin); err == nil {
				allowed = (parsed.Scheme == "http" || parsed.Scheme == "https") && strings.EqualFold(parsed.Host, r.Host)
			}
		} else {
			allowed = r.Header.Get("Sec-Fetch-Site") != "cross-site"
		}
		if !requestscope.Local(r) || !allowed || r.Header.Get("X-BCTools-Local") != "1" {
			response.WriteError(w, http.StatusForbidden, "LOCAL_REQUIRED", "请在本机 BCTools 界面执行此操作")
			return
		}
		next(w, r)
	}
}
