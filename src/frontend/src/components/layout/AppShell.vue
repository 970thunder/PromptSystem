<!-- 文件作用：页面外壳，提供全局站点导航、内容插槽与页脚。 -->
<script setup lang="ts">
import { RouterLink } from 'vue-router'
import SiteHeader from '@/components/navigation/SiteHeader.vue'
</script>

<template>
  <div
    class="app-shell"
    :style="{ backgroundColor: 'var(--prompt-bg)', color: 'var(--prompt-text)' }"
  >
    <SiteHeader />
    <main class="app-shell__main">
      <slot />
    </main>

    <footer class="app-footer">
      <div class="app-footer__inner">
        <div class="app-footer__brand">
          <span class="app-footer__logo">PromptOS</span>
          <span class="app-footer__tagline">可复用的 AI 提示词社区</span>
        </div>
        <nav
          class="app-footer__nav"
          aria-label="页脚导航"
        >
          <RouterLink
            to="/search"
            class="app-footer__link"
          >
            发现
          </RouterLink>
          <RouterLink
            to="/community"
            class="app-footer__link"
          >
            社区
          </RouterLink>
          <RouterLink
            to="/publish"
            class="app-footer__link"
          >
            发布
          </RouterLink>
        </nav>
        <p class="app-footer__copy">
          © 2026 PromptOS
        </p>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.app-shell {
  @apply flex min-h-screen flex-col;
}

.app-shell__main {
  @apply flex-1;
}

.app-footer {
  @apply border-t;
  border-color: var(--prompt-border);
}

.app-footer__inner {
  @apply mx-auto flex w-full flex-col gap-4 px-4 py-8 sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8 xl:max-w-[95%] xl:px-10 2xl:max-w-[96vw];
}

.app-footer__brand {
  @apply flex items-baseline gap-3;
}

.app-footer__logo {
  @apply text-base font-bold tracking-[0.02em] text-[var(--prompt-text)];
}

.app-footer__tagline {
  @apply text-sm text-[var(--prompt-text-faint)];
}

.app-footer__nav {
  @apply flex items-center gap-5;
}

.app-footer__link {
  @apply text-sm text-[var(--prompt-text-muted)] transition-colors hover:text-[var(--prompt-text)];
}

.app-footer__copy {
  @apply text-sm text-[var(--prompt-text-faint)];
}

@media (prefers-reduced-motion: reduce) {
  .app-footer__link {
    transition: none;
  }
}
</style>
