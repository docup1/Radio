import { Page } from '@playwright/test'
import crypto from 'crypto'
import fs from 'fs'
import path from 'path'

const CHUNK_SIZE = 64 * 1024

const FIXTURES_DIR = path.resolve(__dirname, '../../fixtures/audio')

export const FIXTURE = path.join(FIXTURES_DIR, 'song.mp3')

interface SongSpec {
  name: string
  file: string
}

interface SeededMetadata {
  userId: string
  melodyIds: string[]
  songIds: string[]
}

/**
 * Seeds the content library and stream queue of the currently logged-in user
 * through the real gateway API (same origin as the SPA, cookies shared with the
 * page context).
 */
export async function seedLibrary(page: Page, songs: SongSpec[]): Promise<SeededMetadata> {
  const specs = songs.map((s) => {
    const buf = fs.readFileSync(s.file)
    return {
      name: s.name,
      bytes64: buf.toString('base64'),
      size: buf.byteLength,
      hash: crypto.createHash('sha256').update(buf).digest('hex'),
    }
  })

  return page.evaluate(
    async (cfg: {
      songs: { name: string; bytes64: string; size: number; hash: string }[]
      chunkSize: number
    }) => {
      const diag: { step: string; status?: number; body?: string }[] = []
      const j = async <T>(url: string, init?: RequestInit): Promise<T> => {
        const r = await fetch(url, { ...init, credentials: 'include' })
        const text = await r.text()
        const body = text === '' ? null : JSON.parse(text)
        diag.push({ step: url, status: r.status, body: text.slice(0, 120) })
        if (!r.ok) {
          throw new Error(`HTTP ${r.status} ${url} ${text}\nDIAG ${JSON.stringify(diag)}`)
        }
        return body as T
      }

      const me = await j<{ id: string }>('/api/auth/me')
      await j(`/api/streams/`)

      const melodyIds: string[] = []
      const songIds: string[] = []

      for (let i = 0; i < cfg.songs.length; i += 1) {
        const s = cfg.songs[i]
        const totalChunks = Math.ceil(s.size / cfg.chunkSize)
        const up = await j<{ id: string }>('/api/content/uploads', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            media_type: 'audio',
            content_type: 'audio/mpeg',
            total_chunks: totalChunks,
            expected_size: s.size,
            expected_hash: s.hash,
          }),
        })

        const raw = atob(s.bytes64)
        const bytes = new Uint8Array(raw.length)
        for (let k = 0; k < raw.length; k += 1) bytes[k] = raw.charCodeAt(k)

        const hexH = (b: ArrayBuffer) =>
          Array.from(new Uint8Array(b), (x) => x.toString(16).padStart(2, '0')).join('')
        const preHash = await window.crypto.subtle.digest('SHA-256', bytes)
        diag.push({
          step: 'hash-check',
          status: 200,
          body: `want=${s.hash} got=${hexH(preHash)} len=${bytes.length}`,
        })

        for (let ci = 0; ci < totalChunks; ci += 1) {
          const begin = ci * cfg.chunkSize
          const end = Math.min(begin + cfg.chunkSize, s.size)
          await j<string>(`/api/content/uploads/${up.id}/chunks/${ci}`, {
            method: 'PUT',
            body: bytes.slice(begin, end),
          })
        }

        const confirmed = await j<{ id: string }>(`/api/content/uploads/${up.id}/confirm`, {
          method: 'POST',
        })
        melodyIds.push(confirmed.id)

        const song = await j<{ id: string }>('/api/content/songs', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: s.name,
            melody_id: confirmed.id,
            is_public: false,
          }),
        })
        songIds.push(song.id)
      }

      for (const songId of songIds) {
        await j(`/api/streams/${me.id}/queue`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ song_id: songId }),
        })
      }

      return { userId: me.id, melodyIds, songIds, diag }
    },
    { songs: specs, chunkSize: CHUNK_SIZE },
  )
}

/** State of the radio audio element as seen from the page (fallback or MSE). */
export interface RadioAudioState {
  found: boolean
  paused: boolean | null
  currentTime: number | null
  readyState: number | null
  src: string | null
  allAudio: string[]
  activeRadio: boolean
}

export async function radioAudio(page: Page): Promise<RadioAudioState> {
  return page.evaluate(() => {
    const audios = Array.from(document.querySelectorAll<HTMLAudioElement>('audio'))
    const el =
      audios.find((a) => (a.src || '').includes('/api/content/songs/')) ??
      audios.find((a) => (a.src || '').startsWith('blob:')) ??
      null
    return {
      found: el !== null,
      paused: el ? el.paused : null,
      currentTime: el ? el.currentTime : null,
      readyState: el ? el.readyState : null,
      src: el ? el.src : null,
      allAudio: audios.map((a) => a.src),
      activeRadio: typeof window !== 'undefined' && Boolean((window as never as { [k: string]: unknown }).radioActiveRadio),
    }
  })
}