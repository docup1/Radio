<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useFeed } from '../composables/useFeed'
import StreamCard from '@/features/streams/components/StreamCard.vue'

const router = useRouter()
const { streams, loading, load } = useFeed()

onMounted(() => load())

function onPlay(id: string) {
  router.push({ name: 'listen', params: { id } })
}
</script>

<template>
  <div class="feed-page">
    <h1>Стримы</h1>

    <div v-if="loading" class="feed-page__loading">Загрузка...</div>

    <div v-else-if="streams.length === 0" class="feed-page__empty">
      Нет активных стримов
    </div>

    <div v-else class="feed-page__grid">
      <StreamCard
        v-for="s in streams"
        :key="s.id"
        :stream="s"
        :show-play="true"
        @play="onPlay"
      />
    </div>
  </div>
</template>

<style scoped>
.feed-page h1 {
  font-size: 22px;
  font-weight: 700;
  color: var(--text);
  margin: 0 0 24px;
}
.feed-page__loading,
.feed-page__empty {
  color: var(--muted);
  text-align: center;
  padding: 48px 0;
}
.feed-page__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}
</style>
