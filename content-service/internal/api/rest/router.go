package rest

import (
	"net/http"

	hsong "radio/content-service/internal/api/rest/handlers/song"
	hmelody "radio/content-service/internal/api/rest/handlers/melody"
	himage "radio/content-service/internal/api/rest/handlers/image"
	hplaylist "radio/content-service/internal/api/rest/handlers/playlist"
	hupload "radio/content-service/internal/api/rest/handlers/uploads"
	"radio/content-service/internal/application"
)

// NewHandler builds the public user-facing REST API. Every request carries the
// owner identity in the X-Owner-ID header (set by the gateway).
func NewHandler(svc *application.Services) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /songs", hsong.Create(svc))
	mux.HandleFunc("GET /songs", hsong.List(svc))
	mux.HandleFunc("GET /songs/search", hsong.Search(svc))
	mux.HandleFunc("GET /songs/{id}", hsong.Get(svc))
	mux.HandleFunc("PUT /songs/{id}", hsong.Update(svc))
	mux.HandleFunc("DELETE /songs/{id}", hsong.Delete(svc))

	mux.HandleFunc("GET /melodies", hmelody.List(svc))
	mux.HandleFunc("GET /melodies/{id}", hmelody.Get(svc))
	mux.HandleFunc("PUT /melodies/{id}", hmelody.Update(svc))
	mux.HandleFunc("DELETE /melodies/{id}", hmelody.Delete(svc))

	mux.HandleFunc("POST /images", himage.Create(svc))
	mux.HandleFunc("GET /images", himage.List(svc))
	mux.HandleFunc("GET /images/{id}", himage.Get(svc))
	mux.HandleFunc("PUT /images/{id}", himage.Update(svc))
	mux.HandleFunc("DELETE /images/{id}", himage.Delete(svc))

	mux.HandleFunc("POST /playlists", hplaylist.Create(svc))
	mux.HandleFunc("GET /playlists", hplaylist.List(svc))
	mux.HandleFunc("GET /playlists/{id}", hplaylist.Get(svc))
	mux.HandleFunc("PUT /playlists/{id}", hplaylist.Update(svc))
	mux.HandleFunc("DELETE /playlists/{id}", hplaylist.Delete(svc))

	mux.HandleFunc("POST /playlists/{id}/songs", hplaylist.AddSong(svc))
	mux.HandleFunc("DELETE /playlists/{id}/songs/{song_id}", hplaylist.RemoveSong(svc))
	mux.HandleFunc("PUT /playlists/{id}/songs/{song_id}", hplaylist.MoveSong(svc))

	mux.HandleFunc("POST /uploads", hupload.Init(svc))
	mux.HandleFunc("PUT /uploads/{id}/chunks/{index}", hupload.Chunk(svc))
	mux.HandleFunc("POST /uploads/{id}/confirm", hupload.Confirm(svc))

	return mux
}
