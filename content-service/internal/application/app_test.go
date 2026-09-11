package application_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"

	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
	"radio/content-service/internal/infrastructure/inmem"
	infraos "radio/content-service/internal/infrastructure/os"
)

const (
	testMaxChunkSize = 1 << 20
	testMaxFileSize  = 50 << 20
)

type testApp struct {
	services *application.Services
	repos    application.Repos
	songs    *inmem.SongRepository
	melodies *inmem.MelodyRepository
	images   *inmem.ImageRepository
	uploads  *inmem.UploadSessionRepository
	chunks   interfaces.ChunkStore
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	return newTestAppWith(t, nil)
}

func newTestAppWith(t *testing.T, tweak func(*application.Repos)) *testApp {
	t.Helper()
	chunkDir := t.TempDir()
	finalDir := t.TempDir()

	repos := application.Repos{
		Songs:          inmem.NewSongRepository(),
		Melodies:       inmem.NewMelodyRepository(),
		Images:         inmem.NewImageRepository(),
		Playlists:      inmem.NewPlaylistRepository(),
		UploadSessions: inmem.NewUploadSessionRepository(),
	}
	repos.PlaylistSongs = inmem.NewPlaylistSongRepository(repos.Songs.(*inmem.SongRepository))
	if tweak != nil {
		tweak(&repos)
	}

	storage := application.StorageParams{
		ChunkStore:   infraos.NewChunkStore(chunkDir, finalDir),
		FinalRoot:    finalDir,
		MaxChunkSize: testMaxChunkSize,
		MaxFileSize:  testMaxFileSize,
	}

	return &testApp{
		services: application.New(repos, storage),
		repos:    repos,
		songs:    repos.Songs.(*inmem.SongRepository),
		melodies: repos.Melodies.(*inmem.MelodyRepository),
		images:   repos.Images.(*inmem.ImageRepository),
		uploads:  repos.UploadSessions.(*inmem.UploadSessionRepository),
		chunks:   storage.ChunkStore,
	}
}

func (a *testApp) createMelody(t *testing.T) *models.Melody {
	t.Helper()
	m, err := a.services.Melodies.Create(context.Background(), application.CreateFileInput{
		Path: "/media/a.mp3", ContentType: "audio/mpeg",
	})
	if err != nil {
		t.Fatalf("create melody: %v", err)
	}
	return m
}

func (a *testApp) createSong(t *testing.T, owner uuid.UUID, name string, isPublic bool) (*models.Song, *models.Melody) {
	t.Helper()
	m := a.createMelody(t)
	song, err := a.services.Songs.Create(context.Background(), owner, application.CreateSongInput{
		Name: name, MelodyID: m.ID, IsPublic: isPublic,
	})
	if err != nil {
		t.Fatalf("create song: %v", err)
	}
	return song, m
}

// sha256Hex mirrors the melody hash format produced on upload.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

var (
	_ = models.MediaTypeAudio
	_ = uuid.Nil
)
