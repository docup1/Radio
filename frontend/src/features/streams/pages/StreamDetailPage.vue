<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useStreams } from '../composables/useStreams'
import { useStreamQueue } from '../composables/useStreamQueue'
import { useSongs } from '@/features/content/composables/useSongs'
import type { QueueItem } from '@/shared/api/types'

const props = defineProps<{ id: string }>()
const router = useRouter()

const { current, currentState, get, start, stop, update } = useStreams()
const { queue, load: loadQueue, add, remove, reorder } = useStreamQueue(props.id)
const { songs, query, loading: songsLoading, load: loadSongs, setQuery } = useSongs()

const tab = ref<'queue' | 'settings'>('queue')
const name = ref('')
const description = ref('')
const loop = ref(false)
const saving = ref(false)
const showSongSearch = ref(false)

onMounted(async () => {
  await Promise.all([get(props.id), loadQueue()])
  if (current.value) {
    name.value = current.value.name
    description.value = current.value.description
    loop.value = current.value.loop
  }
})

function onSearchSongs() {
  showSongSearch.value = true
  loadSongs(true)
}

async function onAddSong(songId: string) {
  await add(songId)
  showSongSearch.value = false
  query.value = ''
}

async function onRemoveSong(itemId: string) {
  await remove(itemId)
}

async function onDragEnd() {
  await reorder(queue.value)
}

async function onSaveSettings() {
  saving.value = true
  try {
    await update(props.id, { name: name.value.trim(), description: description.value.trim(), loop: loop.value })
  } finally {
    saving.value = false
  }
}

async function onStart() {
  await start(props.id)
}

async function onStop() {
  await stop(props.id)
}
</script>

<template>
  <div class="stream-detail" v-if="current">
    <div class="stream-detail__header">
      <div>
        <h1>{{ current.name }}</h1>
        <p class="stream-detail__status">
          <span :class="['status-dot', currentState?.is_active ? 'status-dot--active' : '']" />
          {{ currentState?.is_active ? 'Активен' : 'Остановлен' }}
        </p>
      </div>
      <div class="stream-detail__controls">
        <button v-if="!currentState?.is_active" class="btn-primary" @click="onStart">▶ Старт</button>
        <button v-else class="btn-danger" @click="onStop">⏹ Стоп</button>
        <RouterLink :to="{ name: 'feed' }" class="btn-ghost">Лента</RouterLink>
      </div>
    </div>

    <div class="stream-detail__tabs">
      <button :class="['tab', tab === 'queue' && 'tab--active']" @click="tab = 'queue'">Очередь</button>
      <button :class="['tab', tab === 'settings' && 'tab--active']" @click="tab = 'settings'">Настройки</button>
    </div>

    <div v-if="tab === 'queue'" class="stream-detail__queue">
      <div v-if="queue.length === 0" class="empty">Очередь пуста</div>

      <div
        v-for="(item, idx) in queue"
        :key="item.id"
        class="queue-item"
        draggable="true"
        @dragend="onDragEnd"
      >
        <span class="queue-item__pos">{{ idx + 1 }}</span>
        <span class="queue-item__name">Песня {{ item.song_id.slice(0, 8) }}</span>
        <button class="queue-item__remove" @click="onRemoveSong(item.id)">&times;</button>
      </div>

      <button class="btn-add" @click="onSearchSongs">+ Добавить песню</button>

      <div v-if="showSongSearch" class="song-search">
        <input
          v-model="query"
          class="song-search__input"
          placeholder="Поиск песен..."
          @input="setQuery(query)"
        />
        <div v-if="songsLoading" class="song-search__loading">Загрузка...</div>
        <div v-else class="song-search__list">
          <div
            v-for="song in songs"
            :key="song.id"
            class="song-search__item"
            @click="onAddSong(song.id)"
          >
            {{ song.name }}
          </div>
        </div>
        <button class="btn-ghost" @click="showSongSearch = false">Закрыть</button>
      </div>
    </div>

    <div v-if="tab === 'settings'" class="stream-detail__settings">
      <label class="field">
        <span class="field__label">Название</span>
        <input v-model="name" class="field__input" />
      </label>
      <label class="field">
        <span class="field__label">Описание</span>
        <textarea v-model="description" class="field__input field__textarea" rows="3" />
      </label>
      <label class="field field--inline">
        <input v-model="loop" type="checkbox" />
        <span class="field__label">Зациклить</span>
      </label>
      <button class="btn-primary" :disabled="saving" @click="onSaveSettings">
        {{ saving ? 'Сохранение...' : 'Сохранить' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.stream-detail__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 20px;
}
.stream-detail__header h1 {
  font-size: 22px;
  font-weight: 700;
  color: var(--text);
  margin: 0;
}
.stream-detail__status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--muted);
  margin: 4px 0 0;
}
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #555;
}
.status-dot--active {
  background: #22c55e;
}
.stream-detail__controls {
  display: flex;
  gap: 8px;
}
.btn-primary {
  background: var(--primary);
  color: #fff;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
.btn-danger {
  background: var(--danger);
  color: #fff;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
.btn-ghost {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  text-decoration: none;
}
.stream-detail__tabs {
  display: flex;
  gap: 0;
  border-bottom: 1px solid #242833;
  margin-bottom: 20px;
}
.tab {
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--muted);
  padding: 10px 20px;
  cursor: pointer;
  font-size: 14px;
}
.tab--active {
  color: var(--text);
  border-bottom-color: var(--primary);
}
.empty {
  color: var(--muted);
  text-align: center;
  padding: 32px 0;
}
.queue-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: #1a1d27;
  border-radius: 6px;
  margin-bottom: 6px;
  cursor: grab;
}
.queue-item__pos {
  color: var(--muted);
  font-size: 13px;
  min-width: 20px;
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
  cursor: pointer;
  font-size: 18px;
  padding: 0 4px;
}
.btn-add {
  background: transparent;
  border: 1px dashed #333;
  color: var(--muted);
  padding: 10px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  width: 100%;
  margin-top: 8px;
}
.btn-add:hover {
  border-color: var(--primary);
  color: var(--text);
}
.song-search {
  margin-top: 12px;
  padding: 12px;
  background: #1a1d27;
  border-radius: 6px;
  border: 1px solid #333;
}
.song-search__input {
  width: 100%;
  background: #0f1115;
  border: 1px solid #333;
  border-radius: 6px;
  padding: 8px 12px;
  color: var(--text);
  font-size: 14px;
  margin-bottom: 8px;
}
.song-search__input:focus {
  outline: none;
  border-color: var(--primary);
}
.song-search__loading {
  color: var(--muted);
  font-size: 13px;
  padding: 8px 0;
}
.song-search__list {
  max-height: 200px;
  overflow-y: auto;
}
.song-search__item {
  padding: 8px;
  color: var(--text);
  font-size: 14px;
  cursor: pointer;
  border-radius: 4px;
}
.song-search__item:hover {
  background: #242833;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 12px;
}
.field--inline {
  flex-direction: row;
  align-items: center;
  gap: 8px;
}
.field__label {
  font-size: 13px;
  color: var(--muted);
}
.field__input {
  background: #1a1d27;
  border: 1px solid #333;
  border-radius: 6px;
  padding: 8px 12px;
  color: var(--text);
  font-size: 14px;
}
.field__input:focus {
  outline: none;
  border-color: var(--primary);
}
.field__textarea {
  resize: vertical;
}
</style>
