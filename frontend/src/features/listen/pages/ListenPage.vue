<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { streamApi } from '@/shared/api/stream'
import { useWebSocketPlayer } from '../composables/useWebSocketPlayer'
import StreamPlayer from '../components/StreamPlayer.vue'
import type { Stream } from '@/shared/api/types'

const props = defineProps<{ id: string }>()
const router = useRouter()
const { state, connect, disconnect } = useWebSocketPlayer()

const stream = ref<Stream | null>(null)

onMounted(async () => {
  stream.value = await streamApi.get(props.id)
  connect(props.id)
})

onUnmounted(() => disconnect())

function onBack() {
  disconnect()
  router.back()
}
</script>

<template>
  <StreamPlayer
    v-if="stream"
    :stream="stream"
    :current-song="state.currentSong"
    :pulse-scale="state.pulseScale"
    @back="onBack"
  />
</template>
