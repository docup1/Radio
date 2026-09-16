/* Isolated playback-interruption repro.
 *
 * Phase A (REST prep): register a throwaway user, generate + upload 3 distinct
 * short MP3 tracks, fill the queue, start the stream via WS.
 * Phase B (wire-level): record every WS frame in order with timestamps and
 * verify the invariant that a binary chunk is NEVER observed before the
 * "song" announce that owns it.
 * Phase C (browser MSE): run a store.ts-equivalent MSE pipeline in Chromium
 * and measure audible gaps (playhead stalls > 700ms) across song boundaries
 * and after forced mid-song WS drops.
 */

const { execSync } = require('child_process')
const fs = require('fs')
const path = require('path')
const os = require('os')
const crypto = require('crypto')
const http = require('http')

const GATEWAY = 'http://localhost:18000'
const WS_GATEWAY = 'ws://localhost:18000'

const GAP_MS = 700
const USER = `repro_${crypto.randomBytes(4).toString('hex')}`
const PASS = 'TestPass!123'
const DUR_S = 15 // each track ~15s long

function api(method, p, body, cookie) {
  return new Promise((resolve, reject) => {
    const data = body ? JSON.stringify(body) : null
    const req = http.request(
      GATEWAY + p,
      { method, headers: { 'Content-Type': 'application/json', ...(cookie ? { Cookie: cookie } : {}) } },
      (res) => {
        let buf = ''
        res.on('data', (c) => (buf += c))
        res.on('end', () => {
          let j = null
          try {
            j = JSON.parse(buf || 'null')
          } catch {}
          resolve({ status: res.statusCode, headers: res.headers, body: buf, json: j, setCookie: res.headers['set-cookie'] })
        })
      },
    )
    req.on('error', reject)
    if (data) req.write(data)
    req.end()
  })
}

function chunkUpload(route, sessionId, index, buf, cookie) {
  return new Promise((resolve, reject) => {
    const req = http.request(
      GATEWAY + route,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/octet-stream', 'Content-Length': buf.length, ...(cookie ? { Cookie: cookie } : {}) },
      },
      (res) => {
        res.resume()
        res.on('end', () => resolve(res.statusCode))
      },
    )
    req.on('error', reject)
    req.write(buf)
    req.end()
  })
}

function genTone(name, freq, dur) {
  const out = path.join(os.tmpdir(), `${name}.mp3`)
  execSync(`ffmpeg -y -f lavfi -i "sine=frequency=${freq}:duration=${dur}" -codec:a libmp3lame -b:a 128k -ar 44100 -ac 1 "${out}" 2>/dev/null`)
  return out
}

function cookieFrom(res) {
  return (res.setCookie || []).map((x) => x.split(';')[0]).join('; ')
}

