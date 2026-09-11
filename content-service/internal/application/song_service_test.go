package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

func TestSong_CreateValidates(t *testing.T) {
	a := newTestApp(t)
	owner := uuid.New()
	m := a.createMelody(t)

	cases := []struct {
		name string
		mut  func(*application.CreateSongInput)
		want error
	}{
		{"empty name", func(in *application.CreateSongInput) { in.Name = "  " }, interfaces.ErrInvalid},
		{"name too long", func(in *application.CreateSongInput) { in.Name = strings.Repeat("x", models.SongNameMaxLength+1) }, interfaces.ErrInvalid},
		{"desc too long", func(in *application.CreateSongInput) {
			in.Name = "ok"
			d := strings.Repeat("x", models.SongDescriptionMaxLength+1)
			in.Description = &d
		}, interfaces.ErrInvalid},
		{"missing melody", func(in *application.CreateSongInput) {
			in.Name = "ok"
			in.MelodyID = uuid.New()
		}, interfaces.ErrInvalid},
		{"missing image", func(in *application.CreateSongInput) {
			in.Name = "ok"
			img := uuid.New()
			in.ImageID = &img
		}, interfaces.ErrInvalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := application.CreateSongInput{Name: "valid", MelodyID: m.ID}
			tc.mut(&in)
			if _, err := a.services.Songs.Create(context.Background(), owner, in); err != tc.want {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestSong_CreateStoresOwner(t *testing.T) {
	a := newTestApp(t)
	owner := uuid.New()
	song, _ := a.createSong(t, owner, "Track", false)

	got, err := a.services.Songs.Get(context.Background(), song.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if got.OwnerID != owner || got.Name != "Track" || got.MelodyID != song.MelodyID {
		t.Fatalf("song = %+v", got)
	}
}

func TestSong_Visibility(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	bob := uuid.New()

	private, _ := a.createSong(t, alice, "Secret", false)
	public, _ := a.createSong(t, alice, "Public", true)

	// owner sees private
	if _, err := a.services.Songs.Get(context.Background(), private.ID, alice); err != nil {
		t.Fatalf("owner get = %v", err)
	}
	// other user cannot see private
	if _, err := a.services.Songs.Get(context.Background(), private.ID, bob); err != interfaces.ErrNotFound {
		t.Fatalf("bob private get = %v, want not found", err)
	}
	// other user sees public
	if _, err := a.services.Songs.Get(context.Background(), public.ID, bob); err != nil {
		t.Fatalf("bob public get = %v", err)
	}
}

func TestSong_ListScopes(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	bob := uuid.New()

	a.createSong(t, alice, "A-private", false)
	a.createSong(t, alice, "A-public", true)
	a.createSong(t, bob, "B-public", true)
	a.createSong(t, bob, "B-private", false)

	mine, err := a.services.Songs.List(context.Background(), alice, models.SongScopeMine, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range mine {
		if s.OwnerID != alice {
			t.Fatalf("mine scope leaked foreign song %+v", s)
		}
	}
	if len(mine) != 2 {
		t.Fatalf("mine len = %d, want 2", len(mine))
	}

	pub, err := a.services.Songs.List(context.Background(), alice, models.SongScopePublic, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pub) != 1 || pub[0].OwnerID != bob {
		t.Fatalf("public scope = %+v, want only bob's public song", pub)
	}
}

func TestSong_Search(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()

	a.createSong(t, alice, "Deep House Mix", false)
	a.createSong(t, alice, "Rock Anthems", false)

	if _, err := a.services.Songs.Search(context.Background(), "  ", alice, models.SongScopeMine, 10, 0); err != interfaces.ErrInvalid {
		t.Fatalf("empty query = %v, want invalid", err)
	}

	hit, err := a.services.Songs.Search(context.Background(), "deep", alice, models.SongScopeMine, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(hit) != 1 || hit[0].Name != "Deep House Mix" {
		t.Fatalf("search = %+v", hit)
	}
}

func TestSong_UpdateAndDeleteOwnership(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	bob := uuid.New()
	song, _ := a.createSong(t, alice, "Track", false)

	// foreign update/delete -> not found (existence masked)
	name := "Renamed"
	if err := a.services.Songs.Update(context.Background(), song.ID, bob, interfaces.SongPatch{Name: &name}); err != interfaces.ErrNotFound {
		t.Fatalf("bob update = %v", err)
	}
	if err := a.services.Songs.Delete(context.Background(), song.ID, bob); err != interfaces.ErrNotFound {
		t.Fatalf("bob delete = %v", err)
	}

	// validation errors precede persistence
	if err := a.services.Songs.Update(context.Background(), song.ID, alice, interfaces.SongPatch{Name: &name}); err != nil {
		t.Fatal(err)
	}
	got, _ := a.services.Songs.Get(context.Background(), song.ID, alice)
	if got.Name != "Renamed" {
		t.Fatalf("name = %q", got.Name)
	}

	if err := a.services.Songs.Delete(context.Background(), song.ID, alice); err != nil {
		t.Fatal(err)
	}
	if _, err := a.services.Songs.Get(context.Background(), song.ID, alice); err != interfaces.ErrNotFound {
		t.Fatalf("after delete get = %v", err)
	}
}

func TestSong_UpdateNonexistentMelody(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	song, _ := a.createSong(t, alice, "Track", false)

	newMelody := uuid.New()
	if err := a.services.Songs.Update(context.Background(), song.ID, alice, interfaces.SongPatch{MelodyID: &newMelody}); err != interfaces.ErrInvalid {
		t.Fatalf("missing melody update = %v, want invalid", err)
	}
}
