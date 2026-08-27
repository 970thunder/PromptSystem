<!-- 文件作用：产品式首页入口。首屏只有品牌、标语、一个主 CTA 与弱化发布入口；
     底部露出真实信息流预览，分类/标签作为真实发现入口。 -->
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import AppShell from '@/components/layout/AppShell.vue'
import PromptGrid from '@/components/prompt/PromptGrid.vue'
import PageError from '@/components/feedback/PageError.vue'

const promptStore = usePromptStore()
const userStore = useUserStore()
const router = useRouter()

const feed = computed(() => promptStore.prompts.slice(0, 8))
const hasMore = computed(() => promptStore.hasMore)

const goPublish = async () => {
  if (!userStore.isLoggedIn) {
    await router.push('/login?redirect=/publish')
    return
  }
  await router.push('/publish')
}

onMounted(() => {
  void promptStore.loadHomeFeed()
})
</script>

<template>
  <AppShell>
    <div class="home-page">
      <div class="home-container">
        <!-- 首屏：产品式入口 -->
        <section
          class="home-hero"
          aria-labelledby="home-hero-title"
        >
          <p class="home-hero__eyebrow">
            AI 提示词社区
          </p>
          <h1
            id="home-hero-title"
            class="home-hero__title"
          >
            好提示词，马上复用。
          </h1>
          <p class="home-hero__desc">
            发现、拆解并分享可复用的 AI 提示词。
          </p>
          <div class="home-hero__actions">
            <RouterLink
              to="/search"
              class="home-hero__primary"
            >
              开始探索
            </RouterLink>
            <button
              type="button"
              class="home-hero__secondary"
              @click="goPublish"
            >
              或发布你自己的提示词
            </button>
          </div>
        </section>

        <!-- 真实发现入口：分类与标签 -->
        <section
          v-if="promptStore.categories.length > 0"
          class="home-discover"
          aria-label="按分类浏览"
        >
          <div class="home-discover__row">
            <RouterLink
              v-for="category in promptStore.categories"
              :key="category.id"
              :to="`/search?categoryId=${category.id}`"
              class="home-chip"
            >
              {{ category.name }}
            </RouterLink>
          </div>
          <div
            v-if="promptStore.hotTags.length > 0"
            class="home-discover__row"
          >
            <RouterLink
              v-for="tag in promptStore.hotTags"
              :key="tag.name"
              :to="`/search?tag=${encodeURIComponent(tag.name)}`"
              class="home-chip home-chip--tag"
            >
              #{{ tag.name }}
            </RouterLink>
          </div>
        </section>

        <!-- 真实信息流预览 -->
        <section
          class="home-feed"
          aria-labelledby="home-feed-title"
        >
          <div class="home-feed__head">
            <div>
              <p class="section-eyebrow">
                社区精选
              </p>
              <h2
                id="home-feed-title"
                class="home-feed__title"
              >
                最新提示词
              </h2>
            </div>
            <RouterLink
              to="/search"
              class="home-feed__more"
            >
              查看更多
            </RouterLink>
          </div>

          <PromptGrid
            :prompts="feed"
            :loading="promptStore.loading"
            :has-more="hasMore"
            end-label="已展示精选内容"
            @load-more="promptStore.loadMorePrompts()"
          >
            <template #empty>
              <PageError
                kind="empty"
                title="暂时没有可展示的提示词"
                description="清除筛选后继续浏览，或者发布第一条可复用内容。"
                action-label="去搜索"
                @action="router.push('/search')"
              />
            </template>
          </PromptGrid>
        </section>

        <div
          v-if="promptStore.usingMockData"
          class="home-demo"
          role="status"
        >
          <span>当前为演示数据 / 离线预览，未连接实时服务。</span>
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.home-page {
  @apply view-page;
}

.home-container {
  @apply view-container--home;
}

/* 首屏：单一视觉焦点，充足留白 */
.home-hero {
  @apply flex flex-col items-start rounded-[28px] border border-[var(--prompt-border)] bg-[var(--prompt-surface)] px-6 py-12 sm:px-10 sm:py-16;
  box-shadow: var(--prompt-shadow-1);
}

.home-hero__eyebrow {
  @apply text-xs font-semibold uppercase tracking-[0.22em] text-[var(--prompt-text-faint)];
}

.home-hero__title {
  @apply mt-3 text-4xl font-semibold leading-tight sm:text-5xl text-[var(--prompt-text)];
}

.home-hero__desc {
  @apply mt-4 max-w-xl text-base leading-7 text-[var(--prompt-text-muted)];
}

.home-hero__actions {
  @apply mt-7 flex flex-wrap items-center gap-4;
}

.home-hero__primary,
.home-hero__secondary {
  @apply inline-flex min-h-[44px] items-center justify-center rounded-full px-6 py-3 text-sm font-medium transition;
}

.home-hero__primary {
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
}

.home-hero__primary:hover {
  background-color: var(--prompt-primary-hover);
}

.home-hero__secondary {
  color: var(--prompt-text-muted);
  text-decoration: underline;
  text-underline-offset: 4px;
  text-decoration-thickness: 1px;
}

.home-hero__secondary:hover {
  color: var(--prompt-text);
}

/* 发现入口 */
.home-discover {
  @apply mt-8 grid gap-3;
}

.home-discover__row {
  @apply flex flex-wrap gap-2;
}

.home-chip {
  @apply rounded-full border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] px-3.5 py-2 text-sm text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border-strong)] hover:text-[var(--prompt-text)];
}

.home-chip--tag {
  border-color: transparent;
  background-color: var(--prompt-surface);
}

/* 信息流 */
.home-feed {
  @apply mt-10;
}

.home-feed__head {
  @apply mb-4 flex items-end justify-between gap-4;
}

.home-feed__title {
  @apply mt-1 text-xl font-semibold sm:text-2xl text-[var(--prompt-text)];
}

.home-feed__more {
  @apply shrink-0 rounded-full border border-[var(--prompt-border)] bg-[var(--prompt-surface)] px-4 py-2 text-sm text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border-strong)] hover:text-[var(--prompt-text)];
}

.home-demo {
  @apply mt-8 rounded-[16px] border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] px-4 py-3 text-sm text-[var(--prompt-text-muted)];
}

@media (prefers-reduced-motion: reduce) {
  .home-hero__primary,
  .home-hero__secondary,
  .home-chip,
  .home-feed__more {
    transition: none;
  }
}
</style>
