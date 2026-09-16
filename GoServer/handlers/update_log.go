package handlers

import (
	"net/http"

	"broadcast-tool/response"
	"broadcast-tool/services"
)

type UpdateLogHandler struct {
	service *services.UpdateLogService
}

func NewUpdateLogHandler(service *services.UpdateLogService) *UpdateLogHandler {
	return &UpdateLogHandler{service: service}
}

func (h *UpdateLogHandler) HandleLatest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, h.service.Latest())
}
