<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { content, imageURL } from '@/shared/api/content'
import { usePlayer } from '@/features/content/composables/usePlayer'
import type { Song, Playlist } from '@/shared/api/types'
import { user } from '@/shared/store/auth'

const route = useRoute()
const router = useRouter()
const { playSong } = usePlayer()

const id = route.params.id as string
const song = ref<Song | null>(null)
const error = ref('')
const playlists = ref<Playlist[]>([])
const selectedPlaylist = ref('')
const addError = ref('')

async function load() {
  try {
    song.value = await content.getSong(id)
    playlists.value = await content.listPlaylists()
    if (playlists.value.length) selectedPlaylist.value = playlists.value[0].id
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  }
}
onMounted(load)

function play() {
  if (song.value) playSong(song.value, song.value ? [song.value] : [])
}

async function onDelete() {
  if (!confirm('Удалить песню?')) return
  try {
    await content.deleteSong(id)
    router.push('/content/songs')
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка удаления'
  }
}

async function addToPlaylist() {
  if (!selectedPlaylist.value || !song.value) return
  addError.value = ''
  try {
    await content.addSongToPlaylist(selectedPlaylist.value, song.value.id)
    addError.value = 'Добавлено'
    setTimeout(() => (addError.value = ''), 1500)
  } catch (e: unknown) {
    addError.value = e instanceof Error ? e.message : 'Ошибка'
  }
}

const isOwner = () => song.value && user.value && song.value.owner_id === user.value.id
</script>

<template>
  <div class="detail">
    <RouterLink to="/content/songs" class="back">← Назад</RouterLink>

    <div v-if="error" class="error">{{ error }}</div>

    <div v-else-if="song" class="card">
      <img v-if="song.image_id" :src="imageURL(song.image_id)" class="cover" alt="" />
      <div v-else class="cover placeholder" />

      <div class="meta">
        <h1>{{ song.name }}</h1>
        <p v-if="song.description" class="desc">{{ song.description }}</p>
        <p class="muted">{{ song.is_public ? 'Публичная' : 'Личная' }}</p>

        <div class="actions">
          <button class="primary" @click="play">▶ Играть</button>
          <RouterLink
            v-if="isOwner()"
            :to="`/content/songs/${song.id}/edit`"
            class="secondary"
            >Редактировать</RouterLink
          >
          <button v-if="isOwner()" class="danger" @click="onDelete">Удалить</button>
        </div>

        <div v-if="playlists.length" class="add-row">
          <select v-model="selectedPlaylist" class="select">
            <option v-for="p in playlists" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
          <button class="secondary" @click="addToPlaylist">В плейлист</button>
          <span v-if="addError" class="hint">{{ addError }}</span>
        </div>
      </div>
    </div>

    <div v-else class="muted">Загрузка…</div>
  </div>
</template>

<style scoped>
.back {
  display: inline-block;
  margin-bottom: 12px;
  color: var(--muted);
  text-decoration: none;
  font-size: 13px;
}
.card {
  display: flex;
  gap: 20px;
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: var(--radius);
  padding: 20px;
}
.cover {
  width: 220px;
  height: 220px;
  object-fit: cover;
  border-radius: 10px;
  background: #0f1115;
}
.cover.placeholder {
  background: #1a1d24;
  border: 1px dashed #2a2e3a;
}
.meta {
  flex: 1;
  min-width: 0;
}
.meta h1 {
  margin: 0 0 8px;
  font-size: 20px;
}
.desc {
  color: var(--text);
  white-space: pre-wrap;
}
.actions {
  display: flex;
  gap: 10px;
  margin-top: 16px;
  flex-wrap: wrap;
}
.primary {
  background: var(--primary);
  color: #fff;
  border: 0;
  padding: 8px 14px;
  border-radius: 8px;
  cursor: pointer;
}
.secondary {
  background: transparent;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 8px 14px;
  border-radius: 8px;
  cursor: pointer;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
}
.danger {
  background: transparent;
  border: 1px solid var(--danger);
  color: var(--danger);
  padding: 8px 14px;
  border-radius: 8px;
  cursor: pointer;
}
.add-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-top: 16px;
}
.select {
  background: #0f1115;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 8px;
  border-radius: 8px;
}
.hint {
  color: var(--ok);
  font-size: 13px;
}
</style>
