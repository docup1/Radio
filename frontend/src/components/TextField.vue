<script setup lang="ts">
import FloatLabel from 'primevue/floatlabel'
import InputText from 'primevue/inputtext'

const model = defineModel<string>()
withDefaults(
  defineProps<{ label: string; type?: string; autocomplete?: string }>(),
  { type: 'text' },
)
</script>

<template>
  <FloatLabel class="ui-field">
    <InputText :type="type" v-model="model" :autocomplete="autocomplete" fluid />
    <label>{{ label }}</label>
  </FloatLabel>
</template>

<style scoped>
.ui-field {
  display: block;
  margin-bottom: 14px;
}
</style>
