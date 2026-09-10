<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useSongs } from '@/features/content/composables/useSongs'
import { usePlayer } from '@/features/content/composables/usePlayer'
import { content } from '@/shared/api/content'
import { streamApi } from '@/shared/api/stream'
import { user } from '@/shared/store/auth'
import { imageURL } from '@/shared/api/content'
import { t } from '@/shared/i18n'
import type { Playlist, SongScope } from '@/shared/api/types'

const { songs, scope, loading, hasMore, load, setScope, setQuery } = useSongs()
const { playSong } = usePlayer()

const localQuery = ref('')
const addedIds = ref<Set<string>>(new Set())
const streamId = computed(() => user.value?.id ?? '')

const pickerFor = ref<string | null>(null)
const playlists = ref<Playlist[]>([])
const existingPlaylistIds = ref<Set<string>>(new Set())
const pickerMsg = ref('')
const newPlaylistName = ref('')
const pickerBusy = ref(false)
let debounce: ReturnType<typeof setTimeout> | null = null

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

function onSearchInput(e: Event) {
  const v = (e.target as HTMLInputElement).value
  localQuery.value = v
  if (debounce) clearTimeout(debounce)
  debounce = setTimeout(() => setQuery(v), 300)
}

function onScope(s: SongScope) {
  setScope(s)
}

async function addToStream(songId: string) {
  if (!streamId.value) return
  try {
    await streamApi.addToQueue(streamId.value, songId)
    const next = new Set(addedIds.value)
    next.add(songId)
    addedIds.value = next
  } catch {
    // ignore
  }
}

async function openPicker(songId: string) {
  pickerFor.value = songId
  pickerMsg.value = ''
  try {
    if (!playlists.value.length) {
      playlists.value = await content.listPlaylists(100, 0)
    }
    const memberships = await Promise.all(
      playlists.value.map(async (p) => ({
        id: p.id,
        has: (await content.listPlaylistSongs(p.id)).some((s) => s.id === songId),
      })),
    )
    existingPlaylistIds.value = new Set(memberships.filter((m) => m.has).map((m) => m.id))
  } catch {
    existingPlaylistIds.value = new Set()
  }
}

const availablePlaylists = computed(() =>
  playlists.value.filter((p) => !existingPlaylistIds.value.has(p.id)),
)

function closePicker() {
  pickerFor.value = null
  newPlaylistName.value = ''
  pickerMsg.value = ''
}

async function pick(playlistId: string) {
  if (!pickerFor.value) return
  pickerBusy.value = true
  try {
    await content.addSongToPlaylist(playlistId, pickerFor.value)
    pickerMsg.value = t('playlist.added')
    setTimeout(() => {
      if (pickerFor.value) closePicker()
    }, 800)
  } catch {
    pickerMsg.value = t('common.error')
  } finally {
    pickerBusy.value = false
  }
}

async function createAndPick() {
  if (!pickerFor.value) return
  const name = newPlaylistName.value.trim()
  if (!name) return
  pickerBusy.value = true
  try {
    const p = await content.createPlaylist(name)
    playlists.value.push(p)
    await content.addSongToPlaylist(p.id, pickerFor.value)
    pickedPlaylists.value = new Set(pickedPlaylists.value).add(p.id)
    closePicker()
  } catch {
    pickerMsg.value = t('common.error')
  } finally {
    pickerBusy.value = false
  }
}

const pickedPlaylists = ref<Set<string>>(new Set())
</script>

