package inmem

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

func paginate[T any](s []T, limit, offset int) []T {
	if offset >= len(s) {
		return []T{}
	}
	end := offset + limit
	if end > len(s) {
		end = len(s)
	}
	return s[offset:end]
}

func sortCreatedDesc[T any](s []T, get func(T) time.Time) {
	sort.Slice(s, func(i, j int) bool { return get(s[i]).After(get(s[j])) })
}

// MelodyRepository is an in-memory implementation of interfaces.MelodyRepository.
type MelodyRepository struct {
	mu sync.RWMutex
	m  map[uuid.UUID]models.Melody
}

func NewMelodyRepository() *MelodyRepository {
	return &MelodyRepository{m: make(map[uuid.UUID]models.Melody)}
}

func (r *MelodyRepository) Create(_ context.Context, m models.Melody) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[m.ID] = m
	return nil
}

func (r *MelodyRepository) Get(_ context.Context, id uuid.UUID) (*models.Melody, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.m[id]
	if !ok {
		return nil, interfaces.ErrNotFound
	}
	return &m, nil
}

func (r *MelodyRepository) List(_ context.Context, limit, offset int) ([]models.Melody, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]models.Melody, 0, len(r.m))
	for _, m := range r.m {
		out = append(out, m)
	}
	sortCreatedDesc(out, func(m models.Melody) time.Time { return m.CreatedAt })
	return paginate(out, limit, offset), nil
}

func (r *MelodyRepository) Update(_ context.Context, id uuid.UUID, patch interfaces.MelodyPatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.m[id]
	if !ok {
		return interfaces.ErrNotFound
	}
	if patch.Path != nil {
		m.Path = *patch.Path
	}
	if patch.ContentType != nil {
		m.ContentType = *patch.ContentType
	}
	r.m[id] = m
	return nil
}

func (r *MelodyRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[id]; !ok {
		return interfaces.ErrNotFound
	}
	delete(r.m, id)
	return nil
}

func (r *MelodyRepository) Exists(_ context.Context, id uuid.UUID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.m[id]
	return ok, nil
}

// ImageRepository is an in-memory implementation of interfaces.ImageRepository.
type ImageRepository struct {
	mu sync.RWMutex
	m  map[uuid.UUID]models.Image
}

func NewImageRepository() *ImageRepository {
	return &ImageRepository{m: make(map[uuid.UUID]models.Image)}
}

func (r *ImageRepository) Create(_ context.Context, m models.Image) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[m.ID] = m
	return nil
}

func (r *ImageRepository) Get(_ context.Context, id uuid.UUID) (*models.Image, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.m[id]
	if !ok {
		return nil, interfaces.ErrNotFound
	}
	return &m, nil
}

func (r *ImageRepository) List(_ context.Context, limit, offset int) ([]models.Image, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]models.Image, 0, len(r.m))
	for _, m := range r.m {
		out = append(out, m)
	}
	sortCreatedDesc(out, func(m models.Image) time.Time { return m.CreatedAt })
	return paginate(out, limit, offset), nil
}

func (r *ImageRepository) Update(_ context.Context, id uuid.UUID, patch interfaces.ImagePatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.m[id]
	if !ok {
		return interfaces.ErrNotFound
	}
	if patch.Path != nil {
		m.Path = *patch.Path
	}
	if patch.ContentType != nil {
		m.ContentType = *patch.ContentType
	}
	r.m[id] = m
	return nil
}

func (r *ImageRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[id]; !ok {
		return interfaces.ErrNotFound
	}
	delete(r.m, id)
	return nil
}

func (r *ImageRepository) Exists(_ context.Context, id uuid.UUID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.m[id]
	return ok, nil
}

// SongRepository is an in-memory implementation of interfaces.SongRepository.
type SongRepository struct {
	mu sync.RWMutex
	m  map[uuid.UUID]models.Song
}

func NewSongRepository() *SongRepository {
	return &SongRepository{m: make(map[uuid.UUID]models.Song)}
}

func (r *SongRepository) Create(_ context.Context, s models.Song) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[s.ID] = s
	return nil
}

func (r *SongRepository) GetVisible(_ context.Context, id, viewer uuid.UUID) (*models.Song, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.m[id]
	if !ok || !(s.OwnerID == viewer || s.IsPublic) {
		return nil, interfaces.ErrNotFound
	}
	return &s, nil
}

