<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Card, FormField, TextField } from '@/components'
import { ApiError } from '@/shared/api/client'
import { login, register } from '@/shared/store/auth'
import { t } from '@/shared/i18n'

const props = defineProps<{ mode: 'login' | 'register' }>()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const route = useRoute()
const router = useRouter()

const title = () => (props.mode === 'login' ? t('auth.signin') : t('auth.signup'))

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
    error.value = e instanceof ApiError ? e.message : t('common.error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <Card>
      <h1>{{ title() }}</h1>
      <FormField
        :submit-label="props.mode === 'login' ? t('auth.submit') : t('auth.register')"
        :loading="loading"
        :error="error"
        @submit="onSubmit"
      >
        <TextField v-model="username" :label="t('auth.username')" autocomplete="username" />
        <TextField
          v-model="password"
          :label="t('auth.password')"
          type="password"
          :autocomplete="props.mode === 'login' ? 'current-password' : 'new-password'"
        />
      </FormField>
      <p class="alt">
        <template v-if="props.mode === 'login'">
          {{ t('auth.noAccount') }} <RouterLink to="/register">{{ t('auth.register') }}</RouterLink>
        </template>
        <template v-else>
          {{ t('auth.hasAccount') }} <RouterLink to="/login">{{ t('auth.signin') }}</RouterLink>
        </template>
      </p>
    </Card>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 100px 24px 40px;
}
.alt {
  color: var(--muted);
  font-size: 14px;
  margin-top: 12px;
  text-align: center;
}
.alt a {
  color: var(--primary);
}
</style>