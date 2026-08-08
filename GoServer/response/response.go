package response

import (
	"encoding/json"
	"net/http"

	"broadcast-tool/models"
)

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// WriteSuccess writes a simple success response
func WriteSuccess(w http.ResponseWriter) {
	WriteJSON(w, http.StatusOK, models.SuccessResponse{Success: true})
}

// WriteCreated writes a 201 response with data
func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, data)
}

// WriteNoContent writes a 204 No Content response
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteError writes a standard error response
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// WriteValidationError writes a 400 validation error
func WriteValidationError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", message)
}

// WriteNotFoundError writes a 404 not found error
func WriteNotFoundError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, "NOT_FOUND", message)
}

// WriteInternalError writes a 500 internal error
func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}
