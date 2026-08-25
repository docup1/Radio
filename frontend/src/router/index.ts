import { createRouter, createWebHistory } from 'vue-router'
import LoginPage from '@/features/auth/pages/LoginPage.vue'
import RegisterPage from '@/features/auth/pages/RegisterPage.vue'
import ProfilePage from '@/features/profile/pages/ProfilePage.vue'
import SongsPage from '@/features/content/pages/SongsPage.vue'
import SongNewPage from '@/features/content/pages/SongNewPage.vue'
import SongDetailPage from '@/features/content/pages/SongDetailPage.vue'
import SongEditPage from '@/features/content/pages/SongEditPage.vue'
import PlaylistsPage from '@/features/content/pages/PlaylistsPage.vue'
import PlaylistDetailPage from '@/features/content/pages/PlaylistDetailPage.vue'
import StreamsPage from '@/features/streams/pages/StreamsPage.vue'
import StreamDetailPage from '@/features/streams/pages/StreamDetailPage.vue'
import FeedPage from '@/features/listen/pages/FeedPage.vue'
import ListenPage from '@/features/listen/pages/ListenPage.vue'
import { isAuthenticated } from '@/shared/store/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/content/songs' },
    { path: '/login', name: 'login', component: LoginPage, meta: { public: true } },
    { path: '/register', name: 'register', component: RegisterPage, meta: { public: true } },
    { path: '/profile', name: 'profile', component: ProfilePage },
    { path: '/content/songs', name: 'songs', component: SongsPage },
    { path: '/content/songs/new', name: 'song-new', component: SongNewPage },
    { path: '/content/songs/:id', name: 'song-detail', component: SongDetailPage, props: true },
    {
      path: '/content/songs/:id/edit',
      name: 'song-edit',
      component: SongEditPage,
      props: true,
    },
    { path: '/content/playlists', name: 'playlists', component: PlaylistsPage },
    {
      path: '/content/playlists/:id',
      name: 'playlist-detail',
      component: PlaylistDetailPage,
      props: true,
    },
    { path: '/content/streams', name: 'streams', component: StreamsPage },
    {
      path: '/content/streams/:id',
      name: 'stream-detail',
      component: StreamDetailPage,
      props: true,
    },
    { path: '/streams', name: 'feed', component: FeedPage, meta: { public: true, publicAlways: true } },
    {
      path: '/streams/:id/listen',
      name: 'listen',
      component: ListenPage,
      props: true,
      meta: { public: true, publicAlways: true },
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
