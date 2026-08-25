<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useStreams } from '../composables/useStreams'
import StreamCard from '../components/StreamCard.vue'

const router = useRouter()
const { streams, loading, load, remove } = useStreams()

onMounted(() => load())

function goNew() {
  router.push({ name: 'stream-new' })
}

function goDetail(id: string) {
  router.push({ name: 'stream-detail', params: { id } })
}

async function onDelete(id: string) {
  if (!confirm('Удалить стрим?')) return
  await remove(id)
}
</script>

<template>
  <div class="streams-page">
    <div class="streams-page__header">
      <h1>Мои стримы</h1>
      <button class="btn-primary" @click="goNew">+ Создать стрим</button>
    </div>

    <div v-if="loading" class="streams-page__loading">Загрузка...</div>

    <div v-else-if="streams.length === 0" class="streams-page__empty">
      Нет стримов. Создайте первый!
    </div>

    <div v-else class="streams-page__list">
      <div v-for="s in streams" :key="s.id" class="streams-page__item">
        <StreamCard :stream="s" @click="goDetail(s.id)" />
        <div class="streams-page__actions">
          <button class="btn-ghost" @click="goDetail(s.id)">Управление</button>
          <button class="btn-danger" @click="onDelete(s.id)">Удалить</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.streams-page__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}
.streams-page__header h1 {
  font-size: 22px;
  font-weight: 700;
  color: var(--text);
  margin: 0;
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
.btn-primary:hover {
  opacity: 0.85;
}
.streams-page__loading,
.streams-page__empty {
  color: var(--muted);
  text-align: center;
  padding: 48px 0;
}
.streams-page__list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.streams-page__item {
  display: flex;
  align-items: center;
  gap: 12px;
}
.streams-page__item > :first-child {
  flex: 1;
  cursor: pointer;
}
.streams-page__actions {
  display: flex;
  gap: 8px;
}
.btn-ghost {
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
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
