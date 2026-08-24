<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { content, imageURL } from '@/shared/api/content'
import { useUpload } from '@/features/content/composables/useUpload'

const route = useRoute()
const router = useRouter()
const id = route.params.id as string
const { progress, uploadFile } = useUpload()

const name = ref('')
const description = ref('')
const isPublic = ref(false)
const existingImage = ref<string | null>(null)
const audioFile = ref<File | null>(null)
const imageFile = ref<File | null>(null)
const imagePreview = ref<string | null>(null)
const error = ref('')
const loading = ref(true)
const submitting = ref(false)

async function load() {
  try {
    const s = await content.getSong(id)
    name.value = s.name
    description.value = s.description ?? ''
    isPublic.value = s.is_public
    existingImage.value = s.image_id ? imageURL(s.image_id) : null
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    loading.value = false
  }
}
onMounted(load)

function onAudio(e: Event) {
  audioFile.value = (e.target as HTMLInputElement).files?.[0] ?? null
}
function onImage(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0] ?? null
  imageFile.value = f
  imagePreview.value = f ? URL.createObjectURL(f) : null
}

async function submit() {
  error.value = ''
  if (!name.value.trim()) {
    error.value = 'Название обязательно'
    return
  }
  submitting.value = true
  try {
    const patch: Record<string, unknown> = {
      name: name.value.trim(),
      description: description.value.trim(),
      is_public: isPublic.value,
    }
    if (audioFile.value) {
      const m = await uploadFile(audioFile.value, 'audio')
      patch.melody_id = m.id
    }
    if (imageFile.value) {
      const img = await uploadFile(imageFile.value, 'image')
      patch.image_id = img.id
    }
    await content.updateSong(id, patch as never)
    router.push(`/content/songs/${id}`)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка сохранения'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="edit-page">
    <RouterLink :to="`/content/songs/${id}`" class="back">← Назад</RouterLink>
    <h1>Редактировать песню</h1>

    <div v-if="loading" class="muted">Загрузка…</div>

    <div v-else class="form">
      <label class="field">
        <span>Название *</span>
        <input v-model="name" />
      </label>

      <label class="field">
        <span>Описание</span>
        <textarea v-model="description" rows="3" />
      </label>

      <label class="check">
        <input v-model="isPublic" type="checkbox" />
        Публичная
      </label>

      <div class="field">
        <span>Заменить аудио (необязательно)</span>
        <input type="file" accept="audio/*" @change="onAudio" />
      </div>

      <div class="field">
        <span>Заменить обложку (необязательно)</span>
        <div v-if="existingImage && !imagePreview" class="current">
          <img :src="existingImage" class="thumb" alt="" />
          <span class="muted">Текущая обложка</span>
        </div>
        <img v-if="imagePreview" :src="imagePreview" class="thumb" alt="" />
        <input type="file" accept="image/*" @change="onImage" />
      </div>

      <div v-if="submitting" class="progress">Загрузка… {{ progress }}%</div>
      <div v-if="error" class="error">{{ error }}</div>

      <div class="actions">
        <button :disabled="submitting" class="primary" @click="submit">Сохранить</button>
        <RouterLink :to="`/content/songs/${id}`" class="secondary">Отмена</RouterLink>
      </div>
    </div>
  </div>
</template>

<style scoped>
.back {
  display: inline-block;
  margin-bottom: 12px;
  color: var(--muted);
  text-decoration: none;
  font-size: 13px;
}
.edit-page {
  max-width: 640px;
}
.form {
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: var(--radius);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field span {
  font-size: 13px;
  color: var(--muted);
}
.field input,
.field textarea {
  background: #0f1115;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 10px 12px;
  border-radius: 8px;
}
.check {
  display: flex;
  gap: 8px;
  align-items: center;
  cursor: pointer;
}
.thumb {
  width: 120px;
  height: 120px;
  object-fit: cover;
  border-radius: 8px;
}
.current {
  display: flex;
  gap: 10px;
  align-items: center;
}
.progress {
  color: var(--primary);
  font-size: 13px;
}
.actions {
  display: flex;
  gap: 10px;
}
.primary {
  background: var(--primary);
  color: #fff;
  border: 0;
  padding: 10px 16px;
  border-radius: 8px;
  cursor: pointer;
}
.secondary {
  background: transparent;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 10px 16px;
  border-radius: 8px;
  text-decoration: none;
}
</style>
