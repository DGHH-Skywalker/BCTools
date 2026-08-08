package handlers

import (
	"net/http"
	"time"

	"broadcast-tool/models"
	"broadcast-tool/response"
	"broadcast-tool/store/settingstore"
)

type UpdateHandler struct {
	Settings *settingstore.SettingsStore
}

func NewUpdateHandler(settings *settingstore.SettingsStore) *UpdateHandler {
	return &UpdateHandler{Settings: settings}
}

func (h *UpdateHandler) HandleCheck(w http.ResponseWriter, r *http.Request) {
	settings := h.Settings.GetSettingsRaw()
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
