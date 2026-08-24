<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { content } from '@/shared/api/content'
import type { Playlist } from '@/shared/api/types'

const playlists = ref<Playlist[]>([])
const name = ref('')
const error = ref('')
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    playlists.value = await content.listPlaylists()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function create() {
  error.value = ''
  if (!name.value.trim()) {
    error.value = 'Название обязательно'
    return
  }
  try {
    const p = await content.createPlaylist(name.value.trim())
    playlists.value.push(p)
    name.value = ''
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка создания'
  }
}

async function remove(id: string) {
  if (!confirm('Удалить плейлист?')) return
  try {
    await content.deletePlaylist(id)
    playlists.value = playlists.value.filter((p) => p.id !== id)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка удаления'
  }
}
</script>

<template>
  <div class="playlists-page">
    <h1>Плейлисты</h1>

    <div class="create">
      <input v-model="name" placeholder="Название плейлиста" @keydown.enter="create" />
      <button class="primary" @click="create">Создать</button>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
    <div v-if="loading" class="muted">Загрузка…</div>

    <div v-else class="list">
      <div v-for="p in playlists" :key="p.id" class="row">
        <RouterLink :to="`/content/playlists/${p.id}`" class="name">{{ p.name }}</RouterLink>
        <button class="danger" @click="remove(p.id)">Удалить</button>
      </div>
      <div v-if="!playlists.length && !loading" class="muted">Плейлистов нет</div>
    </div>
  </div>
</template>

<style scoped>
.create {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}
.create input {
  flex: 1;
  background: #0f1115;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 10px 12px;
  border-radius: 8px;
}
.primary {
  background: var(--primary);
  color: #fff;
  border: 0;
  padding: 10px 16px;
  border-radius: 8px;
  cursor: pointer;
}
.list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--surface);
  border: 1px solid #242833;
  padding: 12px 16px;
  border-radius: 8px;
}
.name {
  color: var(--text);
  text-decoration: none;
  font-weight: 600;
}
.danger {
  background: transparent;
  border: 1px solid var(--danger);
  color: var(--danger);
  padding: 6px 10px;
  border-radius: 6px;
  cursor: pointer;
}
</style>
