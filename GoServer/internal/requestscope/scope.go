package requestscope

import (
	"context"
	"net"
	"net/http"

	"broadcast-tool/network"
)

type desktopKey struct{}

func Desktop(r *http.Request) bool {
	value, _ := r.Context().Value(desktopKey{}).(bool)
	return value
}

func Local(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	return Desktop(r) || (err == nil && network.IsLocalhost(host))
}

func InDesktop(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), desktopKey{}, true)))
	})
}
