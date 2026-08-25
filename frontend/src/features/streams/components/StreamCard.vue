<script setup lang="ts">
import type { Stream } from '@/shared/api/types'

defineProps<{ stream: Stream; showPlay?: boolean }>()
defineEmits<{ play: [id: string] }>()
</script>

<template>
  <div class="stream-card" @click="$emit('play', stream.id)">
    <div class="stream-card__header">
      <span class="stream-card__live">LIVE</span>
    </div>
    <div class="stream-card__body">
      <h3 class="stream-card__name">{{ stream.name }}</h3>
      <p v-if="stream.description" class="stream-card__desc">{{ stream.description }}</p>
    </div>
    <div v-if="showPlay" class="stream-card__footer">
      <button class="stream-card__play" @click.stop="$emit('play', stream.id)">&#9654;</button>
    </div>
  </div>
</template>

<style scoped>
.stream-card {
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: var(--radius);
  padding: 16px;
  cursor: pointer;
  transition: border-color 0.15s;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.stream-card:hover {
  border-color: var(--primary);
}
.stream-card__header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.stream-card__live {
  font-size: 11px;
  font-weight: 700;
  color: #22c55e;
  background: #22c55e20;
  padding: 2px 8px;
  border-radius: 4px;
  letter-spacing: 0.5px;
}
.stream-card__body {
  flex: 1;
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
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.stream-card__footer {
  display: flex;
  justify-content: flex-end;
}
.stream-card__play {
  background: var(--primary);
  color: #fff;
  border: none;
  border-radius: 50%;
  width: 36px;
  height: 36px;
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.stream-card__play:hover {
  opacity: 0.85;
}
</style>
