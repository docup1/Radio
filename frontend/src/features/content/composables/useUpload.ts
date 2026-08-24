import { ref } from 'vue'
import { content } from '@/shared/api/content'
import type { InitUploadInput } from '@/shared/api/types'

const CHUNK_SIZE = 5 * 1024 * 1024

export function useUpload() {
  const progress = ref(0)

  async function uploadFile(
    file: File,
    mediaType: 'audio' | 'image',
  ): Promise<{ id: string }> {
    const totalChunks = Math.max(1, Math.ceil(file.size / CHUNK_SIZE))
    const input: InitUploadInput = {
      media_type: mediaType,
      content_type: file.type || 'application/octet-stream',
      total_chunks: totalChunks,
      expected_size: file.size,
      expected_hash: '',
    }
    const session = await content.createUpload(input)
    for (let i = 0; i < totalChunks; i++) {
      const start = i * CHUNK_SIZE
      const blob = file.slice(start, start + CHUNK_SIZE)
      await content.uploadChunk(session.id, i, blob)
      progress.value = Math.round(((i + 1) / totalChunks) * 100)
    }
    return content.confirmUpload(session.id)
  }

  return { progress, uploadFile }
}
