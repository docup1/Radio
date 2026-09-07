<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { radio, isOwnerFor, start, stop, skip, resume, closeRadio } from '@/shared/radio/store'

const router = useRouter()

const visible = computed(() => !!radio.streamId && radio.phase !== 'idle')
const isOwner = computed(() => isOwnerFor(radio.streamId))
const title = computed(() => radio.stream?.name ?? radio.streamId ?? '')
const songLabel = computed(() => radio.song?.name ?? radio.song?.id ?? '')
const statusLabel = computed(() => {
  switch (radio.phase) {
    case 'playing':
      return 'Играет'
    case 'connecting':
      return 'Подключение…'
    case 'ended':
      return 'Стрим завершён'
    case 'stopped':
      return 'Остановлен'
    case 'error':
      return radio.error || 'Ошибка'
    default:
      return ''
  }
})

function open() {
  if (radio.streamId) router.push({ name: 'listen', params: { id: radio.streamId } })
}
</script>

<template>
  <div v-if="visible" class="radio-bar">
    <button class="radio-bar__main" @click="open">
      <span class="radio-bar__pulse" :class="{ 'radio-bar__pulse--idle': !radio.isActive }" />
      <span class="radio-bar__meta">
        <span class="radio-bar__title">{{ title }}</span>
        <span class="radio-bar__song">{{ songLabel }}</span>
      </span>
      <span class="radio-bar__status">{{ statusLabel }}</span>
      <span v-if="radio.resumeRequired" class="radio-bar__resume" @click.stop="resume">&#9654; Вкл. звук</span>
    </button>
    <div v-if="isOwner" class="radio-bar__controls">
      <button v-if="!radio.isActive" class="radio-bar__btn" @click="start">▶</button>
      <template v-else>
        <button class="radio-bar__btn" title="Стоп" @click="stop">&#9632;</button>
        <button class="radio-bar__btn" title="Скип" @click="skip">&#9197;</button>
      </template>
    </div>
    <button class="radio-bar__close" title="Скрыть" @click="closeRadio">&times;</button>
  </div>
</template>

<style scoped>
.radio-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: #181019;
  border-top: 1px solid #38304a;
  z-index: 60;
  display: flex;
  align-items: center;
  min-height: 52px;
}
.radio-bar__main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  background: transparent;
  border: none;
  color: inherit;
  text-align: left;
  cursor: pointer;
  padding: 8px 12px;
}
.radio-bar__pulse {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  flex-shrink: 0;
  background: radial-gradient(circle, #7c3aed 0%, #4c1d95 70%, transparent 100%);
  box-shadow: 0 0 18px 6px #7c3aed50;
  animation: breathe 1.2s ease-in-out infinite;
}
.radio-bar__pulse--idle {
  animation: none;
  opacity: 0.5;
}
@keyframes breathe {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.08); opacity: 0.85; }
}
.radio-bar__meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.radio-bar__title {
  color: #fff;
  font-weight: 600;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.radio-bar__song {
  color: #b9a8e6;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.radio-bar__status {
  color: #9aa0aa;
  font-size: 12px;
  white-space: nowrap;
}
.radio-bar__resume {
  background: var(--primary);
  color: #fff;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  margin-left: 8px;
  white-space: nowrap;
}
.radio-bar__controls {
  display: flex;
  gap: 6px;
  padding: 0 8px;
}
.radio-bar__btn {
  background: transparent;
  border: 1px solid #4a3f66;
  color: #fff;
  width: 34px;
  height: 34px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 15px;
}
.radio-bar__btn:hover {
  background: #2a2236;
}
.radio-bar__close {
  background: transparent;
  border: none;
  color: #9aa0aa;
  font-size: 22px;
  cursor: pointer;
  padding: 8px 14px;
  line-height: 1;
}
.radio-bar__close:hover {
  color: #fff;
}
</style>