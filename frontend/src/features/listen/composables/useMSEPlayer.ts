import { reactive, onUnmounted } from 'vue'

export type PlayerState = 'idle' | 'connecting' | 'playing' | 'ended' | 'error'

interface PlayerStateData {
  state: PlayerState
  currentSong: string | null
  error: string
  pulseScale: number
}

const WS_BASE = (() => {
  if (typeof window === 'undefined') return ''
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}`
})()

export function useMSEPlayer() {
  const st = reactive<PlayerStateData>({
    state: 'idle',
    currentSong: null,
    error: '',
    pulseScale: 1,
  })

  let ws: WebSocket | null = null
  let audio: HTMLAudioElement | null = null
  let mediaSource: MediaSource | null = null
  let sourceBuffer: SourceBuffer | null = null
  let audioCtx: AudioContext | null = null
  let analyser: AnalyserNode | null = null
  let animFrame = 0
  let blobUrl: string | null = null
  let url: string | null = null

  // Queue of chunks waiting to be appended to SourceBuffer
  const appendQueue: Uint8Array[] = []
  let pendingRemove: { start: number; end: number } | null = null
  let pendingAbort = false
  let draining = false

  // Fallback (no MSE): accumulate chunks and play
  let useFallback = false
  let chunks: Uint8Array[] = []
  let totalBytes = 0
  let blobRefreshTimer = 0

  function connect(streamId: string) {
    disconnect()

    st.state = 'connecting'
    st.error = ''
    st.currentSong = null

    audioCtx = new AudioContext()
    analyser = audioCtx.createAnalyser()
    analyser.fftSize = 256

    if (audioCtx.state === 'suspended') {
      audioCtx.resume()
    }

    const mseSupported = typeof MediaSource !== 'undefined' &&
      MediaSource.isTypeSupported('audio/mpeg')

    if (mseSupported) {
      useFallback = false
      setupMSE(streamId)
    } else {
      useFallback = true
      setupFallback(streamId)
    }
  }

  // ── MSE path ────────────────────────────────────────────────────

  function setupMSE(streamId: string) {
    audio = new Audio()
    audio.autoplay = true

    mediaSource = new MediaSource()
    url = URL.createObjectURL(mediaSource)
    audio.src = url

    const source = audioCtx!.createMediaElementSource(audio)
    source.connect(analyser!)
    analyser!.connect(audioCtx!.destination)

    mediaSource.addEventListener('sourceopen', () => {
      sourceBuffer = mediaSource!.addSourceBuffer('audio/mpeg')
      sourceBuffer.addEventListener('updateend', onSourceUpdateEnd)

      st.state = 'playing'
      startPulse()

      // Drain any chunks that arrived before SourceBuffer was ready
      if (appendQueue.length > 0 && !sourceBuffer.updating) {
        drainNext()
      }
    }, { once: true })

    audio.addEventListener('error', () => {
      if (st.state !== 'idle') {
        st.state = 'error'
        st.error = 'Audio error'
      }
    })

    openWebSocket(streamId)
  }

  // ── Fallback: Blob URL accumulation ─────────────────────────────

  function setupFallback(streamId: string) {
    audio = new Audio()
    audio.autoplay = true

    const source = audioCtx!.createMediaElementSource(audio)
    source.connect(analyser!)
    analyser!.connect(audioCtx!.destination)

    audio.addEventListener('error', () => {
      if (st.state !== 'idle') {
        st.state = 'error'
        st.error = 'Audio error'
      }
    })

    openWebSocket(streamId)
  }

  // ── WebSocket ───────────────────────────────────────────────────

  function openWebSocket(streamId: string) {
    ws = new WebSocket(`${WS_BASE}/api/streams/${streamId}/ws`)
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      if (st.state !== 'playing') {
        st.state = 'playing'
        startPulse()
      }
    }

    ws.onmessage = (ev) => {
      if (typeof ev.data === 'string') {
        try {
          const msg = JSON.parse(ev.data)
          if (msg.type === 'song') {
            handleNewSong(msg.songId)
          }
        } catch { /* ignore */ }
        return
      }

      if (!(ev.data instanceof ArrayBuffer)) return

      if (useFallback) {
        handleFallbackChunk(ev.data)
      } else {
        handleMSEChunk(ev.data)
      }
    }

    ws.onerror = () => {
      st.state = 'error'
      st.error = 'WebSocket error'
    }

    ws.onclose = () => {
      if (st.state === 'playing') {
        st.state = 'ended'
      }
      stopPulse()
    }
  }

  // ── Song change ─────────────────────────────────────────────────

  function handleNewSong(_songId: string) {
    st.currentSong = _songId

    if (useFallback) {
      chunks = []
      totalBytes = 0
      if (blobRefreshTimer) {
        clearTimeout(blobRefreshTimer)
        blobRefreshTimer = 0
      }
      if (audio) {
        audio.pause()
      }
      if (blobUrl) {
        URL.revokeObjectURL(blobUrl)
        blobUrl = null
      }
    } else {
      // Clear queue and abort current SourceBuffer operation
      appendQueue.length = 0
      pendingRemove = null
      pendingAbort = true
      draining = false
      if (sourceBuffer && !sourceBuffer.updating) {
        applyAbort()
      }
    }
  }

  function applyAbort() {
    if (!sourceBuffer) return
    pendingAbort = false
    pendingRemove = null
    try {
      sourceBuffer.abort()
    } catch { /* ignore */ }
  }

  // ── SourceBuffer drain loop ─────────────────────────────────────

  function onSourceUpdateEnd() {
    if (!sourceBuffer || sourceBuffer.updating) return

    // Handle abort from song change
    if (pendingAbort) {
      applyAbort()
      return
    }

    // Handle pending remove
    if (pendingRemove) {
      const { start, end } = pendingRemove
      pendingRemove = null
      try {
        sourceBuffer.remove(start, end)
      } catch { /* ignore */ }
      return
    }

    // Drain next chunk from queue
    if (appendQueue.length > 0) {
      const chunk = appendQueue.shift()!
      try {
        sourceBuffer.appendBuffer(chunk)
      } catch (e) {
        console.warn('[mse] appendBuffer error:', e)
      }
      return
    }

    draining = false

    // Trim old data from buffer (only before current position)
    if (sourceBuffer.buffered.length > 0 && audio) {
      const bufEnd = sourceBuffer.buffered.end(sourceBuffer.buffered.length - 1)
      const bufStart = sourceBuffer.buffered.start(0)
      // Remove data that's more than 2s behind playback
      if (audio.currentTime - bufStart > 2) {
        pendingRemove = { start: bufStart, end: audio.currentTime - 1 }
      }
    }
  }

  // ── Chunk handling ──────────────────────────────────────────────

  function handleMSEChunk(data: ArrayBuffer) {
    const chunk = new Uint8Array(data)
    appendQueue.push(chunk)

    // If SourceBuffer is ready and not draining, start the drain loop
    if (sourceBuffer && !draining && !sourceBuffer.updating && !pendingAbort) {
      drainNext()
    }
  }

  function drainNext() {
    if (!sourceBuffer || sourceBuffer.updating || pendingAbort) return
    if (appendQueue.length === 0) {
      draining = false
      return
    }

    draining = true
    const chunk = appendQueue.shift()!
    try {
      sourceBuffer.appendBuffer(chunk)
    } catch (e) {
      console.warn('[mse] appendBuffer error:', e)
      draining = false
    }
  }

  function handleFallbackChunk(data: ArrayBuffer) {
    const bytes = new Uint8Array(data)
    chunks.push(bytes)
    totalBytes += bytes.length

    if (!blobRefreshTimer) {
      blobRefreshTimer = window.setTimeout(() => {
        blobRefreshTimer = 0
        refreshBlob()
      }, 500)
    }
  }

  function refreshBlob() {
    if (!audio || chunks.length === 0) return

    const wasPlaying = !audio.paused
    const currentTime = audio.currentTime

    const blob = new Blob(chunks, { type: 'audio/mpeg' })

    if (blobUrl) {
      URL.revokeObjectURL(blobUrl)
    }
    blobUrl = URL.createObjectURL(blob)
    audio.src = blobUrl

    if (currentTime > 0 && isFinite(currentTime)) {
      audio.currentTime = currentTime
    }

    if (wasPlaying) {
      audio.play().catch(() => {})
    }
  }

  // ── Pulse visualization ─────────────────────────────────────────

  function startPulse() {
    if (!analyser) return
    const data = new Uint8Array(analyser.frequencyBinCount)

    function tick() {
      if (!analyser) return
      analyser.getByteFrequencyData(data)
      let sum = 0
      for (let i = 0; i < data.length; i++) sum += data[i]
      const avg = sum / data.length / 255
      st.pulseScale = 0.8 + avg * 0.6
      animFrame = requestAnimationFrame(tick)
    }
    tick()
  }

  function stopPulse() {
    cancelAnimationFrame(animFrame)
    st.pulseScale = 1
  }

  // ── Cleanup ─────────────────────────────────────────────────────

  function disconnect() {
    ws?.close()
    ws = null

    if (blobRefreshTimer) {
      clearTimeout(blobRefreshTimer)
      blobRefreshTimer = 0
    }

    appendQueue.length = 0
    pendingRemove = null
    pendingAbort = false
    draining = false

    if (sourceBuffer) {
      sourceBuffer.removeEventListener('updateend', onSourceUpdateEnd)
      try {
        if (mediaSource && mediaSource.readyState === 'open') {
          mediaSource.endOfStream()
        }
      } catch { /* ignore */ }
      sourceBuffer = null
    }

    if (audio) {
      audio.pause()
      audio.src = ''
      audio = null
    }

    if (mediaSource) {
      mediaSource = null
    }

    if (url) {
      URL.revokeObjectURL(url)
      url = null
    }

    if (blobUrl) {
      URL.revokeObjectURL(blobUrl)
      blobUrl = null
    }

    if (audioCtx) {
      audioCtx.close().catch(() => {})
      audioCtx = null
    }

    analyser = null
    chunks = []
    totalBytes = 0
    useFallback = false
    stopPulse()
    st.state = 'idle'
    st.currentSong = null
    st.error = ''
  }

  onUnmounted(() => disconnect())

  return { state: st, connect, disconnect }
}
