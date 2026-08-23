package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"radio/content-service/internal/domain/interfaces"
)

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// WriteServiceError maps domain errors to HTTP status codes.
func WriteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, interfaces.ErrNotFound):
		WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, interfaces.ErrConflict):
		WriteError(w, http.StatusConflict, "conflict")
	case errors.Is(err, interfaces.ErrForbidden):
		WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, interfaces.ErrInvalid):
		WriteError(w, http.StatusBadRequest, "invalid input")
	default:
		WriteError(w, http.StatusInternalServerError, "internal error")
	}
}
