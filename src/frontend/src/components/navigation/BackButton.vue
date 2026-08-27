<script setup lang="ts">
import { useBackNavigation } from '@/composables/useBackNavigation'

const props = withDefaults(defineProps<{
  fallback?: string
  label?: string
  ariaLabel?: string
}>(), {
  fallback: '/',
  label: '返回',
  ariaLabel: ''
})

const { goBack } = useBackNavigation(props.fallback)
</script>

<template>
  <button
    type="button"
    class="back-button btn-pill-secondary bg-[var(--prompt-surface)]"
    :aria-label="props.ariaLabel || props.label"
    @click="goBack"
  >
    <span aria-hidden="true">←</span>
    {{ props.label }}
  </button>
</template>

<style scoped>
.back-button {
  @apply inline-flex items-center gap-2;
}
</style>
