package application

import (
	"radio/content-service/internal/domain/interfaces"
)

// Repos bundles the persistence ports the application services depend on.
type Repos struct {
	Songs         interfaces.SongRepository
	Melodies      interfaces.MelodyRepository
	Images        interfaces.ImageRepository
	Playlists     interfaces.PlaylistRepository
	PlaylistSongs interfaces.PlaylistSongRepository
	UploadSessions interfaces.UploadSessionRepository
}

// Services groups the use-case services exposed to the transport layer.
type Services struct {
	Songs         *SongService
	Melodies      *MelodyService
	Images        *ImageService
	Playlists     *PlaylistService
	PlaylistSongs *PlaylistSongService
	Uploads       *UploadService
}

// StorageParams configures the upload service.
type StorageParams struct {
	ChunkStore    interfaces.ChunkStore
	FinalRoot     string
	MaxChunkSize  int64
	MaxFileSize   int64
}

func New(repos Repos, storage StorageParams) *Services {
	return &Services{
		Songs: &SongService{
			Songs:    repos.Songs,
			Melodies: repos.Melodies,
			Images:   repos.Images,
		},
		Melodies: &MelodyService{
			Melodies: repos.Melodies,
			Songs:    repos.Songs,
		},
		Images:  &ImageService{Images: repos.Images},
		Playlists: &PlaylistService{
			Playlists: repos.Playlists,
		},
		PlaylistSongs: &PlaylistSongService{
			PlaylistSongs: repos.PlaylistSongs,
			Playlists:     repos.Playlists,
		},
		Uploads: &UploadService{
			UploadSessions: repos.UploadSessions,
			Melodies:       repos.Melodies,
			Images:         repos.Images,
			Chunks:         storage.ChunkStore,
			FinalRoot:      storage.FinalRoot,
			MaxChunkSize:   storage.MaxChunkSize,
			MaxFileSize:    storage.MaxFileSize,
		},
	}
}
