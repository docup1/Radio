package os

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestChunkStore_WriteExistsAssemble(t *testing.T) {
	root := t.TempDir()
	final := t.TempDir()
	cs := NewChunkStore(root, final)

	session := uuid.New()
	if err := cs.Write(session, 0, []byte("AAA")); err != nil {
		t.Fatal(err)
	}
	if err := cs.Write(session, 1, []byte("BBB")); err != nil {
		t.Fatal(err)
	}
	if err := cs.Write(session, 2, []byte("CCC")); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		ok, err := cs.Exists(session, i)
		if err != nil || !ok {
			t.Fatalf("Exists(%d) = %v, %v", i, ok, err)
		}
	}
	if ok, _ := cs.Exists(session, 9); ok {
		t.Fatal("chunk 9 must not exist")
	}

	finalPath := filepath.Join(final, "out.mp3")
	if err := cs.Assemble(session, 3, finalPath); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(finalPath)
	if err != nil || string(got) != "AAABBBCCC" {
		t.Fatalf("assembled = %q, err %v", got, err)
	}
}

func TestChunkStore_AssembleMissingChunkFails(t *testing.T) {
	cs := NewChunkStore(t.TempDir(), t.TempDir())
	session := uuid.New()
	_ = cs.Write(session, 1, []byte("B")) // index 0 missing
	if err := cs.Assemble(session, 2, filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatal("want error assembling with missing chunk")
	}
}

func TestChunkStore_DeleteAndOverwrite(t *testing.T) {
	root := t.TempDir()
	cs := NewChunkStore(root, t.TempDir())
	session := uuid.New()
	_ = cs.Write(session, 0, []byte("old"))
	_ = cs.Write(session, 0, []byte("new"))
	if ok, _ := cs.Exists(session, 0); !ok {
		t.Fatal("chunk missing after overwrite")
	}

	if err := cs.Delete(session); err != nil {
		t.Fatal(err)
	}
	if ok, _ := cs.Exists(session, 0); ok {
		t.Fatal("chunk still present after delete")
	}
	if _, err := os.Stat(filepath.Join(root, session.String())); !os.IsNotExist(err) {
		t.Fatalf("session dir not removed: %v", err)
	}
}

func TestChunkStore_HashAndSize(t *testing.T) {
	cs := NewChunkStore(t.TempDir(), t.TempDir())
	data := make([]byte, 4096)
	_, _ = rand.Read(data)

	path := filepath.Join(t.TempDir(), "media.bin")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, err := cs.HashAndSize(path)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(data)
	if hash != hex.EncodeToString(want[:]) || size != int64(len(data)) {
		t.Fatalf("hash=%q size=%d", hash, size)
	}

	if _, _, err := cs.HashAndSize(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("want error for missing file")
	}
}

func TestFileOpener_Open(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, size, _, err := NewFileOpener().Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if size != 5 {
		t.Fatalf("size = %d", size)
	}
	buf := make([]byte, 5)
	if _, err := f.Read(buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "hello" {
		t.Fatalf("read = %q", buf)
	}

	if _, _, _, err := NewFileOpener().Open(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("want error for missing file")
	}
}
