<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useStreams } from '@/features/streams/composables/useStreams'
import { useStreamQueue } from '@/features/streams/composables/useStreamQueue'
import { getSongMetaCached } from '@/shared/api/songMeta'
import { radio, openStream, start, stop, skip } from '@/shared/radio/store'
import { user } from '@/shared/store/auth'
import { t } from '@/shared/i18n'
import type { QueueItem } from '@/shared/api/types'

const router = useRouter()
const streamId = computed(() => user.value?.id ?? '')

const { stream, loading, loadMine, remove } = useStreams()
const { queue, load: loadQueue, remove: removeFromQueue, reorder } = useStreamQueue(streamId.value)

const songNames = ref<Record<string, string>>({})
const dragIndex = ref<number | null>(null)

const isOnline = computed(() => radio.streamId === streamId.value && radio.isActive)
const statusText = computed(() => (isOnline.value ? t('stream.online') : t('stream.offline')))
const currentSong = computed(() => (radio.streamId === streamId.value ? radio.song?.name ?? '' : ''))

onMounted(async () => {
  if (!streamId.value) return
  await Promise.all([loadMine(), loadQueue()])
  openStream(streamId.value)
})

async function refreshSongNames(items: QueueItem[]) {
  const next: Record<string, string> = {}
  await Promise.all(
    items.map(async (it) => {
      const song = await getSongMetaCached(it.song_id)
      if (song) next[it.song_id] = song.name
    }),
  )
  songNames.value = { ...songNames.value, ...next }
}

watch(queue, (q) => void refreshSongNames(q))

async function onRemove(itemId: string) {
  await removeFromQueue(itemId)
}

async function onDeleteStream() {
  if (!stream.value) return
  if (!confirm(t('settings.deleteConfirm'))) return
  await remove(stream.value.id)
}

function onDragStart(index: number) {
  dragIndex.value = index
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
}

function onDrop(index: number) {
  const from = dragIndex.value
  dragIndex.value = null
  if (from === null || from === index) return
  const next = [...queue.value]
  const [moved] = next.splice(from, 1)
  next.splice(index, 0, moved)
  void reorder(next)
}

function onListen() {
  router.push({ name: 'listen', params: { id: streamId.value } })
}
</script>

<template>
  <div class="stream-tab">
    <template v-if="loading">
      <div class="state">{{ t('common.loading') }}</div>
    </template>

    <template v-else-if="stream">
      <div class="status-card">
        <div class="status-card__row">
          <span :class="['dot', { 'dot--live': isOnline }]" />
          <span class="status-text">{{ statusText }}</span>
          <span v-if="isOnline && currentSong" class="status-song">· {{ currentSong }}</span>
        </div>
        <div class="controls">
          <button
            v-if="!isOnline"
            class="btn btn--primary"
            @click="start"
          >{{ t('stream.start') }}</button>
          <template v-else>
            <button class="btn btn--danger" @click="stop">{{ t('stream.stop') }}</button>
            <button class="btn btn--ghost" @click="skip">{{ t('stream.skip') }}</button>
          </template>
          <button class="btn btn--ghost" @click="onListen">{{ t('stream.listen') }}</button>
        </div>
      </div>

      <h2 class="section-title">{{ t('stream.queue') }}</h2>

      <div v-if="queue.length === 0" class="state">
        <p class="state__text">{{ t('stream.queueEmpty') }}</p>
        <p class="state__hint">{{ t('stream.queueHint') }}</p>
      </div>

      <div v-else class="queue">
        <div
          v-for="(item, idx) in queue"
          :key="item.id"
          :class="['queue-item', { 'queue-item--active': radio.currentItemId === item.id }]"
          draggable="true"
          @dragstart="onDragStart(idx)"
          @dragover="onDragOver"
          @drop="onDrop(idx)"
        >
          <span class="queue-item__grip">⠿</span>
          <span class="queue-item__pos">{{ idx + 1 }}</span>
          <span class="queue-item__name">{{ songNames[item.song_id] || t('stream.noQueueSong') }}</span>
          <button class="queue-item__remove" :title="t('stream.remove')" @click="onRemove(item.id)">
            &times;
          </button>
        </div>
      </div>

      <div class="danger-zone">
        <button class="btn btn--danger-ghost" @click="onDeleteStream">{{ t('common.delete') }}</button>
      </div>
    </template>

    <div v-else class="state">{{ t('stream.offline') }}</div>
  </div>
</template>

<style scoped>
.stream-tab {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.status-card {
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: 14px;
  padding: 18px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.status-card__row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #555;
}
.dot--live {
  background: #22c55e;
  box-shadow: 0 0 10px 2px #22c55e80;
}
.status-text {
  color: var(--text);
  font-weight: 600;
  font-size: 15px;
}
.status-song {
  color: var(--primary);
  font-size: 14px;
}
.controls {
  display: flex;
  gap: 8px;
}
.btn {
  background: transparent;
  border: 1px solid #333;
  color: var(--text);
  padding: 9px 18px;
  border-radius: 999px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
}
.btn--primary {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.btn--danger {
  background: var(--danger);
  border-color: var(--danger);
  color: #fff;
}
.btn--ghost:hover {
  border-color: var(--primary);
}
.btn--danger-ghost {
  color: var(--danger);
  border-color: var(--danger);
}
.btn--danger-ghost:hover {
  background: var(--danger);
  color: #fff;
}
.section-title {
  font-size: 13px;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 8px 0 0;
}
.state {
  color: var(--muted);
  text-align: center;
  padding: 32px 0;
}
.state__text {
  margin: 0 0 4px;
}
.state__hint {
  margin: 0;
  font-size: 13px;
}
.queue {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.queue-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: 10px;
  cursor: grab;
}
.queue-item:hover {
  border-color: #3a4050;
}
.queue-item--active {
  border-left: 3px solid var(--primary);
}
.queue-item__grip {
  color: var(--muted);
}
.queue-item__pos {
  color: var(--muted);
  font-size: 13px;
  min-width: 18px;
}
.queue-item__name {
  flex: 1;
  color: var(--text);
  font-size: 14px;
}
.queue-item__remove {
  background: transparent;
  border: none;
  color: var(--danger);
  font-size: 18px;
  cursor: pointer;
  padding: 0 6px;
}
.danger-zone {
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
}
</style>