async function main() {
  console.log(`= user ${USER}`)
  const reg = await api('POST', '/api/auth/register', { username: USER, password: PASS })
  if (reg.status !== 200 && reg.status !== 201) throw new Error(`register failed ${reg.status}: ${reg.body}`)
  const auth = cookieFrom(reg) || cookieFrom(await api('POST', '/api/auth/login', { username: USER, password: PASS }))
  if (!auth) throw new Error('no auth cookie')
  const stream = (await api('GET', '/api/streams/', null, auth)).json
  const streamId = stream.id
  console.log(`= stream ${streamId} name="${stream.name}" loop=${stream.loop}`)

  // 1. generate + upload 3 songs
  const songFiles = [genTone('repro_a', 440, DUR_S), genTone('repro_b', 554, DUR_S), genTone('repro_c', 659, DUR_S)]
  const songIds = []
  for (let i = 0; i < songFiles.length; i++) {
    const buf = fs.readFileSync(songFiles[i])
    const chunks = []
    for (let off = 0; off < buf.length; off += 1024 * 1024) chunks.push(buf.subarray(off, off + 1024 * 1024))
    const init = await api('POST', '/api/content/uploads', {
      media_type: 'audio',
      content_type: 'audio/mpeg',
      total_chunks: Math.max(1, chunks.length),
      expected_size: buf.length,
      expected_hash: '',
    }, auth)
    if (init.status !== 201) throw new Error(`init upload ${init.status}: ${init.body}`)
    const sessionId = init.json.id
    for (let c = 0; c < chunks.length; c++) {
      const s = await chunkUpload(`/api/content/uploads/${sessionId}/chunks/${c}`, sessionId, c, Buffer.from(chunks[c]), auth)
      if (s !== 200 && s !== 201 && s !== 204) throw new Error(`chunk ${c} status ${s}`)
    }
    const confirm = await api('POST', `/api/content/uploads/${sessionId}/confirm`, { media_type: 'audio' }, auth)
    if (confirm.status !== 200 && confirm.status !== 201) throw new Error(`confirm ${confirm.status}: ${confirm.body}`)
    const melodyId = confirm.json.id
    const songRes = await api('POST', '/api/content/songs', {
      name: `repro_${i}`,
      description: 'e2e repro tone',
      melody_id: melodyId,
      is_public: true,
    }, auth)
    if (songRes.status !== 200 && songRes.status !== 201) throw new Error(`createSong ${songRes.status}: ${songRes.body}`)
    songIds.push(songRes.json.id)
    console.log(`= melody ${melodyId} -> song ${songRes.json.id} (${buf.length} bytes)`)
  }
  songFiles.forEach((f) => fs.unlinkSync(f))

  // 2. fill the queue so seams exercise both different-song and same-song repeats
  for (const sid of [songIds[0], songIds[1], songIds[2], songIds[0]]) {
    const r = await api('POST', `/api/streams/${streamId}/queue`, { song_id: sid }, auth)
    if (r.status !== 201) throw new Error(`queue add ${r.status}: ${r.body}`)
  }
  const q = await api('GET', `/api/streams/${streamId}/queue`, null, auth)
  console.log(`= queue: ${q.json.length} items -> ${q.json.map((x) => x.song_id.slice(0, 8)).join(', ')}`)

  // 3. phase B: wire-level WS capture (no decoder) — ordering invariant
  console.log('= phase B: wire-level WS capture')
  const wire = await captureWire(streamId, auth, 3 * DUR_S * 1000 + 5000)
  const summary = analyzeWire(wire)
  console.log(`  frames=${wire.frames.length} boundaries=${summary.boundaries} chunkBeforeAnnounce=${summary.chunkBeforeAnnounce}`)
  console.log(`  firstChunkDelayAfterSong_ms=${summary.firstChunkDelayMs} maxGapBetweenChunks_ms=${summary.maxChunkGapMs}`)
  if (process.env.REPRO_DUMP_WIRE) {
    for (const f of wire.frames) {
      if (f.bin) console.log(`    t=+${f.t}ms BIN len=${f.len}`)
      else {
        const j = f.text
        console.log(`    t=+${f.t}ms TEXT ${j && j.type}${j && j.type === 'song' ? ' song=' + String(j.songId).slice(0, 8) : ''}`)
      }
    }
  }

  // 4. phase C: browser MSE gap measurement (the actual symptom)
  console.log('= phase C: browser MSE playback-gap measurement')
  const mse = await runBrowser(streamId, auth)
  console.log(`  seamContinuity=${mse.seamContinuityCount} stallEvents=${mse.stalls.length} jumps>${GAP_MS}ms=${mse.badJumps} maxStallMs=${mse.maxStallMs}`)
  if (mse.error) console.log(`  phaseC-ERROR: ${mse.error}`)
  for (const s of mse.stalls.slice(0, 12)) {
    console.log(`    ${s.type} at ${s.songIdx}/${Math.round(s.at)}s +${s.spanMs}ms @${s.msSinceSeamMs}ms-after-seam`)
  }

  const failures = []
  if (summary.chunkBeforeAnnounce > 0) failures.push(`wire: ${summary.chunkBeforeAnnounce} chunk(s) before announce`)
  if (mse.badJumps > 0) failures.push(`browser: ${mse.badJumps} stalls > ${GAP_MS}ms`)
  console.log(failures.length ? `FAIL:\n  ${failures.join('\n  ')}` : 'PASS')
  await api('POST', `/api/streams/${streamId}/stop`, {}, auth)
  process.exit(failures.length ? 1 : 0)
}

// --- phase B ---

function captureWire(streamId, auth, ms) {
  return new Promise((resolve, reject) => {
    const WS = require('ws')
    const cookie = auth.split('; ').find((c) => c.startsWith('access_token='))
    const ws = new WS(`${WS_GATEWAY}/api/streams/${streamId}/ws`, { headers: cookie ? { Cookie: cookie } : {} })
    const frames = []
    const t0 = Date.now()
    ws.on('open', () => {
      ws.send(JSON.stringify({ type: 'start', loop: false }))
    })
    ws.on('message', (data, isBinary) => {
      const t = Date.now() - t0
      if (isBinary) frames.push({ t, bin: true, len: data.length })
      else {
        let j = null
        try {
          j = JSON.parse(data.toString())
        } catch {}
        frames.push({ t, bin: false, text: j })
      }
    })
    ws.on('error', reject)
    setTimeout(() => {
      try {
        ws.close()
      } catch {}
      setTimeout(() => resolve({ frames }), 250)
    }, ms)
  })
}

