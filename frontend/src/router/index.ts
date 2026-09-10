import { createRouter, createWebHistory } from 'vue-router'
import LoginPage from '@/features/auth/pages/LoginPage.vue'
import RegisterPage from '@/features/auth/pages/RegisterPage.vue'
import ProfilePage from '@/features/profile/pages/ProfilePage.vue'
import SettingsPage from '@/features/profile/pages/SettingsPage.vue'
import SongNewPage from '@/features/content/pages/SongNewPage.vue'
import SongDetailPage from '@/features/content/pages/SongDetailPage.vue'
import SongEditPage from '@/features/content/pages/SongEditPage.vue'
import PlaylistDetailPage from '@/features/content/pages/PlaylistDetailPage.vue'
import FeedPage from '@/features/listen/pages/FeedPage.vue'
import ListenPage from '@/features/listen/pages/ListenPage.vue'
import { isAuthenticated } from '@/shared/store/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'feed', component: FeedPage, meta: { public: true, publicAlways: true } },
    { path: '/login', name: 'login', component: LoginPage, meta: { public: true } },
    { path: '/register', name: 'register', component: RegisterPage, meta: { public: true } },
    { path: '/profile', name: 'profile', component: ProfilePage },
    { path: '/profile/settings', name: 'settings', component: SettingsPage },
    { path: '/content/songs/new', name: 'song-new', component: SongNewPage },
    { path: '/content/songs/:id', name: 'song-detail', component: SongDetailPage, props: true },
    {
      path: '/content/songs/:id/edit',
      name: 'song-edit',
      component: SongEditPage,
      props: true,
    },
    {
      path: '/content/playlists/:id',
      name: 'playlist-detail',
      component: PlaylistDetailPage,
      props: true,
    },
    {
      path: '/streams/:id/listen',
      name: 'listen',
      component: ListenPage,
      props: true,
      meta: { public: true, publicAlways: true, hidePlayer: true },
    },
  ],
})

router.beforeEach((to) => {
  const publicAlways = to.meta.publicAlways as boolean | undefined
  if (!to.meta.public && !isAuthenticated.value) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.public && !publicAlways && isAuthenticated.value) {
    return { name: 'profile' }
  }
  return true
})

export default router