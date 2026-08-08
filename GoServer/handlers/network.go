package handlers

import (
	"fmt"
	"net/http"

	"broadcast-tool/hotspot"
	"broadcast-tool/network"
	"broadcast-tool/response"

	"github.com/skip2/go-qrcode"
)

// NetworkHandler provides network info, QR codes, and mobile hotspot control.
type NetworkHandler struct {
	Port    int
	Hotspot *hotspot.Manager
}

// NewNetworkHandler creates a new NetworkHandler.
func NewNetworkHandler(port int, hotspot *hotspot.Manager) *NetworkHandler {
	return &NetworkHandler{Port: port, Hotspot: hotspot}
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

// HandleGetQR returns a PNG QR code for the app URL or the hotspot WiFi.
func (h *NetworkHandler) HandleGetQR(w http.ResponseWriter, r *http.Request) {
	qrType := r.URL.Query().Get("type")
	var data string
	switch qrType {
	case "hotspot":
		data = fmt.Sprintf("WIFI:T:WPA;S:%s;P:%s;;", h.Hotspot.SSID(), h.Hotspot.Password())
	case "app", "":
		data = fmt.Sprintf("http://%s:%d/", network.GetLocalIP(), h.Port)
	default:
		response.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid type"})
		return
	}

	png, err := qrcode.Encode(data, qrcode.Medium, 256)
	if err != nil {
		response.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(png)
}

// HandleStartHotspot requests to start the mobile hotspot.
func (h *NetworkHandler) HandleStartHotspot(w http.ResponseWriter, r *http.Request) {
	if err := h.Hotspot.Start(); err != nil {
		response.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	response.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "starting"})
}

// HandleGetHotspotStatus returns the current hotspot state.
func (h *NetworkHandler) HandleGetHotspotStatus(w http.ResponseWriter, r *http.Request) {
	status, ssid, password, message := h.Hotspot.Snapshot()
	response.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":   status,
		"ssid":     ssid,
		"password": password,
		"message":  message,
		"fallback": "ms-settings:network-mobilehotspot",
	})
}
