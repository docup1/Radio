package common

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

const MaxPageLimit = 100

func DecodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func ParseID(r *http.Request, name string) (uuid.UUID, error) {
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

func ParsePagination(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

func UUIDQuery(r *http.Request, name string) (uuid.UUID, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return uuid.Nil, errors.New("missing " + name)
	}
	return uuid.Parse(v)
}
