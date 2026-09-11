package application_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

func uploadAll(t *testing.T, a *testApp, owner, sessionID uuid.UUID, chunks [][]byte) {
	t.Helper()
	for i, data := range chunks {
		if err := a.services.Uploads.AddChunk(context.Background(), owner, sessionID, application.AddChunkInput{
			Index: i, Data: data,
		}); err != nil {
			t.Fatalf("AddChunk(%d): %v", i, err)
		}
	}
}

func TestUpload_InitValidations(t *testing.T) {
	a := newTestApp(t)
	owner := uuid.New()

	cases := []struct {
		name string
		in   application.InitUploadInput
	}{
		{"bad media type", application.InitUploadInput{MediaType: "video", ContentType: "video/mp4", TotalChunks: 1}},
		{"empty content type", application.InitUploadInput{MediaType: models.MediaTypeAudio, ContentType: "", TotalChunks: 1}},
		{"zero chunks", application.InitUploadInput{MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 0}},
		{"negative chunks", application.InitUploadInput{MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: -1}},
		{"negative size", application.InitUploadInput{MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 1, ExpectedSize: -1}},
		{"oversize", application.InitUploadInput{MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 1, ExpectedSize: testMaxFileSize + 1}},
		{"long hash", application.InitUploadInput{MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 1, ExpectedHash: strings.Repeat("a", 65)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := a.services.Uploads.Init(context.Background(), owner, tc.in); err != interfaces.ErrInvalid {
				t.Fatalf("err = %v, want invalid", err)
			}
		})
	}

	s, err := a.services.Uploads.Init(context.Background(), owner, application.InitUploadInput{
		MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != models.UploadStatusInitialized || s.OwnerID != owner || s.ReceivedChunks != 0 {
		t.Fatalf("session = %+v", s)
	}
}

func TestUpload_AddChunkOwnershipAndState(t *testing.T) {
	a := newTestApp(t)
	alice := uuid.New()
	bob := uuid.New()

	s, _ := a.services.Uploads.Init(context.Background(), alice, application.InitUploadInput{
		MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 2,
	})

	// foreign owner -> not found (existence masked)
	if err := a.services.Uploads.AddChunk(context.Background(), bob, s.ID, application.AddChunkInput{Index: 0, Data: []byte("x")}); err != interfaces.ErrNotFound {
		t.Fatalf("foreign add = %v", err)
	}

	// duplicate chunk is idempotent: ReceivedChunks increments once
	if err := a.services.Uploads.AddChunk(context.Background(), alice, s.ID, application.AddChunkInput{Index: 0, Data: []byte("A")}); err != nil {
		t.Fatal(err)
	}
	if err := a.services.Uploads.AddChunk(context.Background(), alice, s.ID, application.AddChunkInput{Index: 0, Data: []byte("OVERRIDE")}); err != nil {
		t.Fatal(err)
	}
	got, _ := a.uploads.Get(context.Background(), s.ID)
	if got.ReceivedChunks != 1 {
		t.Fatalf("received = %d, want 1 (idempotent)", got.ReceivedChunks)
	}

	// index bounds and size limits
	if err := a.services.Uploads.AddChunk(context.Background(), alice, s.ID, application.AddChunkInput{Index: 5, Data: []byte("x")}); err != interfaces.ErrInvalid {
		t.Fatalf("index oob = %v", err)
	}
	if err := a.services.Uploads.AddChunk(context.Background(), alice, s.ID, application.AddChunkInput{Index: -1, Data: []byte("x")}); err != interfaces.ErrInvalid {
		t.Fatalf("negative index = %v", err)
	}
	big := make([]byte, testMaxChunkSize+1)
	if err := a.services.Uploads.AddChunk(context.Background(), alice, s.ID, application.AddChunkInput{Index: 1, Data: big}); err != interfaces.ErrInvalid {
		t.Fatalf("oversized chunk = %v", err)
	}
}

func TestUpload_ConfirmAudioHappyPath(t *testing.T) {
	a := newTestApp(t)
	owner := uuid.New()
	data := []byte("Mp3BytesChunk1Mp3BytesChunk2")
	chunks := [][]byte{data[:17], data[17:]}
	wantSz := int64(len(data))
	wantHash := sha256Hex(data)

	s, _ := a.services.Uploads.Init(context.Background(), owner, application.InitUploadInput{
		MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 2,
		ExpectedSize: wantSz, ExpectedHash: wantHash,
	})
	uploadAll(t, a, owner, s.ID, chunks)

	res, err := a.services.Uploads.Confirm(context.Background(), owner, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Melody == nil || res.Image != nil {
		t.Fatalf("result = %+v", res)
	}
	if res.Melody.ContentType != "audio/mpeg" || res.Melody.Size != wantSz || res.Melody.Hash != wantHash {
		t.Fatalf("melody = %+v", res.Melody)
	}
	if !strings.Contains(res.Melody.Path, owner.String()) || !strings.Contains(res.Melody.Path, "audio") {
		t.Fatalf("final path = %q", res.Melody.Path)
	}
	if _, err := os.Stat(res.Melody.Path); err != nil {
		t.Fatalf("assembled file missing: %v", err)
	}

	// session completed; re-confirm conflicts
	got, _ := a.uploads.Get(context.Background(), s.ID)
	if got.Status != models.UploadStatusCompleted {
		t.Fatalf("status = %q", got.Status)
	}
	if _, err := a.services.Uploads.Confirm(context.Background(), owner, s.ID); err != interfaces.ErrConflict {
		t.Fatalf("re-confirm = %v", err)
	}
}

func TestUpload_ConfirmImageHappyPath(t *testing.T) {
	a := newTestApp(t)
	owner := uuid.New()
	data := []byte("PNG-bin")
	chunks := [][]byte{data}

	s, _ := a.services.Uploads.Init(context.Background(), owner, application.InitUploadInput{
		MediaType: models.MediaTypeImage, ContentType: "image/png", TotalChunks: 1,
	})
	uploadAll(t, a, owner, s.ID, chunks)

	res, err := a.services.Uploads.Confirm(context.Background(), owner, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Image == nil || res.Melody != nil {
		t.Fatalf("result = %+v", res)
	}
	if res.Image.ContentType != "image/png" {
		t.Fatalf("image = %+v", res.Image)
	}
	if _, err := os.Stat(res.Image.Path); err != nil {
		t.Fatalf("image file missing: %v", err)
	}
}

func TestUpload_ConfirmValidations(t *testing.T) {
	a := newTestApp(t)
	owner := uuid.New()

	// incomplete upload -> invalid
	s, _ := a.services.Uploads.Init(context.Background(), owner, application.InitUploadInput{
		MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 2,
	})
	_ = a.services.Uploads.AddChunk(context.Background(), owner, s.ID, application.AddChunkInput{Index: 0, Data: []byte("A")})
	if _, err := a.services.Uploads.Confirm(context.Background(), owner, s.ID); err != interfaces.ErrInvalid {
		t.Fatalf("incomplete confirm = %v", err)
	}

	// foreign owner
	s2, _ := a.services.Uploads.Init(context.Background(), owner, application.InitUploadInput{
		MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 1,
	})
	uploadAll(t, a, owner, s2.ID, [][]byte{[]byte("A")})
	if _, err := a.services.Uploads.Confirm(context.Background(), uuid.New(), s2.ID); err != interfaces.ErrNotFound {
		t.Fatalf("foreign confirm = %v", err)
	}
}

func TestUpload_ConfirmSizeAndHashMismatch(t *testing.T) {
	a := newTestApp(t)
	owner := uuid.New()
	data := []byte("ABCDE")

	t.Run("size mismatch", func(t *testing.T) {
		s, _ := a.services.Uploads.Init(context.Background(), owner, application.InitUploadInput{
			MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 1,
			ExpectedSize: 999,
		})
		uploadAll(t, a, owner, s.ID, [][]byte{data})
		if _, err := a.services.Uploads.Confirm(context.Background(), owner, s.ID); err != interfaces.ErrInvalid {
			t.Fatalf("size mismatch = %v", err)
		}
	})

	t.Run("hash mismatch", func(t *testing.T) {
		s, _ := a.services.Uploads.Init(context.Background(), owner, application.InitUploadInput{
			MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 1,
			ExpectedHash: sha256Hex([]byte("XXXXX")),
		})
		uploadAll(t, a, owner, s.ID, [][]byte{data})
		if _, err := a.services.Uploads.Confirm(context.Background(), owner, s.ID); err != interfaces.ErrInvalid {
			t.Fatalf("hash mismatch = %v", err)
		}
	})
}

func TestUpload_SessionPathIsOwnerScoped(t *testing.T) {
	a := newTestApp(t)
	owner := uuid.New()
	s, _ := a.services.Uploads.Init(context.Background(), owner, application.InitUploadInput{
		MediaType: models.MediaTypeAudio, ContentType: "audio/mpeg", TotalChunks: 1,
	})
	uploadAll(t, a, owner, s.ID, [][]byte{[]byte("song")})
	res, err := a.services.Uploads.Confirm(context.Background(), owner, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(a.services.Uploads.FinalRoot, res.Melody.Path)
	if err != nil {
		t.Fatal(err)
	}
	if rel != filepath.Join(owner.String(), "audio", s.ID.String()) {
		t.Fatalf("rel path = %q", rel)
	}
}
