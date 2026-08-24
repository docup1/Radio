import { ref } from 'vue'
import { content } from '@/shared/api/content'
import type { Playlist, Song } from '@/shared/api/types'

export function usePlaylists() {
  const playlists = ref<Playlist[]>([])

  async function load() {
    playlists.value = await content.listPlaylists()
  }

  async function create(name: string) {
    const p = await content.createPlaylist(name)
    playlists.value.push(p)
    return p
  }

  return { playlists, load, create }
}

export function usePlaylistSongs(id: string) {
  const songs = ref<Song[]>([])

  async function load() {
    songs.value = await content.listPlaylistSongs(id)
  }

  return { songs, load }
}
