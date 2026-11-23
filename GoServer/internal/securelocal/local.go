package securelocal

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"broadcast-tool/internal/requestscope"
	"broadcast-tool/network"
	"broadcast-tool/response"
)

// RequireLoopbackHost also rejects DNS rebinding for private local data.
func RequireLoopbackHost(next http.HandlerFunc) http.HandlerFunc {
	return Require(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = strings.Trim(r.Host, "[]")
		}
		if !requestscope.Desktop(r) && !network.IsLocalhost(strings.ToLower(host)) {
			response.WriteError(w, http.StatusForbidden, "LOCAL_REQUIRED", "请使用本机 BCTools 地址")
			return
		}
		next(w, r)
	})
}

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
