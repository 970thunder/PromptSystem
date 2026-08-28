<!-- 文件作用：产品式首页入口。首屏品牌 + 标语 + 单主 CTA；下方为「今日精选」
     大屏展示图（全幅封面 + 前景信息），替代圆角卡片信息流；分类/标签作为真实
     发现入口保留在展示图之下。 -->
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import AppShell from '@/components/layout/AppShell.vue'
import PageLoading from '@/components/feedback/PageLoading.vue'
import PageError from '@/components/feedback/PageError.vue'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'

const promptStore = usePromptStore()
const userStore = useUserStore()
const router = useRouter()

// 大屏展示图取真实信息流第一条；无数据时回退到分类/标签入口，不伪造内容。
const featured = computed(() => promptStore.prompts[0] ?? null)
const fallbackPrompts = computed(() => promptStore.prompts.slice(1, 4))

const featuredCover = computed(() =>
  featured.value && isDisplayableCover(featured.value.cover)
    ? resolveMediaUrl(featured.value.cover)
    : '',
)

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
        <!-- 首屏大屏：用全幅区块 + 分割线构图，无圆角边框包裹 -->
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

        <hr
          class="home-divider"
          aria-hidden="true"
        >

        <!-- 大屏展示图：今日精选 -->
        <section
          class="home-showcase"
          aria-labelledby="home-showcase-title"
        >
          <div class="home-showcase__head">
            <div>
              <p class="section-eyebrow">
                社区精选
              </p>
              <h2
                id="home-showcase-title"
                class="home-showcase__title"
              >
                今日精选
              </h2>
            </div>
            <RouterLink
              to="/search"
              class="home-showcase__more"
            >
              查看更多
            </RouterLink>
          </div>

          <PageLoading
            v-if="promptStore.loading && !featured"
            label="正在加载精选内容"
            variant="blocks"
            class="home-showcase__state"
          />

          <!-- 无精选数据时的回退：真实横向预览，不伪造统计 -->
          <div
            v-else-if="fallbackPrompts.length"
            class="home-showcase__fallback-row"
          >
            <RouterLink
              v-for="prompt in fallbackPrompts"
              :key="prompt.id"
              :to="`/prompt/${prompt.id}`"
              class="home-showcase__mini"
            >
              {{ prompt.title }}
            </RouterLink>
          </div>

          <RouterLink
            v-else-if="featured"
            :to="`/prompt/${featured.id}`"
            class="home-showcase__card"
            :style="featuredCover ? { backgroundImage: `url(${featuredCover})` } : undefined"
          >
            <div
              v-if="!featuredCover"
              class="home-showcase__fallback"
              aria-hidden="true"
            />
            <div class="home-showcase__scrim" />
            <div class="home-showcase__content">
              <p class="home-showcase__tag">
                {{ featured.categoryName || '社区推荐' }}
                <span
                  v-if="featured.tags.length"
                  class="home-showcase__tag-sep"
                >·</span>
                <span v-if="featured.tags.length">{{ featured.tags[0] }}</span>
              </p>
              <h3 class="home-showcase__name">
                {{ featured.title }}
              </h3>
              <p class="home-showcase__desc">
                {{ featured.description || '复制即用的 AI 提示词。' }}
              </p>
              <div class="home-showcase__meta">
                <span class="home-showcase__author">
                  {{ featured.user?.username ?? '作者' }}
                </span>
                <span>{{ featured.model }}</span>
                <span>{{ featured.likes }} 点赞</span>
                <span>{{ featured.views }} 浏览</span>
              </div>
              <span class="home-showcase__cta">
                查看详情 →
              </span>
            </div>
          </RouterLink>

          <PageError
            v-else
            kind="empty"
            title="暂时没有可展示的提示词"
            description="去发现页浏览，或者发布第一条可复用内容。"
            action-label="去搜索"
            @action="router.push('/search')"
          />
        </section>

        <!-- 真实发现入口：分类与标签 -->
        <hr
          class="home-divider"
          aria-hidden="true"
        >
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

