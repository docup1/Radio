package models

import "testing"

func TestParseSongScope(t *testing.T) {
	cases := map[string]SongScope{
		"public":  SongScopePublic,
		"PRIVATE": SongScopeMine,
		"mine":    SongScopeMine,
		"":        SongScopeMine,
		"bogus":   SongScopeMine,
	}
	for in, want := range cases {
		if got := ParseSongScope(in); got != want {
			t.Errorf("ParseSongScope(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMediaTypeConstants(t *testing.T) {
	if MediaTypeAudio != "audio" || MediaTypeImage != "image" {
		t.Fatalf("media types = %q/%q", MediaTypeAudio, MediaTypeImage)
	}
	if UploadStatusInitialized != "initialized" || UploadStatusCompleted != "completed" || UploadStatusAborted != "aborted" {
		t.Fatalf("upload statuses = %q/%q/%q", UploadStatusInitialized, UploadStatusCompleted, UploadStatusAborted)
	}
}

func TestLengthConstants(t *testing.T) {
	if SongNameMaxLength != 128 || SongDescriptionMaxLength != 512 {
		t.Fatalf("song lengths = %d/%d", SongNameMaxLength, SongDescriptionMaxLength)
	}
	if PlaylistNameMaxLength != 128 {
		t.Fatalf("playlist name length = %d", PlaylistNameMaxLength)
	}
	if MelodyPathMaxLength != 512 || MelodyContentTypeMaxLength != 128 ||
		ImagePathMaxLength != 512 || ImageContentTypeMaxLength != 128 {
		t.Fatalf("file lengths wrong")
	}
}