function analyzeWire(s) {
  const frames = s.frames
  let announced = false
  let chunkBeforeAnnounce = 0
  let maxChunkGapMs = 0
  let firstChunkDelayMs = null
  let lastChunkT = null
  let boundaries = 0
  for (let i = 0; i < frames.length; i++) {
    const f = frames[i]
    if (f.bin) {
      if (!announced) chunkBeforeAnnounce++
      if (lastChunkT != null) maxChunkGapMs = Math.max(maxChunkGapMs, f.t - lastChunkT)
      lastChunkT = f.t
      continue
    }
    if (f.text && f.text.type === 'song') {
      boundaries++
      announced = true
      const next = frames.slice(i + 1).find((g) => g.bin && g.t > f.t)
      if (next) firstChunkDelayMs = firstChunkDelayMs == null ? next.t - f.t : Math.min(firstChunkDelayMs, next.t - f.t)
    }
  }
  return { boundaries, chunkBeforeAnnounce, maxChunkGapMs, firstChunkDelayMs }
}

// --- phase C (browser) ---

const BROWSER_PIPELINE = `
  const log = (x) => console.log('[repro] ' + x)
  const wsBase = 'ws://localhost:18000'
  const GAP_MS = __GAP_MS__
  let audio = null, ms = null, sb = null, wantAudio = false, activeSongId = null
  const queue = []
  const stalls = []
  let lastT = 0, stallStartWall = 0, songIdx = -1
  let lastSeamWall = 0
  const wall0 = Date.now()
  let pumpTimer = 0
  let drops = [0]

  function stallRecord(type, span) {
    const wall = Date.now() - wall0
    stalls.push({ type, spanMs: Math.round(span), at: Math.round(lastT), msSinceSeamMs: Math.round(wall - lastSeamWall), songIdx })
  }

  function resetPlayback(sid) {
    // mirror store.ts resetPlayback incl. the same-song continuity fix
    if (sid && activeSongId === sid && audio && ms) {
      wantAudio = true
      audio.play().catch(() => {})
      log('same song repeat -> continuity (no rebuild)')
      return
    }
    activeSongId = sid || null
    if (!audio) {
      setupMse()
      return
    }
    queue.length = 0
    teardownMse()
    setupMse()
  }

  function setupMse() {
    audio = new Audio()
    audio.autoplay = true
    audio.style.display = 'none'
    document.body.appendChild(audio)
    ms = new MediaSource()
    audio.src = URL.createObjectURL(ms)
    ms.addEventListener('sourceopen', () => {
      sb = ms.addSourceBuffer('audio/mpeg')
      clearInterval(pumpTimer)
      pumpTimer = setInterval(pump, 80)
    }, { once: true })
    audio.addEventListener('timeupdate', () => {
      const now = audio.currentTime
      if (now === lastT) {
        if (!stallStartWall && wantAudio) stallStartWall = Date.now()
        return
      }
      if (stallStartWall) {
        const span = Date.now() - stallStartWall
        stallStartWall = 0
        if (span > 100) stallRecord('gap', span)
      }
      const d = (now - lastT) * 1000
      if (d > GAP_MS && wantAudio) stallRecord('jump', d)
      lastT = now
    })
    audio.play().catch(() => {})
  }

  function teardownMse() {
    clearInterval(pumpTimer)
    sb = null
    ms = null
    if (audio) { audio.pause(); audio.remove(); audio = null }
  }

  function pump() {
    if (!sb || !ms || ms.readyState !== 'open' || sb.updating) return
    const b = queue.shift()
    if (!b) return
    try {
      sb.appendBuffer(b)
      if (sb.buffered.length) {
        const start = sb.buffered.start(0)
        if (start > 0.4 && start > 0 && Math.abs((audio && audio.currentTime) || 0 - start) > 1) audio.currentTime = start
      }
    } catch {}
  }

  function handleText(j) {
    if (j.type === 'song') {
      songIdx++
      if (songIdx > 0) lastSeamWall = Date.now() - wall0
      resetPlayback(j.songId)
      wantAudio = true
    } else if (j.type === 'state' && j.data && j.data.is_active && !wantAudio) {
      wantAudio = true
      if (!audio) resetPlayback(activeSongId)
    }
  }

  function attach(ws) {
    ws.binaryType = 'arraybuffer'
    ws.onmessage = (ev) => {
      if (typeof ev.data === 'string') {
        let j = null
        try { j = JSON.parse(ev.data) } catch {}
        log('TEXT ' + (j ? (j.type + (j.error ? ' err=' + JSON.stringify(j.error) : j.message ? ' msg=' + j.message : '')) : '?'))
        if (j) handleText(j)
      } else {
        queue.push(ev.data)
        pump()
      }
    }
  }

  window.__run = () => {
    const socket = () => {
      const ws = new WebSocket(wsBase + '/api/streams/' + __STREAM_ID__ + '/ws')
      attach(ws)
      return ws
    }
    let current = socket()
    let started = false
    current.onopen = () => {
      if (started) return
      started = true
      current.send(JSON.stringify({ type: 'start', loop: false }))
      log('ws open, start sent')
      // force a mid-song drop, then reconnect
      setTimeout(() => {
        log('forcing mid-song drop')
        try { current.close() } catch {}
      }, 6000)
      setTimeout(() => {
        log('reconnecting after drop')
        current = socket()
        current.onopen = () => current.send(JSON.stringify({ type: 'start', loop: false }))
      }, 7800)
    }
    setTimeout(() => {
      try {
        window.__done = {
          stalls,
          maxStallMs: stalls.reduce((m, s) => Math.max(m, s.spanMs), 0),
          badJumps: stalls.filter((s) => (s.type === 'jump' || s.type === 'gap') && s.spanMs > GAP_MS).length,
        }
        log('done set')
      } catch (e) {
        log('done ERR ' + e.message)
      }
    }, 3 * __DUR_S__ * 1000 + 8000)
  }
`

