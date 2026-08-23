package rest

import (
	"net/http"

	"github.com/google/uuid"

	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
)

// NewPublicHandler builds the user-facing HTTP API. Every request carries the
// owner identity in the X-Owner-ID header (set by the gateway).
func NewPublicHandler(svc *application.Services) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /songs", createSong(svc))
	mux.HandleFunc("GET /songs", listSongs(svc))
	mux.HandleFunc("GET /songs/search", searchSongs(svc))
	mux.HandleFunc("GET /songs/{id}", getSong(svc))
	mux.HandleFunc("PUT /songs/{id}", updateSong(svc))
	mux.HandleFunc("DELETE /songs/{id}", deleteSong(svc))

	mux.HandleFunc("POST /melodies", createMelody(svc))
	mux.HandleFunc("GET /melodies", listMelodies(svc))
	mux.HandleFunc("GET /melodies/{id}", getMelody(svc))
	mux.HandleFunc("PUT /melodies/{id}", updateMelody(svc))
	mux.HandleFunc("DELETE /melodies/{id}", deleteMelody(svc))

	mux.HandleFunc("POST /images", createImage(svc))
	mux.HandleFunc("GET /images", listImages(svc))
	mux.HandleFunc("GET /images/{id}", getImage(svc))
	mux.HandleFunc("PUT /images/{id}", updateImage(svc))
	mux.HandleFunc("DELETE /images/{id}", deleteImage(svc))

	mux.HandleFunc("POST /playlists", createPlaylist(svc))
	mux.HandleFunc("GET /playlists", listPlaylists(svc))
	mux.HandleFunc("GET /playlists/{id}", getPlaylist(svc))
	mux.HandleFunc("PUT /playlists/{id}", updatePlaylist(svc))
	mux.HandleFunc("DELETE /playlists/{id}", deletePlaylist(svc))

	mux.HandleFunc("POST /playlists/{id}/songs", addPlaylistSong(svc))
	mux.HandleFunc("DELETE /playlists/{id}/songs/{song_id}", removePlaylistSong(svc))
	mux.HandleFunc("PUT /playlists/{id}/songs/{song_id}", movePlaylistSong(svc))

	return mux
}

