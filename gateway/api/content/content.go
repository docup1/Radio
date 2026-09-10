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

	rest := strings.TrimPrefix(r.URL.Path, "/api/content")
	if r.Method == http.MethodGet && publicReadPath(rest) {
		// Public read endpoints (song meta, image files) work anonymously too so
		// the public stream feed can resolve names and thumbnails by guid. The
		// uuid.Nil owner makes content-service enforce visibility itself:
		// private songs/images are simply not visible to the Nil owner.
		if token != "" {
			if uid, err := h.auth.Validate(r.Context(), token); err == nil {
				r.Header.Set("X-Owner-ID", uid)
			}
		}
		// if r.Header.Get("X-Owner-ID") == "" {
		// 	r.Header.Set("X-Owner-ID", "00000000-0000-0000-0000-000000000000")
		// }
		h.restProxy.ServeHTTP(w, r)
		return
	}

	if token == "" {
		infra.WriteError(w, http.StatusUnauthorized, "missing authentication")
		return
	}
	uid, err := h.auth.Validate(r.Context(), token)
	if err != nil {
		infra.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

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

// publicReadPath reports whether an unauthenticated GET is allowed for the
// given path (relative to /api/content).
func publicReadPath(path string) bool {
	if isSongMetaPath(path) || isImageFilePath(path) {
		return true
	}
	return false
}

// isSongMetaPath matches GET /songs/{id} (not audio).
func isSongMetaPath(path string) bool {
	const prefix = "/songs/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" || strings.Contains(rest, "/") {
		return false
	}
	return !strings.HasSuffix(path, "/audio")
}

// isImageFilePath matches GET /images/{id}/file.
func isImageFilePath(path string) bool {
	const prefix = "/images/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	rest := strings.TrimPrefix(path, prefix)
	return strings.HasSuffix(rest, "/file") && !strings.Contains(strings.TrimSuffix(rest, "/file"), "/")
}
