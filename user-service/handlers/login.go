package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"radio/user-service/infra"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

const selectUserByUsername = `
	SELECT id, password
	FROM users
	WHERE username = $1`

// LoginHandler authenticates a user and issues a session JWT.
//
//	@Summary	Login
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		loginRequest	true	"Login payload"
//	@Success	200		{object}	loginResponse
//	@Failure	400		{object}	map[string]string
//	@Failure	401		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/login [post]
func LoginHandler(db *sql.DB, cfg *infra.Config, hasher *infra.Hasher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			infra.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Username == "" || req.Password == "" {
			infra.WriteError(w, http.StatusBadRequest, "username and password are required")
			return
		}

		var id, hash string
		err := db.QueryRowContext(r.Context(), selectUserByUsername, req.Username).Scan(&id, &hash)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			infra.WriteError(w, http.StatusUnauthorized, "invalid username or password")
			return
		case err != nil:
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if hasher.Compare(r.Context(), []byte(hash), []byte(req.Password)) != nil {
			infra.WriteError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}

		signed, err := infra.IssueToken(db, cfg, id, req.Username)
		if err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		infra.WriteJSON(w, http.StatusOK, loginResponse{Token: signed})
	}
}
