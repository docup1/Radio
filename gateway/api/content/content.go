package content

import (
	"net/http"
	"strings"

	"radio/gateway/infra"
)

// Content godoc
//
//	@Summary	Proxy to content-service
//	@Tags		content
//	@Security	bearerAuth
//	@Produce	json
//	@Success	200
//	@Failure	401	{object}	ErrorResponse
//	@Router		/api/content/songs [get]
//	@Router		/api/content/songs [post]
//	@Router		/api/content/songs/{id} [get]
//	@Router		/api/content/songs/{id}/audio [get]
func (h *Handler) content(w http.ResponseWriter, r *http.Request) {
	token := h.auth.ExtractToken(r)

	// Never trust client-supplied identity headers.
	r.Header.Del("X-Owner-ID")
	r.Header.Del("Authorization")
	if token == "" {
		infra.WriteError(w, http.StatusUnauthorized, "missing authentication")
		return
	}
	uid, err := h.auth.Validate(r.Context(), token)
	if err != nil {
		infra.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/api/content")
	target := h.restProxy
	if infra.IsAudioPath(rest) {
		target = h.streamProxy
		// stream service expects user_id as query param (legacy) and/or X-Owner-ID header;
		// set both for compatibility.
		q := r.URL.Query()
		q.Set("user_id", uid)
		r.URL.RawQuery = q.Encode()
	}
	r.Header.Set("X-Owner-ID", uid)
	target.ServeHTTP(w, r)
}
