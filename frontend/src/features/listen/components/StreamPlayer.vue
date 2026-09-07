<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { radio, isOwnerFor, start, stop, skip, resume } from '@/shared/radio/store'

defineProps<{ id: string }>()
const router = useRouter()

const isOwner = computed(() => isOwnerFor(radio.streamId))
const name = computed(() => radio.stream?.name ?? radio.streamId ?? 'Стрим')
const description = computed(() => radio.stream?.description ?? '')
const songLabel = computed(() => radio.song?.name ?? radio.song?.id ?? '')
const isPlaying = computed(() => radio.phase === 'playing' && radio.isActive)

const statusLabel = computed(() => {
  switch (radio.phase) {
    case 'connecting':
      return 'Подключение…'
    case 'playing':
      return radio.isActive ? 'Играет' : 'Пауза'
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

const needsPlay = computed(() => radio.phase === 'idle' || radio.phase === 'stopped' || radio.phase === 'ended')

function onBack() {
  router.back()
}
</script>

<template>
  <div class="stream-player">
    <button class="stream-player__back" @click="onBack">&#8592; Назад</button>

    <div class="stream-player__content">
      <div
        class="stream-player__pulse"
        :class="{ 'stream-player__pulse--idle': !isPlaying }"
        :style="{ transform: `scale(${radio.pulseScale})` }"
      />
      <h1 class="stream-player__name">{{ name }}</h1>
      <p v-if="description" class="stream-player__desc">{{ description }}</p>
      <p class="stream-player__song" :class="{ 'stream-player__song--empty': !songLabel }">
        {{ songLabel || '— нет песен в очереди —' }}
      </p>

      <div class="stream-player__actions">
        <button
          v-if="radio.resumeRequired"
          class="stream-player__play"
          @click="resume"
        >
          &#9654; Включить звук
        </button>
        <template v-else-if="isOwner">
          <button v-if="!radio.isActive" class="stream-player__play" @click="start">
            &#9654; Запустить стрим
          </button>
          <template v-else>
            <button class="stream-player__ctrl" @click="stop">&#9632; Стоп</button>
            <button class="stream-player__ctrl" @click="skip">&#9197; Скип</button>
          </template>
        </template>
        <span v-else class="stream-player__status">{{ statusLabel }}</span>
      </div>

      <div v-if="radio.feed.length > 0" class="stream-player__feed">
        <div v-for="(item, i) in radio.feed" :key="item.at + '-' + i" class="feed-item">
          <span class="feed-item__icon" :class="`feed-item__icon--${item.type}`">
            {{ iconFor(item.type) }}
          </span>
          <span class="feed-item__text">
            {{ labelFor(item) }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
function iconFor(type: string): string {
  switch (type) {
    case 'song':
      return '♫'
    case 'song_ended':
      return '▶'
    case 'stream_ended':
      return '●'
    case 'stream_stopped':
      return '■'
    default:
      return '⚠'
  }
}

function labelFor(item: { type: string; songName?: string; songId?: string; message?: string }): string {
  switch (item.type) {
    case 'song':
      return item.songName ? `Играет: ${item.songName}` : 'Следующая песня'
    case 'song_ended':
      return item.songName ? `${item.songName} — закончилась` : 'Песня закончилась'
    case 'stream_ended':
      return item.message ?? 'Стрим завершён'
    case 'stream_stopped':
      return 'Стрим остановлен'
    default:
      return item.message ?? 'Ошибка'
  }
}
</script>

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
  gap: 16px;
  padding: 80px 24px 24px;
  overflow-y: auto;
}
.stream-player__pulse {
  width: 180px;
  height: 180px;
  border-radius: 50%;
  background: radial-gradient(circle, #7c3aed 0%, #4c1d95 60%, transparent 100%);
  box-shadow: 0 0 60px 20px #7c3aed40;
  transition: transform 0.1s ease-out;
}
.stream-player__pulse--idle {
  opacity: 0.35;
  animation: none;
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
.stream-player__song--empty {
  color: var(--muted);
}
.stream-player__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 8px;
}
.stream-player__play {
  background: var(--primary);
  color: #fff;
  border: none;
  padding: 14px 36px;
  border-radius: 8px;
  font-size: 18px;
  font-weight: 600;
  cursor: pointer;
}
.stream-player__play:hover {
  opacity: 0.9;
}
.stream-player__ctrl {
  background: transparent;
  border: 1px solid var(--primary);
  color: var(--primary);
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 16px;
  cursor: pointer;
}
.stream-player__ctrl:hover {
  background: var(--primary);
  color: #fff;
}
.stream-player__status {
  font-size: 14px;
  color: var(--muted);
}
.stream-player__feed {
  width: 100%;
  max-width: 420px;
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.feed-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--muted);
}
.feed-item__icon {
  width: 20px;
  text-align: center;
  flex-shrink: 0;
}
.feed-item__icon--song {
  color: var(--primary);
}
.feed-item__icon--error {
  color: #ef4444;
}
.feed-item__text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>