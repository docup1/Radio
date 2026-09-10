<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { radio, isOwnerFor, start, stop, skip, resume } from '@/shared/radio/store'
import { t } from '@/shared/i18n'

defineProps<{ id: string }>()
const router = useRouter()

const isOwner = computed(() => isOwnerFor(radio.streamId))
const name = computed(() => radio.stream?.name ?? radio.streamId ?? t('profile.tab.stream'))
const description = computed(() => radio.stream?.description ?? '')
const songLabel = computed(() => radio.song?.name ?? radio.song?.id ?? '')
const isPlaying = computed(() => radio.phase === 'playing' && radio.isActive)

const statusLabel = computed(() => {
  switch (radio.phase) {
    case 'connecting':
      return t('player.status.connecting')
    case 'playing':
      return radio.isActive ? t('player.status.playing') : t('player.status.paused')
    case 'ended':
      return t('player.status.ended')
    case 'stopped':
      return t('player.status.stopped')
    case 'error':
      return radio.error || t('player.status.error')
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
    <button class="stream-player__back" @click="onBack">{{ t('player.back') }}</button>

    <div class="stream-player__content">
      <div
        class="stream-player__pulse"
        :class="{ 'stream-player__pulse--idle': !isPlaying }"
        :style="{ transform: `scale(${radio.pulseScale})` }"
      />
      <h1 class="stream-player__name">{{ name }}</h1>
      <p v-if="description" class="stream-player__desc">{{ description }}</p>
      <p class="stream-player__song" :class="{ 'stream-player__song--empty': !songLabel }">
        {{ songLabel || t('player.emptySong') }}
      </p>

      <div class="stream-player__actions">
        <button
          v-if="radio.resumeRequired"
          class="stream-player__play"
          @click="resume"
        >
          &#9654; {{ t('player.resume') }}
        </button>
        <template v-else-if="isOwner">
          <button v-if="!radio.isActive" class="stream-player__play" @click="start">
            &#9654; {{ t('stream.start') }}
          </button>
          <template v-else>
            <button class="stream-player__ctrl" @click="stop">{{ t('stream.stop') }}</button>
            <button class="stream-player__ctrl" @click="skip">{{ t('stream.skip') }}</button>
          </template>
        </template>
        <span v-else class="stream-player__status">{{ statusLabel }}</span>
      </div>
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
</style>