package song

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

func Search(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		q := r.URL.Query().Get("q")
		if q == "" {
			common.WriteError(w, http.StatusBadRequest, "missing q parameter")
			return
		}
		limit, offset := common.ParsePagination(r)
		songs, err := svc.Songs.Search(r.Context(), q, owner, limit, offset)
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, songs)
	}
}
