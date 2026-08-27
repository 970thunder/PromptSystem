<!-- 文件作用：统一的 Prompt 卡片，负责封面占位、图片懒加载、断行截断和稳定高度。 -->
<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import type { Prompt } from '@/types'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'

type PromptCardVariant = 'gallery' | 'result' | 'compact'

const props = withDefaults(defineProps<{
  prompt: Prompt
  index?: number
  variant?: PromptCardVariant
  eager?: boolean
  target?: string
}>(), {
  index: 0,
  variant: 'result',
  eager: false,
  target: ''
})

defineSlots<{
  actions?: () => unknown
}>()

const IMAGE_ASPECT = '4 / 3'
const FALLBACK_COVER_URLS = [
  'https://images.unsplash.com/photo-1516321497487-e288fb19713f?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1498050108023-c5249f4df085?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1526379095098-d400fd0bf935?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80'
]

const formatCount = (value: number) => {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}k`
  }
  return `${value}`
}

const coverUrl = computed(() => {
  if (isDisplayableCover(props.prompt.cover)) {
    return resolveMediaUrl(props.prompt.cover)
  }
  return FALLBACK_COVER_URLS[props.index % FALLBACK_COVER_URLS.length]
})

const imageFailed = ref(false)
</script>

<template>
  <article
    class="prompt-card"
    :class="`prompt-card--${variant}`"
  >
    <RouterLink
      :to="target ?? `/prompt/${prompt.id}`"
      class="prompt-card__link"
      :aria-label="`查看提示词：${prompt.title}`"
    >
      <div
        v-if="variant === 'gallery' || variant === 'result'"
        class="prompt-card__cover"
        :style="{ aspectRatio: IMAGE_ASPECT }"
      >
        <img
          v-if="!imageFailed"
          :src="coverUrl"
          :alt="prompt.title"
          class="prompt-card__image"
          :loading="eager ? 'eager' : 'lazy'"
          width="1200"
          height="900"
          decoding="async"
          @error="imageFailed = true"
        >
        <div
          v-else
          class="prompt-card__cover-fallback"
          role="img"
          :aria-label="`${prompt.title} 的封面图片无法显示`"
        >
          <span class="prompt-card__fallback-text">图片暂不可用</span>
        </div>
      </div>

      <div class="prompt-card__body">
        <div class="prompt-card__meta">
          <span>{{ prompt.categoryName }}</span>
          <span>{{ prompt.model }}</span>
        </div>
        <h3
          class="prompt-card__title"
          :title="prompt.title"
        >
          {{ prompt.title }}
        </h3>
        <p
          v-if="variant !== 'compact'"
          class="prompt-card__desc"
        >
          {{ prompt.description }}
        </p>
        <div class="prompt-card__footer">
          <span
            class="prompt-card__author"
            :title="prompt.user.username"
          >
            {{ prompt.user.username }}
          </span>
          <div class="prompt-card__stats">
            <span
              v-if="variant !== 'compact'"
              :title="`${prompt.likes} 赞`"
            >{{ formatCount(prompt.likes) }} 赞</span>
            <span
              v-if="variant === 'gallery'"
              :title="`${prompt.favorites} 收藏`"
            >{{ formatCount(prompt.favorites) }} 收藏</span>
            <span
              :title="`${prompt.views} 浏览`"
            >{{ formatCount(prompt.views) }} 浏览</span>
          </div>
        </div>
      </div>
    </RouterLink>

    <div
      v-if="$slots.actions"
      class="prompt-card__actions"
    >
      <slot name="actions" />
    </div>
  </article>
</template>

<style scoped>
.prompt-card {
  @apply relative overflow-hidden rounded-[24px] border border-[var(--prompt-border)] bg-[var(--prompt-surface)] transition;
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.05);
}

.prompt-card__link {
  @apply flex h-full flex-col;
}

.prompt-card__cover {
  @apply relative w-full overflow-hidden bg-[var(--prompt-surface-muted)];
}

.prompt-card__image {
  @apply h-full w-full object-cover;
}

.prompt-card__cover-fallback {
  @apply flex h-full min-h-[120px] w-full flex-col items-center justify-center bg-[var(--prompt-surface-muted)] text-center;
}

.prompt-card__fallback-text {
  @apply px-4 text-sm text-[var(--prompt-text-faint)];
}

.prompt-card__body {
  @apply flex min-w-0 flex-1 flex-col p-5;
}

.prompt-card__meta {
  @apply flex items-center gap-3 text-xs uppercase tracking-[0.14em] text-[var(--prompt-text-faint)];
}

.prompt-card__title {
  @apply mt-3 line-clamp-2 text-lg font-semibold text-[var(--prompt-text)];
}

.prompt-card__desc {
  @apply mt-2 line-clamp-2 text-sm leading-6 text-[var(--prompt-text-faint)];
}

.prompt-card__footer {
  @apply mt-auto flex items-center justify-between gap-3 pt-4 text-sm text-[var(--prompt-text-faint)];
}

.prompt-card__author {
  @apply min-w-0 truncate;
}

.prompt-card__stats {
  @apply flex shrink-0 items-center gap-3;
}

.prompt-card__actions {
  @apply absolute right-3 top-3 z-10 flex items-center gap-2;
}

@media (prefers-reduced-motion: reduce) {
  .prompt-card {
    transition: none;
  }
}
</style>
