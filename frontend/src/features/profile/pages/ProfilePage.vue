<script setup lang="ts">
import { ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import StreamTab from '../components/StreamTab.vue'
import MusicTab from '../components/MusicTab.vue'
import PlaylistsTab from '../components/PlaylistsTab.vue'
import { t } from '@/shared/i18n'

type Tab = 'stream' | 'music' | 'playlists'
const tab = ref<Tab>('stream')

const tabs: { id: Tab; label: string }[] = [
  { id: 'stream', label: t('profile.tab.stream') },
  { id: 'music', label: t('profile.tab.music') },
  { id: 'playlists', label: t('profile.tab.playlists') },
]

watch(tab, () => {})
</script>

<template>
  <div class="profile">
    <div class="profile__header">
      <h1 class="profile__title">{{ t('profile.title') }}</h1>
      <RouterLink :to="{ name: 'settings' }" class="settings-btn">
        ⚙ {{ t('profile.settings') }}
      </RouterLink>
    </div>

    <nav class="tabs">
      <button
        v-for="tb in tabs"
        :key="tb.id"
        :class="['tab', { 'tab--active': tab === tb.id }]"
        @click="tab = tb.id"
      >
        {{ tb.label }}
      </button>
    </nav>

    <section class="tab-body">
      <StreamTab v-if="tab === 'stream'" />
      <MusicTab v-else-if="tab === 'music'" />
      <PlaylistsTab v-else />
    </section>
  </div>
</template>

<style scoped>
.profile {
  min-height: 100vh;
  padding: 104px 24px 96px;
  max-width: 980px;
  margin: 0 auto;
}
.profile__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.profile__title {
  font-size: 30px;
  font-weight: 800;
  color: #fff;
  margin: 0;
}
.settings-btn {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 8px 16px;
  border-radius: 999px;
  text-decoration: none;
  font-size: 14px;
  transition: border-color 0.15s, color 0.15s;
}
.settings-btn:hover {
  border-color: var(--primary);
  color: var(--text);
}
.tabs {
  display: flex;
  gap: 4px;
  border-bottom: 1px solid #242833;
  margin-bottom: 24px;
}
.tab {
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--muted);
  padding: 12px 18px;
  cursor: pointer;
  font-size: 15px;
  font-weight: 600;
}
.tab:hover {
  color: var(--text);
}
.tab--active {
  color: #fff;
  border-bottom-color: var(--primary);
}
</style>