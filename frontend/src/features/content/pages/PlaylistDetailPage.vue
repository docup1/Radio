<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import draggable from 'vuedraggable'
import { content, imageURL } from '@/shared/api/content'
import { usePlayer } from '@/features/content/composables/usePlayer'
import type { Playlist, Song } from '@/shared/api/types'

const route = useRoute()
const id = route.params.id as string
const { playSong } = usePlayer()

const playlist = ref<Playlist | null>(null)
const songs = ref<Song[]>([])
const allSongs = ref<Song[]>([])
const selectedSong = ref('')
const error = ref('')
const addMsg = ref('')
const editingName = ref(false)
const draftName = ref('')

async function load() {
  try {
    playlist.value = await content.getPlaylist(id)
    draftName.value = playlist.value.name
    songs.value = await content.listPlaylistSongs(id)
    allSongs.value = await content.listSongs('mine', 100, 0)
    if (allSongs.value.length) selectedSong.value = allSongs.value[0].id
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  }
}

function startRename() {
  editingName.value = true
}
async function saveRename() {
  if (!draftName.value.trim() || !playlist.value) return
  try {
    playlist.value = await content.updatePlaylist(id, { name: draftName.value.trim() })
    editingName.value = false
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка переименования'
  }
}
onMounted(load)

async function onAdd() {
  if (!selectedSong.value) return
  addMsg.value = ''
  try {
    await content.addSongToPlaylist(id, selectedSong.value)
    songs.value = await content.listPlaylistSongs(id)
    addMsg.value = 'Добавлено'
    setTimeout(() => (addMsg.value = ''), 1500)
  } catch (e: unknown) {
    addMsg.value = e instanceof Error ? e.message : 'Ошибка'
  }
}

async function onRemove(songId: string) {
  try {
    await content.removeSongFromPlaylist(id, songId)
    songs.value = songs.value.filter((s) => s.id !== songId)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка удаления'
  }
}

async function onDragEnd() {
  try {
    await Promise.all(songs.value.map((s, idx) => content.moveSongInPlaylist(id, s.id, idx)))
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка сортировки'
    songs.value = await content.listPlaylistSongs(id)
  }
}

function play(index: number) {
  playSong(songs.value[index], songs.value)
}
</script>

<template>
  <div class="playlist-detail">
    <RouterLink to="/content/playlists" class="back">← Плейлисты</RouterLink>

    <h1 v-if="playlist && !editingName" @click="startRename" class="editable">
      {{ playlist.name }}
    </h1>
    <div v-else-if="editingName" class="rename">
      <input v-model="draftName" @keydown.enter="saveRename" @keydown.escape="editingName = false" />
      <button class="primary" @click="saveRename">Сохранить</button>
      <button class="secondary" @click="editingName = false">Отмена</button>
    </div>
    <h1 v-else class="muted">Плейлист</h1>

    <div v-if="error" class="error">{{ error }}</div>

    <div class="add-row">
      <select v-model="selectedSong" class="select">
        <option v-for="s in allSongs" :key="s.id" :value="s.id">{{ s.name }}</option>
      </select>
      <button class="primary" @click="onAdd">Добавить песню</button>
      <span v-if="addMsg" class="hint">{{ addMsg }}</span>
    </div>

    <draggable
      v-model="songs"
      item-key="id"
      handle=".drag"
      class="song-list"
      @end="onDragEnd"
    >
      <template #item="{ element: s, index }">
        <div class="song-row">
          <span class="drag" title="Перетащить">⠿</span>
          <img v-if="s.image_id" :src="imageURL(s.image_id)" class="thumb" alt="" />
          <div class="info" @click="play(index)">
            <div class="name">{{ s.name }}</div>
            <div class="desc">{{ s.description }}</div>
          </div>
          <button class="danger" @click="onRemove(s.id)">Убрать</button>
        </div>
      </template>
    </draggable>

    <div v-if="!songs.length" class="muted">В плейлисте пока нет песен</div>
  </div>
</template>

<style scoped>
.editable {
  cursor: pointer;
}
.rename {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}
.rename input {
  background: #0f1115;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 8px 12px;
  border-radius: 8px;
  flex: 1;
}
.secondary {
  background: transparent;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 8px 14px;
  border-radius: 8px;
  cursor: pointer;
}
.back {
  display: inline-block;
  margin-bottom: 12px;
  color: var(--muted);
  text-decoration: none;
  font-size: 13px;
}
.add-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin: 16px 0;
}
.select {
  flex: 1;
  background: #0f1115;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 8px;
  border-radius: 8px;
}
.primary {
  background: var(--primary);
  color: #fff;
  border: 0;
  padding: 8px 14px;
  border-radius: 8px;
  cursor: pointer;
}
.hint {
  color: var(--ok);
  font-size: 13px;
}
.song-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.song-row {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--surface);
  border: 1px solid #242833;
  padding: 10px;
  border-radius: 8px;
}
.drag {
  cursor: grab;
  color: var(--muted);
  padding: 0 6px;
}
.thumb {
  width: 44px;
  height: 44px;
  border-radius: 6px;
  object-fit: cover;
  background: #0f1115;
}
.info {
  flex: 1;
  min-width: 0;
  cursor: pointer;
}
.name {
  color: var(--text);
  font-weight: 600;
}
.desc {
  color: var(--muted);
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
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
