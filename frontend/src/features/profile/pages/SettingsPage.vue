<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useConfirm } from 'primevue/useconfirm'
import ConfirmDialog from 'primevue/confirmdialog'
import { Card, Button, TextField } from '@/components'
import { useStreams } from '@/features/streams/composables/useStreams'
import {
  user,
  logout,
  changePassword,
  deleteAccount,
} from '@/shared/store/auth'
import { setLocale, t, type Locale } from '@/shared/i18n'
import { ApiError } from '@/shared/api/client'

const router = useRouter()
const confirm = useConfirm()

const streamId = computed(() => user.value?.id ?? '')
const { stream, loading: streamLoading, get, update } = useStreams()

const err = ref('')
const msg = ref('')
const current = ref('')
const next = ref('')
const streamName = ref('')
const streamDesc = ref('')
const loop = ref(false)
const savingStream = ref(false)

const languages: { id: Locale; label: string }[] = [
  { id: 'ru', label: t('settings.language.ru') },
]

const selectedLanguage = ref<Locale>('ru')

onMounted(async () => {
  selectedLanguage.value = (localStorage.getItem('radio:lang') as Locale) || 'ru'
  await loadStream()
})

async function loadStream() {
  if (!streamId.value) return
  await get(streamId.value)
  if (stream.value) {
    streamName.value = stream.value.name
    streamDesc.value = stream.value.description
    loop.value = stream.value.loop
  }
}

async function onBack() {
  await router.push({ name: 'profile' })
}

async function onLogout() {
  await logout()
  await router.push('/login')
}

async function onChangePassword() {
  msg.value = ''
  err.value = ''
  try {
    await changePassword({ current_password: current.value, new_password: next.value })
    msg.value = t('settings.passwordUpdated')
    current.value = ''
    next.value = ''
  } catch (e) {
    err.value = e instanceof ApiError ? e.message : t('common.error')
  }
}

function onDelete() {
  confirm.require({
    message: t('settings.deleteConfirm'),
    accept: async () => {
      try {
        await deleteAccount()
        await router.push('/login')
      } catch (e) {
        err.value = e instanceof ApiError ? e.message : t('common.error')
      }
    },
  })
}

function onLanguageChange(e: Event) {
  const l = (e.target as HTMLSelectElement).value as Locale
  selectedLanguage.value = l
  setLocale(l)
  err.value = ''
}

async function onSaveStream() {
  if (!streamId.value) return
  savingStream.value = true
  try {
    await update(streamId.value, {
      name: streamName.value.trim(),
      description: streamDesc.value.trim(),
      loop: loop.value,
    })
    msg.value = t('settings.saved')
  } finally {
    savingStream.value = false
  }
}
</script>

<template>
  <div class="settings-page">
    <button class="back" @click="onBack">{{ t('settings.back') }}</button>
    <h1 class="title">{{ t('settings.title') }}</h1>

    <Card wide>
      <section v-if="user" class="section">
        <h2>{{ t('settings.profile') }}</h2>
        <div class="field-text">
          <span class="field-label">{{ t('settings.username') }}</span>
          <span class="field-value">{{ user.username }}</span>
        </div>
        <button class="btn-ghost" @click="onLogout">{{ t('settings.logout') }}</button>
      </section>

      <section class="section">
        <h2>{{ t('settings.password') }}</h2>
        <TextField
          v-model="current"
          :label="t('settings.currentPassword')"
          type="password"
          autocomplete="current-password"
        />
        <TextField
          v-model="next"
          :label="t('settings.newPassword')"
          type="password"
          autocomplete="new-password"
        />
        <div class="row-actions">
          <Button :label="t('settings.updatePassword')" variant="primary" @click="onChangePassword" />
        </div>
      </section>

      <section class="section">
        <h2>{{ t('settings.language') }}</h2>
        <select :value="selectedLanguage" class="select" @change="onLanguageChange">
          <option v-for="l in languages" :key="l.id" :value="l.id">{{ l.label }}</option>
        </select>
      </section>

      <section class="section">
        <h2>{{ t('settings.stream') }}</h2>
        <template v-if="streamLoading">
          <div class="muted">{{ t('common.loading') }}</div>
        </template>
        <template v-else-if="stream">
          <label class="field">
            <span class="field-label">{{ t('settings.streamName') }}</span>
            <input v-model="streamName" class="input" />
          </label>
          <label class="field">
            <span class="field-label">{{ t('settings.streamDescription') }}</span>
            <textarea v-model="streamDesc" class="input textarea" rows="3" />
          </label>
          <label class="field field--inline">
            <input v-model="loop" type="checkbox" />
            <span class="field-label">{{ t('settings.streamLoop') }}</span>
          </label>
          <div class="row-actions">
            <Button
              :label="savingStream ? t('settings.saving') : t('settings.save')"
              variant="primary"
              :disabled="savingStream"
              @click="onSaveStream"
            />
          </div>
        </template>
      </section>

      <section class="section">
        <div v-if="msg" class="ok">{{ msg }}</div>
        <div v-if="err" class="error">{{ err }}</div>
        <div class="row-actions">
          <Button :label="t('settings.deleteAccount')" variant="danger" @click="onDelete" />
        </div>
      </section>

      <ConfirmDialog />
    </Card>
  </div>
</template>

<style scoped>
.settings-page {
  min-height: 100vh;
  padding: 104px 24px 96px;
  max-width: 720px;
  margin: 0 auto;
}
.back {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 8px 16px;
  border-radius: 999px;
  cursor: pointer;
  font-size: 14px;
  margin-bottom: 16px;
}
.back:hover {
  border-color: var(--primary);
  color: var(--text);
}
.title {
  font-size: 30px;
  font-weight: 800;
  color: #fff;
  margin: 0 0 24px;
}
.section {
  margin-bottom: 26px;
}
.section h2 {
  font-size: 13px;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 14px;
}
.field-text {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 14px;
}
.field-label {
  font-size: 13px;
  color: var(--muted);
}
.field-value {
  color: var(--text);
  font-weight: 600;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}
.field--inline {
  flex-direction: row;
  align-items: center;
  gap: 10px;
}
.input {
  background: #1a1d24;
  border: 1px solid #333;
  border-radius: 10px;
  padding: 10px 14px;
  color: var(--text);
  font-size: 14px;
  outline: none;
}
.input:focus {
  border-color: var(--primary);
}
.textarea {
  resize: vertical;
}
.select {
  background: #1a1d24;
  border: 1px solid #333;
  color: var(--text);
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 14px;
  min-width: 220px;
}
.row-actions {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}
.btn-ghost {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 8px 18px;
  border-radius: 999px;
  cursor: pointer;
  font-size: 14px;
}
.btn-ghost:hover {
  color: var(--text);
  border-color: #555;
}
.ok {
  color: var(--ok);
  font-size: 13px;
  margin-bottom: 12px;
}
.error {
  color: var(--error);
  font-size: 13px;
  margin-bottom: 12px;
}
</style>