func (r *SongRepository) ListVisible(_ context.Context, viewer uuid.UUID, limit, offset int) ([]models.Song, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]models.Song, 0, len(r.m))
	for _, s := range r.m {
		if s.OwnerID == viewer || s.IsPublic {
			out = append(out, s)
		}
	}
	sortCreatedDesc(out, func(s models.Song) time.Time { return s.CreatedAt })
	return paginate(out, limit, offset), nil
}

func (r *SongRepository) Update(_ context.Context, id, owner uuid.UUID, patch interfaces.SongPatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.m[id]
	if !ok || s.OwnerID != owner {
		return interfaces.ErrNotFound
	}
	if patch.Name != nil {
		s.Name = *patch.Name
	}
	if patch.Description != nil {
		s.Description = *patch.Description
	}
	if patch.MelodyID != nil {
		s.MelodyID = *patch.MelodyID
	}
	if patch.ImageID != nil {
		s.ImageID = *patch.ImageID
	}
	if patch.IsPublic != nil {
		s.IsPublic = *patch.IsPublic
	}
	s.UpdatedAt = time.Now()
	r.m[id] = s
	return nil
}

func (r *SongRepository) Delete(_ context.Context, id, owner uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.m[id]
	if !ok || s.OwnerID != owner {
		return interfaces.ErrNotFound
	}
	delete(r.m, id)
	return nil
}

func (r *SongRepository) SearchVisible(_ context.Context, q string, viewer uuid.UUID, limit, offset int) ([]models.Song, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	needle := strings.ToLower(q)
	out := make([]models.Song, 0, len(r.m))
	for _, s := range r.m {
		if !(s.OwnerID == viewer || s.IsPublic) {
			continue
		}
		hay := strings.ToLower(s.Name + " " + s.Description)
		if strings.Contains(hay, needle) {
			out = append(out, s)
		}
	}
	sortCreatedDesc(out, func(s models.Song) time.Time { return s.CreatedAt })
	return paginate(out, limit, offset), nil
}

func (r *SongRepository) Exists(_ context.Context, id uuid.UUID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.m[id]
	return ok, nil
}

func (r *SongRepository) SongByMelody(_ context.Context, melodyID uuid.UUID) (*models.Song, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.m {
		if s.MelodyID == melodyID {
			return &s, nil
		}
	}
	return nil, interfaces.ErrNotFound
}

// PlaylistRepository is an in-memory implementation of interfaces.PlaylistRepository.
type PlaylistRepository struct {
	mu sync.RWMutex
	m  map[uuid.UUID]models.Playlist
}

func NewPlaylistRepository() *PlaylistRepository {
	return &PlaylistRepository{m: make(map[uuid.UUID]models.Playlist)}
}

func (r *PlaylistRepository) Create(_ context.Context, p models.Playlist) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[p.ID] = p
	return nil
}

func (r *PlaylistRepository) GetVisible(_ context.Context, id, viewer uuid.UUID) (*models.Playlist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.m[id]
	if !ok || !(p.OwnerID == viewer || p.IsPublic) {
		return nil, interfaces.ErrNotFound
	}
	return &p, nil
}

func (r *PlaylistRepository) ListVisible(_ context.Context, viewer uuid.UUID, limit, offset int) ([]models.Playlist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]models.Playlist, 0, len(r.m))
	for _, p := range r.m {
		if p.OwnerID == viewer || p.IsPublic {
			out = append(out, p)
		}
	}
	sortCreatedDesc(out, func(p models.Playlist) time.Time { return p.CreatedAt })
	return paginate(out, limit, offset), nil
}

func (r *PlaylistRepository) Update(_ context.Context, id, owner uuid.UUID, patch interfaces.PlaylistPatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.m[id]
	if !ok || p.OwnerID != owner {
		return interfaces.ErrNotFound
	}
	if patch.Name != nil {
		p.Name = *patch.Name
	}
	if patch.IsPublic != nil {
		p.IsPublic = *patch.IsPublic
	}
	p.UpdatedAt = time.Now()
	r.m[id] = p
	return nil
}

func (r *PlaylistRepository) Delete(_ context.Context, id, owner uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.m[id]
	if !ok || p.OwnerID != owner {
		return interfaces.ErrNotFound
	}
	delete(r.m, id)
	return nil
}

func (r *PlaylistRepository) Exists(_ context.Context, id uuid.UUID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.m[id]
	return ok, nil
}

