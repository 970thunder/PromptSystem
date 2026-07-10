<!-- 文件作用：展示 PromptOS 首页瀑布流、分类筛选和 Phase 2 Skill 入口占位。 -->
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
const activeTag = ref<string>('')
const userMenuOpen = ref(false)
const mobileNavOpen = ref(false)
const { visible: categoryBarVisible } = useScrollCategoryBar()

const navItems = [
  { label: '发现', to: '/' },
  { label: '图像', to: '/' },
  { label: '工作流', to: '/search?tab=workflow' },
  { label: '智能体', to: '/search?tab=agent' }
]

const skillCategoryPlaceholders = [
  { name: '内容创作', desc: 'Prompt 编排与发布流程', icon: 'Content' },
  { name: '电商运营', desc: '商品文案、客服与活动 SOP', icon: 'Ops' },
  { name: '数据分析', desc: '分析任务拆解与报告模板', icon: 'Data' },
  { name: '研发自动化', desc: '代码审查、测试与发布清单', icon: 'Dev' }
] as const

const fallbackCoverMap: Record<number, string> = {
  101: 'https://images.unsplash.com/photo-1516321497487-e288fb19713f?auto=format&fit=crop&w=1200&q=80',
  102: 'https://images.unsplash.com/photo-1498050108023-c5249f4df085?auto=format&fit=crop&w=1200&q=80',
  103: 'https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1200&q=80',
  104: 'https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=1200&q=80',
  105: 'https://images.unsplash.com/photo-1526379095098-d400fd0bf935?auto=format&fit=crop&w=1200&q=80',
  106: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80'
}

const cardSizePattern = ['gallery-card--large', 'gallery-card--medium', 'gallery-card--tall', 'gallery-card--small']

const categoryBtnClass = (active: boolean) => ({
  'category-btn': true,
  'category-btn--active': active
})

onMounted(() => {
  promptStore.loadHomeFeed()
})

const handleCategoryChange = async (categoryId: number | 'all') => {
  activeCategoryId.value = categoryId
  activeTag.value = ''
  await promptStore.loadHomeFeed(categoryId === 'all' ? undefined : categoryId)
}

const visiblePrompts = computed(() => {
  return promptStore.prompts
})

const featuredPrompt = computed(() => visiblePrompts.value[0] ?? promptStore.prompts[0] ?? null)

const activityCards = computed(() => {
  const [first, second, third] = visiblePrompts.value
  return [
    {
      id: 'weekly-visual',
      title: first?.title ?? '本周视觉 Prompt 精选',
      desc: first?.description ?? '精选高完成度图像案例，适合直接拆解参数和镜头语言。',
      meta: 'Weekly Pick',
      image: first ? resolveCover(first, 0) : fallbackCoverMap[101],
      to: first ? `/prompt/${first.id}` : '/search'
    },
    {
      id: 'creator-lab',
      title: second?.title ?? '创作者工作台开放',
      desc: second?.description ?? '从灵感、草稿到发布，把可复用 Prompt 沉淀成个人作品集。',
      meta: 'Creator Lab',
      image: second ? resolveCover(second, 1) : fallbackCoverMap[102],
      to: second ? `/prompt/${second.id}` : '/publish'
    },
    {
      id: 'model-focus',
      title: third?.title ?? '模型参数灵感包',
      desc: third?.description ?? '收藏高质量案例，快速找到适配不同模型的参数组合。',
      meta: 'Model Focus',
      image: third ? resolveCover(third, 2) : fallbackCoverMap[103],
      to: third ? `/prompt/${third.id}` : '/search'
    }
  ]
})

const featuredRailPrompts = computed(() => visiblePrompts.value.slice(0, 8))

const curatedPrompts = computed(() =>
  visiblePrompts.value.map((prompt, index) => ({
    ...prompt,
    image: resolveCover(prompt, index),
    cardClass: cardSizePattern[index % cardSizePattern.length]
  }))
)