<template>
  <div class="music-tab">
    <div class="toolbar">
      <div class="tabs">
        <button :class="{ active: scope === 'mine' }" @click="onScope('mine')">
          {{ t('music.tab.mine') }}
        </button>
        <button :class="{ active: scope === 'public' }" @click="onScope('public')">
          {{ t('music.tab.public') }}
        </button>
      </div>
      <input
        :value="localQuery"
        class="search"
        :placeholder="t('music.search')"
        @input="onSearchInput"
      />
      <RouterLink to="/content/songs/new" class="add">{{ t('music.add') }}</RouterLink>
    </div>

    <div v-if="loading && songs.length === 0" class="hint">{{ t('music.loading') }}</div>
    <div v-else-if="songs.length === 0" class="hint">{{ t('music.empty') }}</div>

    <div v-else class="list" @scroll="onScroll">
      <div v-for="(s, i) in songs" :key="s.id" class="row">
        <img v-if="s.image_id" :src="imageURL(s.image_id)" class="thumb" alt="" @click="play(i)" />
        <div v-else class="thumb placeholder" @click="play(i)" />
        <div class="info" @click="play(i)">
          <div class="name">{{ s.name }}</div>
          <div class="desc">{{ s.description }}</div>
        </div>
        <div class="actions">
          <button
            class="btn-to-stream"
            :class="{ added: addedIds.has(s.id) }"
            :disabled="addedIds.has(s.id)"
            @click="addToStream(s.id)"
          >
            {{ addedIds.has(s.id) ? t('music.added') : t('music.toStream') }}
          </button>
          <button class="btn-to-playlist" :disabled="pickerFor === s.id" @click="openPicker(s.id)">
            {{ t('music.toPlaylist') }}
          </button>
          <RouterLink :to="`/content/songs/${s.id}`" class="open">
            {{ t('music.open') }}
          </RouterLink>
        </div>
      </div>
    </div>

    <div v-if="pickerFor" class="overlay" @click.self="closePicker">
      <div class="picker">
        <div class="picker__title">{{ t('music.pickPlaylist') }}</div>
        <p class="picker__song">{{ songs.find((s) => s.id === pickerFor)?.name }}</p>

        <div v-if="availablePlaylists.length" class="picker__list">
          <button
            v-for="p in availablePlaylists"
            :key="p.id"
            class="picker__item"
            :disabled="pickerBusy || pickedPlaylists.has(p.id)"
            @click="pick(p.id)"
          >
            <span class="picker__item-name">{{ p.name }}</span>
            <span v-if="pickedPlaylists.has(p.id)" class="picker__item-done">{{ t('playlist.added') }}</span>
          </button>
        </div>
        <div v-else class="picker__empty">
          {{
            playlists.length
              ? t('music.inAllPlaylists')
              : t('playlists.empty')
          }}
        </div>

        <div class="picker__create">
          <input
            v-model="newPlaylistName"
            :placeholder="t('playlists.namePlaceholder')"
            :disabled="pickerBusy"
            @keydown.enter="createAndPick"
          />
          <button class="picker__create-btn" :disabled="pickerBusy" @click="createAndPick">
            {{ t('playlists.create') }}
          </button>
        </div>

        <div v-if="pickerMsg" class="picker__msg">{{ pickerMsg }}</div>
        <button class="picker__close" @click="closePicker">{{ t('common.cancel') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.music-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.tabs {
  display: inline-flex;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid #242833;
  border-radius: 999px;
  padding: 3px;
}
.tabs button {
  background: transparent;
  border: none;
  color: var(--muted);
  padding: 7px 16px;
  border-radius: 999px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
}
.tabs button.active {
  background: var(--primary);
  color: #fff;
}
.search {
  flex: 1;
  min-width: 160px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid #242833;
  color: #fff;
  padding: 9px 14px;
  border-radius: 999px;
  outline: none;
}
.search:focus {
  border-color: var(--primary);
}
.add {
  background: var(--primary);
  color: #fff;
  padding: 9px 16px;
  border-radius: 999px;
  text-decoration: none;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}
.list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 55vh;
  overflow-y: auto;
}
.row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: 10px;
}
.row:hover {
  border-color: #3a4050;
}
.thumb {
  width: 46px;
  height: 46px;
  border-radius: 8px;
  object-fit: cover;
  cursor: pointer;
  flex-shrink: 0;
}
.thumb.placeholder {
  background: #26242c;
}
.info {
  flex: 1;
  min-width: 0;
  cursor: pointer;
}
.name {
  color: #fff;
  font-weight: 600;
  font-size: 14px;
}
.desc {
  color: var(--muted);
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.btn-to-stream,
.btn-to-playlist {
  background: transparent;
  border: 1px solid var(--primary);
  color: var(--primary);
  padding: 6px 12px;
  border-radius: 999px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.btn-to-stream:hover,
.btn-to-playlist:hover {
  background: var(--primary);
  color: #fff;
}
.btn-to-stream.added {
  border-color: var(--ok);
  color: var(--ok);
}
.btn-to-playlist {
  border-color: #7c3aed;
  color: #a78bfa;
}
.btn-to-playlist:hover {
  background: #7c3aed;
  color: #fff;
}
.open {
  color: var(--muted);
  text-decoration: none;
  font-size: 13px;
  white-space: nowrap;
}
.open:hover {
  color: var(--text);
}
.hint {
  color: var(--muted);
  text-align: center;
  padding: 40px 0;
}
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 80;
}
.picker {
  width: min(420px, calc(100vw - 40px));
  background: #17161c;
  border: 1px solid #2c2a33;
  border-radius: 14px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 80vh;
  overflow-y: auto;
}
.picker__title {
  color: #fff;
  font-weight: 700;
  font-size: 17px;
}
.picker__song {
  color: var(--muted);
  font-size: 13px;
  margin: -6px 0 4px;
}
.picker__list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.picker__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  background: #201e27;
  border: 1px solid #2c2a33;
  border-radius: 8px;
  color: var(--text);
  padding: 10px 12px;
  cursor: pointer;
  font-size: 14px;
}
.picker__item:hover:not(:disabled) {
  border-color: #a78bfa;
}
.picker__item:disabled {
  opacity: 0.7;
  cursor: default;
}
.picker__item-done {
  color: var(--ok);
  font-size: 12px;
}
.picker__empty {
  color: var(--muted);
  font-size: 13px;
  text-align: center;
  padding: 14px 0;
}
.picker__create {
  display: flex;
  gap: 8px;
}
.picker__create input {
  flex: 1;
  min-width: 0;
  background: #0f1115;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 9px 12px;
  border-radius: 8px;
  outline: none;
}
.picker__create input:focus {
  border-color: var(--primary);
}
.picker__create-btn {
  background: var(--primary);
  border: none;
  color: #fff;
  padding: 9px 14px;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 600;
  white-space: nowrap;
}
.picker__create-btn:disabled {
  opacity: 0.6;
}
.picker__msg {
  color: var(--ok);
  font-size: 13px;
  text-align: center;
}
.picker__close {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 8px;
  border-radius: 8px;
  cursor: pointer;
}
</style>