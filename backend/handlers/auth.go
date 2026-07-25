package handlers

import (
	"encoding/json"
	"net/http"

	"broadcast-tool/models"
	"broadcast-tool/response"
	"broadcast-tool/store"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	Store *store.Store
}

func NewAuthHandler(s *store.Store) *AuthHandler {
	return &AuthHandler{Store: s}
}

func (h *AuthHandler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	hash := h.Store.GetAdminPasswordHash()
	if hash == "" {
		response.WriteJSON(w, http.StatusOK, models.AuthResponse{Success: true})
		return
	}
	hint := h.Store.GetSettings().AdminPasswordHint
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		response.WriteJSON(w, http.StatusOK, models.AuthResponse{Success: false, Hint: hint})
		return
	}
	response.WriteJSON(w, http.StatusOK, models.AuthResponse{Success: true})
}
