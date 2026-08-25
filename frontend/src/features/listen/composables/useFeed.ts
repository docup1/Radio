import { ref } from 'vue'
import { streamApi } from '@/shared/api/stream'
import type { Stream } from '@/shared/api/types'

export function useFeed() {
  const streams = ref<Stream[]>([])
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      streams.value = await streamApi.list()
    } finally {
      loading.value = false
    }
  }

  return { streams, loading, load }
}
