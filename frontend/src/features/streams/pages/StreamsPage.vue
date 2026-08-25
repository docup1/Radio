<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useStreams } from '../composables/useStreams'

const router = useRouter()
const { stream, loading, loadMine, remove } = useStreams()

onMounted(() => loadMine())

function goDetail() {
  if (stream.value) {
    router.push({ name: 'stream-detail', params: { id: stream.value.id } })
  }
}

async function onDelete() {
  if (!stream.value) return
  if (!confirm('Удалить стрим?')) return
  await remove(stream.value.id)
}
</script>

<template>
  <div class="streams-page">
    <h1>Мой стрим</h1>

    <div v-if="loading" class="streams-page__loading">Загрузка...</div>

    <div v-else-if="stream" class="streams-page__stream">
      <div class="stream-card" @click="goDetail">
        <h3 class="stream-card__name">{{ stream.name }}</h3>
        <p v-if="stream.description" class="stream-card__desc">{{ stream.description }}</p>
      </div>
      <div class="streams-page__actions">
        <button class="btn-ghost" @click="goDetail">Управление</button>
        <RouterLink :to="{ name: 'listen', params: { id: stream.id } }" class="btn-ghost">Слушать</RouterLink>
        <button class="btn-danger" @click="onDelete">Удалить</button>
      </div>
    </div>

    <div v-else class="streams-page__empty">
      Стрим не найден
    </div>
  </div>
</template>

<style scoped>
.streams-page h1 {
  font-size: 22px;
  font-weight: 700;
  color: var(--text);
  margin: 0 0 24px;
}
.streams-page__loading,
.streams-page__empty {
  color: var(--muted);
  text-align: center;
  padding: 48px 0;
}
.stream-card {
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: var(--radius);
  padding: 16px;
  cursor: pointer;
  transition: border-color 0.15s;
}
.stream-card:hover {
  border-color: var(--primary);
}
.stream-card__name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
  margin: 0 0 4px;
}
.stream-card__desc {
  font-size: 13px;
  color: var(--muted);
  margin: 0;
}
.streams-page__actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}
.btn-ghost {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  text-decoration: none;
}
.btn-ghost:hover {
  color: var(--text);
  border-color: #555;
}
.btn-danger {
  background: transparent;
  border: 1px solid var(--danger);
  color: var(--danger);
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}
.btn-danger:hover {
  background: var(--danger);
  color: #fff;
}
</style>
