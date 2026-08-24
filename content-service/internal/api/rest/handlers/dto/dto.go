package dto

type CreateSongRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	MelodyID    string  `json:"melody_id"`
	ImageID     *string `json:"image_id"`
	IsPublic    bool    `json:"is_public"`
}

type UpdateSongRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	MelodyID    *string `json:"melody_id"`
	ImageID     *string `json:"image_id"`
	IsPublic    *bool   `json:"is_public"`
}

type CreateImageRequest struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
}

type UpdateMelodyRequest struct {
	Path        *string `json:"path"`
	ContentType *string `json:"content_type"`
}

type CreatePlaylistRequest struct {
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
}

type UpdatePlaylistRequest struct {
	Name     *string `json:"name"`
	IsPublic *bool   `json:"is_public"`
}

type AddSongRequest struct {
	SongID   string `json:"song_id"`
	Position *int   `json:"position"`
}

type MoveSongRequest struct {
	Position int `json:"position"`
}

type InitUploadRequest struct {
	MediaType    string `json:"media_type"`
	ContentType  string `json:"content_type"`
	TotalChunks  int    `json:"total_chunks"`
	ExpectedSize int64  `json:"expected_size"`
	ExpectedHash string `json:"expected_hash"`
}
