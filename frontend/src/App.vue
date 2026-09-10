<script setup lang="ts">
import { watch } from 'vue'
import { useRoute, RouterLink, RouterView } from 'vue-router'
import { isAuthenticated } from '@/shared/store/auth'
import { initRadio, radio } from '@/shared/radio/store'
import { stopPlaying } from '@/features/content/composables/usePlayer'
import NowPlayingBar from '@/shared/components/NowPlayingBar.vue'
import { t } from '@/shared/i18n'

initRadio()

const route = useRoute()

watch(
  () => radio.streamId,
  (id) => {
    if (id) stopPlaying()
  },
)
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <RouterLink to="/" class="brand">{{ t('app.name') }}</RouterLink>
      <div class="account">
        <RouterLink v-if="!isAuthenticated" :to="{ name: 'login' }" class="btn-ghost">
          {{ t('nav.signin') }}
        </RouterLink>
        <RouterLink v-else :to="{ name: 'profile' }" class="btn-primary">
          {{ t('nav.cabinet') }}
        </RouterLink>
      </div>
    </header>

    <main class="shell-content">
      <RouterView :key="route.fullPath" />
    </main>

    <NowPlayingBar />
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.topbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 40;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  background: linear-gradient(180deg, rgba(10, 10, 14, 0.85) 0%, rgba(10, 10, 14, 0) 100%);
  pointer-events: none;
}
.topbar > * {
  pointer-events: auto;
}
.brand {
  color: #fff;
  font-weight: 800;
  font-size: 20px;
  letter-spacing: 0.02em;
  text-decoration: none;
  text-transform: uppercase;
}
.brand:hover {
  opacity: 0.85;
}
.account {
  display: flex;
}
.btn-ghost {
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.18);
  color: #fff;
  padding: 8px 18px;
  border-radius: 999px;
  text-decoration: none;
  font-size: 14px;
  font-weight: 600;
  backdrop-filter: blur(8px);
}
.btn-ghost:hover {
  background: rgba(255, 255, 255, 0.16);
}
.btn-primary {
  background: var(--primary);
  border: 1px solid var(--primary);
  color: #fff;
  padding: 8px 18px;
  border-radius: 999px;
  text-decoration: none;
  font-size: 14px;
  font-weight: 600;
}
.btn-primary:hover {
  opacity: 0.9;
}
.shell-content {
  flex: 1;
  min-height: 100vh;
}
</style>

<style>
.app {
  min-height: 100vh;
  display: block;
  padding: 0;
}
</style>