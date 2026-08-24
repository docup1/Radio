<script setup lang="ts">
import { useRoute } from 'vue-router'
const route = useRoute()
</script>

<template>
  <div class="placeholder-page">
    <h2>Редактировать #{{ route.params.id }}</h2>
    <p>Форма редактирования в разработке.</p>
  </div>
</template>
