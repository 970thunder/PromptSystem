<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useScrollCategoryBar } from '@/composables/useScrollCategoryBar'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import type { Prompt } from '@/types'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'

const promptStore = usePromptStore()
const userStore = useUserStore()
const router = useRouter()
const activeCategoryId = ref<number | 'all'>('all')
const { visible: categoryBarVisible } = useScrollCategoryBar()

const navItems = [
  { label: '发现', to: '/' },
  { label: '图像', to: '/' },
  { label: '工作流', to: '/search?tab=workflow' },
  { label: '智能体', to: '/search?tab=agent' }
]

const fallbackCoverMap: Record<number, string> = {
  101: 'https://images.unsplash.com/photo-1516321497487-e288fb19713f?auto=format&fit=crop&w=1200&q=80',
  102: 'https://images.unsplash.com/photo-1498050108023-c5249f4df085?auto=format&fit=crop&w=1200&q=80',
  103: 'https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1200&q=80',
  104: 'https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=1200&q=80',
  105: 'https://images.unsplash.com/photo-1526379095098-d400fd0bf935?auto=format&fit=crop&w=1200&q=80',
  106: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80'
}

const cardSizePattern = [
  'gallery-card--large',
  'gallery-card--small',
  'gallery-card--tall',
  'gallery-card--small',
  'gallery-card--wide',
  'gallery-card--small'
]

const categoryBtnClass = (active: boolean) => ({
  'category-btn': true,
  'category-btn--active': active
})

onMounted(() => {
  promptStore.loadHomeFeed()
})

const visiblePrompts = computed(() => {
  if (activeCategoryId.value === 'all') {
    return promptStore.prompts
  }

  return promptStore.prompts.filter((prompt) => prompt.categoryId === activeCategoryId.value)
})

const featuredPrompt = computed(() => visiblePrompts.value[0] ?? promptStore.prompts[0] ?? null)

const curatedPrompts = computed(() =>
  visiblePrompts.value.map((prompt, index) => ({
    ...prompt,
    image: resolveCover(prompt, index),
    cardClass: cardSizePattern[index % cardSizePattern.length]
  }))
)

const communityStats = computed(() => [
  { label: '精选提示词', value: `${promptStore.prompts.length * 24}+` },
  { label: '活跃创作者', value: '320+' },
  { label: '本周收藏', value: '1.8k' }
])

