package handlers

import (
	"fmt"
	"net/http"

	"broadcast-tool/network"
	"broadcast-tool/response"

	"github.com/skip2/go-qrcode"
)

// NetworkHandler provides network info and QR codes for mobile access.
type NetworkHandler struct {
	Port int
}

// NewNetworkHandler creates a new NetworkHandler.
func NewNetworkHandler(port int) *NetworkHandler {
	return &NetworkHandler{Port: port}
}

// HandleGetInfo returns the local IP and port for mobile access.
func (h *NetworkHandler) HandleGetInfo(w http.ResponseWriter, r *http.Request) {
	ip := network.GetLocalIP()
	url := fmt.Sprintf("http://%s:%d/", ip, h.Port)
	response.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ip":   ip,
		"port": h.Port,
		"url":  url,
	})
}

// HandleGetQR returns a PNG QR code for the app URL.
func (h *NetworkHandler) HandleGetQR(w http.ResponseWriter, r *http.Request) {
	data := fmt.Sprintf("http://%s:%d/", network.GetLocalIP(), h.Port)

	png, err := qrcode.Encode(data, qrcode.Medium, 256)
	if err != nil {
		response.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(png)
}
