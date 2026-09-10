<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePlayer } from '@/features/content/composables/usePlayer'
import { imageURL } from '@/shared/api/content'
import { radio, closeRadio, isOwnerFor, start, stop, skip, resume } from '@/shared/radio/store'
import { t } from '@/shared/i18n'

const route = useRoute()
const router = useRouter()
const { state, toggle, next, prev, seekByRatio, formatTime } = usePlayer()

const showTrack = computed(() => !!state.current)
const showRadio = computed(
  () => !state.current && !!radio.streamId && radio.phase !== 'idle',
)
const visible = computed(() => (route.meta.hidePlayer ? false : showTrack.value || showRadio.value))
const isRadio = computed(() => !state.current && showRadio.value && !showTrack.value)

const cover = computed(() =>
  state.current?.image_id ? imageURL(state.current.image_id) : null,
)
const progressPercent = computed(() =>
  state.duration ? (state.currentTime / state.duration) * 100 : 0,
)
const timeLabel = computed(
  () => `${formatTime(state.currentTime)} / ${formatTime(state.duration)}`,
)

const radioIsOwner = computed(() => isOwnerFor(radio.streamId))
const radioTitle = computed(() => radio.stream?.name ?? '')
const radioSong = computed(() => radio.song?.name ?? '')
const radioStatus = computed(() => {
  switch (radio.phase) {
    case 'playing':
      return t('player.status.playing')
    case 'connecting':
      return t('player.status.connecting')
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

function onProgressClick(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  const ratio = (e.clientX - rect.left) / rect.width
  seekByRatio(Math.max(0, Math.min(1, ratio)))
}

function openRadio() {
  if (radio.streamId) router.push({ name: 'listen', params: { id: radio.streamId } })
}
</script>

<template>
  <div v-if="visible" :class="['now-playing', isRadio ? 'now-playing--radio' : '']">
    <!-- Трек -->
    <template v-if="showTrack">
      <div class="main-row">
        <img v-if="cover" :src="cover" class="cover" alt="" />
        <div v-else class="cover placeholder" />
        <div class="meta">
          <div class="title">{{ state.current.name }}</div>
          <div class="sub">{{ timeLabel }}</div>
          <div v-if="state.error" class="error">{{ state.error }}</div>
          <div v-else-if="state.loading" class="loading">{{ t('common.loading') }}</div>
        </div>
        <div class="controls">
          <button :title="t('player.prev')" @click="prev">⏮</button>
          <button :title="state.isPlaying ? t('player.pause') : t('player.play')" @click="toggle">
            {{ state.isPlaying ? '⏸' : '▶' }}
          </button>
          <button :title="t('player.next')" @click="next">⏭</button>
        </div>
      </div>
      <div class="progress" @click="onProgressClick">
        <div class="progress-filled" :style="{ width: progressPercent + '%' }" />
      </div>
    </template>

    <!-- Радио -->
    <template v-else>
      <div class="radio-row">
        <button class="radio-main" @click="openRadio">
          <span
            class="pulse"
            :class="{ 'pulse--idle': !radio.isActive }"
            :style="{ transform: `scale(${radio.pulseScale})` }"
          />
          <span class="radio-meta">
            <span class="radio-title">{{ radioTitle || (radio.streamId ?? '') }}</span>
            <span class="radio-song">{{ radioSong || t('player.closedQueue') }}</span>
          </span>
          <span class="radio-status">{{ radioStatus }}</span>
          <span v-if="radio.resumeRequired" class="resume" @click.stop="resume">
            &#9654; {{ t('player.resume') }}
          </span>
        </button>
        <div v-if="radioIsOwner" class="controls">
          <button v-if="!radio.isActive" :title="t('player.play')" @click="start">▶</button>
          <template v-else>
            <button :title="t('player.stop')" @click="stop">⏹</button>
            <button :title="t('player.skip')" @click="skip">⏭</button>
          </template>
        </div>
        <button class="close" :title="t('player.hideHint')" @click="closeRadio">&times;</button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.now-playing {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 50;
  display: flex;
  flex-direction: column;
  background: #17161b;
  border-top: 1px solid #2c2a33;
}
.now-playing--radio {
  background: #191022;
  border-top-color: #3b2f4e;
}
.main-row,
.radio-row {
  min-height: 60px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 16px;
}
.cover {
  width: 44px;
  height: 44px;
  object-fit: cover;
  border-radius: 6px;
  flex-shrink: 0;
}
.cover.placeholder {
  background: #26242c;
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
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.sub,
.time {
  color: var(--muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.error {
  color: var(--danger);
  font-size: 12px;
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
  border: 1px solid #3a3842;
  color: #fff;
  font-size: 15px;
  cursor: pointer;
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: 8px;
}
.controls button:hover {
  background: #26232d;
}
.progress {
  height: 4px;
  background: #27252c;
  cursor: pointer;
}
.progress-filled {
  height: 100%;
  background: var(--primary);
  transition: width 0.1s linear;
}
.radio-main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  background: transparent;
  border: none;
  color: inherit;
  text-align: left;
  cursor: pointer;
  padding: 8px 0;
}
.pulse {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  flex-shrink: 0;
  background: radial-gradient(circle, #7c3aed 0%, #4c1d95 70%, transparent 100%);
  box-shadow: 0 0 18px 6px #7c3aed50;
  transition: transform 0.1s ease-out;
}
.pulse--idle {
  opacity: 0.5;
}
.radio-meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.radio-title {
  color: #fff;
  font-weight: 600;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.radio-song {
  color: #c3b1ee;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.radio-status {
  color: var(--muted);
  font-size: 12px;
  white-space: nowrap;
}
.resume {
  background: var(--primary);
  color: #fff;
  border: none;
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  margin-left: 8px;
  white-space: nowrap;
}
.close {
  background: transparent;
  border: none;
  color: var(--muted);
  font-size: 22px;
  cursor: pointer;
  padding: 8px 14px;
  line-height: 1;
}
.close:hover {
  color: #fff;
}
</style>