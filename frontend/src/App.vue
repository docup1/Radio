<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { isAuthenticated, user, logout } from '@/shared/store/auth'
import PlayerBar from '@/features/content/components/PlayerBar.vue'

const router = useRouter()
const username = computed(() => user.value?.username ?? '')
const showHeader = computed(() => true)

async function onLogout() {
  await logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="app-shell">
    <header v-if="showHeader" class="topbar">
      <div class="brand">
        <RouterLink to="/content/songs" class="brand-link">Radio</RouterLink>
      </div>
      <nav v-if="isAuthenticated" class="nav">
        <RouterLink to="/content/songs" class="nav-link">Песни</RouterLink>
        <RouterLink to="/content/playlists" class="nav-link">Плейлисты</RouterLink>
        <RouterLink to="/content/streams" class="nav-link">Стримы</RouterLink>
        <RouterLink to="/streams" class="nav-link">Лента</RouterLink>
        <RouterLink to="/profile" class="nav-link">Профиль</RouterLink>
      </nav>
      <nav v-else class="nav">
        <RouterLink to="/streams" class="nav-link">Лента</RouterLink>
        <RouterLink to="/login" class="nav-link">Вход</RouterLink>
        <RouterLink to="/register" class="nav-link">Регистрация</RouterLink>
      </nav>
      <div class="user">
        <span v-if="username" class="username">{{ username }}</span>
        <button v-if="isAuthenticated" class="logout" @click="onLogout">Выход</button>
      </div>
    </header>

    <main class="shell-content">
      <RouterView />
    </main>

    <PlayerBar />
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.topbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 0 16px;
  height: 48px;
  background: var(--surface);
  border-bottom: 1px solid #242833;
  position: sticky;
  top: 0;
  z-index: 40;
}
.brand-link {
  color: var(--text);
  font-weight: 700;
  font-size: 18px;
  text-decoration: none;
}
.nav {
  display: flex;
  gap: 12px;
  flex: 1;
}
.nav-link {
  color: var(--muted);
  text-decoration: none;
  font-size: 14px;
}
.nav-link.router-link-active {
  color: var(--text);
  font-weight: 600;
}
.user {
  display: flex;
  align-items: center;
  gap: 10px;
}
.username {
  color: var(--muted);
  font-size: 13px;
}
.logout {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 4px 10px;
  border-radius: 6px;
  cursor: pointer;
}
.shell-content {
  flex: 1;
  padding: 16px;
  padding-bottom: 80px;
  max-width: 1100px;
  width: 100%;
  margin: 0 auto;
}
</style>

<style>
/* override global .app centering for shell layout */
.app {
  min-height: 100vh;
  display: block;
  padding: 0;
}
</style>
