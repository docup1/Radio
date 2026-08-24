package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"radio/user-service/infra"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

const insertUser = `
	INSERT INTO users (id, username, password)
	VALUES ($1, $2, $3)
	RETURNING id`

// RegisterHandler registers a new user.
//
//	@Summary	Register a new user
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		registerRequest	true	"Registration payload"
//	@Success	201		{object}	registerResponse
//	@Failure	400		{object}	map[string]string
//	@Failure	409		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/register [post]
func RegisterHandler(db *sql.DB, cfg *infra.Config, hasher *infra.Hasher) http.HandlerFunc {
	usernamePattern := regexp.MustCompile(fmt.Sprintf(
		`^[a-zA-Z0-9_]{%d,%d}$`,
		cfg.Validation.UsernameMinLength,
		cfg.Validation.UsernameMaxLength,
	))

	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			infra.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if !usernamePattern.MatchString(req.Username) {
			infra.WriteError(w, http.StatusBadRequest, fmt.Sprintf(
				"username must be %d-%d characters: letters, digits, underscore",
				cfg.Validation.UsernameMinLength,
				cfg.Validation.UsernameMaxLength,
			))
			return
		}
		if !cfg.Validation.ValidPassword(req.Password) {
			infra.WriteError(w, http.StatusBadRequest, fmt.Sprintf(
				"password must be %d-%d characters",
				cfg.Validation.PasswordMinLength,
				cfg.Validation.PasswordMaxLength,
			))
			return
		}

		hash, err := hasher.Generate(r.Context(), req.Password, cfg.Bcrypt.Cost)
		if err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		id, err := uuid.NewV7()
		if err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		var createdID string
		err = db.QueryRowContext(r.Context(), insertUser, id, req.Username, string(hash)).Scan(&createdID)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				infra.WriteError(w, http.StatusConflict, "username already taken")
				return
			}
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		signed, err := infra.IssueToken(db, cfg, createdID, req.Username)
		if err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		infra.WriteJSON(w, http.StatusCreated, registerResponse{ID: createdID, Username: req.Username, Token: signed})
	}
}
