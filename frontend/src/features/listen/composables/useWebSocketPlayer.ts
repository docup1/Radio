import { reactive, onUnmounted } from 'vue'

export type PlayerState = 'idle' | 'connecting' | 'playing' | 'ended' | 'error'

interface WSPlayerState {
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

export function useWebSocketPlayer() {
  const st = reactive<WSPlayerState>({
    state: 'idle',
    currentSong: null,
    error: '',
    pulseScale: 1,
  })

  let ws: WebSocket | null = null
  let audioCtx: AudioContext | null = null
  let analyser: AnalyserNode | null = null
  let nextPlayTime = 0
  let bufferQueue: AudioBuffer[] = []
  let playing = false
  let animFrame = 0

  function connect(streamId: string) {
    disconnect()

    st.state = 'connecting'
    st.error = ''
    st.currentSong = null

    audioCtx = new AudioContext()
    analyser = audioCtx.createAnalyser()
    analyser.fftSize = 256
    analyser.connect(audioCtx.destination)

    // Resume after user gesture (browser blocks autoplay)
    if (audioCtx.state === 'suspended') {
      audioCtx.resume()
    }

    ws = new WebSocket(`${WS_BASE}/api/streams/${streamId}/ws`)
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      st.state = 'playing'
      startPulse()
    }

    ws.onmessage = async (ev) => {
      if (!(ev.data instanceof ArrayBuffer)) return
      try {
        const audioBuffer = await audioCtx!.decodeAudioData(ev.data)
        bufferQueue.push(audioBuffer)
        schedulePlay()
      } catch (e) {
        console.warn('[ws-player] decode error:', e)
      }
    }

    ws.onerror = () => {
      st.state = 'error'
      st.error = 'WebSocket ошибка'
    }

    ws.onclose = () => {
      if (st.state === 'playing') {
        st.state = 'ended'
      }
      stopPulse()
    }
  }

  function schedulePlay() {
    if (!audioCtx || playing) return
    playNext()
  }

  function playNext() {
    if (!audioCtx || !analyser || bufferQueue.length === 0) {
      playing = false
      return
    }

    playing = true
    const buffer = bufferQueue.shift()!
    const source = audioCtx.createBufferSource()
    source.buffer = buffer
    source.connect(analyser)

    const now = audioCtx.currentTime
    const startAt = Math.max(nextPlayTime, now)
    source.start(startAt)
    nextPlayTime = startAt + buffer.duration

    source.onended = () => {
      playNext()
    }
  }

  function startPulse() {
    if (!analyser) return
    const data = new Uint8Array(analyser.frequencyBinCount)

    function tick() {
      analyser!.getByteFrequencyData(data)
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

  function disconnect() {
    ws?.close()
    ws = null
    if (audioCtx) {
      audioCtx.close().catch(() => {})
      audioCtx = null
    }
    analyser = null
    bufferQueue = []
    playing = false
    nextPlayTime = 0
    stopPulse()
    st.state = 'idle'
    st.currentSong = null
    st.error = ''
  }

  onUnmounted(() => disconnect())

  return { state: st, connect, disconnect }
}
