<!-- 文件作用：产品式首页。首屏品牌 + 搜索主 CTA + 标题飘带；下方依次为
     「今日精选」大屏展示（1 张大卡 + 2 张横排小卡）、「最新发布」卡片网格（支持
     加载更多）、分类/标签发现入口与发布引导。所有内容均来自实时服务（测试环境除外）。 -->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { ChevronDown, ChevronUp } from 'lucide-vue-next'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import AppShell from '@/components/layout/AppShell.vue'
import PageLoading from '@/components/feedback/PageLoading.vue'
import PageError from '@/components/feedback/PageError.vue'
import PromptGrid from '@/components/prompt/PromptGrid.vue'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'
import { fallbackCoverUrl } from '@/utils/coverFallback'

const promptStore = usePromptStore()
const userStore = useUserStore()
const router = useRouter()

const searchKeyword = ref('')
const categoryExpanded = ref(false)
const collapsedCategoryCount = 14

// 大屏展示取信息流前三条：1 张主卡 + 2 张小卡；最新发布网格展示其余内容。
const featured = computed(() => promptStore.prompts[0] ?? null)
const sideFeatures = computed(() => promptStore.prompts.slice(1, 3))
const latestPrompts = computed(() => promptStore.prompts.slice(3))

const resolveCover = (prompt: { id: number; cover: string }) =>
  isDisplayableCover(prompt.cover) ? resolveMediaUrl(prompt.cover) : fallbackCoverUrl(prompt.id)

const featuredCover = computed(() => (featured.value ? resolveCover(featured.value) : ''))

const shuffle = <T,>(items: T[]) => {
  const shuffled = [...items]
  for (let index = shuffled.length - 1; index > 0; index -= 1) {
    const swapIndex = Math.floor(Math.random() * (index + 1))
    const current = shuffled[index]
    shuffled[index] = shuffled[swapIndex]
    shuffled[swapIndex] = current
  }
  return shuffled
}

// 飘带只从当前已加载提示词的标题中取值，避免展示脱离内容流的文案。
const heroTitles = computed(() => {
  const prompts = promptStore.prompts.filter((prompt) => prompt.title.trim())
  return shuffle(prompts.map((prompt) => ({
    id: prompt.id,
    title: prompt.title.trim(),
    cover: resolveCover(prompt)
  })))
    .slice(0, 8)
})

const heroTitleRows = computed(() => {
  const rows: Array<{ items: Array<{ id: number; title: string; cover: string }>; duration: number }> = []
  const titlePool = heroTitles.value
  if (!titlePool.length) {
    return rows
  }

  // 用已有标题循环铺满首屏，保持每条轨道内容顺序略有变化。
  for (let rowIndex = 0; rowIndex < 9; rowIndex += 1) {
    const items = Array.from({ length: 3 }, (_, itemIndex) =>
      titlePool[(rowIndex * 2 + itemIndex) % titlePool.length],
    )
    rows.push({
      items,
      duration: 24 + ((rowIndex * 7) % 19)
    })
  }
  return rows
})

const displayCategories = computed(() => (
  categoryExpanded.value
    ? promptStore.categories
    : promptStore.categories.slice(0, collapsedCategoryCount)
))
const hasHiddenCategories = computed(() => promptStore.categories.length > collapsedCategoryCount)

