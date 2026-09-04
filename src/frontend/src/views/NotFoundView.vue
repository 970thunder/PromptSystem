<!-- 文件作用：404 兜底页。未匹配路由不再白屏，给出返回首页与搜索两个出路。 -->
<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="notfound-page">
    <div class="notfound-container">
      <p class="notfound-code">
        404
      </p>
      <h1 class="notfound-title">
        这个页面不存在
      </h1>
      <p class="notfound-desc">
        链接可能已失效，或者内容已被下架。
      </p>
      <div class="notfound-actions">
        <RouterLink
          to="/"
          class="notfound-primary"
        >
          返回首页
        </RouterLink>
        <RouterLink
          to="/search"
          class="notfound-secondary"
        >
          去发现页逛逛
        </RouterLink>
      </div>
    </div>
  </div>
</template>

<style scoped>
.notfound-page {
  @apply view-page;
}

.notfound-container {
  @apply view-container flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center;
}

.notfound-code {
  @apply text-6xl font-bold tabular-nums text-[var(--prompt-text-faint)];
}

.notfound-title {
  @apply text-2xl font-semibold text-[var(--prompt-text)];
}

.notfound-desc {
  @apply text-sm text-[var(--prompt-text-muted)];
}

.notfound-actions {
  @apply mt-2 flex flex-wrap items-center justify-center gap-4;
}

.notfound-primary {
  @apply inline-flex min-h-[44px] items-center rounded-full px-6 py-3 text-sm font-medium transition;
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
}

.notfound-primary:hover {
  background-color: var(--prompt-primary-hover);
}

.notfound-secondary {
  @apply inline-flex min-h-[44px] items-center rounded-full px-6 py-3 text-sm text-[var(--prompt-text-muted)] underline underline-offset-4 transition hover:text-[var(--prompt-text)];
}

@media (prefers-reduced-motion: reduce) {
  .notfound-primary,
  .notfound-secondary {
    transition: none;
  }
}
</style>
