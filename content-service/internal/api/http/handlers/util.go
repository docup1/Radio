package handlers

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
)

func parseID(r *http.Request, name string) (uuid.UUID, error) {
	v := r.PathValue(name)
	if v == "" {
		return uuid.Nil, errors.New("missing id")
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, errors.New("invalid id")
	}
	return id, nil
}

func uuidQuery(r *http.Request, name string) (uuid.UUID, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return uuid.Nil, errors.New("missing " + name)
	}
	return uuid.Parse(v)
}
