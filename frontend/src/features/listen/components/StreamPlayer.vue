<script setup lang="ts">
import type { Stream } from '@/shared/api/types'

defineProps<{ stream: Stream; currentSong: string | null; pulseScale: number }>()
defineEmits<{ back: [] }>()
</script>

<template>
  <div class="stream-player">
    <button class="stream-player__back" @click="$emit('back')">&#8592; Назад</button>

    <div class="stream-player__content">
      <div
        class="stream-player__pulse"
        :style="{ transform: `scale(${pulseScale})` }"
      />
      <h1 class="stream-player__name">{{ stream.name }}</h1>
      <p v-if="stream.description" class="stream-player__desc">{{ stream.description }}</p>
      <p v-if="currentSong" class="stream-player__song">&#9835; {{ currentSong }}</p>
    </div>
  </div>
</template>

<style scoped>
.stream-player {
  position: fixed;
  inset: 0;
  background: #0a0a0f;
  display: flex;
  flex-direction: column;
  z-index: 100;
}
.stream-player__back {
  position: absolute;
  top: 16px;
  left: 16px;
  background: transparent;
  border: 1px solid #333;
  color: var(--muted);
  padding: 6px 14px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  z-index: 101;
}
.stream-player__back:hover {
  color: var(--text);
  border-color: #555;
}
.stream-player__content {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 24px;
  padding: 80px 24px 24px;
}
.stream-player__pulse {
  width: 180px;
  height: 180px;
  border-radius: 50%;
  background: radial-gradient(circle, #7c3aed 0%, #4c1d95 60%, transparent 100%);
  box-shadow: 0 0 60px 20px #7c3aed40;
  transition: transform 0.1s ease-out;
}
.stream-player__name {
  font-size: 28px;
  font-weight: 700;
  color: var(--text);
  margin: 0;
  text-align: center;
}
.stream-player__desc {
  font-size: 15px;
  color: var(--muted);
  margin: 0;
  text-align: center;
  max-width: 400px;
}
.stream-player__song {
  font-size: 16px;
  color: var(--primary);
  margin: 0;
}
</style>
