import { reactive } from 'vue'
import type { Song } from '@/shared/api/types'
import { audioURL } from '@/shared/api/content'

const state = reactive({
  current: null as Song | null,
  queue: [] as Song[],
  index: -1,
  isPlaying: false,
})

let audio: HTMLAudioElement | null = null

function ensure(): HTMLAudioElement {
  if (!audio) {
    audio = new Audio()
    audio.addEventListener('play', () => (state.isPlaying = true))
    audio.addEventListener('pause', () => (state.isPlaying = false))
    audio.addEventListener('ended', () => next())
  }
  return audio
}

function playSong(song: Song, queue: Song[] = []) {
  const a = ensure()
  const q = queue.length ? queue : [song]
  let idx = q.findIndex((s) => s.id === song.id)
  if (idx < 0) {
    q.unshift(song)
    idx = 0
  }
  state.queue = q
  state.index = idx
  state.current = song
  a.src = audioURL(song.id)
  void a.play()
}

function toggle() {
  const a = ensure()
  if (!state.current) return
  if (a.paused) void a.play()
  else a.pause()
}

function next() {
  if (state.index < state.queue.length - 1) {
    playSong(state.queue[state.index + 1], state.queue)
  } else {
    ensure().pause()
    state.isPlaying = false
  }
}

function prev() {
  if (state.index > 0) playSong(state.queue[state.index - 1], state.queue)
}

function stop() {
  ensure().pause()
  state.current = null
  state.isPlaying = false
}

export function usePlayer() {
  return { state, playSong, toggle, next, prev, stop }
}
