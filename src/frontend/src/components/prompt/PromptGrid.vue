<!-- 文件作用：稳定的 Prompt 卡片网格，统一加载/空/到底态与「加载更多」。
     封面缺失、作者、模型、统计、键盘可达与 hover 由 PromptCard 负责。 -->
<script setup lang="ts">
import type { Prompt } from '@/types'
import PromptCard from './PromptCard.vue'
import PageLoading from '@/components/feedback/PageLoading.vue'

withDefaults(
  defineProps<{
    prompts: Prompt[]
    loading?: boolean
    hasMore?: boolean
    endLabel?: string
    skeletonRows?: number
  }>(),
  {
    loading: false,
    hasMore: false,
    endLabel: '',
    skeletonRows: 6
  }
)

const emit = defineEmits<{
  (e: 'load-more'): void
}>()
</script>

<template>
  <section class="prompt-grid">
    <PageLoading
      v-if="loading && prompts.length === 0"
      variant="grid"
      :rows="skeletonRows"
      label="正在加载提示词"
    />

    <template v-else>
      <slot name="error" />

      <div
        v-if="prompts.length > 0"
        class="prompt-grid__items"
      >
        <PromptCard
          v-for="(prompt, index) in prompts"
          :key="prompt.id"
          :prompt="prompt"
          :index="index"
          variant="result"
          :eager="index < 2"
        />
      </div>

      <slot
        v-else
        name="empty"
      />

      <div
        v-if="prompts.length > 0"
        class="prompt-grid__footer"
      >
        <button
          v-if="hasMore"
          type="button"
          class="prompt-grid__more"
          :disabled="loading"
          :aria-busy="loading"
          @click="emit('load-more')"
        >
          {{ loading ? '加载中...' : '加载更多' }}
        </button>
        <span
          v-else-if="endLabel"
          class="prompt-grid__end"
        >{{ endLabel }}</span>
      </div>
    </template>
  </section>
</template>

<style scoped>
.prompt-grid__items {
  @apply grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5;
}

.prompt-grid__footer {
  @apply flex min-h-[52px] items-center justify-center pb-10 pt-6;
}

.prompt-grid__more {
  @apply rounded-full border border-[var(--prompt-border)] bg-[var(--prompt-primary)] px-6 py-3 text-sm font-medium text-[var(--prompt-primary-contrast)] transition disabled:cursor-wait disabled:opacity-70;
}

.prompt-grid__more:hover:not(:disabled) {
  background-color: var(--prompt-primary-hover);
}

.prompt-grid__end {
  @apply rounded-full border border-[var(--prompt-border)] px-5 py-2 text-sm text-[var(--prompt-text-faint)];
}

@media (prefers-reduced-motion: reduce) {
  .prompt-grid__more {
    transition: none;
  }
}
</style>
