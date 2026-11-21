package handlers

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"broadcast-tool/internal/requestscope"
	"broadcast-tool/network"
	"broadcast-tool/platform"
	"broadcast-tool/response"

	"github.com/skip2/go-qrcode"
)

// NetworkHandler provides network info and QR codes for mobile access.
type NetworkHandler struct {
	Port    int
	failure atomic.Pointer[string]
}

// NewNetworkHandler creates a new NetworkHandler.
func NewNetworkHandler(port int) *NetworkHandler {
	return &NetworkHandler{Port: port}
}

func (h *NetworkHandler) SetUnavailable(err error) {
	message := err.Error()
	h.failure.Store(&message)
}

// HandleGetInfo returns the local IP, port, mobile URL, and server/client OS info.
func (h *NetworkHandler) HandleGetInfo(w http.ResponseWriter, r *http.Request) {
	ip := network.GetLocalIP()
	url := fmt.Sprintf("http://%s:%d/", ip, h.Port)

	errorMessage := ""
	if failure := h.failure.Load(); failure != nil {
		url = ""
		errorMessage = *failure
	}
	isLocalRequest := requestscope.Local(r)
	isWindows, windowsVersion := platform.WindowsInfo()

	response.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ip":             ip,
		"port":           h.Port,
		"url":            url,
		"error":          errorMessage,
		"isLocalRequest": isLocalRequest,
		"isWindows":      isWindows,
		"windowsVersion": windowsVersion,
	})
}

// HandleGetQR returns a PNG QR code for the app URL.
func (h *NetworkHandler) HandleGetQR(w http.ResponseWriter, r *http.Request) {
	if h.failure.Load() != nil {
		response.WriteError(w, http.StatusServiceUnavailable, "LAN_UNAVAILABLE", "局域网服务未启动")
		return
	}
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
