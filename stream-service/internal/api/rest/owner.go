package rest

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
)

var (
	ErrMissingOwner = errors.New("missing X-Owner-ID header")
	ErrInvalidOwner = errors.New("invalid X-Owner-ID header")
)

func OwnerID(r *http.Request) (uuid.UUID, error) {
	v := r.Header.Get("X-Owner-ID")
	if v == "" {
		return uuid.Nil, ErrMissingOwner
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, ErrInvalidOwner
	}
	return id, nil
}

func ownerOrError(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := OwnerID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return uuid.Nil, false
	}
	return id, true
}