const formatCount = (value: number) => {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}k`
  }

  return `${value}`
}

const resolveCover = (prompt: Prompt, index: number) => {
  if (isDisplayableCover(prompt.cover)) {
    return resolveMediaUrl(prompt.cover)
  }

  return fallbackCoverMap[prompt.id] ?? fallbackCoverMap[101 + (index % Object.keys(fallbackCoverMap).length)]
}

const handlePublishClick = async () => {
  if (!userStore.isLoggedIn) {
    await router.push('/login?redirect=/publish')
    return
  }

  await router.push('/publish')
}

const handleLogout = async () => {
  userStore.logout()
  await router.push('/')
}
</script>

<template>
  <div class="home-page">
    <div class="home-container">
      <header class="home-header">
        <div class="home-header__inner">
          <div class="home-header__brand">
            <RouterLink
              to="/"
              class="home-logo"
            >
              PromptOS
            </RouterLink>
            <div class="home-badge">
              视觉提示词库
            </div>
          </div>

          <nav class="home-nav">
            <RouterLink
              v-for="item in navItems"
              :key="item.label"
              :to="item.to"
              class="home-nav__link"
            >
              {{ item.label }}
            </RouterLink>
          </nav>

          <div class="home-actions">
            <RouterLink
              to="/search"
              class="home-actions__search"
            >
              搜索
            </RouterLink>
            <RouterLink
              v-if="!userStore.isLoggedIn"
              to="/login"
              class="home-actions__link"
            >
              登录
            </RouterLink>
            <RouterLink
              v-else
              to="/profile"
              class="home-actions__link"
            >
              {{ userStore.userInfo?.username ?? '个人主页' }}
            </RouterLink>
            <button
              class="btn-pill-primary"
              @click="handlePublishClick"
            >
              发布
            </button>
            <button
              v-if="userStore.isLoggedIn"
              class="home-actions__link home-actions__link--outline"
              @click="handleLogout"
            >
              退出
            </button>
          </div>
        </div>
      </header>

      <section class="stats-row">
        <div class="stats-grid">
          <article
            v-for="stat in communityStats"
            :key="stat.label"
            class="stat-card"
          >
            <div class="stat-card__label">
              {{ stat.label }}
            </div>
            <div class="stat-card__value">
              {{ stat.value }}
            </div>
          </article>
        </div>
      </section>

      <div
        class="category-bar"
        :class="{ 'category-bar--hidden': !categoryBarVisible }"
      >
        <div class="category-bar__header">
          <span class="category-bar__title">分类</span>
          <span class="category-bar__hint">浏览</span>
        </div>
        <div class="category-bar__scroll">
          <button
            :class="categoryBtnClass(activeCategoryId === 'all')"
            @click="activeCategoryId = 'all'"
          >
            全部
          </button>
          <button
            v-for="category in promptStore.categories"
            :key="category.id"
            :class="categoryBtnClass(activeCategoryId === category.id)"
            @click="activeCategoryId = category.id"
          >
            {{ category.name }} · {{ category.count }}
          </button>
        </div>
      </div>

      <section
        v-if="featuredPrompt"
        class="featured-section"
      >
        <RouterLink
          :to="`/prompt/${featuredPrompt.id}`"
          class="featured-card"
        >
          <img
            :src="resolveCover(featuredPrompt, 0)"
            :alt="featuredPrompt.title"
            class="featured-card__image"
          >
          <div class="featured-card__overlay" />
          <div class="featured-card__content">
            <div class="featured-card__meta">
              <span>{{ featuredPrompt.categoryName }}</span>
              <span>{{ featuredPrompt.model }}</span>
              <span v-if="promptStore.usingMockData">演示数据</span>
            </div>
            <h2 class="featured-card__title">
              {{ featuredPrompt.title }}
            </h2>
            <p class="featured-card__desc">
              {{ featuredPrompt.description }}
            </p>
            <div class="featured-card__stats">
              <span>{{ featuredPrompt.user.username }}</span>
              <span>{{ formatCount(featuredPrompt.likes) }} 赞</span>
              <span>{{ formatCount(featuredPrompt.favorites) }} 收藏</span>
            </div>
          </div>
        </RouterLink>

        <div class="creator-panel">
          <div class="creator-panel__label">
            精选创作者
          </div>
          <div class="creator-panel__name">
            {{ featuredPrompt.user.username }}
          </div>
          <p class="creator-panel__bio">
            {{ featuredPrompt.user.bio || '专注可复用提示词体系，偏向生产落地。' }}
          </p>
          <div class="creator-panel__stats">
            <div class="creator-stat">
              <div class="creator-stat__value">
                {{ formatCount(featuredPrompt.views) }}
              </div>
              <div class="creator-stat__label">
                浏览
              </div>
            </div>
            <div class="creator-stat">
              <div class="creator-stat__value">
                {{ formatCount(featuredPrompt.likes) }}
              </div>
              <div class="creator-stat__label">
                点赞
              </div>
            </div>
            <div class="creator-stat">
              <div class="creator-stat__value">
                {{ featuredPrompt.model }}
              </div>
              <div class="creator-stat__label">
                模型
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="gallery-header">
        <div class="gallery-header__label">
          画廊
        </div>
        <div class="gallery-header__count">
          {{ visiblePrompts.length }} 条结果
        </div>
      </section>

      <section
        v-if="promptStore.loading"
        class="gallery-grid"
      >
        <div
          v-for="index in 6"
          :key="index"
          class="gallery-skeleton"
          :class="cardSizePattern[(index - 1) % cardSizePattern.length]"
        />
      </section>

      <section
        v-else-if="curatedPrompts.length > 0"
        class="gallery-grid"
      >
        <RouterLink
          v-for="prompt in curatedPrompts"
          :key="prompt.id"
          :to="`/prompt/${prompt.id}`"
          class="gallery-card"
          :class="prompt.cardClass"
        >
          <img
            :src="prompt.image"
            :alt="prompt.title"
            class="gallery-card__image"
          >
          <div class="gallery-card__overlay" />
          <div class="gallery-card__content">
            <div class="gallery-card__meta">
              <span>{{ prompt.categoryName }}</span>
              <span>{{ prompt.model }}</span>
            </div>
            <h3 class="gallery-card__title">
              {{ prompt.title }}
            </h3>
            <p class="gallery-card__desc">
              {{ prompt.description }}
            </p>
            <div class="gallery-card__stats">
              <span>{{ prompt.user.username }}</span>
              <span>{{ formatCount(prompt.likes) }} 赞</span>
              <span>{{ formatCount(prompt.views) }} 浏览</span>
            </div>
          </div>
        </RouterLink>
      </section>

      <section
        v-else
        class="empty-state"
      >
        <div class="empty-state__title">
          该分类下还没有提示词
        </div>
        <p class="empty-state__desc">
          切换分类，或发布第一条内容。
        </p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.home-page {
  @apply min-h-screen bg-[#f5f3ee] text-[#111111];
}

.home-container {
  @apply mx-auto max-w-[1160px] px-3 pb-12 pt-4 sm:px-4 lg:px-5;
}

.home-header {
  @apply sticky top-3 z-30 rounded-[20px] border border-black/10 bg-white/90 px-4 py-3 shadow-[0_14px_30px_rgba(15,23,42,0.06)] backdrop-blur md:px-5;
}

.home-header__inner {
  @apply flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between;
}

.home-header__brand {
  @apply flex items-center justify-between gap-4;
}

.home-logo {
  @apply text-lg font-semibold tracking-[0.08em] text-[#111111];
}

.home-badge {
  @apply hidden rounded-full border border-black/10 bg-black px-3 py-1 text-xs text-white sm:inline-flex;
}

.home-nav {
  @apply flex flex-wrap items-center gap-2;
}

.home-nav__link {
  @apply rounded-full px-3 py-2 text-sm text-[#555555] transition hover:bg-black hover:text-white;
}

.home-actions {
  @apply flex flex-wrap items-center justify-end gap-2;
}

.home-actions__search {
  @apply rounded-full border border-black/10 bg-[#f7f5f0] px-4 py-2 text-sm text-[#555555] transition hover:border-black/20 hover:text-black;
}

.home-actions__link {
  @apply rounded-full px-4 py-2 text-sm text-[#555555] transition hover:bg-black hover:text-white;
}

.home-actions__link--outline {
  @apply border border-black/10;
}

.stats-row {
  @apply pb-5 pt-5;
}

.stats-grid {
  @apply grid gap-3 sm:grid-cols-3;
}

.stat-card__label {
  @apply text-sm text-[#777777];
}

.stat-card__value {
  @apply mt-1.5 text-2xl font-semibold text-black;
}

.category-bar {
  @apply sticky top-[72px] z-20 mb-6 rounded-[20px] border border-black/10 bg-white px-4 py-3 shadow-[0_8px_24px_rgba(15,23,42,0.04)];
  transform: translateY(0);
  opacity: 1;
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.category-bar--hidden {
  transform: translateY(calc(-100% - 12px));
  opacity: 0;
  pointer-events: none;
}

.category-bar__header {
  @apply mb-3 flex items-center justify-between gap-3;
}

.category-bar__title {
  @apply text-sm text-[#777777];
}

.category-bar__hint {
  @apply text-xs uppercase tracking-[0.2em] text-[#999999];
}

.category-bar__scroll {
  @apply flex gap-2 overflow-x-auto pb-1;
  scrollbar-width: none;
}

.category-bar__scroll::-webkit-scrollbar {
  display: none;
}

.category-btn {
  @apply shrink-0 rounded-full px-3.5 py-2 text-sm transition;
  @apply border border-black/10 bg-[#f6f4ef] text-[#555555] hover:border-black/20 hover:text-black;
}

.category-btn--active {
  @apply border-transparent bg-black text-white;
}

.featured-section {
  @apply grid gap-3 pb-6 lg:grid-cols-[1.25fr_0.75fr];
}

.featured-card {
  @apply relative block h-[470px] overflow-hidden rounded-[24px] bg-black;
}

.featured-card__image {
  @apply h-full w-full max-h-full max-w-full object-cover transition duration-500;
}

.featured-card:hover .featured-card__image {
  @apply scale-[1.02];
}

.featured-card__overlay {
  @apply absolute inset-0 bg-gradient-to-t from-black via-black/20 to-transparent;
}

.featured-card__content {
  @apply absolute inset-x-0 bottom-0 p-5 text-white sm:p-6;
}

.featured-card__meta {
  @apply flex flex-wrap items-center gap-2 text-xs uppercase tracking-[0.2em] text-white/70;
}

.featured-card__title {
  @apply mt-2 max-w-2xl text-2xl font-semibold sm:text-[30px];
}

.featured-card__desc {
  @apply mt-2 max-w-2xl text-sm leading-6 text-white/70;
}

.featured-card__stats {
  @apply mt-4 flex flex-wrap items-center gap-4 text-sm text-white/70;
}

.creator-panel {
  @apply rounded-[24px] border border-black/10 bg-[#111111] p-5 text-white;
}

.creator-panel__label {
  @apply text-sm text-white/60;
}

.creator-panel__name {
  @apply mt-2 text-xl font-semibold;
}

.creator-panel__bio {
  @apply mt-3 text-sm leading-6 text-white/70;
}

.creator-panel__stats {
  @apply mt-5 grid grid-cols-3 gap-2 text-center;
}

.creator-stat {
  @apply rounded-[16px] bg-white/10 px-3 py-3;
}

.creator-stat__value {
  @apply text-lg font-semibold;
}

.creator-stat__label {
  @apply mt-1 text-xs text-white/60;
}

.gallery-header {
  @apply flex items-end justify-between gap-3 pb-3;
}

.gallery-header__label,
.gallery-header__count {
  @apply text-sm text-[#777777];
}

.gallery-grid {
  @apply grid auto-rows-[168px] grid-cols-1 gap-3 pb-8 md:grid-cols-3;
}

.gallery-skeleton {
  @apply animate-pulse rounded-[20px] bg-black/5;
}

.gallery-card {
  @apply relative h-full min-h-0 overflow-hidden rounded-[20px] bg-black;
}

.gallery-card--large {
  @apply md:col-span-2 md:row-span-2;
}

.gallery-card--small {
  @apply md:col-span-1 md:row-span-1;
}

.gallery-card--tall {
  @apply md:col-span-1 md:row-span-2;
}

.gallery-card--wide {
  @apply md:col-span-2 md:row-span-1;
}

.gallery-card__image {
  @apply absolute inset-0 h-full w-full max-h-full max-w-full object-cover transition duration-500;
}

.gallery-card:hover .gallery-card__image {
  @apply scale-[1.03];
}

.gallery-card__overlay {
  @apply absolute inset-0 bg-gradient-to-t from-black/90 via-black/20 to-transparent;
}

.gallery-card__content {
  @apply absolute inset-x-0 bottom-0 p-4 text-white;
}

.gallery-card__meta {
  @apply flex items-center justify-between gap-3 text-xs uppercase tracking-[0.16em] text-white/70;
}

.gallery-card__title {
  @apply mt-2 text-lg font-semibold leading-tight;
}

.gallery-card__desc {
  @apply mt-2 line-clamp-2 text-sm leading-5 text-white/70;
}

.gallery-card__stats {
  @apply mt-3 flex flex-wrap items-center gap-3 text-sm text-white/70;
}

.empty-state__title {
  @apply text-lg font-semibold text-black;
}

.empty-state__desc {
  @apply mt-2 text-sm text-[#777777];
}
</style>
