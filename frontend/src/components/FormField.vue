<script setup lang="ts">
import { Button } from '@/components'

withDefaults(
  defineProps<{ submitLabel?: string; loading?: boolean; error?: string; ok?: string }>(),
  { submitLabel: 'Submit' },
)
const emit = defineEmits<{ submit: [] }>()
</script>

<template>
  <form @submit.prevent="emit('submit')">
    <slot />
    <Button :label="submitLabel" type="submit" :disabled="loading" :loading="loading" />
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="ok" class="ok">{{ ok }}</p>
  </form>
</template>
