package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"broadcast-tool/internal/contracts"
	"broadcast-tool/internal/plugins"
	"broadcast-tool/internal/repo"
	"broadcast-tool/internal/updates"
	"broadcast-tool/models"
	"broadcast-tool/response"
	"broadcast-tool/services"

	"github.com/go-chi/chi/v5"
)

type ExtensionsHandler struct {
	Repository *repo.Repository
	Library    *services.MusicLibrary
	Plugins    *plugins.Registry
	Updates    *updates.Manager
	Stop       func()
	Desktop    bool
}

func (h *ExtensionsHandler) Retention(w http.ResponseWriter, r *http.Request) {
	settings := h.Repository.Retention()
	entries, err := h.Repository.Recoveries()
	if err != nil {
		extensionError(w, err)
		return
	}
	type recovery struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"createdAt"`
		SongCount int       `json:"songCount"`
		FileCount int       `json:"fileCount"`
		Restored  bool      `json:"restored"`
	}
	report := make([]recovery, 0, len(entries))
	for _, entry := range entries {
		report = append(report, recovery{entry.ID, entry.CreatedAt, entry.SongCount, entry.FileCount, entry.Restored})
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"settings": settings, "recoveries": report})
}

func (h *ExtensionsHandler) SetRetention(w http.ResponseWriter, r *http.Request) {
	var settings contracts.RetentionSettings
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&settings); err != nil {
		extensionError(w, err)
		return
	}
	if err := h.Repository.SetRetention(models.RetentionSettings{AudioWeeks: settings.AudioWeeks, MetadataYears: settings.MetadataYears}); err != nil {
		extensionError(w, err)
		return
	}
	h.Retention(w, r)
}

func (h *ExtensionsHandler) Collect(w http.ResponseWriter, r *http.Request) {
	if err := h.Library.CollectRetiredSongs(time.Now()); err != nil {
		extensionError(w, err)
		return
	}
	h.Retention(w, r)
}

func (h *ExtensionsHandler) Restore(w http.ResponseWriter, r *http.Request) {
	if err := h.Library.RestoreRetiredSongs(chi.URLParam(r, "id"), time.Now()); err != nil {
		extensionError(w, err)
		return
	}
	h.Retention(w, r)
}

func (h *ExtensionsHandler) PluginState(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, h.Plugins.Snapshot())
}

func (h *ExtensionsHandler) InstallPlugin(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, plugins.MaxPackageSize))
	if err == nil {
		_, err = h.Plugins.Install(r.Context(), b)
	}
	if err != nil {
		extensionError(w, err)
		return
	}
	h.PluginState(w, r)
}

func (h *ExtensionsHandler) ConfigurePlugin(w http.ResponseWriter, r *http.Request) {
	var request contracts.PluginConfig
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&request); err != nil {
		extensionError(w, err)
		return
	}
	if err := h.Plugins.Configure(chi.URLParam(r, "id"), request.Enabled, request.Layout); err != nil {
		extensionError(w, err)
		return
	}
	h.PluginState(w, r)
}

func (h *ExtensionsHandler) RemovePlugin(w http.ResponseWriter, r *http.Request) {
	if err := h.Plugins.Remove(chi.URLParam(r, "id")); err != nil {
		extensionError(w, err)
		return
	}
	h.PluginState(w, r)
}

func (h *ExtensionsHandler) CallPlugin(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		extensionError(w, err)
		return
	}
	out, err := h.Plugins.Call(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "command"), b)
	if err != nil {
		extensionError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"output": string(out)})
}

func (h *ExtensionsHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, h.Updates.Status())
}

func (h *ExtensionsHandler) UpdateResources(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, h.Updates.Resources())
}

func (h *ExtensionsHandler) CheckUpdates(w http.ResponseWriter, r *http.Request) {
	if err := h.Updates.Check(r.Context()); err != nil {
		extensionError(w, err)
		return
	}
	h.UpdateStatus(w, r)
}

func (h *ExtensionsHandler) ApplyUpdate(w http.ResponseWriter, r *http.Request) {
	if err := h.Updates.Apply(r.Context(), chi.URLParam(r, "id")); err != nil {
		extensionError(w, err)
		return
	}
	h.UpdateStatus(w, r)
}

func (h *ExtensionsHandler) ConfirmWebUpdate(w http.ResponseWriter, r *http.Request) {
	if err := h.Updates.ConfirmWeb(chi.URLParam(r, "digest")); err != nil {
		extensionError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ExtensionsHandler) RollbackWeb(w http.ResponseWriter, r *http.Request) {
	if err := h.Updates.RollbackWeb(); err != nil {
		extensionError(w, err)
		return
	}
	h.UpdateStatus(w, r)
}

func (h *ExtensionsHandler) RestartForUpdate(w http.ResponseWriter, r *http.Request) {
	if h.Stop == nil {
		extensionError(w, fmt.Errorf("应用退出不可用"))
		return
	}
	if err := h.Updates.RestartIntoUpdate(r.Context(), h.Desktop); err != nil {
		extensionError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "restarting"})
	time.AfterFunc(200*time.Millisecond, h.Stop)
}

func extensionError(w http.ResponseWriter, err error) {
	response.WriteError(w, http.StatusBadRequest, "OPERATION_FAILED", err.Error())
}

func (h *ExtensionsHandler) InstallExample(w http.ResponseWriter, r *http.Request) {
	b, err := plugins.ExamplePackage()
	if err == nil {
		_, err = h.Plugins.Install(r.Context(), b)
	}
	if err != nil {
		extensionError(w, err)
		return
	}
	h.PluginState(w, r)
}
