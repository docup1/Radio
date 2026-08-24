<script setup lang="ts">
import { useRoute } from 'vue-router'
const route = useRoute()
</script>

<template>
  <div class="placeholder-page">
    <h2>Песня #{{ route.params.id }}</h2>
    <p>Страница песни в разработке.</p>
  </div>
</template>