const communityStats = computed(() => [
  { label: '精选提示词', value: `${promptStore.total || promptStore.prompts.length}` },
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
  userMenuOpen.value = false
  mobileNavOpen.value = false
  if (!userStore.isLoggedIn) {
    await router.push('/login?redirect=/publish')
    return
  }

  await router.push('/publish')
}

const handleLogout = async () => {
  userMenuOpen.value = false
  mobileNavOpen.value = false
  userStore.logout()
  await router.push('/')
}

const closeMenus = () => {
  userMenuOpen.value = false
  mobileNavOpen.value = false
}

const handleTagChange = async (tag: string) => {
  activeTag.value = activeTag.value === tag ? '' : tag
  await promptStore.loadHomeFeed(
    activeCategoryId.value === 'all' ? undefined : activeCategoryId.value,
    activeTag.value || undefined
  )
}

const clearFilters = async () => {
  activeCategoryId.value = 'all'
  activeTag.value = ''
  await promptStore.loadHomeFeed()
}

const loadMore = async () => {
  await promptStore.loadMorePrompts()
}
</script>

<template>
  <div class="home-page">
    <div class="home-container">
      <header class="home-header">
        <div class="home-header__inner">
          <div class="home-header__brand">
            <div class="home-header__brand-main">
              <RouterLink
                to="/"
                class="home-logo"
                @click="closeMenus"
              >
                PromptOS
              </RouterLink>
              <div class="home-badge">
                视觉提示词库
              </div>
            </div>
            <button
              class="home-mobile-toggle"
              type="button"
              :aria-expanded="mobileNavOpen"
              @click="mobileNavOpen = !mobileNavOpen"
            >
              <span />
              <span />
              <span />
            </button>
          </div>

          <nav class="home-nav">
            <RouterLink
              v-for="item in navItems"
              :key="item.label"
              :to="item.to"
              class="home-nav__link"
              @click="closeMenus"
            >
              {{ item.label }}
            </RouterLink>
          </nav>

          <div class="home-actions">
            <RouterLink
              to="/search"
              class="home-actions__search"
              @click="closeMenus"
            >
              搜索
            </RouterLink>
            <RouterLink
              v-if="!userStore.isLoggedIn"
              to="/login"
              class="home-actions__link"
              @click="closeMenus"
            >
              登录
            </RouterLink>
            <div
              v-else
              class="home-user-menu"
            >
              <button
                class="home-user-menu__trigger"
                type="button"
                @click="userMenuOpen = !userMenuOpen"
              >
                <span class="home-user-menu__avatar">
                  {{ userStore.userInfo?.username?.slice(0, 1) ?? 'U' }}
                </span>
                <span>{{ userStore.userInfo?.username ?? '个人主页' }}</span>
              </button>
              <div
                v-if="userMenuOpen"
                class="home-user-menu__panel"
              >
                <RouterLink
                  to="/profile"
                  class="home-user-menu__item"
                  @click="closeMenus"
                >
                  个人主页
                </RouterLink>
                <RouterLink
                  to="/publish"
                  class="home-user-menu__item"
                  @click="closeMenus"
                >
                  发布 Prompt
                </RouterLink>
                <button
                  class="home-user-menu__item home-user-menu__item--danger"
                  @click="handleLogout"
                >
                  退出登录
                </button>
              </div>
            </div>
            <button
              class="btn-pill-primary"
              @click="handlePublishClick"
            >
              发布
            </button>
          </div>
        </div>

        <div
          v-if="mobileNavOpen"
          class="home-mobile-panel"
        >
          <RouterLink
            v-for="item in navItems"
            :key="item.label"
            :to="item.to"
            class="home-mobile-panel__link"
            @click="closeMenus"
          >
            {{ item.label }}
          </RouterLink>
          <RouterLink
            to="/search"
            class="home-mobile-panel__link"
            @click="closeMenus"
          >
            搜索
          </RouterLink>
          <RouterLink
            v-if="!userStore.isLoggedIn"
            to="/login"
            class="home-mobile-panel__link"
            @click="closeMenus"
          >
            登录
          </RouterLink>
          <RouterLink
            v-else
            to="/profile"
            class="home-mobile-panel__link"
            @click="closeMenus"
          >
            个人主页
          </RouterLink>
          <button
            class="home-mobile-panel__primary"
            @click="handlePublishClick"
          >
            发布 Prompt
          </button>
          <button
            v-if="userStore.isLoggedIn"
            class="home-mobile-panel__link home-mobile-panel__danger"
            @click="handleLogout"
          >
            退出登录
          </button>
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
        <!-- <div class="category-bar__header">
          <span class="category-bar__title">分类</span>
          <span class="category-bar__hint">浏览</span>
        </div> -->
        <div class="category-bar__scroll">
          <button
            :class="categoryBtnClass(activeCategoryId === 'all')"
            @click="handleCategoryChange('all')"
          >
            全部
          </button>
          <button
            v-for="category in promptStore.categories"
            :key="category.id"
            :class="categoryBtnClass(activeCategoryId === category.id)"
            @click="handleCategoryChange(category.id)"
          >
            {{ category.name }} · {{ category.count }}
          </button>
        </div>
      </div>

      <section class="skill-entry">
        <div class="skill-entry__head">
          <div>
            <div class="gallery-header__label">
              Skill
            </div>
            <h2 class="skill-entry__title">
              技能分类即将开放
            </h2>
          </div>
          <span class="skill-entry__badge">Phase 2</span>
        </div>
        <div class="skill-entry__grid">
          <article
            v-for="item in skillCategoryPlaceholders"
            :key="item.name"
            class="skill-entry__card"
          >
            <div class="skill-entry__icon">
              {{ item.icon }}
            </div>
            <div class="skill-entry__name">
              {{ item.name }}
            </div>
            <p class="skill-entry__desc">
              {{ item.desc }}
            </p>
          </article>
        </div>
      </section>

      <section
        v-if="promptStore.hotTags.length > 0"
        class="tag-filter"
      >
        <div class="tag-filter__head">
          <span class="tag-filter__title">热门标签</span>
          <button
            v-if="activeTag || activeCategoryId !== 'all'"
            class="tag-filter__clear"
            @click="clearFilters"
          >
            清除筛选
          </button>
        </div>
        <div class="tag-filter__list">
          <button
            v-for="tag in promptStore.hotTags"
            :key="tag.name"
            class="tag-filter__chip"
            :class="{ 'tag-filter__chip--active': activeTag === tag.name }"
            @click="handleTagChange(tag.name)"
          >
            #{{ tag.name }}
            <span>{{ tag.count }}</span>
          </button>
        </div>
      </section>

      <section class="activity-section">
        <div class="activity-section__head">
          <div>
            <div class="gallery-header__label">
              活动
            </div>
            <h2 class="activity-section__title">
              正在被收藏的创作方向
            </h2>
          </div>
          <RouterLink
            to="/search"
            class="activity-section__more"
          >
            查看更多
          </RouterLink>
        </div>

        <div class="activity-banners">
          <RouterLink
            v-for="activity in activityCards"
            :key="activity.id"
            :to="activity.to"
            class="activity-banner"
          >
            <img
              :src="activity.image"
              :alt="activity.title"
              class="activity-banner__image"
            >
            <div class="activity-banner__overlay" />
            <div class="activity-banner__content">
              <div class="activity-banner__meta">
                {{ activity.meta }}
              </div>
              <h3 class="activity-banner__title">
                {{ activity.title }}
              </h3>
              <p class="activity-banner__desc">
                {{ activity.desc }}
              </p>
            </div>
          </RouterLink>
        </div>

        <div
          v-if="featuredRailPrompts.length > 0"
          class="featured-rail"
        >
          <RouterLink
            v-for="(prompt, index) in featuredRailPrompts"
            :key="prompt.id"
            :to="`/prompt/${prompt.id}`"
            class="featured-rail__item"
          >
            <img
              :src="resolveCover(prompt, index)"
              :alt="prompt.title"
              class="featured-rail__image"
            >
            <div class="featured-rail__body">
              <div class="featured-rail__meta">
                {{ prompt.categoryName }} · {{ prompt.model }}
              </div>
              <div class="featured-rail__title">
                {{ prompt.title }}
              </div>
            </div>
          </RouterLink>
        </div>
      </section>

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
          {{ promptStore.total || visiblePrompts.length }} 条结果
          <span v-if="activeTag"> · #{{ activeTag }}</span>
        </div>
      </section>

      <section
        v-if="promptStore.loading"
        class="gallery-masonry"
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
        class="gallery-masonry"
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

      <div
        v-if="!promptStore.loading && curatedPrompts.length > 0"
        class="gallery-pagination"
      >
        <button
          v-if="promptStore.hasMore"
          class="gallery-load-more"
          :disabled="promptStore.loadingMore"
          @click="loadMore"
        >
          {{ promptStore.loadingMore ? '加载中...' : '加载更多' }}
        </button>
        <div
          v-else
          class="gallery-end"
        >
          已经到底了
        </div>
      </div>

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
  @apply view-page;
}

