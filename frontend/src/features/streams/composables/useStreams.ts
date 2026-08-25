import { ref } from 'vue'
import { streamApi } from '@/shared/api/stream'
import type { Stream, StreamState } from '@/shared/api/types'

export function useStreams() {
  const streams = ref<Stream[]>([])
  const current = ref<Stream | null>(null)
  const currentState = ref<StreamState | null>(null)
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      streams.value = await streamApi.list()
    } finally {
      loading.value = false
    }
  }

  async function get(id: string) {
    loading.value = true
    try {
      current.value = await streamApi.get(id)
      currentState.value = await streamApi.getState(id)
    } finally {
      loading.value = false
    }
  }

  async function create(data: { name: string; description?: string; loop?: boolean }) {
    const s = await streamApi.create(data)
    streams.value.push(s)
    return s
  }

  async function update(id: string, patch: { name?: string; description?: string; loop?: boolean }) {
    const s = await streamApi.update(id, patch)
    const idx = streams.value.findIndex((x) => x.id === id)
    if (idx >= 0) streams.value[idx] = s
    if (current.value?.id === id) current.value = s
    return s
  }

  async function remove(id: string) {
    await streamApi.delete(id)
    streams.value = streams.value.filter((x) => x.id !== id)
    if (current.value?.id === id) current.value = null
  }

  async function start(id: string) {
    currentState.value = await streamApi.start(id)
  }

  async function stop(id: string) {
    currentState.value = await streamApi.stop(id)
  }

  return { streams, current, currentState, loading, load, get, create, update, remove, start, stop }
}
