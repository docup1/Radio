import { createRouter, createWebHistory } from 'vue-router'
import LoginPage from '@/features/auth/pages/LoginPage.vue'
import RegisterPage from '@/features/auth/pages/RegisterPage.vue'
import ProfilePage from '@/features/profile/pages/ProfilePage.vue'
import { isAuthenticated } from '@/shared/store/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/profile' },
    { path: '/login', name: 'login', component: LoginPage, meta: { public: true } },
    { path: '/register', name: 'register', component: RegisterPage, meta: { public: true } },
    { path: '/profile', name: 'profile', component: ProfilePage },
  ],
})

router.beforeEach((to) => {
  if (!to.meta.public && !isAuthenticated.value) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.public && isAuthenticated.value) {
    return { name: 'profile' }
  }
  return true
})

export default router
