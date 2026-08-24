import { ref } from 'vue'
import { content } from '@/shared/api/content'
import type { Song, SongScope } from '@/shared/api/types'

const LIMIT = 20

export function useSongs() {
  const songs = ref<Song[]>([])
  const scope = ref<SongScope>('mine')
  const query = ref('')
  const loading = ref(false)
  const hasMore = ref(true)

  async function load(reset = false) {
    if (loading.value) return
    loading.value = true
    try {
      if (reset) songs.value = []
      const offset = songs.value.length
      const items = query.value.trim()
        ? await content.searchSongs(query.value.trim(), scope.value, LIMIT, offset)
        : await content.listSongs(scope.value, LIMIT, offset)
      hasMore.value = items.length >= LIMIT
      songs.value.push(...items)
    } finally {
      loading.value = false
    }
  }

  function setScope(s: SongScope) {
    scope.value = s
    void load(true)
  }

  function setQuery(q: string) {
    query.value = q
    void load(true)
  }

  return { songs, scope, query, loading, hasMore, load, setScope, setQuery }
}
