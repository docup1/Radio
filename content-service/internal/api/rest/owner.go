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

// OwnerID extracts the requesting user's uuid from the X-Owner-ID header. The
// public server sits behind the gateway which authenticates the user and
// forwards the identity, so the header is treated as authoritative.
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
