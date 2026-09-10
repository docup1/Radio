import { ref } from 'vue'
import { streamApi } from '@/shared/api/stream'
import type { Stream } from '@/shared/api/types'

const LIMIT = 20

export function useFeed() {
  const streams = ref<Stream[]>([])
  const loading = ref(false)
  const hasMore = ref(true)
  const query = ref('')

  async function load(reset = false) {
    if (loading.value) return
    loading.value = true
    try {
      if (reset) {
        streams.value = []
        hasMore.value = true
      }
      const offset = streams.value.length
      const items = await streamApi.getFeed(query.value.trim(), LIMIT, offset)
      hasMore.value = items.length >= LIMIT
      streams.value.push(...items.filter((s) => !streams.value.some((x) => x.id === s.id)))
    } finally {
      loading.value = false
    }
  }

  function setQuery(q: string) {
    query.value = q
    void load(true)
  }

  return { streams, loading, hasMore, query, load, setQuery }
}