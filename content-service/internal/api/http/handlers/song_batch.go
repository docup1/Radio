package handlers

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
)

// SongAudio streams the audio (melody) of a specific song to the caller in
// HTTP byte-range batches. Access is enforced via the user_id query parameter:
// the song must be public or owned by that user.
//
// Supports:
//   - Range requests (bytes=start-end) for chunked delivery
//   - ETag (weak, based on size+modtime) with If-None-Match → 304
//   - Content-Length (total file size) in every response
//   - 416 Range Not Satisfiable when range is out of bounds
//   - Cache-Control: private, max-age=0, must-revalidate
//
//	@Summary	Stream song audio
//	@Tags		songs
//	@Produce	audio/mpeg
//	@Param		id			path	string	true	"Song ID"
//	@Param		user_id		query	string	true	"Requesting user UUID (must own the song or song must be public)"
//	@Success	200
//	@Failure	304
//	@Failure	400			{object}	map[string]string
//	@Failure	416
//	@Failure	404			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/songs/{id}/audio [get]
func SongAudio(svc *application.Services, files interfaces.FileOpener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		songID, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Prefer X-Owner-ID header (set by gateway), fall back to user_id query for direct calls.
		userIDStr := r.Header.Get("X-Owner-ID")
		if userIDStr == "" {
			userIDStr = r.URL.Query().Get("user_id")
		}
		if userIDStr == "" {
			WriteError(w, http.StatusBadRequest, "missing user_id")
			return
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid user_id")
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
		rc, size, modTime, err := files.Open(melody.Path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, http.StatusNotFound, "file not found")
				return
			}
			WriteError(w, http.StatusInternalServerError, "cannot open file")
			return
		}
		defer rc.Close()

		// ETag: weak format based on size + modtime
		etag := fmt.Sprintf(`W"%d-%d"`, size, modTime.Unix())

		// If-None-Match → 304
		if match := r.Header.Get("If-None-Match"); match != "" {
			if match == etag || match == "*" {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Validate Range header if present
		if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
			if !isValidRange(rangeHeader, size) {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
		}

		// Set headers
		w.Header().Set("Content-Type", melody.ContentType)
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")

		// Serve with Range support
		http.ServeContent(w, r, melody.Path, modTime, rc)
	}
}

// isValidRange checks if the Range header is valid for the given file size.
// Format: "bytes=start-end" or "bytes=start-" or "bytes=-suffix"
func isValidRange(rangeHeader string, size int64) bool {
	rangeHeader = strings.TrimSpace(rangeHeader)
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return false
	}
	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")

	parts := strings.SplitN(rangeSpec, "-", 2)
	if len(parts) != 2 {
		return false
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	if startStr == "" {
		// Suffix range: bytes=-500 (last 500 bytes)
		suffix, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil || suffix <= 0 {
			return false
		}
		return true
	}

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil || start < 0 {
		return false
	}

	if endStr == "" {
		// Open-ended: bytes=1000-
		return start < size
	}

	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil || end < start {
		return false
	}

	return start < size
}
