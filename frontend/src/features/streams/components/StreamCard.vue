<script setup lang="ts">
import { computed } from 'vue'
import type { Stream } from '@/shared/api/types'
import { useSongMeta } from '@/shared/api/songMeta'
import { t } from '@/shared/i18n'

const props = defineProps<{ stream: Stream; showPlay?: boolean }>()
defineEmits<{ play: [id: string] }>()

const { song: nowPlaying } = useSongMeta(computed(() => props.stream.current_song_id))

const gradient = computed(() => {
  let h = 0
  for (let i = 0; i < props.stream.name.length; i++) {
    h = (h * 31 + props.stream.name.charCodeAt(i)) >>> 0
  }
  const hue = h % 360
  return `linear-gradient(135deg, hsl(${hue} 60% 28%) 0%, hsl(${(hue + 70) % 360} 60% 16%) 55%, #101018 100%)`
})

const initials = computed(() => {
  const parts = props.stream.name.trim().split(/\s+/).slice(0, 2)
  return parts.map((p) => p[0]?.toUpperCase() ?? '').join('')
})
</script>

<template>
  <div class="stream-card" @click="$emit('play', stream.id)">
    <div class="banner" :style="{ background: gradient }">
      <span class="initials">{{ initials }}</span>
      <span class="live"><span class="live-dot" />{{ t('feed.live') }}</span>
    </div>
    <div class="body">
      <h3 class="name">{{ stream.name }}</h3>
      <p v-if="stream.description" class="desc">{{ stream.description }}</p>
      <p v-if="nowPlaying" class="now">
        {{ t('feed.nowPlaying') }} <span class="now-song">{{ nowPlaying.name }}</span>
      </p>
      <p v-else class="now muted">{{ t('feed.nowPlaying') }} —</p>
    </div>
    <div v-if="showPlay" class="footer">
      <button class="play" @click.stop="$emit('play', stream.id)">&#9654;</button>
    </div>
  </div>
</template>

<style scoped>
.stream-card {
  position: relative;
  background: var(--surface);
  border: 1px solid #242833;
  border-radius: 14px;
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.15s, border-color 0.15s;
  display: flex;
  flex-direction: column;
}
.stream-card:hover {
  transform: translateY(-2px);
  border-color: var(--primary);
}
.banner {
  height: 120px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 14px;
  position: relative;
}
.initials {
  color: rgba(255, 255, 255, 0.92);
  font-size: 34px;
  font-weight: 800;
  letter-spacing: 0.04em;
}
.live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  color: #fff;
  background: rgba(0, 0, 0, 0.45);
  padding: 4px 10px;
  border-radius: 999px;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  backdrop-filter: blur(4px);
}
.live-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #22c55e;
  box-shadow: 0 0 8px 2px #22c55e80;
  animation: blink 1.4s ease-in-out infinite;
}
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
.body {
  flex: 1;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.name {
  font-size: 16px;
  font-weight: 700;
  color: var(--text);
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.desc {
  font-size: 13px;
  color: var(--muted);
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.now {
  font-size: 12px;
  color: var(--primary);
  margin: 2px 0 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.now-song {
  color: var(--text);
  font-weight: 600;
}
.muted {
  color: var(--muted);
}
.footer {
  display: flex;
  justify-content: flex-end;
  padding: 0 14px 14px;
}
.play {
  background: var(--primary);
  color: #fff;
  border: none;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  font-size: 15px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 6px 16px -6px var(--primary);
}
.play:hover {
  opacity: 0.88;
}
</style>