<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useStreams } from '../composables/useStreams'

const router = useRouter()
const { create } = useStreams()

const name = ref('')
const description = ref('')
const loop = ref(false)
const saving = ref(false)
const error = ref('')

async function onSubmit() {
  if (!name.value.trim()) return
  saving.value = true
  error.value = ''
  try {
    const s = await create({ name: name.value.trim(), description: description.value.trim(), loop: loop.value })
    router.push({ name: 'stream-detail', params: { id: s.id } })
  } catch (e: any) {
    error.value = e?.message || 'Ошибка создания'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="stream-new">
    <h1>Новый стрим</h1>
    <form class="stream-new__form" @submit.prevent="onSubmit">
      <label class="field">
        <span class="field__label">Название</span>
        <input v-model="name" class="field__input" placeholder="Название стрима" required />
      </label>
      <label class="field">
        <span class="field__label">Описание</span>
        <textarea v-model="description" class="field__input field__textarea" placeholder="Описание (необязательно)" rows="3" />
      </label>
      <label class="field field--inline">
        <input v-model="loop" type="checkbox" />
        <span class="field__label">Зациклить</span>
      </label>
      <p v-if="error" class="error">{{ error }}</p>
      <div class="stream-new__actions">
        <button type="button" class="btn-ghost" @click="router.back()">Отмена</button>
        <button type="submit" class="btn-primary" :disabled="saving || !name.trim()">
          {{ saving ? 'Создание...' : 'Создать' }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.stream-new h1 {
  font-size: 22px;
  font-weight: 700;
  color: var(--text);
  margin: 0 0 24px;
}
.stream-new__form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 400px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.field--inline {
  flex-direction: row;
  align-items: center;
  gap: 8px;
}
.field__label {
  font-size: 13px;
  color: var(--muted);
}
.field__input {
  background: #1a1d27;
  border: 1px solid #333;
  border-radius: 6px;
  padding: 8px 12px;
  color: var(--text);
  font-size: 14px;
}
.field__input:focus {
  outline: none;
  border-color: var(--primary);
}
.field__textarea {
  resize: vertical;
}
.error {
  color: var(--danger);
  font-size: 13px;
  margin: 0;
}
.stream-new__actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
.btn-primary {
  background: var(--primary);
  color: #fff;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn-ghost {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
</style>
