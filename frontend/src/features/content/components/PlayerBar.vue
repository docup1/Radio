<script setup lang="ts">
import { computed } from 'vue'
import { usePlayer } from '@/features/content/composables/usePlayer'
import { imageURL } from '@/shared/api/content'

const { state, toggle, next, prev, seekByRatio, formatTime } = usePlayer()

const cover = computed(() =>
  state.current?.image_id ? imageURL(state.current.image_id) : null,
)
const progressPercent = computed(() =>
  state.duration ? (state.currentTime / state.duration) * 100 : 0,
)
const timeLabel = computed(
  () => `${formatTime(state.currentTime)} / ${formatTime(state.duration)}`,
)

function onProgressClick(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  const ratio = (e.clientX - rect.left) / rect.width
  seekByRatio(Math.max(0, Math.min(1, ratio)))
}
</script>

<template>
  <div v-if="state.current" class="player-bar">
    <div class="main-row">
      <img v-if="cover" :src="cover" class="cover" alt="" />
      <div v-else class="cover placeholder" />
      <div class="meta">
        <div class="title">{{ state.current.name }}</div>
        <div class="time">{{ timeLabel }}</div>
        <div v-if="state.error" class="error">{{ state.error }}</div>
        <div v-else-if="state.loading" class="loading">Загрузка…</div>
      </div>
      <div class="controls">
        <button title="Назад" @click="prev">⏮</button>
        <button :title="state.isPlaying ? 'Пауза' : 'Играть'" @click="toggle">
          {{ state.isPlaying ? '⏸' : '▶' }}
        </button>
        <button title="Вперёд" @click="next">⏭</button>
      </div>
    </div>
    <div class="progress" @click="onProgressClick">
      <div class="progress-filled" :style="{ width: progressPercent + '%' }" />
    </div>
  </div>
</template>

<style scoped>
.player-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: #1e1e1e;
  border-top: 1px solid #333;
  z-index: 50;
  display: flex;
  flex-direction: column;
}
.main-row {
  height: 64px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 16px;
}
.cover {
  width: 48px;
  height: 48px;
  object-fit: cover;
  border-radius: 6px;
  flex-shrink: 0;
}
.cover.placeholder {
  background: #2a2a2a;
}
.meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.title {
  color: #fff;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
}
.time {
  color: #9aa0aa;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.error {
  color: var(--danger);
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.loading {
  color: var(--muted);
  font-size: 12px;
}
.controls {
  display: flex;
  gap: 4px;
}
.controls button {
  background: transparent;
  border: 0;
  color: #fff;
  font-size: 18px;
  cursor: pointer;
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border-radius: 6px;
}
.controls button:hover {
  background: #2a2a2a;
}
.progress {
  height: 4px;
  background: #2a2a2a;
  cursor: pointer;
  position: relative;
}
.progress-filled {
  height: 100%;
  background: var(--primary);
  transition: width 0.1s linear;
}
</style>
