import { ref } from 'vue'
import { streamApi } from '@/shared/api/stream'
import type { QueueItem } from '@/shared/api/types'

export function useStreamQueue(streamId: string) {
  const queue = ref<QueueItem[]>([])
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      queue.value = await streamApi.listQueue(streamId)
    } finally {
      loading.value = false
    }
  }

  async function add(songId: string) {
    const item = await streamApi.addToQueue(streamId, songId)
    queue.value.push(item)
  }

  async function remove(itemId: string) {
    await streamApi.removeFromQueue(streamId, itemId)
    queue.value = queue.value.filter((x) => x.id !== itemId)
  }

  async function reorder(items: QueueItem[]) {
    queue.value = items
    for (let i = 0; i < items.length; i++) {
      if (items[i].position !== i) {
        await streamApi.moveInQueue(streamId, items[i].id, i)
        items[i].position = i
      }
    }
  }

  return { queue, loading, load, add, remove, reorder }
}
