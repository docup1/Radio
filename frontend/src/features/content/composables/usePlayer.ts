import { reactive } from 'vue'
import type { Song } from '@/shared/api/types'
import { audioURL } from '@/shared/api/content'

export const state = reactive({
  current: null as Song | null,
  queue: [] as Song[],
  index: -1,
  isPlaying: false,
  currentTime: 0,
  duration: 0,
  error: '' as string,
  loading: false,
})

let audio: HTMLAudioElement | null = null

function formatTime(sec: number): string {
  if (!isFinite(sec) || sec < 0) return '0:00'
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}

function ensure(): HTMLAudioElement {
  if (!audio) {
    audio = new Audio()
    audio.preload = 'metadata'
    audio.addEventListener('play', () => {
      state.isPlaying = true
      state.loading = false
    })
    audio.addEventListener('pause', () => (state.isPlaying = false))
    audio.addEventListener('waiting', () => (state.loading = true))
    audio.addEventListener('canplay', () => (state.loading = false))
    audio.addEventListener('timeupdate', () => {
      state.currentTime = audio!.currentTime
    })
    audio.addEventListener('durationchange', () => {
      state.duration = isFinite(audio!.duration) ? audio!.duration : 0
    })
    audio.addEventListener('loadedmetadata', () => {
      state.duration = isFinite(audio!.duration) ? audio!.duration : 0
    })
    audio.addEventListener('ended', () => {
      if (state.index < state.queue.length - 1) {
        next()
      } else {
        state.isPlaying = false
      }
    })
    audio.addEventListener('error', () => {
      const msg =
        audio?.error?.message || 'Ошибка загрузки аудио (нет доступа или файл повреждён)'
      state.error = msg
      state.isPlaying = false
      state.loading = false
      // auto-next on error after short delay
      setTimeout(() => {
        if (state.error && state.index < state.queue.length - 1) {
          state.error = ''
          next()
        }
      }, 2000)
    })
  }
  return audio
}

function playSong(song: Song, queue: Song[] = []) {
  const a = ensure()
  const q = queue.length ? [...queue] : [song]
  let idx = q.findIndex((s) => s.id === song.id)
  if (idx < 0) {
    q.unshift(song)
    idx = 0
  }
  state.queue = q
  state.index = idx
  state.current = song
  state.error = ''
  state.loading = true
  state.currentTime = 0
  state.duration = 0
  a.src = audioURL(song.id)
  a.currentTime = 0
  const p = a.play()
  if (p && typeof p.catch === 'function') {
    p.catch((e: unknown) => {
      const msg = e instanceof Error ? e.message : 'Не удалось воспроизвести'
      // NotFound / Forbidden from gateway will surface as network error here
      state.error = msg
      state.loading = false
    })
  }
}

function toggle() {
  const a = ensure()
  if (!state.current) return
  if (a.paused) {
    state.error = ''
    void a.play().catch((e: unknown) => {
      state.error = e instanceof Error ? e.message : 'Ошибка воспроизведения'
    })
  } else {
    a.pause()
  }
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
  if (state.index > 0) {
    playSong(state.queue[state.index - 1], state.queue)
  } else {
    // restart current
    const a = ensure()
    a.currentTime = 0
    state.currentTime = 0
  }
}

function seek(time: number) {
  const a = ensure()
  if (!state.current || !isFinite(time)) return
  const t = Math.max(0, Math.min(time, state.duration || time))
  a.currentTime = t
  state.currentTime = t
}

function seekByRatio(ratio: number) {
  if (!state.duration) return
  seek(ratio * state.duration)
}

function stop() {
  const a = ensure()
  a.pause()
  a.removeAttribute('src')
  a.load()
  state.current = null
  state.isPlaying = false
  state.currentTime = 0
  state.duration = 0
  state.error = ''
  state.loading = false
}

export function usePlayer() {
  return { state, playSong, toggle, next, prev, stop, seek, seekByRatio, formatTime }
}
