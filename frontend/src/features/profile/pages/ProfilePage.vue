<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useConfirm } from 'primevue/useconfirm'
import ConfirmDialog from 'primevue/confirmdialog'
import { Card, Button, TextField, FormField } from '@/components'
import ProfileInfo from '@/features/profile/components/ProfileInfo.vue'
import { user, logout, changePassword, deleteAccount } from '@/shared/store/auth'
import { ApiError } from '@/shared/api/client'

const router = useRouter()
const confirm = useConfirm()

const msg = ref('')
const err = ref('')
const current = ref('')
const next = ref('')

async function onLogout() {
  await logout()
  await router.push('/login')
}

async function onChangePassword() {
  msg.value = ''
  err.value = ''
  try {
    await changePassword({ current_password: current.value, new_password: next.value })
    msg.value = 'Password updated'
    current.value = ''
    next.value = ''
  } catch (e) {
    err.value = e instanceof ApiError ? e.message : 'Update failed'
  }
}

function onDelete() {
  confirm.require({
    message: 'Delete your account? This cannot be undone.',
    accept: async () => {
      try {
        await deleteAccount()
        await router.push('/login')
      } catch (e) {
        err.value = e instanceof ApiError ? e.message : 'Delete failed'
      }
    },
  })
}
</script>

<template>
  <Card wide>
    <h1>Profile</h1>
      <ProfileInfo v-if="user" :username="user.username" />

      <section>
        <h2>Change password</h2>
        <FormField
          submit-label="Update password"
          :error="err"
          :ok="msg"
          @submit="onChangePassword"
        >
          <TextField v-model="current" label="Current password" type="password" autocomplete="current-password" />
          <TextField v-model="next" label="New password" type="password" autocomplete="new-password" />
        </FormField>
      </section>

      <div class="actions">
        <Button label="Logout" variant="ghost" @click="onLogout" />
        <Button label="Delete account" variant="danger" @click="onDelete" />
      </div>

      <ConfirmDialog />
  </Card>
</template>