const formatCount = (value: number) => {
  if (value >= 10000) {
    return `${(value / 10000).toFixed(1)} 万`
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}k`
  }
  return `${value}`
}

const submitSearch = async () => {
  const keyword = searchKeyword.value.trim()
  await router.push(keyword ? `/search?keyword=${encodeURIComponent(keyword)}` : '/search')
}

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
        <!-- 首屏：标题飘带作为全宽背景层，内容始终位于其上方 -->
        <section
          class="home-hero"
          aria-labelledby="home-hero-title"
        >
          <div
            v-if="heroTitleRows.length"
            class="home-hero__title-barrage"
            aria-hidden="true"
          >
            <div
              v-for="(row, rowIndex) in heroTitleRows"
              :key="`title-row-${rowIndex}`"
              class="home-hero__title-row"
              :class="{ 'home-hero__title-row--reverse': rowIndex % 2 === 1 }"
              :style="`--marquee-duration: ${row.duration}s`"
            >
              <div class="home-hero__title-track">
                <template
                  v-for="pass in 2"
                  :key="pass"
                >
                  <div class="home-hero__title-sequence">
                    <span
                      v-for="prompt in row.items"
                      :key="`${pass}-${prompt.id}`"
                      class="home-hero__title-item"
                    >
                      <img
                        :src="prompt.cover"
                        alt=""
                        class="home-hero__title-cover"
                        loading="lazy"
                        decoding="async"
                      >
                      <span>{{ prompt.title }}</span>
                    </span>
                  </div>
                </template>
              </div>
            </div>
          </div>

          <div class="home-hero__grid">
            <div class="home-hero__content">
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
                发现、拆解并分享可复用的 AI 提示词，从搜索到复制只要一次点击。
              </p>
              <form
                class="home-search"
                role="search"
                aria-label="搜索提示词"
                @submit.prevent="submitSearch"
              >
                <input
                  v-model="searchKeyword"
                  type="search"
                  class="home-search__input"
                  placeholder="搜索提示词、标签或模型…"
                  aria-label="搜索关键词"
                  autocomplete="off"
                >
                <button
                  type="submit"
                  class="home-search__submit"
                >
                  搜索
                </button>
              </form>
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
                  发布你的提示词
                </button>
              </div>
            </div>
          </div>
        </section>

        <hr
          class="home-divider"
          aria-hidden="true"
        >

        <!-- 今日精选：1 张主卡 + 2 张横排小卡 -->
        <section
          class="home-showcase"
          aria-labelledby="home-showcase-title"
        >
          <div class="home-section__head">
            <div>
              <p class="section-eyebrow">
                社区精选
              </p>
              <h2
                id="home-showcase-title"
                class="home-section__title"
              >
                今日精选
              </h2>
            </div>
            <RouterLink
              to="/search"
              class="home-section__more"
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

          <PageError
            v-else-if="promptStore.feedError"
            kind="error"
            title="内容加载失败"
            :description="promptStore.feedError"
            action-label="重新加载"
            :busy="promptStore.loading"
            @action="promptStore.loadHomeFeed()"
          />

          <div
            v-else-if="featured"
            class="home-showcase__grid"
            :class="{ 'home-showcase__grid--with-side': sideFeatures.length > 0 }"
          >
            <RouterLink
              :to="`/prompt/${featured.id}`"
              class="home-feature"
              :class="sideFeatures.length > 0 ? 'lg:col-span-3' : ''"
            >
              <img
                :src="featuredCover"
                alt=""
                class="home-feature__img"
                loading="eager"
                decoding="async"
              >
              <div class="home-feature__scrim" />
              <div class="home-feature__content">
                <p class="home-feature__tag">
                  {{ featured.categoryName || '社区推荐' }}
                  <template v-if="featured.tags.length">
                    <span class="home-feature__tag-sep">·</span>
                    <span>{{ featured.tags[0] }}</span>
                  </template>
                </p>
                <h3 class="home-feature__name">
                  {{ featured.title }}
                </h3>
                <p class="home-feature__desc">
                  {{ featured.description || '复制即用的 AI 提示词。' }}
                </p>
                <div class="home-feature__meta">
                  <span class="home-feature__author">
                    {{ featured.user?.username ?? '作者' }}
                  </span>
                  <span>{{ featured.model }}</span>
                  <span>{{ formatCount(featured.likes) }} 点赞</span>
                  <span>{{ formatCount(featured.views) }} 浏览</span>
                </div>
                <span class="home-feature__cta">
                  查看详情 →
                </span>
              </div>
            </RouterLink>

            <div
              v-if="sideFeatures.length"
              class="home-showcase__side lg:col-span-2"
            >
              <RouterLink
                v-for="prompt in sideFeatures"
                :key="prompt.id"
                :to="`/prompt/${prompt.id}`"
                class="home-side-card"
              >
                <span class="home-side-card__thumb">
                  <img
                    :src="resolveCover(prompt)"
                    alt=""
                    loading="lazy"
                    decoding="async"
                  >
                </span>
                <span class="home-side-card__body">
                  <span class="home-side-card__meta">
                    {{ prompt.categoryName || '未分类' }} · {{ prompt.model }}
                  </span>
                  <span class="home-side-card__title">{{ prompt.title }}</span>
                  <span class="home-side-card__stats">
                    {{ prompt.user?.username ?? '作者' }}
                    <span aria-hidden="true">·</span>
                    {{ formatCount(prompt.likes) }} 点赞
                    <span aria-hidden="true">·</span>
                    {{ formatCount(prompt.views) }} 浏览
                  </span>
                </span>
              </RouterLink>
            </div>
          </div>

          <PageError
            v-else
            kind="empty"
            title="暂时没有可展示的提示词"
            description="去发现页浏览，或者发布第一条可复用内容。"
            action-label="去搜索"
            @action="router.push('/search')"
          />
        </section>

        <hr
          class="home-divider"
          aria-hidden="true"
        >

        <!-- 最新发布：卡片网格 + 加载更多 -->
        <section
          v-if="latestPrompts.length > 0 || promptStore.loading"
          class="home-latest"
          aria-labelledby="home-latest-title"
        >
          <div class="home-section__head">
            <div>
              <p class="section-eyebrow">
                最新发布
              </p>
              <h2
                id="home-latest-title"
                class="home-section__title"
              >
                刚刚更新
              </h2>
            </div>
            <RouterLink
              to="/search?sort=latest"
              class="home-section__more"
            >
              去发现页浏览全部
            </RouterLink>
          </div>

          <PromptGrid
            :prompts="latestPrompts"
            :loading="promptStore.loading"
            :has-more="promptStore.hasMore"
            end-label="已经到底了，去发布一条吧"
            @load-more="promptStore.loadMorePrompts()"
          >
            <template #empty>
              <p class="home-latest__empty">
                精选之外还没有更多内容，
                <RouterLink
                  to="/publish"
                  class="home-latest__empty-link"
                >
                  去发布第一条提示词
                </RouterLink>。
              </p>
            </template>
          </PromptGrid>
        </section>

        <hr
          v-if="promptStore.categories.length > 0"
          class="home-divider"
          aria-hidden="true"
        >

        <!-- 发现入口：分类与标签 -->
        <section
          v-if="promptStore.categories.length > 0"
          class="home-discover"
          aria-labelledby="home-discover-title"
        >
          <div class="home-section__head">
            <div>
              <p class="section-eyebrow">
                按分类浏览
              </p>
              <h2
                id="home-discover-title"
                class="home-section__title"
              >
                探索分类
              </h2>
            </div>
            <RouterLink
              to="/search"
              class="home-section__more"
            >
              全部分类
            </RouterLink>
          </div>

          <div
            id="home-category-list"
            class="home-discover__cats"
          >
            <RouterLink
              v-for="category in displayCategories"
              :key="category.id"
              :to="`/search?categoryId=${category.id}`"
              class="home-cat-chip"
            >
              <span>{{ category.name }}</span>
              <span class="home-cat-chip__count">{{ category.count }}</span>
            </RouterLink>
          </div>

          <button
            v-if="hasHiddenCategories"
            type="button"
            class="home-category-toggle"
            :aria-expanded="categoryExpanded"
            aria-controls="home-category-list"
            :title="categoryExpanded ? '收起分类' : '展开全部分类'"
            @click="categoryExpanded = !categoryExpanded"
          >
            <span>{{ categoryExpanded ? '收起分类' : `展开全部分类（${promptStore.categories.length}）` }}</span>
            <ChevronUp
              v-if="categoryExpanded"
              aria-hidden="true"
            />
            <ChevronDown
              v-else
              aria-hidden="true"
            />
          </button>

          <div
            v-if="promptStore.hotTags.length > 0"
            class="home-discover__tags"
          >
            <RouterLink
              v-for="tag in promptStore.hotTags"
              :key="tag.name"
              :to="`/search?tag=${encodeURIComponent(tag.name)}`"
              class="home-tag-chip"
            >
              #{{ tag.name }}
              <span class="home-tag-chip__count">{{ tag.count }}</span>
            </RouterLink>
          </div>
        </section>

        <!-- 发布引导 -->
        <section
          class="home-publish"
          aria-label="发布提示词"
        >
          <div>
            <h2 class="home-publish__title">
              把你的提示词沉淀为可复用模板
            </h2>
            <p class="home-publish__desc">
              结构化填写基本信息、内容与模型参数，一次发布即可被社区搜索、收藏与引用。
            </p>
          </div>
          <button
            type="button"
            class="home-publish__cta"
            @click="goPublish"
          >
            开始发布 →
          </button>
        </section>
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

/* 分区头部：eyebrow + 标题 + 右侧更多链接 */
.home-section__head {
  @apply mb-6 flex items-end justify-between gap-4;
}

.home-section__title {
  @apply mt-1 text-xl font-semibold text-[var(--prompt-text)] sm:text-2xl;
}

.home-section__more {
  @apply shrink-0 text-sm text-[var(--prompt-text-muted)] underline-offset-4 hover:text-[var(--prompt-text)] hover:underline;
}

/* 首屏 */
.home-hero {
  @apply relative isolate w-full overflow-hidden py-16 sm:py-20 lg:py-24;
}

.home-hero__grid {
  @apply relative z-[1] grid items-center gap-10 lg:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)] lg:gap-16;
}

.home-hero__content {
  @apply relative z-[1];
}

.home-hero__eyebrow {
  @apply text-xs font-semibold uppercase tracking-[0.22em] text-[var(--prompt-text-faint)];
}

.home-hero__title {
  @apply mt-4 text-4xl font-semibold leading-tight text-[var(--prompt-text)] sm:text-6xl;
}

.home-hero__desc {
  @apply mt-6 max-w-2xl text-base leading-7 text-[var(--prompt-text-muted)] sm:text-lg;
}

/* 搜索框：整条胶囊，输入与按钮同高 */
.home-search {
  @apply mt-8 flex h-12 max-w-xl overflow-hidden rounded-full border bg-[var(--prompt-surface)] transition focus-within:border-[var(--prompt-border-strong)];
  border-color: var(--prompt-border);
}

.home-search__input {
  @apply h-full min-w-0 flex-1 bg-transparent pl-5 pr-4 text-sm text-[var(--prompt-text)] outline-none placeholder:text-[var(--prompt-text-faint)];
}

.home-search__submit {
  @apply h-full shrink-0 bg-[var(--prompt-primary)] px-6 text-sm font-medium text-[var(--prompt-primary-contrast)] transition hover:bg-[var(--prompt-primary-hover)];
}

.home-hero__actions {
  @apply mt-6 flex flex-wrap items-center gap-4;
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

/* 标题飘带：标题和封面组成一个 3D 序列，在首屏内容后方流动 */
.home-hero__title-barrage {
  @apply pointer-events-none absolute inset-0 z-0 flex h-full w-screen -translate-x-1/2 flex-col justify-between overflow-hidden py-2 opacity-[0.1];
  left: 50%;
  perspective: 1200px;
  transform-style: preserve-3d;
}

.home-hero__title-row {
  @apply flex w-full overflow-hidden;
  transform: perspective(1200px) rotateX(8deg);
  transform-style: preserve-3d;
}

.home-hero__title-track {
  @apply flex w-max shrink-0 items-center;
  transform-style: preserve-3d;
  will-change: transform;
  animation: home-title-marquee var(--marquee-duration, 34s) linear infinite;
}

.home-hero__title-sequence {
  @apply flex shrink-0 items-center;
}

.home-hero__title-row--reverse .home-hero__title-track {
  animation-direction: reverse;
}

.home-hero__title-item {
  @apply inline-flex shrink-0 items-center gap-4 whitespace-nowrap pr-12 text-2xl font-semibold tracking-normal text-[var(--prompt-text)] sm:text-3xl lg:text-4xl;
  transform: translateZ(18px) rotateZ(-2deg);
  transform-style: preserve-3d;
  text-shadow: 0 2px 0 rgba(17, 17, 17, 0.08);
}

.home-hero__title-cover {
  @apply h-12 w-20 shrink-0 object-cover sm:h-16 sm:w-28;
  border-radius: 5px;
  opacity: 0.82;
  filter: saturate(0.78) contrast(0.92);
  box-shadow: 0 10px 22px rgba(17, 17, 17, 0.16);
  transform: rotateY(-10deg) translateZ(8px);
  transform-style: preserve-3d;
}

@keyframes home-title-marquee {
  from {
    transform: translateX(0);
  }

  to {
    transform: translateX(-50%);
  }
}

/* 分割线 */
.home-divider {
  @apply my-12 border-0 border-t border-[var(--prompt-border)];
}

/* 今日精选 */
.home-showcase__state {
  @apply min-h-[360px];
}

.home-showcase__grid {
  @apply grid gap-4;
}

.home-showcase__grid--with-side {
  @apply lg:grid-cols-5;
}

/* 主卡 */
.home-feature {
  @apply relative block overflow-hidden rounded-[var(--prompt-radius-lg)] border bg-[var(--prompt-surface-muted)];
  border-color: var(--prompt-border);
  box-shadow: var(--prompt-shadow-1);
  min-height: 420px;
}

.home-feature__img {
  @apply absolute inset-0 h-full w-full object-cover transition-transform duration-500;
}

.home-feature:hover .home-feature__img {
  transform: scale(1.03);
}

.home-feature__scrim {
  @apply absolute inset-0;
  background: linear-gradient(180deg, rgba(17, 17, 17, 0.16) 0%, rgba(17, 17, 17, 0.74) 100%);
}

.home-feature__content {
  @apply relative flex min-h-[420px] flex-col items-start justify-end p-6 sm:p-10;
}

.home-feature__tag {
  @apply text-xs font-semibold uppercase tracking-[0.18em] text-white/90;
}

.home-feature__tag-sep {
  @apply mx-1 opacity-60;
}

.home-feature__name {
  @apply mt-3 max-w-2xl text-3xl font-semibold leading-tight text-white sm:text-4xl;
  text-shadow: 0 2px 12px rgba(0, 0, 0, 0.3);
}

.home-feature__desc {
  @apply mt-3 line-clamp-2 max-w-xl text-base leading-7 text-white/85;
}

.home-feature__meta {
  @apply mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-white/75;
}

.home-feature__author {
  @apply font-medium text-white;
}

.home-feature__cta {
  @apply mt-6 inline-flex min-h-[44px] items-center rounded-full bg-white px-5 py-2.5 text-sm font-semibold text-slate-900;
}

/* 小卡：横排媒体卡，均分右侧高度 */
.home-showcase__side {
  @apply grid grid-rows-2 gap-4;
}

.home-side-card {
  @apply flex overflow-hidden rounded-[var(--prompt-radius-lg)] border bg-[var(--prompt-surface)] transition;
  border-color: var(--prompt-border);
  box-shadow: var(--prompt-shadow-1);
}

.home-side-card:hover {
  border-color: var(--prompt-border-strong);
}

.home-side-card__thumb {
  @apply w-32 shrink-0 overflow-hidden bg-[var(--prompt-surface-muted)] sm:w-40;
}

.home-side-card__thumb img {
  @apply h-full w-full object-cover transition-transform duration-500;
}

.home-side-card:hover .home-side-card__thumb img {
  transform: scale(1.05);
}

.home-side-card__body {
  @apply flex min-w-0 flex-1 flex-col justify-center gap-1.5 p-5;
}

.home-side-card__meta {
  @apply text-xs uppercase tracking-[0.14em] text-[var(--prompt-text-faint)];
}

.home-side-card__title {
  @apply line-clamp-2 text-base font-semibold text-[var(--prompt-text)];
}

.home-side-card__stats {
  @apply text-sm text-[var(--prompt-text-faint)];
}

/* 最新发布 */
.home-latest__empty {
  @apply rounded-[var(--prompt-radius-lg)] border border-dashed px-6 py-12 text-center text-sm text-[var(--prompt-text-muted)];
  border-color: var(--prompt-border);
}

.home-latest__empty-link {
  @apply font-medium text-[var(--prompt-text)] underline underline-offset-4;
}

/* 发现入口 */
.home-discover__cats {
  @apply flex flex-wrap gap-3;
}

.home-cat-chip {
  @apply inline-flex items-center gap-2 rounded-full border bg-[var(--prompt-surface)] px-4 py-2.5 text-sm font-medium text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border-strong)] hover:text-[var(--prompt-text)];
  border-color: var(--prompt-border);
}

.home-cat-chip__count {
  @apply text-xs tabular-nums text-[var(--prompt-text-faint)];
}

.home-category-toggle {
  @apply mt-4 inline-flex min-h-[40px] items-center gap-1.5 rounded-full border bg-[var(--prompt-surface-muted)] px-4 py-2 text-sm font-medium text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border-strong)] hover:bg-[var(--prompt-surface)] hover:text-[var(--prompt-primary)];
  border-color: var(--prompt-border);
}

.home-category-toggle svg {
  @apply h-4 w-4;
}

.home-discover__tags {
  @apply mt-4 flex flex-wrap gap-2.5;
}

.home-tag-chip {
  @apply inline-flex items-center gap-1.5 rounded-full border border-transparent bg-[var(--prompt-surface-muted)] px-3.5 py-2 text-sm text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border)] hover:text-[var(--prompt-text)];
}

.home-tag-chip__count {
  @apply text-xs tabular-nums text-[var(--prompt-text-faint)];
}

/* 发布引导 */
.home-publish {
  @apply mt-14 flex flex-col items-start gap-6 rounded-[var(--prompt-radius-lg)] border bg-[var(--prompt-surface)] p-8 sm:flex-row sm:items-center sm:justify-between sm:p-10;
  border-color: var(--prompt-border);
  box-shadow: var(--prompt-shadow-1);
}

.home-publish__title {
  @apply text-2xl font-semibold text-[var(--prompt-text)] sm:text-3xl;
}

.home-publish__desc {
  @apply mt-3 max-w-xl text-base leading-7 text-[var(--prompt-text-muted)];
}

.home-publish__cta {
  @apply inline-flex min-h-[48px] shrink-0 items-center rounded-full bg-[var(--prompt-primary)] px-7 py-3 text-sm font-semibold text-[var(--prompt-primary-contrast)] transition hover:bg-[var(--prompt-primary-hover)];
}

@media (prefers-reduced-motion: reduce) {
  .home-feature__img,
  .home-side-card__thumb img,
  .home-hero__title-track {
    transition: none;
    animation: none;
  }

  .home-feature:hover .home-feature__img,
  .home-side-card:hover .home-side-card__thumb img {
    transform: none;
  }

  .home-hero__primary,
  .home-hero__secondary,
  .home-search,
  .home-search__submit,
  .home-cat-chip,
  .home-category-toggle,
  .home-tag-chip,
  .home-side-card,
  .home-publish__cta,
  .home-section__more {
    transition: none;
  }
}
</style>
