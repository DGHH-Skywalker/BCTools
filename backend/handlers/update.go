package handlers

import (
	"net/http"
	"time"

	"broadcast-tool/models"
	"broadcast-tool/response"
	"broadcast-tool/store"
)

type UpdateHandler struct {
	Store *store.Store
}

func NewUpdateHandler(s *store.Store) *UpdateHandler {
	return &UpdateHandler{Store: s}
}

func (h *UpdateHandler) HandleCheck(w http.ResponseWriter, r *http.Request) {
	settings := h.Store.GetSettingsRaw()
	if settings.DownloadURL == "" {
		response.WriteJSON(w, http.StatusOK, models.UpdateCheckResponse{HasUpdate: false})
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(settings.DownloadURL)
	if err != nil {
		response.WriteJSON(w, http.StatusOK, models.UpdateCheckResponse{HasUpdate: false})
		return
	}
	defer resp.Body.Close()
	_ = resp
	// Simplified: just report available version from settings
	response.WriteJSON(w, http.StatusOK, models.UpdateCheckResponse{
		HasUpdate:     false,
		LatestVersion: settings.Version,
		DownloadURL:   settings.DownloadURL,
	})
}
