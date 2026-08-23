package handlers

import (
	"errors"
	"io/fs"
	"net/http"

	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
)

// SongAudio streams the audio (melody) of a specific song to the caller in
// HTTP byte-range batches. Access is enforced via the user_id query parameter:
// the song must be public or owned by that user.
func SongAudio(svc *application.Services, files interfaces.FileOpener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		songID, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		userID, err := uuidQuery(r, "user_id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, "missing user_id")
			return
		}
		song, err := svc.Songs.Get(r.Context(), songID, userID)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		melody, err := svc.Melodies.Get(r.Context(), song.MelodyID)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		rc, _, modTime, err := files.Open(melody.Path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, http.StatusNotFound, "file not found")
				return
			}
			WriteError(w, http.StatusInternalServerError, "cannot open file")
			return
		}
		defer rc.Close()
		w.Header().Set("Content-Type", melody.ContentType)
		w.Header().Set("Accept-Ranges", "bytes")
		http.ServeContent(w, r, melody.Path, modTime, rc)
	}
}
