import { ref } from 'vue'
import { streamApi } from '@/shared/api/stream'
import type { Stream, StreamState } from '@/shared/api/types'

export function useStreams() {
  const stream = ref<Stream | null>(null)
  const currentState = ref<StreamState | null>(null)
  const loading = ref(false)

  async function loadMine() {
    loading.value = true
    try {
      stream.value = await streamApi.getMine()
      currentState.value = await streamApi.getState(stream.value.id)
    } finally {
      loading.value = false
    }
  }

  async function get(id: string) {
    loading.value = true
    try {
      stream.value = await streamApi.get(id)
      currentState.value = await streamApi.getState(id)
    } finally {
      loading.value = false
    }
  }

  async function update(id: string, patch: { name?: string; description?: string; loop?: boolean }) {
    const s = await streamApi.update(id, patch)
    stream.value = s
    return s
  }

  async function remove(id: string) {
    await streamApi.delete(id)
    stream.value = null
  }

  async function start(id: string) {
    currentState.value = await streamApi.start(id)
  }

  async function stop(id: string) {
    currentState.value = await streamApi.stop(id)
  }

  return { stream, currentState, loading, loadMine, get, update, remove, start, stop }
}
