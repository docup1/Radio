<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
</script>

<template>
  <div class="placeholder-page">
    <h2>Плейлист #{{ route.params.id }}</h2>
    <p>Песни плейлиста и drag-drop в разработке.</p>
  </div>
</template>