function runBrowser(streamId, auth) {
  return new Promise((resolve, reject) => {
    const { chromium } = require('playwright')
    chromium
      .launch({ args: ['--autoplay-policy=no-user-gesture-required', '--disable-background-timer-throttling', '--disable-renderer-backgrounding'] })
      .then(async (browser) => {
        const page = await browser.newPage()
        const logs = []
        page.on('console', (msg) => {
          const m = msg.text()
          if (m.startsWith('[repro]')) logs.push(m)
        })
        page.on('pageerror', (e) => logs.push('[pageerror] ' + e.message))
        page.on('load', () => {})
        const srv = http.createServer((_req, res) => {
          res.writeHead(200, { 'Content-Type': 'text/html' })
          res.end('<!doctype html><html><head></head><body>repro origin</body></html>')
        })
        await new Promise((r) => srv.listen(18999, 'localhost', r))
        await page.goto('http://localhost:18999/')
        const token = auth.split('; ').find((c) => c.startsWith('access_token='))
        if (token) {
          const [name, ...rest] = token.split('=')
          await page.context().addCookies([{ name, value: rest.join('='), domain: 'localhost', path: '/' }])
        }
        const src = BROWSER_PIPELINE.replace('__GAP_MS__', GAP_MS).replace('__STREAM_ID__', `'${streamId}'`).replace('__DUR_S__', DUR_S)
        await page.evaluate(src)
        await page.evaluate(() => window.__run())
        for (let i = 0; i < 240 && !(await page.evaluate(() => !!window.__done)); i++) {
          await new Promise((r) => setTimeout(r, 500))
        }
        const timedOut = !(await page.evaluate(() => !!window.__done))
        if (timedOut) logs.push('[timeout] window.__done never set')
        const result = await page.evaluate(() => window.__done)
        const seams = logs.filter((l) => l.includes('same song repeat'))
        console.log('  ' + logs.join('\n  '))
        await browser.close()
        srv.close()
        if (!result) {
          resolve({ stalls: [], maxStallMs: 0, badJumps: 0, seamContinuityCount: 0, dropStallMs: [], error: logs.join(' | ') || 'no data' })
          return
        }
        resolve({ ...result, seamContinuityCount: seams.length, dropStallMs: [] })
      })
      .catch(reject)
  })
}

main().catch((e) => {
  console.error('ERROR', e && e.stack ? e.stack : e)
  process.exit(2)
})