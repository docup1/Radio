package interfaces

import "github.com/google/uuid"

// ChunkStore persists upload chunks on the filesystem and assembles them into a
// single finalized file. It is intentionally storage-agnostic so the
// application layer stays testable.
type ChunkStore interface {
	// Write stores a single chunk for the given session and index.
	Write(sessionID uuid.UUID, index int, data []byte) error
	// Exists reports whether a chunk for the session and index is present.
	Exists(sessionID uuid.UUID, index int) (bool, error)
	// Delete removes all chunks belonging to a session.
	Delete(sessionID uuid.UUID) error
	// Assemble concatenates chunks 0..totalChunks-1 into finalPath.
	Assemble(sessionID uuid.UUID, totalChunks int, finalPath string) error
	// HashAndSize returns the sha256 hex digest and byte size of a file.
	HashAndSize(finalPath string) (hash string, size int64, err error)
}
