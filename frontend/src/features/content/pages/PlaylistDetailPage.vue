<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import draggable from 'vuedraggable'
import { content, imageURL } from '@/shared/api/content'
import { usePlayer } from '@/features/content/composables/usePlayer'
import { t } from '@/shared/i18n'
import type { Playlist, Song } from '@/shared/api/types'

const route = useRoute()
const id = route.params.id as string
const { playSong } = usePlayer()

const playlist = ref<Playlist | null>(null)
const songs = ref<Song[]>([])
const error = ref('')
const editingName = ref(false)
const draftName = ref('')

async function load() {
  try {
    playlist.value = await content.getPlaylist(id)
    draftName.value = playlist.value.name
    songs.value = await content.listPlaylistSongs(id)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : t('songs.loadError')
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
    error.value = e instanceof Error ? e.message : t('playlist.renameError')
  }
}
onMounted(load)

async function onRemove(songId: string) {
  try {
    await content.removeSongFromPlaylist(id, songId)
    songs.value = songs.value.filter((s) => s.id !== songId)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : t('playlist.removeError')
  }
}

async function onDragEnd() {
  try {
    await Promise.all(songs.value.map((s, idx) => content.moveSongInPlaylist(id, s.id, idx)))
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : t('playlist.sortError')
    songs.value = await content.listPlaylistSongs(id)
  }
}

function play(index: number) {
  playSong(songs.value[index], songs.value)
}
</script>

<template>
  <div class="playlist-detail">
    <RouterLink to="/profile" class="back">{{ t('player.back') }}</RouterLink>

    <h1 v-if="playlist && !editingName" @click="startRename" class="editable">
      {{ playlist.name }}
    </h1>
    <div v-else-if="editingName" class="rename">
      <input v-model="draftName" @keydown.enter="saveRename" @keydown.escape="editingName = false" />
      <button class="primary" @click="saveRename">{{ t('songs.save') }}</button>
      <button class="secondary" @click="editingName = false">{{ t('common.cancel') }}</button>
    </div>
    <h1 v-else class="muted">{{ t('playlist.title') }}</h1>

    <div v-if="error" class="error">{{ error }}</div>

    <draggable
      v-model="songs"
      item-key="id"
      handle=".drag"
      class="song-list"
      @end="onDragEnd"
    >
      <template #item="{ element: s, index }">
        <div class="song-row">
          <span class="drag" :title="t('playlist.drag')">⠿</span>
          <img v-if="s.image_id" :src="imageURL(s.image_id)" class="thumb" alt="" />
          <div class="info" @click="play(index)">
            <div class="name">{{ s.name }}</div>
            <div class="desc">{{ s.description }}</div>
          </div>
          <button class="danger" @click="onRemove(s.id)">{{ t('playlist.remove') }}</button>
        </div>
      </template>
    </draggable>

    <div v-if="!songs.length" class="muted">{{ t('playlist.isEmpty') }}</div>
  </div>
</template>

<style scoped>
.playlist-detail {
  width: 100%;
  max-width: 720px;
  margin: 0 auto;
  padding: 104px 24px 96px;
}
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
.primary {
  background: var(--primary);
  color: #fff;
  border: 0;
  padding: 8px 14px;
  border-radius: 8px;
  cursor: pointer;
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
