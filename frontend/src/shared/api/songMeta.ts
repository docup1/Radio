import { ref, type Ref } from 'vue'
import { content } from '@/shared/api/content'
import type { Song } from '@/shared/api/types'

const cache = new Map<string, Promise<Song | null>>()

export function getSongMetaCached(id: string): Promise<Song | null> {
  let p = cache.get(id)
  if (!p) {
    p = content
      .getSong(id)
      .then((s) => s as Song)
      .catch(() => null)
    cache.set(id, p)
  }
  return p
}

export function useSongMeta(id: Ref<string | null>) {
  const song = ref<Song | null>(null)
  const loading = ref(true)

  getSongMetaCached(id.value ?? '').then(
    (s) => {
      song.value = s
      loading.value = false
    },
    () => {
      loading.value = false
    },
  )

  return { song, loading }
}