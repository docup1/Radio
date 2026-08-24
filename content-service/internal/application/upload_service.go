package application

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type UploadService struct {
	UploadSessions interfaces.UploadSessionRepository
	Melodies       interfaces.MelodyRepository
	Images         interfaces.ImageRepository
	Chunks         interfaces.ChunkStore
	FinalRoot      string
	MaxChunkSize   int64
	MaxFileSize    int64
}

type InitUploadInput struct {
	MediaType   string
	ContentType string
	TotalChunks int
	// ExpectedSize and ExpectedHash are optional integrity expectations verified
	// on confirm. ExpectedHash is a hex-encoded sha256 of the full file.
	ExpectedSize  int64
	ExpectedHash  string
}

type AddChunkInput struct {
	Index int
	Data  []byte
}

func (s *UploadService) Init(ctx context.Context, owner uuid.UUID, in InitUploadInput) (*models.UploadSession, error) {
	if in.MediaType != models.MediaTypeAudio && in.MediaType != models.MediaTypeImage {
		return nil, interfaces.ErrInvalid
	}
	maxCT := models.MelodyContentTypeMaxLength
	if in.MediaType == models.MediaTypeImage {
		maxCT = models.ImageContentTypeMaxLength
	}
	if in.ContentType == "" || len(in.ContentType) > maxCT {
		return nil, interfaces.ErrInvalid
	}
	if in.TotalChunks <= 0 {
		return nil, interfaces.ErrInvalid
	}
	if in.ExpectedSize < 0 || in.ExpectedSize > s.MaxFileSize {
		return nil, interfaces.ErrInvalid
	}
	if len(in.ExpectedHash) > 64 {
		return nil, interfaces.ErrInvalid
	}

	session := models.UploadSession{
		ID:            uuid.New(),
		OwnerID:       owner,
		MediaType:     in.MediaType,
		Status:        models.UploadStatusInitialized,
		ContentType:   in.ContentType,
		TotalChunks:   in.TotalChunks,
		ReceivedChunks: 0,
		Size:          in.ExpectedSize,
		Hash:          in.ExpectedHash,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.UploadSessions.Create(ctx, session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *UploadService) AddChunk(ctx context.Context, owner uuid.UUID, sessionID uuid.UUID, in AddChunkInput) error {
	if in.Index < 0 {
		return interfaces.ErrInvalid
	}
	if int64(len(in.Data)) > s.MaxChunkSize {
		return interfaces.ErrInvalid
	}

	session, err := s.UploadSessions.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.OwnerID != owner {
		return interfaces.ErrNotFound
	}
	if session.Status != models.UploadStatusInitialized {
		return interfaces.ErrConflict
	}
	if in.Index >= session.TotalChunks {
		return interfaces.ErrInvalid
	}

	exists, err := s.Chunks.Exists(sessionID, in.Index)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if err := s.Chunks.Write(sessionID, in.Index, in.Data); err != nil {
		return err
	}
	received := session.ReceivedChunks + 1
	if err := s.UploadSessions.Update(ctx, sessionID, interfaces.UploadSessionPatch{
		ReceivedChunks: &received,
	}); err != nil {
		return err
	}
	return nil
}

// ConfirmResult is the outcome of a completed upload. Exactly one of Melody or
// Image is set, matching the session's media type.
type ConfirmResult struct {
	Melody *models.Melody
	Image  *models.Image
}

// Confirm verifies all chunks are present and (when provided) that the assembled
// file matches the expected size and sha256, then creates the melody (audio) or
// image (cover) record depending on the session's media type.
func (s *UploadService) Confirm(ctx context.Context, owner uuid.UUID, sessionID uuid.UUID) (*ConfirmResult, error) {
	session, err := s.UploadSessions.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.OwnerID != owner {
		return nil, interfaces.ErrNotFound
	}
	if session.Status != models.UploadStatusInitialized {
		return nil, interfaces.ErrConflict
	}
	if session.ReceivedChunks != session.TotalChunks {
		return nil, interfaces.ErrInvalid
	}
	for i := 0; i < session.TotalChunks; i++ {
		ok, err := s.Chunks.Exists(sessionID, i)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, interfaces.ErrInvalid
		}
	}

	finalPath := filepath.Join(s.FinalRoot, session.OwnerID.String(), session.MediaType, session.ID.String())
	if err := s.Chunks.Assemble(sessionID, session.TotalChunks, finalPath); err != nil {
		return nil, err
	}

	hash, size, err := s.Chunks.HashAndSize(finalPath)
	if err != nil {
		_ = s.Chunks.Delete(sessionID)
		return nil, err
	}
	if session.Size > 0 && size != session.Size {
		_ = s.Chunks.Delete(sessionID)
		_ = removeFile(finalPath)
		return nil, interfaces.ErrInvalid
	}
	if session.Hash != "" && hash != session.Hash {
		_ = s.Chunks.Delete(sessionID)
		_ = removeFile(finalPath)
		return nil, interfaces.ErrInvalid
	}

	status := models.UploadStatusCompleted
	if err := s.UploadSessions.Update(ctx, sessionID, interfaces.UploadSessionPatch{
		Status:    &status,
		FinalPath: &finalPath,
		Size:      &size,
		Hash:      &hash,
	}); err != nil {
		return nil, err
	}
	_ = s.Chunks.Delete(sessionID)

	if session.MediaType == models.MediaTypeImage {
		img := models.Image{
			ID:          uuid.New(),
			Path:        finalPath,
			ContentType: session.ContentType,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.Images.Create(ctx, img); err != nil {
			_ = removeFile(finalPath)
			return nil, err
		}
		return &ConfirmResult{Image: &img}, nil
	}

	melody := models.Melody{
		ID:          uuid.New(),
		Path:        finalPath,
		ContentType: session.ContentType,
		Size:        size,
		Hash:        hash,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.Melodies.Create(ctx, melody); err != nil {
		_ = removeFile(finalPath)
		return nil, err
	}
	return &ConfirmResult{Melody: &melody}, nil
}

func removeFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
