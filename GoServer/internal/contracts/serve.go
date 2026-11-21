package contracts

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.json
var specification []byte

func Serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(specification)
}
