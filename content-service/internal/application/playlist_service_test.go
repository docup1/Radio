package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
	"radio/content-service/internal/infrastructure/inmem"
)

func TestPlaylist_CreateValidates(t *testing.T) {
	a := newTestApp(t)
	owner := uuid.New()

	if _, err := a.services.Playlists.Create(context.Background(), owner, application.CreatePlaylistInput{Name: " "}); err != interfaces.ErrInvalid {
		t.Fatalf("empty name = %v", err)
	}
	if _, err := a.services.Playlists.Create(context.Background(), owner, application.CreatePlaylistInput{
		Name: strings.Repeat("x", models.PlaylistNameMaxLength+1),
	}); err != interfaces.ErrInvalid {
		t.Fatalf("long name = %v", err)
	}

	p, err := a.services.Playlists.Create(context.Background(), owner, application.CreatePlaylistInput{Name: "Favorites"})
	if err != nil {
		t.Fatal(err)
	}
	if p.OwnerID != owner || p.Name != "Favorites" {
		t.Fatalf("playlist = %+v", p)
	}
}

func TestPlaylist_OwnerOnly(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	bob := uuid.New()

	p, _ := a.services.Playlists.Create(context.Background(), alice, application.CreatePlaylistInput{Name: "Mine"})

	if _, err := a.services.Playlists.Get(context.Background(), p.ID, alice); err != nil {
		t.Fatalf("owner get = %v", err)
	}
	if _, err := a.services.Playlists.Get(context.Background(), p.ID, bob); err != interfaces.ErrNotFound {
		t.Fatalf("bob get = %v", err)
	}

	name := "Renamed"
	if err := a.services.Playlists.Update(context.Background(), p.ID, bob, interfaces.PlaylistPatch{Name: &name}); err != interfaces.ErrNotFound {
		t.Fatalf("bob update = %v", err)
	}
	if err := a.services.Playlists.Update(context.Background(), p.ID, alice, interfaces.PlaylistPatch{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if err := a.services.Playlists.Delete(context.Background(), p.ID, bob); err != interfaces.ErrNotFound {
		t.Fatalf("bob delete = %v", err)
	}
	if err := a.services.Playlists.Delete(context.Background(), p.ID, alice); err != nil {
		t.Fatal(err)
	}
}

func TestPlaylist_ListVisible(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	bob := uuid.New()

	a.services.Playlists.Create(context.Background(), alice, application.CreatePlaylistInput{Name: "A1"})
	a.services.Playlists.Create(context.Background(), bob, application.CreatePlaylistInput{Name: "B1"})

	list, err := a.services.Playlists.List(context.Background(), alice, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "A1" {
		t.Fatalf("list = %+v", list)
	}
}

// strictSongVisibility wraps the inmem repo with a controllable SongVisibleTo.
type strictSongVisibility struct {
	*inmem.PlaylistSongRepository
	visible bool
}

func (s *strictSongVisibility) SongVisibleTo(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return s.visible, nil
}

func TestPlaylistSong_AddPositioning(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()

	p, _ := a.services.Playlists.Create(context.Background(), alice, application.CreatePlaylistInput{Name: "Mix"})
	s1, _ := a.createSong(t, alice, "One", false)
	s2, _ := a.createSong(t, alice, "Two", false)

	// auto position 1 then 2
	if err := a.services.PlaylistSongs.Add(context.Background(), p.ID, alice, application.AddSongInput{SongID: s1.ID}); err != nil {
		t.Fatal(err)
	}
	if err := a.services.PlaylistSongs.Add(context.Background(), p.ID, alice, application.AddSongInput{SongID: s2.ID}); err != nil {
		t.Fatal(err)
	}

	list, err := a.services.PlaylistSongs.List(context.Background(), p.ID, alice)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != s1.ID || list[1].ID != s2.ID {
		t.Fatalf("order = %+v (want s1 then s2 by position)", list)
	}
}

func TestPlaylistSong_ForbiddenForForeignOwner(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	bob := uuid.New()

	p, _ := a.services.Playlists.Create(context.Background(), alice, application.CreatePlaylistInput{Name: "Mix"})
	s, _ := a.createSong(t, alice, "One", false)

	if err := a.services.PlaylistSongs.Add(context.Background(), p.ID, bob, application.AddSongInput{SongID: s.ID}); err != interfaces.ErrForbidden {
		t.Fatalf("foreign add = %v", err)
	}
	if err := a.services.PlaylistSongs.Remove(context.Background(), p.ID, bob, s.ID); err != interfaces.ErrForbidden {
		t.Fatalf("foreign remove = %v", err)
	}
	if err := a.services.PlaylistSongs.Move(context.Background(), p.ID, bob, s.ID, 1); err != interfaces.ErrForbidden {
		t.Fatalf("foreign move = %v", err)
	}
	if _, err := a.services.PlaylistSongs.List(context.Background(), p.ID, bob); err != interfaces.ErrForbidden {
		t.Fatalf("foreign list = %v", err)
	}
}

func TestPlaylistSong_RejectsInvisibleSong(t *testing.T) {
	a := newTestAppWith(t, func(repos *application.Repos) {
		repos.PlaylistSongs = &strictSongVisibility{
			PlaylistSongRepository: repos.PlaylistSongs.(*inmem.PlaylistSongRepository),
			visible:                false,
		}
	})
	alice := uuid.New()
	m := a.createMelody(t)
	foreignSong, err := a.services.Songs.Create(context.Background(), alice, application.CreateSongInput{
		Name: "Private", MelodyID: m.ID, IsPublic: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	p, _ := a.services.Playlists.Create(context.Background(), alice, application.CreatePlaylistInput{Name: "Mix"})
	if err := a.services.PlaylistSongs.Add(context.Background(), p.ID, alice, application.AddSongInput{SongID: foreignSong.ID}); err != interfaces.ErrNotFound {
		t.Fatalf("invisible song add = %v, want not found", err)
	}
}

func TestPlaylistSong_MoveAndRemove(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	p, _ := a.services.Playlists.Create(context.Background(), alice, application.CreatePlaylistInput{Name: "Mix"})

	songs := make([]*models.Song, 3)
	for i := range songs {
		songs[i], _ = a.createSong(t, alice, string(rune('A'+i)), false)
	}
	for _, s := range songs {
		if err := a.services.PlaylistSongs.Add(context.Background(), p.ID, alice, application.AddSongInput{SongID: s.ID}); err != nil {
			t.Fatal(err)
		}
	}

	// free position 1 by removing the first song
	if err := a.services.PlaylistSongs.Remove(context.Background(), p.ID, alice, songs[0].ID); err != nil {
		t.Fatal(err)
	}
	list, _ := a.services.PlaylistSongs.List(context.Background(), p.ID, alice)
	if len(list) != 2 {
		t.Fatalf("after remove len = %d", len(list))
	}

	// move last to the freed position 1
	if err := a.services.PlaylistSongs.Move(context.Background(), p.ID, alice, songs[2].ID, 1); err != nil {
		t.Fatal(err)
	}
	// negative position is invalid
	if err := a.services.PlaylistSongs.Move(context.Background(), p.ID, alice, songs[2].ID, -1); err != interfaces.ErrInvalid {
		t.Fatalf("negative move = %v", err)
	}
	// unknown song -> not found
	if err := a.services.PlaylistSongs.Move(context.Background(), p.ID, alice, uuid.New(), 5); err != interfaces.ErrNotFound {
		t.Fatalf("unknown move = %v", err)
	}

	list, _ = a.services.PlaylistSongs.List(context.Background(), p.ID, alice)
	if list[0].ID != songs[2].ID {
		t.Fatalf("first = %+v, want moved song", list[0])
	}
}
