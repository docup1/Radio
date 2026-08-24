<script setup lang="ts">
import Button from 'primevue/button'

withDefaults(
  defineProps<{
    label?: string
    variant?: 'primary' | 'ghost' | 'danger'
    type?: string
    disabled?: boolean
    loading?: boolean
  }>(),
  { variant: 'primary' },
)
</script>

<template>
  <Button
    :label="label"
    :type="type"
    :disabled="disabled"
    :loading="loading"
    :severity="variant === 'danger' ? 'danger' : variant === 'ghost' ? 'secondary' : undefined"
    :text="variant === 'ghost'"
    :outlined="variant === 'ghost'"
  >
    <slot />
  </Button>
</template>