func createSong(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		var req struct {
			Name        string  `json:"name"`
			Description *string `json:"description"`
			MelodyID    string  `json:"melody_id"`
			ImageID     *string `json:"image_id"`
			IsPublic    bool    `json:"is_public"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		mid, err := uuid.Parse(req.MelodyID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid melody_id")
			return
		}
		var imgID *uuid.UUID
		if req.ImageID != nil {
			id, err := uuid.Parse(*req.ImageID)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "invalid image_id")
				return
			}
			imgID = &id
		}
		song, err := svc.Songs.Create(r.Context(), owner, application.CreateSongInput{
			Name:        req.Name,
			Description: req.Description,
			MelodyID:    mid,
			ImageID:     imgID,
			IsPublic:    req.IsPublic,
		})
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusCreated, song)
	}
}

func listSongs(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		limit, offset := parsePagination(r)
		songs, err := svc.Songs.List(r.Context(), owner, limit, offset)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, songs)
	}
}

func searchSongs(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		q := r.URL.Query().Get("q")
		if q == "" {
			WriteError(w, http.StatusBadRequest, "missing q parameter")
			return
		}
		limit, offset := parsePagination(r)
		songs, err := svc.Songs.Search(r.Context(), q, owner, limit, offset)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, songs)
	}
}

func getSong(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		song, err := svc.Songs.Get(r.Context(), id, owner)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, song)
	}
}

func updateSong(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			MelodyID    *string `json:"melody_id"`
			ImageID     *string `json:"image_id"`
			IsPublic    *bool   `json:"is_public"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		patch := interfaces.SongPatch{}
		if req.Name != nil {
			patch.Name = req.Name
		}
		if req.Description != nil {
			patch.Description = req.Description
		}
		if req.MelodyID != nil {
			mid, err := uuid.Parse(*req.MelodyID)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "invalid melody_id")
				return
			}
			patch.MelodyID = &mid
		}
		if req.ImageID != nil {
			if *req.ImageID == "" {
				WriteError(w, http.StatusBadRequest, "invalid image_id")
				return
			}
			iid, err := uuid.Parse(*req.ImageID)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "invalid image_id")
				return
			}
			patch.ImageID = &uuid.NullUUID{UUID: iid, Valid: true}
		}
		if req.IsPublic != nil {
			patch.IsPublic = req.IsPublic
		}
		if err := svc.Songs.Update(r.Context(), id, owner, patch); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func deleteSong(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.Songs.Delete(r.Context(), id, owner); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func createMelody(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		var req struct {
			Path        string `json:"path"`
			ContentType string `json:"content_type"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		m, err := svc.Melodies.Create(r.Context(), application.CreateFileInput{Path: req.Path, ContentType: req.ContentType})
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusCreated, m)
	}
}

func listMelodies(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		limit, offset := parsePagination(r)
		out, err := svc.Melodies.List(r.Context(), limit, offset)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func getMelody(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		m, err := svc.Melodies.Get(r.Context(), id)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, m)
	}
}

func updateMelody(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req struct {
			Path        *string `json:"path"`
			ContentType *string `json:"content_type"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		patch := interfaces.MelodyPatch{}
		if req.Path != nil {
			patch.Path = req.Path
		}
		if req.ContentType != nil {
			patch.ContentType = req.ContentType
		}
		if err := svc.Melodies.Update(r.Context(), id, patch); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func deleteMelody(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.Melodies.Delete(r.Context(), id); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func createImage(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		var req struct {
			Path        string `json:"path"`
			ContentType string `json:"content_type"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		m, err := svc.Images.Create(r.Context(), application.CreateFileInput{Path: req.Path, ContentType: req.ContentType})
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusCreated, m)
	}
}

func listImages(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		limit, offset := parsePagination(r)
		out, err := svc.Images.List(r.Context(), limit, offset)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func getImage(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		m, err := svc.Images.Get(r.Context(), id)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, m)
	}
}

func updateImage(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req struct {
			Path        *string `json:"path"`
			ContentType *string `json:"content_type"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		patch := interfaces.ImagePatch{}
		if req.Path != nil {
			patch.Path = req.Path
		}
		if req.ContentType != nil {
			patch.ContentType = req.ContentType
		}
		if err := svc.Images.Update(r.Context(), id, patch); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func deleteImage(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ownerOrError(w, r); !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.Images.Delete(r.Context(), id); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func createPlaylist(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		var req struct {
			Name     string `json:"name"`
			IsPublic bool   `json:"is_public"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		p, err := svc.Playlists.Create(r.Context(), owner, application.CreatePlaylistInput{Name: req.Name, IsPublic: req.IsPublic})
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusCreated, p)
	}
}

func listPlaylists(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		limit, offset := parsePagination(r)
		out, err := svc.Playlists.List(r.Context(), owner, limit, offset)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func getPlaylist(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		p, err := svc.Playlists.Get(r.Context(), id, owner)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, p)
	}
}

func updatePlaylist(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req struct {
			Name     *string `json:"name"`
			IsPublic *bool   `json:"is_public"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		patch := interfaces.PlaylistPatch{}
		if req.Name != nil {
			patch.Name = req.Name
		}
		if req.IsPublic != nil {
			patch.IsPublic = req.IsPublic
		}
		if err := svc.Playlists.Update(r.Context(), id, owner, patch); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func deletePlaylist(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		id, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.Playlists.Delete(r.Context(), id, owner); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func addPlaylistSong(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		playlistID, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req struct {
			SongID   string `json:"song_id"`
			Position *int   `json:"position"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		sid, err := uuid.Parse(req.SongID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid song_id")
			return
		}
		if err := svc.PlaylistSongs.Add(r.Context(), playlistID, owner, application.AddSongInput{SongID: sid, Position: req.Position}); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func removePlaylistSong(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		playlistID, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		songID, err := parseID(r, "song_id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.PlaylistSongs.Remove(r.Context(), playlistID, owner, songID); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func movePlaylistSong(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := ownerOrError(w, r)
		if !ok {
			return
		}
		playlistID, err := parseID(r, "id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		songID, err := parseID(r, "song_id")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req struct {
			Position int `json:"position"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := svc.PlaylistSongs.Move(r.Context(), playlistID, owner, songID, req.Position); err != nil {
			WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
