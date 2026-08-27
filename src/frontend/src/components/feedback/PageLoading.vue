<!-- 文件作用：统一的页面加载占位，避免无限旋转动画，并为减少动效偏好提供降级。 -->
<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  rows?: number
  label?: string
  variant?: 'grid' | 'blocks' | 'list'
}>(), {
  rows: 6,
  label: '正在加载',
  variant: 'grid'
})

const skeletonItems = computed(() => {
  const count = Math.max(1, props.rows)
  return Array.from({ length: count }, (_, index) => index + 1)
})
</script>

<template>
  <section
    class="page-loading"
    :class="`page-loading--${variant}`"
    role="status"
    aria-live="polite"
  >
    <span class="page-loading__sr-label">{{ label }}...</span>
    <div
      v-for="item in skeletonItems"
      :key="item"
      class="page-loading__item"
      :style="{ animationDelay: `${(item % 6) * 60}ms` }"
    />
  </section>
</template>

<style scoped>
.page-loading {
  @apply mt-6 grid gap-4;
}

.page-loading--grid {
  @apply md:grid-cols-2 xl:grid-cols-3;
}

.page-loading--list {
  @apply grid-cols-1;
}

.page-loading--blocks {
  @apply grid-cols-1 lg:grid-cols-[1.35fr_0.65fr];
}

.page-loading__sr-label {
  @apply sr-only;
}

.page-loading__item {
  @apply h-[280px] animate-pulse rounded-[20px] bg-[var(--prompt-surface-muted)];
}

.page-loading--list .page-loading__item {
  @apply h-[150px] rounded-[24px];
}

.page-loading--blocks .page-loading__item {
  @apply h-[420px] rounded-[28px];
}

@media (prefers-reduced-motion: reduce) {
  .page-loading__item {
    animation: none;
  }
}
</style>
