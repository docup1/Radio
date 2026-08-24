<script setup lang="ts">
import { onMounted } from 'vue'
import { useSongs } from '@/features/content/composables/useSongs'
import { usePlayer } from '@/features/content/composables/usePlayer'
import { imageURL } from '@/shared/api/content'

const { songs, scope, query, loading, hasMore, load, setScope, setQuery } = useSongs()
const { playSong } = usePlayer()

onMounted(() => load(true))

function onScroll(e: Event) {
  const el = e.target as HTMLElement
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 80 && hasMore.value && !loading.value) {
    load()
  }
}

function play(index: number) {
  playSong(songs.value[index], songs.value)
}
</script>

<template>
  <div class="songs-page">
    <div class="toolbar">
      <div class="tabs">
        <button :class="{ active: scope === 'mine' }" @click="setScope('mine')">Мои</button>
        <button :class="{ active: scope === 'public' }" @click="setScope('public')">
          Публичные
        </button>
      </div>
      <input
        v-model="query"
        class="search"
        placeholder="Поиск…"
        @input="setQuery(query)"
      />
      <RouterLink to="/content/songs/new" class="add">+ Песня</RouterLink>
    </div>

    <div class="list" @scroll="onScroll">
      <div v-for="(s, i) in songs" :key="s.id" class="row" @click="play(i)">
        <img v-if="s.image_id" :src="imageURL(s.image_id)" class="thumb" alt="" />
        <div v-else class="thumb placeholder" />
        <div class="info">
          <div class="name">{{ s.name }}</div>
          <div class="desc">{{ s.description }}</div>
        </div>
        <RouterLink :to="`/content/songs/${s.id}`" class="open" @click.stop>
          Открыть
        </RouterLink>
      </div>

      <div v-if="loading" class="hint">Загрузка…</div>
      <div v-else-if="!songs.length" class="hint">Нет песен</div>
    </div>
  </div>
</template>

<style scoped>
.songs-page {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
}
.tabs button {
  background: transparent;
  border: 1px solid #444;
  color: #ccc;
  padding: 6px 12px;
  cursor: pointer;
}
.tabs button.active {
  background: #2d2d2d;
  color: #fff;
  border-color: #666;
}
.search {
  flex: 1;
  background: #1e1e1e;
  border: 1px solid #444;
  color: #fff;
  padding: 8px 12px;
  border-radius: 6px;
}
.add {
  background: #3b82f6;
  color: #fff;
  padding: 8px 12px;
  border-radius: 6px;
  text-decoration: none;
  white-space: nowrap;
}
.list {
  flex: 1;
  overflow-y: auto;
  max-height: calc(100vh - 220px);
}
.row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border-bottom: 1px solid #262626;
  cursor: pointer;
}
.row:hover {
  background: #1a1a1a;
}
.thumb {
  width: 48px;
  height: 48px;
  border-radius: 6px;
  object-fit: cover;
  background: #2a2a2a;
  flex-shrink: 0;
}
.info {
  flex: 1;
  min-width: 0;
}
.name {
  color: #fff;
  font-weight: 600;
}
.desc {
  color: #999;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.open {
  color: #60a5fa;
  text-decoration: none;
  white-space: nowrap;
}
.hint {
  color: #777;
  padding: 16px;
  text-align: center;
}
</style>
