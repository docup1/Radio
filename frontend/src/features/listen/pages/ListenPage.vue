<script setup lang="ts">
import { onMounted } from 'vue'
import { openStream } from '@/shared/radio/store'
import StreamPlayer from '../components/StreamPlayer.vue'

const props = defineProps<{ id: string }>()

onMounted(() => {
  openStream(props.id)
})
</script>

<template>
  <StreamPlayer :id="props.id" />
</template>