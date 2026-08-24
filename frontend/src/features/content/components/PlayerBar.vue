<script setup lang="ts">
import { computed } from 'vue'
import { usePlayer } from '@/features/content/composables/usePlayer'
import { imageURL } from '@/shared/api/content'

const { state, toggle, next, prev } = usePlayer()
const cover = computed(() =>
  state.current?.image_id ? imageURL(state.current.image_id) : null,
)
</script>

<template>
  <div v-if="state.current" class="player-bar">
    <img v-if="cover" :src="cover" class="cover" alt="" />
    <div class="meta">
      <div class="title">{{ state.current.name }}</div>
    </div>
    <div class="controls">
      <button title="Назад" @click="prev">⏮</button>
      <button :title="state.isPlaying ? 'Пауза' : 'Играть'" @click="toggle">
        {{ state.isPlaying ? '⏸' : '▶' }}
      </button>
      <button title="Вперёд" @click="next">⏭</button>
    </div>
  </div>
</template>

<style scoped>
.player-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  height: 64px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 16px;
  background: #1e1e1e;
  border-top: 1px solid #333;
  z-index: 50;
}
.cover {
  width: 48px;
  height: 48px;
  object-fit: cover;
  border-radius: 6px;
}
.meta {
  flex: 1;
  min-width: 0;
}
.title {
  color: #fff;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.controls button {
  background: transparent;
  border: 0;
  color: #fff;
  font-size: 18px;
  cursor: pointer;
  margin: 0 4px;
}
</style>
