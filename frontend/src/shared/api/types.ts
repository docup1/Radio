export interface Credentials {
  username: string
  password: string
}

export interface User {
  id: string
  username: string
}

export interface PasswordUpdate {
  current_password: string
  new_password: string
}

export interface ApiErrorBody {
  error: string
}

// ---- Content service ----

export type SongScope = 'mine' | 'public'

export interface Song {
  id: string
  owner_id: string
  name: string
  description: string
  melody_id: string
  image_id: string | null
  is_public: boolean
  created_at: string
  updated_at: string
}

export interface Playlist {
  id: string
  owner_id: string
  name: string
  created_at: string
  updated_at: string
}

export interface UploadSession {
  id: string
  owner_id: string
  media_type: 'audio' | 'image'
  status: string
  content_type: string
  total_chunks: number
  received_chunks: number
  size: number
  hash: string
  created_at: string
  updated_at: string
}

export interface CreateSongInput {
  name: string
  description?: string
  melody_id: string
  image_id?: string
  is_public: boolean
}

export interface UpdateSongInput {
  name?: string
  description?: string
  melody_id?: string
  image_id?: string
  is_public?: boolean
}

export interface InitUploadInput {
  media_type: 'audio' | 'image'
  content_type: string
  total_chunks: number
  expected_size: number
  expected_hash?: string
}

// ---- Stream service ----

export interface Stream {
  id: string
  owner_id: string
  name: string
  description: string
  loop: boolean
  created_at: string
  updated_at: string
}

export interface StreamState {
  stream_id: string
  current_queue_id: string | null
  song_id: string | null
  started_at: string | null
  is_active: boolean
  revision: number
}

export interface QueueItem {
  id: string
  stream_id: string
  song_id: string
  position: number
  song?: Song
}

export interface CreateStreamInput {
  name: string
  description?: string
  loop?: boolean
}

export interface UpdateStreamInput {
  name?: string
  description?: string
  loop?: boolean
}
