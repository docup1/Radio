<script setup lang="ts">
import Card from 'primevue/card'

withDefaults(defineProps<{ wide?: boolean }>(), { wide: false })
</script>

<template>
  <Card :class="['ui-card', { 'ui-card--wide': wide }]">
    <template #content>
      <slot />
    </template>
  </Card>
</template>

<style scoped>
.ui-card {
  width: 100%;
  max-width: 380px;
}
.ui-card--wide {
  max-width: 460px;
}
</style>
