<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { content } from '@/shared/api/content'
import { useUpload } from '@/features/content/composables/useUpload'

const router = useRouter()
const { progress, uploadFile } = useUpload()

const name = ref('')
const description = ref('')
const isPublic = ref(false)
const audioFile = ref<File | null>(null)
const imageFile = ref<File | null>(null)
const audioPreview = ref<string | null>(null)
const imagePreview = ref<string | null>(null)
const error = ref('')
const submitting = ref(false)

function onAudio(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0] ?? null
  audioFile.value = f
  audioPreview.value = f ? f.name : null
}
function onImage(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0] ?? null
  imageFile.value = f
  if (f) imagePreview.value = URL.createObjectURL(f)
  else imagePreview.value = null
}
function onDropAudio(e: DragEvent) {
  e.preventDefault()
  const f = e.dataTransfer?.files?.[0] ?? null
  if (f && f.type.startsWith('audio/')) {
    audioFile.value = f
    audioPreview.value = f.name
  }
}
function onDropImage(e: DragEvent) {
  e.preventDefault()
  const f = e.dataTransfer?.files?.[0] ?? null
  if (f && f.type.startsWith('image/')) {
    imageFile.value = f
    imagePreview.value = URL.createObjectURL(f)
  }
}

async function submit() {
  error.value = ''
  if (!name.value.trim()) {
    error.value = 'Название обязательно'
    return
  }
  if (!audioFile.value) {
    error.value = 'Выберите аудио-файл'
    return
  }
  submitting.value = true
  try {
    const melody = await uploadFile(audioFile.value, 'audio')
    let imageId: string | undefined
    if (imageFile.value) {
      const img = await uploadFile(imageFile.value, 'image')
      imageId = img.id
    }
    const song = await content.createSong({
      name: name.value.trim(),
      description: description.value.trim() || undefined,
      melody_id: melody.id,
      image_id: imageId,
      is_public: isPublic.value,
    })
    router.push(`/content/songs/${song.id}`)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Ошибка создания'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="new-page">
    <h1>Новая песня</h1>

    <div class="form">
      <label class="field">
        <span>Название *</span>
        <input v-model="name" placeholder="Как называется трек" />
      </label>

      <label class="field">
        <span>Описание</span>
        <textarea v-model="description" rows="3" placeholder="Необязательно" />
      </label>

      <label class="check">
        <input v-model="isPublic" type="checkbox" />
        Публичная (видна всем)
      </label>

      <div class="dropzone" @dragover.prevent @drop="onDropAudio">
        <div class="dz-title">Аудио *</div>
        <div v-if="audioPreview" class="preview">{{ audioPreview }}</div>
        <div v-else class="hint">Перетащите mp3/wav/ogg сюда или выберите файл</div>
        <input type="file" accept="audio/*" @change="onAudio" />
      </div>

      <div class="dropzone" @dragover.prevent @drop="onDropImage">
        <div class="dz-title">Обложка (необязательно)</div>
        <img v-if="imagePreview" :src="imagePreview" class="img-preview" alt="" />
        <div v-else class="hint">Перетащите картинку или выберите файл</div>
        <input type="file" accept="image/*" @change="onImage" />
      </div>

      <div v-if="submitting" class="progress">Загрузка… {{ progress }}%</div>
      <div v-if="error" class="error">{{ error }}</div>

      <div class="actions">
        <button :disabled="submitting" class="primary" @click="submit">Создать</button>
        <RouterLink to="/content/songs" class="secondary">Отмена</RouterLink>
      </div>
    </div>
  </div>
</template>

<style scoped>
.new-page {
  max-width: 640px;
}
.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: var(--radius);
  padding: 20px;
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
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--text);
  cursor: pointer;
}
.dropzone {
  border: 1px dashed #2a2e3a;
  border-radius: 8px;
  padding: 16px;
  background: #0f1115;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.dz-title {
  font-size: 13px;
  color: var(--muted);
}
.hint {
  color: #6b6f7a;
  font-size: 13px;
}
.preview {
  color: var(--text);
  font-size: 14px;
}
.img-preview {
  max-width: 160px;
  max-height: 160px;
  border-radius: 8px;
  object-fit: cover;
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
.primary:disabled {
  opacity: 0.6;
}
.secondary {
  background: transparent;
  border: 1px solid #2a2e3a;
  color: var(--text);
  padding: 10px 16px;
  border-radius: 8px;
  text-decoration: none;
  text-align: center;
}
</style>