/* 首屏大屏：全幅区块，无圆角边框，无 background 包裹；下方用分割线分隔 */
.home-hero {
  @apply relative w-full py-16 sm:py-24;
}

.home-hero__eyebrow {
  @apply text-xs font-semibold uppercase tracking-[0.22em] text-[var(--prompt-text-faint)];
}

.home-hero__title {
  @apply mt-4 text-4xl font-semibold leading-tight sm:text-6xl text-[var(--prompt-text)];
}

.home-hero__desc {
  @apply mt-6 max-w-2xl text-base leading-7 text-[var(--prompt-text-muted)] sm:text-lg;
}

.home-hero__actions {
  @apply mt-10 flex flex-wrap items-center gap-4;
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

/* 分割线：贯穿页面宽度的横向分隔 */
.home-divider {
  @apply my-12 border-0 border-t border-[var(--prompt-border)];
}

/* 大屏展示图 */
.home-showcase {
  @apply mt-2;
}

.home-showcase__head {
  @apply mb-6 flex items-end justify-between gap-4;
}

.home-showcase__title {
  @apply mt-1 text-xl font-semibold sm:text-2xl text-[var(--prompt-text)];
}

.home-showcase__more {
  @apply shrink-0 text-sm text-[var(--prompt-text-muted)] underline-offset-4 hover:text-[var(--prompt-text)] hover:underline;
}

.home-showcase__state {
  @apply min-h-[360px];
}

.home-showcase__card {
  @apply relative block overflow-hidden;
  min-height: 460px;
  background-color: var(--prompt-surface-muted);
  background-size: cover;
  background-position: center;
}

.home-showcase__fallback {
  @apply absolute inset-0;
  background:
    linear-gradient(135deg, rgba(17, 17, 17, 0.06), rgba(37, 99, 235, 0.1)),
    repeating-linear-gradient(45deg, transparent 0 18px, rgba(17, 17, 17, 0.04) 18px 19px);
}

.home-showcase__scrim {
  @apply absolute inset-0;
  background: linear-gradient(180deg, rgba(17, 17, 17, 0.18) 0%, rgba(17, 17, 17, 0.72) 100%);
}

.home-showcase__content {
  @apply relative flex min-h-[420px] flex-col items-start justify-end p-6 sm:p-10;
}

.home-showcase__tag {
  @apply text-xs font-semibold uppercase tracking-[0.18em] text-white/90;
}

.home-showcase__tag-sep {
  @apply mx-1 opacity-60;
}

.home-showcase__name {
  @apply mt-3 max-w-2xl text-3xl font-semibold leading-tight text-white sm:text-4xl;
  text-shadow: 0 2px 12px rgba(0, 0, 0, 0.3);
}

.home-showcase__desc {
  @apply mt-3 max-w-xl text-base leading-7 text-white/85;
}

.home-showcase__meta {
  @apply mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-white/75;
}

.home-showcase__author {
  @apply font-medium text-white;
}

.home-showcase__cta {
  @apply mt-6 inline-flex min-h-[44px] items-center rounded-full bg-white px-5 py-2.5 text-sm font-semibold text-[#111111];
}

.home-showcase__fallback-row {
  @apply grid gap-3 sm:grid-cols-3;
}

.home-showcase__mini {
  @apply px-4 py-6 text-sm text-[var(--prompt-text-muted)] transition hover:text-[var(--prompt-text)];
}

/* 发现入口 */
.home-discover {
  @apply mt-2 grid gap-3;
}

.home-discover__row {
  @apply flex flex-wrap gap-x-6 gap-y-2;
}

.home-chip {
  @apply text-sm text-[var(--prompt-text-muted)] transition hover:text-[var(--prompt-text)];
}

.home-chip--tag {
  @apply text-[var(--prompt-text-faint)];
}

.home-demo {
  @apply mt-8 text-sm text-[var(--prompt-text-faint)];
}

@media (prefers-reduced-motion: reduce) {
  .home-hero__primary,
  .home-hero__secondary,
  .home-chip,
  .home-showcase__more {
    transition: none;
  }
}
</style>
