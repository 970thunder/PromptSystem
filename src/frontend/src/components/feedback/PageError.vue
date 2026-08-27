<!-- 文件作用：统一的错误/空状态提示，提供下一步操作与手动重试入口。 -->
<script setup lang="ts">
withDefaults(defineProps<{
  kind?: 'error' | 'empty'
  title: string
  description?: string
  actionLabel?: string
  secondaryLabel?: string
  busy?: boolean
}>(), {
  kind: 'error',
  description: '',
  actionLabel: '',
  secondaryLabel: '',
  busy: false
})

const emit = defineEmits<{
  (e: 'action'): void
  (e: 'secondary'): void
}>()
</script>

<template>
  <section
    class="page-feedback"
    :class="`page-feedback--${kind}`"
    :role="kind === 'error' ? 'alert' : undefined"
    aria-live="polite"
  >
    <div class="page-feedback__title">
      {{ title }}
    </div>
    <p
      v-if="description"
      class="page-feedback__desc"
    >
      {{ description }}
    </p>
    <div
      v-if="actionLabel || secondaryLabel"
      class="page-feedback__actions"
    >
      <button
        v-if="secondaryLabel"
        type="button"
        class="page-feedback__btn page-feedback__btn--secondary"
        :disabled="busy"
        @click="emit('secondary')"
      >
        {{ secondaryLabel }}
      </button>
      <button
        v-if="actionLabel"
        type="button"
        class="page-feedback__btn page-feedback__btn--primary"
        :disabled="busy"
        :aria-busy="busy"
        @click="emit('action')"
      >
        {{ busy ? '处理中...' : actionLabel }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.page-feedback {
  @apply rounded-[24px] border border-dashed border-[var(--prompt-border)] bg-[var(--prompt-surface)] px-6 py-14 text-center;
}

.page-feedback--error {
  @apply border-amber-200 bg-amber-50;
}

.page-feedback__title {
  @apply text-lg font-semibold text-[var(--prompt-text)];
}

.page-feedback--error .page-feedback__title {
  @apply text-amber-900;
}

.page-feedback__desc {
  @apply mx-auto mt-2 max-w-md text-sm leading-6 text-[var(--prompt-text-faint)];
}

.page-feedback__actions {
  @apply mt-5 flex flex-wrap items-center justify-center gap-2;
}

.page-feedback__btn {
  @apply inline-flex min-h-10 items-center justify-center rounded-full px-4 py-2 text-sm transition disabled:cursor-wait disabled:opacity-60;
}

.page-feedback__btn--secondary {
  @apply border border-[var(--prompt-border)] bg-[var(--prompt-surface)] text-[var(--prompt-text-muted)] hover:border-[var(--prompt-border-strong)] hover:text-[var(--prompt-text)];
}

.page-feedback__btn--primary {
  @apply border-transparent bg-[var(--prompt-primary)] font-medium text-[var(--prompt-primary-contrast)] hover:bg-[var(--prompt-primary-hover)];
}

@media (prefers-reduced-motion: reduce) {
  .page-feedback__btn {
    transition: none;
  }
}
</style>
