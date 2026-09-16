import { reactive } from 'vue'
import { streamApi } from '@/shared/api/stream'
import { content, audioURL } from '@/shared/api/content'
import { user } from '@/shared/store/auth'
import type { Stream } from '@/shared/api/types'

export type RadioPhase = 'idle' | 'connecting' | 'playing' | 'stopped' | 'ended' | 'error'

export interface RadioFeedItem {
  type: 'song' | 'song_ended' | 'stream_ended' | 'stream_stopped' | 'error'
  songId?: string
  songName?: string
  message?: string
  at: number
}

export interface RadioState {
  phase: RadioPhase
  streamId: string | null
  stream: Stream | null
  song: { id: string; name?: string } | null
  isActive: boolean
  queueLength: number
  currentItemId: string | null
  feed: RadioFeedItem[]
  error: string
  resumeRequired: boolean
  pulseScale: number
}

const WS_BASE = (() => {
  if (typeof window === 'undefined') return ''
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}`
})()

const STORAGE_KEY = 'radio:active'
const MAX_FEED = 30
const MAX_PENDING = 256

export const radio = reactive<RadioState>({
  phase: 'idle',
  streamId: null,
  stream: null,
  song: null,
  isActive: false,
  queueLength: 0,
  currentItemId: null,
  feed: [],
  error: '',
  resumeRequired: false,
  pulseScale: 1,
})

let ws: WebSocket | null = null
let audio: HTMLAudioElement | null = null
let mediaSource: MediaSource | null = null
let sourceBuffer: SourceBuffer | null = null
let audioCtx: AudioContext | null = null
let analyser: AnalyserNode | null = null
let animFrame = 0
let reconnectTimer = 0
let attempts = 0
let wanted = false
let useFallback = false
let drainTimer = 0
let watchdogTimer = 0
let appendErrors = 0
let pendingRemove: { start: number; end: number } | null = null
let resetPending = false
let wantAudio = false
let songFetchSeq = 0
let activeSongId: string | null = null

const appendQueue: ArrayBuffer[] = []

export function isOwnerFor(streamId: string | null): boolean {
  return !!streamId && user.value?.id === streamId
}

export async function openStream(streamId: string) {
  if (radio.streamId === streamId && ws) {
    fetchSongMeta(radio.song?.id ?? null)
    return
  }

  teardownEngine()
  radio.streamId = streamId
  radio.stream = null
  radio.song = null
  radio.feed = []
  radio.error = ''
  radio.resumeRequired = false
  radio.isActive = false
  radio.currentItemId = null
  activeSongId = null

  try {
    localStorage.setItem(STORAGE_KEY, streamId)
  } catch {
    // ignore storage errors
  }

  streamApi
    .get(streamId)
    .then((s) => {
      if (radio.streamId === streamId) radio.stream = s
    })
    .catch(() => {
      if (radio.streamId === streamId) radio.stream = null
    })

  connectWs()
}

export function initRadio() {
  if (typeof window === 'undefined') return
  let id: string | null = null
  try {
    id = localStorage.getItem(STORAGE_KEY)
  } catch {
    // ignore storage errors
  }
  if (id) openStream(id)
}

export function closeRadio() {
  wanted = false
  clearTimeout(reconnectTimer)
  reconnectTimer = 0
  try {
    ws?.close()
  } catch {
    // ignore
  }
  ws = null
  attempts = 0
  teardownEngine()
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // ignore
  }
  radio.streamId = null
  radio.stream = null
  radio.song = null
  radio.feed = []
  radio.isActive = false
  radio.queueLength = 0
  radio.currentItemId = null
  radio.error = ''
  radio.resumeRequired = false
  radio.phase = 'idle'
  activeSongId = null
}

export function start() {
  send({ type: 'start', loop: !!radio.stream?.loop })
}

export function stop() {
  wantAudio = false
  send({ type: 'stop' })
}

export function skip() {
  send({ type: 'skip' })
}

export function resume() {
  radio.resumeRequired = false
  audio?.play().catch(() => {})
}

// --- connection ---

function send(obj: unknown) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(obj))
  }
}

function connectWs() {
  const id = radio.streamId
  if (!id) return

  wanted = true
  attempts = 0
  radio.phase = 'connecting'

  try {
    ws?.close()
  } catch {
    // ignore
  }

  ws = new WebSocket(`${WS_BASE}/api/streams/${id}/ws`)
  ws.binaryType = 'arraybuffer'
  ws.onopen = () => {}
  ws.onmessage = onMessage
  ws.onerror = () => {}
  ws.onclose = onClose
}

function onClose() {
  ws = null
  teardownEngine()
  if (!wanted || !radio.streamId) return

  radio.phase = 'connecting'
  attempts++
  const delay = Math.min(1000 * 2 ** Math.min(attempts, 4), 10000)
  clearTimeout(reconnectTimer)
  reconnectTimer = window.setTimeout(() => {
    if (wanted && radio.streamId) connectWs()
  }, delay)
}

function onMessage(ev: MessageEvent) {
  if (typeof ev.data === 'string') {
    try {
      handleMessage(JSON.parse(ev.data))
    } catch {
      // ignore malformed frames
    }
    return
  }
  if (!(ev.data instanceof ArrayBuffer)) return
  handleChunk(ev.data)
}

function handleMessage(msg: Record<string, unknown>) {
  switch (msg.type) {
    case 'state': {
      const d = (msg.data ?? {}) as Record<string, unknown>
      radio.isActive = !!d.is_active
      wantAudio = radio.isActive
      radio.queueLength = Number(d.queue_length ?? 0) || 0
      radio.currentItemId = typeof d.current_item_id === 'string' ? d.current_item_id : null
      if (radio.phase === 'connecting') {
        radio.phase = radio.isActive ? 'playing' : 'stopped'
      }
      if (typeof d.current_song_id === 'string') fetchSongMeta(d.current_song_id)
      break
    }
    case 'song': {
      if (typeof msg.songId === 'string') {
        fetchSongMeta(msg.songId)
        radio.song = { id: msg.songId }
      }
      radio.phase = 'playing'
      radio.isActive = true
      wantAudio = true
      resetPlayback(typeof msg.songId === 'string' ? msg.songId : undefined)
      break
    }
    case 'song_ended':
      pushFeed('song_ended', msg.songId)
      break
    case 'stream_ended':
      pushFeed('stream_ended', undefined, msg.message)
      radio.isActive = false
      radio.phase = 'ended'
      teardownEngine()
      break
    case 'stream_stopped':
      pushFeed('stream_stopped')
      radio.isActive = false
      radio.phase = 'stopped'
      teardownEngine()
      break
    case 'error':
      pushFeed('error', undefined, msg.message)
      radio.error = String(msg.message ?? 'Ошибка')
      break
    default:
      break
  }
}

function pushFeed(type: RadioFeedItem['type'], songId?: unknown, message?: unknown) {
  const id = typeof songId === 'string' ? songId : undefined
  let songName: string | undefined
  if (type === 'song' && id) {
    songName = radio.song?.id === id ? radio.song.name : undefined
  }
  radio.feed.unshift({
    type,
    songId: id,
    songName,
    message: typeof message === 'string' ? message : undefined,
    at: Date.now(),
  })
  if (radio.feed.length > MAX_FEED) radio.feed.length = MAX_FEED
}

function fetchSongMeta(id: string | null) {
  if (!id) return
  if (radio.song?.id === id && radio.song.name !== undefined) return

  const seq = ++songFetchSeq
  content
    .getSong(id)
    .then((s) => {
      if (seq !== songFetchSeq) return
      radio.song = { id: s.id, name: s.name }
    })
    .catch(() => {
      if (seq !== songFetchSeq) return
      radio.song = { id }
    })
}

// --- playback engine ---

function ensureAudioCtx() {
  if (audioCtx) return
  audioCtx = new AudioContext()
  analyser = audioCtx.createAnalyser()
  analyser.fftSize = 256
}

function hookAudioAnalyser() {
  if (!audio || !audioCtx || !analyser) return
  const source = audioCtx.createMediaElementSource(audio)
  source.connect(analyser)
  analyser.connect(audioCtx.destination)
}

function resetPlayback(songId?: string) {
  // A re-announce of the song that is already playing (repeat in the queue,
  // or a duplicate announce after a reconnect): the live buffer still holds
  // this track's frames, so rebuilding the engine here would reset the
  // timeline and drop the buffered tail — an audible gap at an identical seam.
  // Keep the buffer flowing and just make sure playback is running.
  if (songId && activeSongId === songId && audio && mediaSource) {
    wantAudio = true
    audio.play().catch(handlePlayBlock)
    return
  }
  activeSongId = songId ?? null
  if (!audio) {
    setupMse()
    if (!useFallback) {
      // Engine was torn down (e.g. a mid-song reconnect): the chunks already
      // queued below belong to the announced song and are preserved — a fresh
      // MediaSource drains them as soon as it opens, so playback resumes from
      // the live tail instead of stalling until the next chunk.
      return
    }
    // fallback: fall through to source assignment on the fresh element
  }
  if (useFallback) {
    if (!songId) return
    wantAudio = true
    try {
      audio.currentTime = 0
    } catch {
      // ignore
    }
    if (audio.src !== audioURL(songId)) {
      audio.src = audioURL(songId)
    }
    audio.play().catch(handlePlayBlock)
    return
  }
  appendQueue.length = 0
  pendingRemove = null
  if (sourceBuffer) {
    // A brand-new MediaSource per announced song: SourceBuffers hold the
    // previous track's frames, and removing/rebuilding them in-place leaves the
    // element stuck — the next song's chunks never reach the buffer and the
    // stream goes silent right after the first track.
    disposeMseEngine()
    setupMse()
  }
  restartPosition()
}

// restartPosition rewinds the playhead to the start of the new song. Without
// this the media element keeps sitting on the old track's time code while the
// new song is appended from 0, so it starves and the stream goes silent.
function restartPosition() {
  if (!audio || useFallback) return
  try {
    audio.currentTime = 0
  } catch {
    // ignore
  }
  audio.play().catch(handlePlayBlock)
}

// disposeMseEngine tears the MSE pipeline down but keeps the shared AudioContext.
// setupMse() rebuilds it with a clean element and MediaSource.
function disposeMseEngine() {
  clearInterval(drainTimer)
  drainTimer = 0
  clearInterval(watchdogTimer)
  watchdogTimer = 0
  appendQueue.length = 0
  pendingRemove = null
  resetPending = false
  appendErrors = 0

  if (sourceBuffer) {
    sourceBuffer.removeEventListener('error', onSourceBufferError)
    sourceBuffer = null
  }
  if (mediaSource) {
    try {
      if (mediaSource.readyState === 'open') mediaSource.endOfStream()
    } catch {
      // ignore
    }
    mediaSource = null
  }
  if (audio) {
    audio.pause()
    audio.src = ''
    audio.load()
    audio.remove()
    audio = null
  }
}

function setupMse() {
  const mseSupported =
    typeof MediaSource !== 'undefined' && MediaSource.isTypeSupported('audio/mpeg')

  if (!mseSupported) {
    useFallback = true
    setupFallback()
    return
  }
  useFallback = false

  ensureAudioCtx()
  audio = new Audio()
  audio.autoplay = true
  audio.style.display = 'none'
  document.body.appendChild(audio)
  hookAudioAnalyser()

  mediaSource = new MediaSource()
  const url = URL.createObjectURL(mediaSource)
  audio.src = url

  mediaSource.addEventListener(
    'sourceopen',
    () => {
      if (!mediaSource || sourceBuffer) return
      sourceBuffer = mediaSource.addSourceBuffer('audio/mpeg')
      sourceBuffer.addEventListener('error', onSourceBufferError)
      startDrain()
      doReset()
    },
    { once: true },
  )

  audio.addEventListener('error', () => {
    if (radio.phase !== 'idle') {
      radio.phase = 'error'
      radio.error = 'Ошибка воспроизведения'
    }
  })

  try {
    audio.play().catch(handlePlayBlock)
  } catch {
    // ignore
  }
  startPulse()
}

function setupFallback() {
  ensureAudioCtx()
  audio = new Audio()
  audio.autoplay = true
  audio.style.display = 'none'
  document.body.appendChild(audio)
  hookAudioAnalyser()

  // Browsers without MSE (Chrome/Firefox) cannot stream raw MPEG through a
  // SourceBuffer or a mutable blob: rebuilding the object URL never lets the
  // decoder progress past the first frames. Instead we play each announced
  // song straight from the static audio endpoint — one src per song, replaced
  // on every track change. The WebSocket stays authoritative for skip/end/stop.

  audio.addEventListener('error', () => {
    if (radio.phase !== 'idle') {
      radio.phase = 'error'
      radio.error = 'Ошибка воспроизведения'
    }
  })

  audio.addEventListener('ended', () => {
    // Natural end of the current file: wait for the next 'song' announce.
    // The watchdog must not auto-restart the same track.
    audio?.pause()
  })
  startPulse()
}

function handlePlayBlock(err: unknown) {
  const e = err as { name?: string }
  if (e?.name === 'NotAllowedError' || e?.name === 'NotSupportedError') {
    radio.resumeRequired = true
  }
}

function doReset() {
  if (!sourceBuffer) return
  try {
    sourceBuffer.abort()
  } catch {
    // ignore
  }
  let end = 0
  try {
    if (sourceBuffer.buffered.length > 0) {
      end = sourceBuffer.buffered.end(sourceBuffer.buffered.length - 1)
    }
  } catch {
    // ignore
  }
  if (end > 0) {
    pendingRemove = { start: 0, end }
    try {
      sourceBuffer.remove(0, end)
    } catch {
      pendingRemove = null
    }
  }
  resetPending = false
}

function startDrain() {
  clearInterval(drainTimer)
  drainTimer = window.setInterval(pump, 80)
  clearInterval(watchdogTimer)
  watchdogTimer = window.setInterval(watchdog, 1000)
  pump()
}

function pump() {
  if (!sourceBuffer || !mediaSource) return
  if (mediaSource.readyState !== 'open') return
  if (sourceBuffer.updating) return
  if (resetPending) {
    resetPending = false
    doReset()
    return
  }
  if (pendingRemove) {
    const { start, end } = pendingRemove
    pendingRemove = null
    try {
      sourceBuffer.remove(start, end)
    } catch {
      // ignore
    }
    return
  }
  if (appendQueue.length === 0) return
  const chunk = appendQueue.shift()!
  try {
    sourceBuffer.appendBuffer(chunk)
    appendErrors = 0
    catchUpToLive()
  } catch {
    appendErrors++
    if (appendErrors > 3) {
      appendQueue.length = 0
      resetPending = true
      appendErrors = 0
    }
  }
}

function watchdog() {
  if (!audio) return
  // Failsafe for the fallback: if the stream is expected to play but the
  // element got paused (e.g. after a song switch or an interrupted autoplay),
  // nudge it back into playback. Never restart a track that already ended on
  // its own — the next 'song' announce plays the next file.
  if (wantAudio && audio.paused && !audio.ended) {
    audio.play().catch(handlePlayBlock)
  }
  if (useFallback || audio.paused) return
  if (audio.readyState === 2 && mediaSource?.readyState === 'open') {
    audio.play().catch(handlePlayBlock)
  }
  if (sourceBuffer && !sourceBuffer.updating && radio.resumeRequired) {
    audio.play().catch(handlePlayBlock)
  }
}

// catchUpToLive moves the playhead to the live buffered start after an append:
// a listener joining mid-song only ever receives the tail, so stalling at 0
// would keep them silent. The media stream for the current song always starts
// its own timeline at 0, so early joins are left untouched.
function catchUpToLive() {
  if (!audio || !sourceBuffer) return
  if (sourceBuffer.buffered.length === 0) return
  const liveStart = sourceBuffer.buffered.start(0)
  if (liveStart > 0.4 && liveStart > 0 && Math.abs(audio.currentTime - liveStart) > 1) {
    try {
      audio.currentTime = liveStart
    } catch {
      // ignore
    }
  }
}

function onSourceBufferError() {
  appendErrors++
  if (appendErrors > 3) {
    appendQueue.length = 0
    resetPending = true
    appendErrors = 0
  }
}

function handleChunk(_data: ArrayBuffer) {
  if (useFallback) {
    // Fallback plays announced songs from the static audio endpoint; the
    // real-time-paced chunks are irrelevant here.
    return
  }
  const data = _data
  if (appendQueue.length > MAX_PENDING) {
    appendQueue.length = 0
  }
  appendQueue.push(data)
  pump()
}

function startPulse() {
  if (!analyser) return
  const data = new Uint8Array(analyser.frequencyBinCount)
  function tick() {
    if (!analyser) return
    analyser.getByteFrequencyData(data)
    let sum = 0
    for (let i = 0; i < data.length; i++) sum += data[i]
    const avg = sum / data.length / 255
    radio.pulseScale = 0.8 + avg * 0.6
    animFrame = requestAnimationFrame(tick)
  }
  tick()
}

function stopPulse() {
  cancelAnimationFrame(animFrame)
  radio.pulseScale = 1
}

function teardownEngine() {
  clearTimeout(reconnectTimer)
  reconnectTimer = 0
  clearInterval(drainTimer)
  drainTimer = 0
  clearInterval(watchdogTimer)
  watchdogTimer = 0

  appendQueue.length = 0
  pendingRemove = null
  resetPending = false
  appendErrors = 0
  wantAudio = false

  if (sourceBuffer) {
    sourceBuffer.removeEventListener('error', onSourceBufferError)
    try {
      if (mediaSource && mediaSource.readyState === 'open') mediaSource.endOfStream()
    } catch {
      // ignore
    }
    sourceBuffer = null
  }

  if (audio) {
    audio.pause()
    audio.src = ''
    audio.remove()
    audio = null
  }

  if (mediaSource) {
    mediaSource = null
  }

  if (audioCtx) {
    audioCtx.close().catch(() => {})
    audioCtx = null
    analyser = null
  }
  stopPulse()

  useFallback = false
  radio.resumeRequired = false
}