import { request } from './client'
import type {
  Stream,
  StreamState,
  QueueItem,
  CreateStreamInput,
  UpdateStreamInput,
} from './types'

export const streamApi = {
  list: () => request<Stream[]>('GET', '/api/streams/'),

  create: (input: CreateStreamInput) => request<Stream>('POST', '/api/streams/', input),

  get: (id: string) => request<Stream>('GET', `/api/streams/${id}`),

  update: (id: string, patch: UpdateStreamInput) =>
    request<Stream>('PUT', `/api/streams/${id}`, patch),

  delete: (id: string) => request<void>('DELETE', `/api/streams/${id}`),

  getState: (id: string) => request<StreamState>('GET', `/api/streams/${id}/state`),

  start: (id: string) => request<StreamState>('POST', `/api/streams/${id}/start`),

  stop: (id: string) => request<StreamState>('POST', `/api/streams/${id}/stop`),

  listQueue: (id: string) => request<QueueItem[]>('GET', `/api/streams/${id}/queue`),

  addToQueue: (id: string, songId: string) =>
    request<QueueItem>('POST', `/api/streams/${id}/queue`, { song_id: songId }),

  removeFromQueue: (id: string, itemId: string) =>
    request<void>('DELETE', `/api/streams/${id}/queue/${itemId}`),

  moveInQueue: (id: string, itemId: string, position: number) =>
    request<QueueItem>('PUT', `/api/streams/${id}/queue/${itemId}`, { position }),
}
