<script setup lang="ts">
defineProps<{ username: string }>()
</script>

<template>
  <p><strong>{{ username }}</strong></p>
</template>
