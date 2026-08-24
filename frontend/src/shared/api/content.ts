import { request } from './client'
import type {
  Song,
  SongScope,
  Playlist,
  UploadSession,
  CreateSongInput,
  UpdateSongInput,
  InitUploadInput,
} from './types'

export const content = {
  listSongs: (scope: SongScope = 'mine', limit = 20, offset = 0) =>
    request<Song[]>('GET', `/api/content/songs?scope=${scope}&limit=${limit}&offset=${offset}`),

  searchSongs: (q: string, scope: SongScope = 'mine', limit = 20, offset = 0) =>
    request<Song[]>(
      'GET',
      `/api/content/songs/search?q=${encodeURIComponent(q)}&scope=${scope}&limit=${limit}&offset=${offset}`,
    ),

  getSong: (id: string) => request<Song>('GET', `/api/content/songs/${id}`),

  createSong: (input: CreateSongInput) => request<Song>('POST', '/api/content/songs', input),

  updateSong: (id: string, patch: UpdateSongInput) =>
    request<Song>('PUT', `/api/content/songs/${id}`, patch),

  deleteSong: (id: string) => request<void>('DELETE', `/api/content/songs/${id}`),

  createUpload: (input: InitUploadInput) =>
    request<UploadSession>('POST', '/api/content/uploads', input),

  uploadChunk: (id: string, index: number, body: Blob) =>
    fetch(`/api/content/uploads/${id}/chunks/${index}`, {
      method: 'PUT',
      body,
      credentials: 'include',
    }).then((r) => {
      if (!r.ok) throw new Error('chunk upload failed')
    }),

  confirmUpload: (id: string) =>
    request<{ id: string }>('POST', `/api/content/uploads/${id}/confirm`),

  listPlaylists: (limit = 50, offset = 0) =>
    request<Playlist[]>('GET', `/api/content/playlists?limit=${limit}&offset=${offset}`),

  createPlaylist: (name: string) =>
    request<Playlist>('POST', '/api/content/playlists', { name }),

  deletePlaylist: (id: string) => request<void>('DELETE', `/api/content/playlists/${id}`),

  listPlaylistSongs: (id: string) =>
    request<Song[]>('GET', `/api/content/playlists/${id}/songs`),

  addSongToPlaylist: (id: string, songId: string, position?: number) =>
    request<void>('POST', `/api/content/playlists/${id}/songs`, { song_id: songId, position }),

  removeSongFromPlaylist: (id: string, songId: string) =>
    request<void>('DELETE', `/api/content/playlists/${id}/songs/${songId}`),

  moveSongInPlaylist: (id: string, songId: string, position: number) =>
    request<void>('PUT', `/api/content/playlists/${id}/songs/${songId}`, { position }),
}

export const audioURL = (songId: string) => `/api/content/songs/${songId}/audio`
export const imageURL = (imageId: string) => `/api/content/images/${imageId}/file`
