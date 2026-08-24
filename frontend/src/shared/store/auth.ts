import { computed, reactive } from 'vue'
import { api } from '@/shared/api/client'
import type { Credentials, PasswordUpdate, User } from '@/shared/api/types'

const state = reactive<{ user: User | null; loaded: boolean }>({
  user: null,
  loaded: false,
})

export const user = computed(() => state.user)
export const isAuthenticated = computed(() => state.user !== null)
export const authLoaded = computed(() => state.loaded)

export async function loadMe(): Promise<void> {
  try {
    state.user = await api.me()
  } catch {
    state.user = null
  } finally {
    state.loaded = true
  }
}

export async function login(c: Credentials): Promise<void> {
  await api.login(c)
  await loadMe()
}

export async function register(c: Credentials): Promise<void> {
  await api.register(c)
  await loadMe()
}

export async function logout(): Promise<void> {
  try {
    await api.logout()
  } catch {
    // regardless of result, drop local session state
  }
  state.user = null
}

export async function changePassword(p: PasswordUpdate): Promise<void> {
  await api.changePassword(p)
}

export async function deleteAccount(): Promise<void> {
  await api.deleteAccount()
  state.user = null
}
