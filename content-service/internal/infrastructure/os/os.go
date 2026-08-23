package os

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
)

// FileOpener implements interfaces.FileOpener using the host filesystem.
type FileOpener struct{}

func NewFileOpener() *FileOpener { return &FileOpener{} }

func (o *FileOpener) Open(path string) (io.ReadSeekCloser, int64, time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, time.Time{}, err
	}
	return f, info.Size(), info.ModTime(), nil
}

// ChunkStore implements interfaces.ChunkStore on top of the host filesystem.
// Each session gets its own directory under Root; chunks are named by zero-based
// index. The finalized, assembled file is written into FinalRoot.
type ChunkStore struct {
	Root      string
	FinalRoot string
}

func NewChunkStore(root, finalRoot string) *ChunkStore {
	return &ChunkStore{Root: root, FinalRoot: finalRoot}
}

func (c *ChunkStore) sessionDir(sessionID uuid.UUID) string {
	return filepath.Join(c.Root, sessionID.String())
}

func (c *ChunkStore) chunkPath(sessionID uuid.UUID, index int) string {
	return filepath.Join(c.sessionDir(sessionID), fmt.Sprintf("%d.chunk", index))
}

func (c *ChunkStore) Write(sessionID uuid.UUID, index int, data []byte) error {
	dir := c.sessionDir(sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(c.chunkPath(sessionID, index), data, 0o644)
}

func (c *ChunkStore) Exists(sessionID uuid.UUID, index int) (bool, error) {
	info, err := os.Stat(c.chunkPath(sessionID, index))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

func (c *ChunkStore) Delete(sessionID uuid.UUID) error {
	err := os.RemoveAll(c.sessionDir(sessionID))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (c *ChunkStore) Assemble(sessionID uuid.UUID, totalChunks int, finalPath string) error {
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer out.Close()
	for i := 0; i < totalChunks; i++ {
		data, err := os.ReadFile(c.chunkPath(sessionID, i))
		if err != nil {
			return fmt.Errorf("missing chunk %d: %w", i, err)
		}
		if _, err := out.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func (c *ChunkStore) HashAndSize(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	written, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), written, nil
}

var (
	_ interfaces.FileOpener = (*FileOpener)(nil)
	_ interfaces.ChunkStore = (*ChunkStore)(nil)
)
