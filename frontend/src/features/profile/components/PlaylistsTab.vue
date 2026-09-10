<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { content } from '@/shared/api/content'
import type { Playlist } from '@/shared/api/types'
import { t } from '@/shared/i18n'

const playlists = ref<Playlist[]>([])
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    playlists.value = await content.listPlaylists(100, 0)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : t('common.error')
  } finally {
    loading.value = false
  }
}
onMounted(load)

const initials = (name: string) =>
  name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase() ?? '')
    .join('')

const gradient = (name: string) => {
  let h = 0
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0
  const hue = h % 360
  return `linear-gradient(135deg, hsl(${hue} 55% 30%) 0%, hsl(${(hue + 60) % 360} 55% 18%) 60%, #13131a 100%)`
}
</script>

<template>
  <div class="playlists-tab">
    <div v-if="error" class="error">{{ error }}</div>

    <div v-if="loading" class="hint">{{ t('playlists.loading') }}</div>

    <template v-else>
      <div v-if="playlists.length" class="grid">
        <RouterLink
          v-for="p in playlists"
          :key="p.id"
          :to="`/content/playlists/${p.id}`"
          class="card"
        >
          <div class="banner" :style="{ background: gradient(p.name) }">
            <span class="initials">{{ initials(p.name) }}</span>
          </div>
          <div class="info">
            <span class="name">{{ p.name }}</span>
            <span class="open">{{ t('music.open') }} &#8594;</span>
          </div>
        </RouterLink>
      </div>

      <div v-else class="hint">{{ t('playlists.empty') }}</div>
    </template>
  </div>
</template>

<style scoped>
.playlists-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 14px;
}
.card {
  display: flex;
  flex-direction: column;
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: 12px;
  overflow: hidden;
  text-decoration: none;
  transition: transform 0.15s;
}
.card:hover {
  transform: translateY(-2px);
  border-color: var(--primary);
}
.banner {
  height: 90px;
  display: flex;
  align-items: flex-start;
  padding: 12px;
}
.initials {
  color: rgba(255, 255, 255, 0.92);
  font-size: 26px;
  font-weight: 800;
  letter-spacing: 0.04em;
}
.info {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.name {
  color: var(--text);
  font-weight: 600;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.open {
  color: var(--muted);
  font-size: 12px;
}
.hint {
  color: var(--muted);
  text-align: center;
  padding: 32px 0;
}
</style>