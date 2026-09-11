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

func TestMelody_CRUD(t *testing.T) {
	a := newTestApp(t)
	m, err := a.services.Melodies.Create(context.Background(), application.CreateFileInput{
		Path: "/media/a.mp3", ContentType: "audio/mpeg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.ID == uuid.Nil {
		t.Fatalf("melody = %+v", m)
	}

	got, err := a.services.Melodies.Get(context.Background(), m.ID)
	if err != nil || got.Path != "/media/a.mp3" {
		t.Fatalf("get = %+v, %v", got, err)
	}

	path := "/media/renamed.mp3"
	if err := a.services.Melodies.Update(context.Background(), m.ID, interfaces.MelodyPatch{Path: &path}); err != nil {
		t.Fatal(err)
	}
	got, _ = a.services.Melodies.Get(context.Background(), m.ID)
	if got.Path != path {
		t.Fatalf("path = %q", got.Path)
	}

	if err := a.services.Melodies.Delete(context.Background(), m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.services.Melodies.Get(context.Background(), m.ID); err != interfaces.ErrNotFound {
		t.Fatalf("after delete = %v", err)
	}
}

func TestMelody_UpdateValidation(t *testing.T) {
	a := newTestApp(t)
	m := a.createMelody(t)

	badPath := strings.Repeat("x", models.MelodyPathMaxLength+1)
	if err := a.services.Melodies.Update(context.Background(), m.ID, interfaces.MelodyPatch{Path: &badPath}); err != interfaces.ErrInvalid {
		t.Fatalf("long path = %v", err)
	}
	neg := int64(-1)
	if err := a.services.Melodies.Update(context.Background(), m.ID, interfaces.MelodyPatch{Size: &neg}); err != interfaces.ErrInvalid {
		t.Fatalf("negative size = %v", err)
	}
	longHash := strings.Repeat("a", 65)
	if err := a.services.Melodies.Update(context.Background(), m.ID, interfaces.MelodyPatch{Hash: &longHash}); err != interfaces.ErrInvalid {
		t.Fatalf("long hash = %v", err)
	}

	// sanity: partial patch applies
	ct := "audio/ogg"
	if err := a.services.Melodies.Update(context.Background(), m.ID, interfaces.MelodyPatch{ContentType: &ct}); err != nil {
		t.Fatal(err)
	}
	got, _ := a.services.Melodies.Get(context.Background(), m.ID)
	if got.ContentType != "audio/ogg" {
		t.Fatalf("content type = %q", got.ContentType)
	}
}

func TestMelody_Access(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	bob := uuid.New()

	_, privateMelody := a.createSong(t, alice, "Secret", false)
	_, publicMelody := a.createSong(t, alice, "Public", true)

	// Unreferenced melody: exists but no access.
	unused := a.createMelody(t)
	res, err := a.services.Melodies.Access(context.Background(), unused.ID, alice)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Exists || res.HasAccess {
		t.Fatalf("unused melody access = %+v", res)
	}

	// Owner has access to private.
	res, _ = a.services.Melodies.Access(context.Background(), privateMelody.ID, alice)
	if !res.HasAccess {
		t.Fatalf("owner access = %+v", res)
	}
	// Stranger has no access to private.
	res, _ = a.services.Melodies.Access(context.Background(), privateMelody.ID, bob)
	if res.HasAccess {
		t.Fatalf("stranger private access = %+v", res)
	}
	// Stranger has access to public.
	res, _ = a.services.Melodies.Access(context.Background(), publicMelody.ID, bob)
	if !res.HasAccess || res.ContentType != "audio/mpeg" {
		t.Fatalf("stranger public access = %+v", res)
	}
}

func TestImage_CRUD(t *testing.T) {
	a := newTestApp(t)
	img, err := a.services.Images.Create(context.Background(), application.CreateFileInput{
		Path: "/media/c.jpg", ContentType: "image/jpeg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if img.ID == uuid.Nil || img.Path != "/media/c.jpg" {
		t.Fatalf("image = %+v", img)
	}
	if _, err := a.services.Images.Get(context.Background(), img.ID); err != nil {
		t.Fatal(err)
	}

	bad := ""
	if err := a.services.Images.Update(context.Background(), img.ID, interfaces.ImagePatch{Path: &bad}); err != interfaces.ErrInvalid {
		t.Fatalf("empty path = %v", err)
	}

	if err := a.services.Images.Delete(context.Background(), img.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.services.Images.Get(context.Background(), img.ID); err != interfaces.ErrNotFound {
		t.Fatalf("after delete = %v", err)
	}
}
