package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"radio/stream-service/internal/domain/models"
)

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

func WriteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, models.ErrConflict):
		WriteError(w, http.StatusConflict, "conflict")
	case errors.Is(err, models.ErrForbidden):
		WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, models.ErrInvalid):
		WriteError(w, http.StatusBadRequest, "invalid input")
	default:
		WriteError(w, http.StatusInternalServerError, "internal error")
	}
}
