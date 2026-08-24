<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Card, FormField, TextField } from '@/components'
import { ApiError } from '@/shared/api/client'
import { login, register } from '@/shared/store/auth'

const props = defineProps<{ mode: 'login' | 'register' }>()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const route = useRoute()
const router = useRouter()

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    if (props.mode === 'login') {
      await login({ username: username.value, password: password.value })
    } else {
      await register({ username: username.value, password: password.value })
    }
    const redirect = (route.query.redirect as string) || '/profile'
    await router.push(props.mode === 'login' ? redirect : '/profile')
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Request failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <Card>
    <template #content>
      <h1>{{ mode === 'login' ? 'Login' : 'Register' }}</h1>
      <FormField
        :submit-label="mode === 'login' ? 'Login' : 'Register'"
        :loading="loading"
        :error="error"
        @submit="onSubmit"
      >
        <TextField v-model="username" label="Username" autocomplete="username" />
        <TextField
          v-model="password"
          label="Password"
          type="password"
          :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
        />
      </FormField>
      <p class="alt">
        <template v-if="mode === 'login'">No account? <RouterLink to="/register">Register</RouterLink></template>
        <template v-else>Have an account? <RouterLink to="/login">Login</RouterLink></template>
      </p>
    </template>
  </Card>
</template>