.home-container {
  @apply view-container--home;
}

.home-header {
  @apply sticky top-3 z-30 rounded-[20px] border border-black/10 bg-white/90 px-4 py-3 backdrop-blur md:px-5;
  box-shadow: 0 14px 30px rgba(15, 23, 42, 0.06);
}

.home-header__inner {
  @apply flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between;
}

.home-header__brand {
  @apply flex items-center justify-between gap-4;
}

.home-header__brand-main {
  @apply flex items-center gap-4;
}

.home-logo {
  @apply text-lg font-semibold tracking-[0.08em] text-[#111111];
}

.home-badge {
  @apply hidden rounded-full border border-black/10 bg-black px-3 py-1 text-xs text-white sm:inline-flex;
}

.home-mobile-toggle {
  @apply flex h-10 w-10 flex-col items-center justify-center gap-1.5 rounded-full border border-black/10 bg-[#f7f5f0] lg:hidden;
}

.home-mobile-toggle span {
  @apply h-0.5 w-4 rounded-full bg-black;
}

.home-nav {
  @apply hidden flex-wrap items-center gap-2 lg:flex;
}

.home-nav__link {
  @apply rounded-full px-3 py-2 text-sm text-[#555555] transition hover:bg-black hover:text-white;
}

.home-actions {
  @apply hidden flex-wrap items-center justify-end gap-2 lg:flex;
}

.home-actions__search {
  @apply rounded-full border border-black/10 bg-[#f7f5f0] px-4 py-2 text-sm text-[#555555] transition hover:border-black/20 hover:text-black;
}

.home-actions__link {
  @apply rounded-full px-4 py-2 text-sm text-[#555555] transition hover:bg-black hover:text-white;
}

.home-user-menu {
  @apply relative;
}

.home-user-menu__trigger {
  @apply flex items-center gap-2 rounded-full border border-black/10 bg-[#f7f5f0] px-3 py-2 text-sm text-[#333333] transition hover:border-black/20 hover:text-black;
}

.home-user-menu__avatar {
  @apply flex h-6 w-6 items-center justify-center rounded-full bg-black text-xs font-semibold text-white;
}

.home-user-menu__panel {
  @apply absolute right-0 top-[calc(100%+8px)] z-40 grid min-w-[160px] gap-1 rounded-[18px] border border-black/10 bg-white p-2;
  box-shadow: 0 18px 38px rgba(15, 23, 42, 0.12);
}

.home-user-menu__item {
  @apply rounded-[12px] px-3 py-2 text-left text-sm text-[#444444] transition hover:bg-[#f6f4ef] hover:text-black;
}

.home-user-menu__item--danger {
  @apply text-red-600 hover:bg-red-50 hover:text-red-700;
}

.home-mobile-panel {
  @apply mt-3 grid gap-2 border-t border-black/10 pt-3 lg:hidden;
}

.home-mobile-panel__link,
.home-mobile-panel__primary {
  @apply rounded-[14px] px-4 py-3 text-left text-sm text-[#444444] transition hover:bg-[#f6f4ef] hover:text-black;
}

.home-mobile-panel__primary {
  @apply bg-black text-center font-medium text-white hover:bg-black/80 hover:text-white;
}

.home-mobile-panel__danger {
  @apply text-red-600 hover:bg-red-50 hover:text-red-700;
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
  @apply sticky top-[72px] z-20 mb-6 rounded-[20px] border border-black/10 bg-white px-4 py-3;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
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

.tag-filter {
  @apply mb-6 rounded-[20px] border border-black/10 bg-white px-4 py-3;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
}

.skill-entry {
  @apply mb-6 rounded-[20px] border border-black/10 bg-white px-4 py-4;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
}

.skill-entry__head {
  @apply mb-4 flex items-end justify-between gap-3;
}

.skill-entry__title {
  @apply mt-1 text-lg font-semibold text-black;
}

.skill-entry__badge {
  @apply shrink-0 rounded-full border border-black/10 bg-[#111111] px-3 py-1 text-xs font-medium text-white;
}

.skill-entry__grid {
  @apply grid gap-3 sm:grid-cols-2 xl:grid-cols-4;
}

.skill-entry__card {
  @apply rounded-[18px] border border-black/10 bg-[#faf8f4] p-4 transition hover:border-black/20 hover:bg-white;
}

.skill-entry__icon {
  @apply inline-flex rounded-full bg-black px-3 py-1 text-xs font-medium text-white;
}

.skill-entry__name {
  @apply mt-3 text-sm font-semibold text-black;
}

.skill-entry__desc {
  @apply mt-2 text-sm leading-5 text-[#666666];
}

.tag-filter__head {
  @apply mb-3 flex items-center justify-between gap-3;
}

.tag-filter__title {
  @apply text-sm text-[#777777];
}

.tag-filter__clear {
  @apply rounded-full px-3 py-1 text-xs text-[#777777] transition hover:bg-[#f6f4ef] hover:text-black;
}

.tag-filter__list {
  @apply flex flex-wrap gap-2;
}

.tag-filter__chip {
  @apply rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1.5 text-sm text-[#555555] transition hover:border-black/20 hover:text-black;
}

.tag-filter__chip span {
  @apply ml-1 text-xs text-[#999999];
}

.tag-filter__chip--active {
  @apply border-transparent bg-black text-white hover:text-white;
}

.tag-filter__chip--active span {
  @apply text-white/60;
}

.activity-section {
  @apply mb-6 rounded-[24px] border border-black/10 bg-white p-4;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.05);
}

.activity-section__head {
  @apply mb-4 flex items-end justify-between gap-4;
}

.activity-section__title {
  @apply mt-1 text-xl font-semibold text-black;
}

.activity-section__more {
  @apply shrink-0 rounded-full border border-black/10 bg-[#f6f4ef] px-4 py-2 text-sm text-[#555555] transition hover:border-black/20 hover:text-black;
}

.activity-banners {
  @apply grid gap-3 lg:grid-cols-3;
}

.activity-banner {
  @apply relative block h-[220px] overflow-hidden rounded-[20px] bg-black;
}

.activity-banner__image {
  @apply h-full w-full object-cover transition duration-500;
}

.activity-banner:hover .activity-banner__image {
  @apply scale-[1.03];
}

.activity-banner__overlay {
  @apply absolute inset-0 bg-gradient-to-t from-black/85 via-black/25 to-transparent;
}

.activity-banner__content {
  @apply absolute inset-x-0 bottom-0 p-4 text-white;
}

.activity-banner__meta {
  @apply text-xs uppercase tracking-[0.18em] text-white/65;
}

.activity-banner__title {
  @apply mt-2 line-clamp-2 text-lg font-semibold leading-tight;
}

.activity-banner__desc {
  @apply mt-2 line-clamp-2 text-sm leading-5 text-white/70;
}

.featured-rail {
  @apply mt-3 flex gap-3 overflow-x-auto pb-1;
  scroll-snap-type: x mandatory;
  scrollbar-width: none;
}

.featured-rail::-webkit-scrollbar {
  display: none;
}

.featured-rail__item {
  @apply grid w-[260px] shrink-0 grid-cols-[86px_1fr] overflow-hidden rounded-[18px] border border-black/10 bg-[#f8f6f1] transition hover:border-black/20 hover:bg-white;
  scroll-snap-align: start;
}

.featured-rail__image {
  @apply h-full min-h-[92px] w-full object-cover;
}

.featured-rail__body {
  @apply min-w-0 p-3;
}

.featured-rail__meta {
  @apply truncate text-xs text-[#777777];
}

.featured-rail__title {
  @apply mt-2 line-clamp-2 text-sm font-medium leading-5 text-black;
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

.gallery-masonry {
  column-count: 1;
  column-gap: 0.75rem;
  padding-bottom: 1rem;
}

@media (min-width: 640px) {
  .gallery-masonry {
    column-count: 2;
  }
}

@media (min-width: 1024px) {
  .gallery-masonry {
    column-count: 3;
  }
}

@media (min-width: 1440px) {
  .gallery-masonry {
    column-count: 4;
  }
}

.gallery-skeleton {
  @apply mb-3 h-[280px] break-inside-avoid animate-pulse rounded-[20px] bg-black/5;
}

.gallery-card {
  @apply relative mb-3 block h-[320px] min-h-0 break-inside-avoid overflow-hidden rounded-[20px] bg-black transition;
  transform: translateZ(0);
}

.gallery-card--large {
  @apply h-[430px];
}

.gallery-card--small {
  @apply h-[260px];
}

.gallery-card--tall {
  @apply h-[520px];
}

.gallery-card--medium {
  @apply h-[360px];
}

.gallery-card__image {
  @apply absolute inset-0 h-full w-full max-h-full max-w-full object-cover transition duration-500;
}

.gallery-card:hover .gallery-card__image {
  @apply scale-[1.03];
}

.gallery-card:hover {
  box-shadow: 0 18px 38px rgba(15, 23, 42, 0.16);
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

.gallery-pagination {
  @apply flex items-center justify-center pb-10 pt-2;
}

.gallery-load-more {
  @apply rounded-full border border-black/10 bg-black px-6 py-3 text-sm font-medium text-white transition hover:bg-black/80 disabled:cursor-wait disabled:opacity-60;
}

.gallery-end {
  @apply rounded-full border border-black/10 bg-white px-5 py-2 text-sm text-[#777777];
}

.empty-state {
  @apply rounded-[24px] border border-dashed border-black/10 bg-white px-6 py-16 text-center;
}

.empty-state__title {
  @apply text-lg font-semibold text-black;
}

.empty-state__desc {
  @apply mt-2 text-sm text-[#777777];
}
</style>