func (r *PlaylistRepository) OwnerOf(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.m[id]
	if !ok {
		return uuid.Nil, interfaces.ErrNotFound
	}
	return p.OwnerID, nil
}

// PlaylistSongRepository is an in-memory implementation of interfaces.PlaylistSongRepository.
type PlaylistSongRepository struct {
	mu sync.RWMutex
	m  map[uuid.UUID][]models.PlaylistSong
}

func NewPlaylistSongRepository() *PlaylistSongRepository {
	return &PlaylistSongRepository{m: make(map[uuid.UUID][]models.PlaylistSong)}
}

func (r *PlaylistSongRepository) Add(_ context.Context, ps models.PlaylistSong) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.m[ps.PlaylistID] {
		if e.SongID == ps.SongID {
			return interfaces.ErrConflict
		}
		if e.Position == ps.Position {
			return interfaces.ErrConflict
		}
	}
	r.m[ps.PlaylistID] = append(r.m[ps.PlaylistID], ps)
	return nil
}

func (r *PlaylistSongRepository) Remove(_ context.Context, playlistID, songID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	before := len(r.m[playlistID])
	kept := r.m[playlistID][:0]
	for _, e := range r.m[playlistID] {
		if e.SongID != songID {
			kept = append(kept, e)
		}
	}
	r.m[playlistID] = kept
	if len(kept) == before {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *PlaylistSongRepository) Move(_ context.Context, playlistID, songID uuid.UUID, position int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.m[playlistID]
	for i, e := range list {
		if e.SongID == songID {
			for _, o := range list {
				if o.SongID != songID && o.Position == position {
					return interfaces.ErrConflict
				}
			}
			list[i].Position = position
			return nil
		}
	}
	return interfaces.ErrNotFound
}

func (r *PlaylistSongRepository) List(_ context.Context, playlistID uuid.UUID) ([]models.PlaylistSong, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := append([]models.PlaylistSong(nil), r.m[playlistID]...)
	sort.Slice(list, func(i, j int) bool { return list[i].Position < list[j].Position })
	return list, nil
}

func (r *PlaylistSongRepository) MaxPosition(_ context.Context, playlistID uuid.UUID) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	max := 0
	for _, e := range r.m[playlistID] {
		if e.Position > max {
			max = e.Position
		}
	}
	return max, nil
}

func (r *PlaylistSongRepository) SongVisibleTo(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	// In-memory store assumes the referenced song is visible. The Postgres
	// implementation performs the real ownership/publicity check.
	return true, nil
}

// UploadSessionRepository is an in-memory implementation of interfaces.UploadSessionRepository.
type UploadSessionRepository struct {
	mu sync.RWMutex
	m  map[uuid.UUID]models.UploadSession
}

func NewUploadSessionRepository() *UploadSessionRepository {
	return &UploadSessionRepository{m: make(map[uuid.UUID]models.UploadSession)}
}

func (r *UploadSessionRepository) Create(_ context.Context, s models.UploadSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[s.ID] = s
	return nil
}

func (r *UploadSessionRepository) Get(_ context.Context, id uuid.UUID) (*models.UploadSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.m[id]
	if !ok {
		return nil, interfaces.ErrNotFound
	}
	return &s, nil
}

func (r *UploadSessionRepository) Update(_ context.Context, id uuid.UUID, patch interfaces.UploadSessionPatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.m[id]
	if !ok {
		return interfaces.ErrNotFound
	}
	if patch.Status != nil {
		s.Status = *patch.Status
	}
	if patch.ReceivedChunks != nil {
		s.ReceivedChunks = *patch.ReceivedChunks
	}
	if patch.FinalPath != nil {
		s.FinalPath = *patch.FinalPath
	}
	if patch.Size != nil {
		s.Size = *patch.Size
	}
	if patch.Hash != nil {
		s.Hash = *patch.Hash
	}
	r.m[id] = s
	return nil
}

func (r *UploadSessionRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[id]; !ok {
		return interfaces.ErrNotFound
	}
	delete(r.m, id)
	return nil
}

func (r *UploadSessionRepository) ListByOwner(_ context.Context, owner uuid.UUID, limit, offset int) ([]models.UploadSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]models.UploadSession, 0, len(r.m))
	for _, s := range r.m {
		if s.OwnerID == owner {
			out = append(out, s)
		}
	}
	sortCreatedDesc(out, func(s models.UploadSession) time.Time { return s.CreatedAt })
	return paginate(out, limit, offset), nil
}
