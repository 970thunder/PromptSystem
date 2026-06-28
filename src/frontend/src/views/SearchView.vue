<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { promptApi } from '@/api/promptApi'
import { mockPrompts } from '@/mock/prompts'
import { usePromptStore } from '@/stores/prompt'
import type { Prompt } from '@/types'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'

const route = useRoute()
const router = useRouter()
const promptStore = usePromptStore()

const loading = ref(false)
const hasLoaded = ref(false)
const syncingRoute = ref(false)
const total = ref(0)
const results = ref<Prompt[]>([])
const searchHistory = ref<string[]>([])

const searchHistoryKey = 'promptos:search-history'
const maxSearchHistory = 8

const filters = reactive({
  keyword: '',
  categoryId: 0,
  model: '',
  sort: 'latest'
})

const fallbackCoverMap: Record<number, string> = {
  101: 'https://images.unsplash.com/photo-1516321497487-e288fb19713f?auto=format&fit=crop&w=1200&q=80',
  102: 'https://images.unsplash.com/photo-1498050108023-c5249f4df085?auto=format&fit=crop&w=1200&q=80',
  103: 'https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1200&q=80',
  104: 'https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=1200&q=80',
  105: 'https://images.unsplash.com/photo-1526379095098-d400fd0bf935?auto=format&fit=crop&w=1200&q=80',
  106: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80'
}

const modelOptions = computed(() => {
  const set = new Set<string>()
  promptStore.prompts.forEach((prompt) => set.add(prompt.model))
  results.value.forEach((prompt) => set.add(prompt.model))
  return Array.from(set).sort((a, b) => a.localeCompare(b))
})

const sortLabels: Record<string, string> = {
  latest: '最新',
  popular: '最热'
}

const formatSortLabel = (sort: string) => sortLabels[sort] ?? sort

const keywordPool = computed(() => {
  const counts = new Map<string, number>()
  const addKeyword = (value: string, weight = 1) => {
    const keyword = value.trim()
    if (!keyword || keyword.length > 24) {
      return
    }
    counts.set(keyword, (counts.get(keyword) ?? 0) + weight)
  }

  const source = [...promptStore.prompts, ...results.value]
  source.forEach((prompt) => {
    addKeyword(prompt.model, 2)
    addKeyword(prompt.categoryName, 2)
    prompt.tags.forEach((tag) => addKeyword(tag, 3))
    prompt.title
      .split(/[\s,，、:：|｜/]+/)
      .filter((part) => part.length >= 2)
      .slice(0, 4)
      .forEach((part) => addKeyword(part))
  })

  return Array.from(counts.entries())
    .sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0], 'zh-Hans-CN'))
    .map(([name]) => name)
})

const hotSearches = computed(() => keywordPool.value.slice(0, 10))

const searchSuggestions = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase()
  if (!keyword) {
    return []
  }

  return keywordPool.value
    .filter((item) => item.toLowerCase().includes(keyword) && item.toLowerCase() !== keyword)
    .slice(0, 6)
})

const activeFilterCount = computed(() => {
  let count = 0
  if (filters.keyword.trim()) count++
  if (filters.categoryId > 0) count++
  if (filters.model.trim()) count++
  if (filters.sort !== 'latest') count++
  return count
})

const resolveCover = (prompt: Prompt, index: number) => {
  if (isDisplayableCover(prompt.cover)) {
    return resolveMediaUrl(prompt.cover)
  }

  return fallbackCoverMap[prompt.id] ?? fallbackCoverMap[101 + (index % Object.keys(fallbackCoverMap).length)]
}

const formatCount = (value: number) => {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}k`
  }

  return `${value}`
}

const parseRouteQuery = () => {
  filters.keyword = typeof route.query.keyword === 'string' ? route.query.keyword : ''
  filters.categoryId = typeof route.query.categoryId === 'string' ? Number(route.query.categoryId) || 0 : 0
  filters.model = typeof route.query.model === 'string' ? route.query.model : ''
  filters.sort = typeof route.query.sort === 'string' ? route.query.sort : 'latest'
}

const updateRouteQuery = async () => {
  syncingRoute.value = true
  const query: Record<string, string> = {}
  if (filters.keyword.trim()) query.keyword = filters.keyword.trim()
  if (filters.categoryId > 0) query.categoryId = String(filters.categoryId)
  if (filters.model.trim()) query.model = filters.model.trim()
  if (filters.sort !== 'latest') query.sort = filters.sort

  await router.replace({ path: '/search', query })
  syncingRoute.value = false
}

const loadSearchHistory = () => {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(searchHistoryKey) ?? '[]')
    if (Array.isArray(parsed)) {
      searchHistory.value = parsed.filter((item): item is string => typeof item === 'string').slice(0, maxSearchHistory)
    }
  } catch {
    searchHistory.value = []
  }
}

const saveSearchHistory = (keyword: string) => {
  const cleanKeyword = keyword.trim()
  if (!cleanKeyword) {
    return
  }

  searchHistory.value = [
    cleanKeyword,
    ...searchHistory.value.filter((item) => item.toLowerCase() !== cleanKeyword.toLowerCase())
  ].slice(0, maxSearchHistory)
  localStorage.setItem(searchHistoryKey, JSON.stringify(searchHistory.value))
}

const clearSearchHistory = () => {
  searchHistory.value = []
  localStorage.removeItem(searchHistoryKey)
}

const filterMockPrompts = () => {
  const keyword = filters.keyword.trim().toLowerCase()
  const model = filters.model.trim().toLowerCase()

  let list = mockPrompts.filter((prompt) => {
    if (filters.categoryId > 0 && prompt.categoryId !== filters.categoryId) {
      return false
    }
    if (model && !prompt.model.toLowerCase().includes(model)) {
      return false
    }
    if (!keyword) {
      return true
    }

    return [
      prompt.title,
      prompt.description,
      prompt.content,
      prompt.systemPrompt,
      prompt.categoryName,
      prompt.model,
      prompt.user.username,
      ...prompt.tags
    ].some((field) => field.toLowerCase().includes(keyword))
  })

  list = [...list].sort((left, right) => {
    if (filters.sort === 'popular') {
      return right.likes - left.likes
    }

    return right.createdAt.localeCompare(left.createdAt)
  })

  return list
}

const loadResults = async () => {
  loading.value = true
  try {
    const response = await promptApi.searchPrompts({
      keyword: filters.keyword.trim(),
      categoryId: filters.categoryId || undefined,
      model: filters.model.trim() || undefined,
      sort: filters.sort,
      page: 1,
      pageSize: 24
    })

    results.value = response.data.list
    total.value = response.data.total
  } catch {
    const fallback = filterMockPrompts()
    results.value = fallback
    total.value = fallback.length
  } finally {
    hasLoaded.value = true
    loading.value = false
  }
}

const submitSearch = async () => {
  saveSearchHistory(filters.keyword)
  await updateRouteQuery()
  await loadResults()
}

const applyKeyword = async (keyword: string) => {
  filters.keyword = keyword
  await submitSearch()
}

const clearFilters = async () => {
  filters.keyword = ''
  filters.categoryId = 0
  filters.model = ''
  filters.sort = 'latest'
  await submitSearch()
}

watch(
  () => route.fullPath,
  async () => {
    if (syncingRoute.value) {
      return
    }
    parseRouteQuery()
    if (hasLoaded.value) {
      await loadResults()
    }
  }
)

onMounted(async () => {
  loadSearchHistory()
  if (promptStore.categories.length === 0 || promptStore.prompts.length === 0) {
    await promptStore.loadHomeFeed()
  }

  parseRouteQuery()
  await loadResults()
})
</script>

<template>
  <div class="search-page">
    <div class="search-container">
      <header class="search-header">
        <div>
          <div class="section-eyebrow">
            搜索
          </div>
          <h1 class="search-header__title">
            按意图、模型或分类查找提示词
          </h1>
        </div>
        <RouterLink
          to="/"
          class="btn-pill-secondary bg-white"
        >
          返回首页
        </RouterLink>
      </header>

      <section class="search-layout">
        <aside class="search-sidebar panel-card">
          <div class="search-sidebar__head">
            <div class="search-sidebar__title">
              筛选
            </div>
            <button
              class="search-sidebar__reset"
              @click="clearFilters"
            >
              重置
            </button>
          </div>

          <div class="search-sidebar__form">
            <label class="search-field">
              <span class="search-field__label">关键词</span>
              <input
                v-model="filters.keyword"
                type="text"
                class="field-input"
                placeholder="搜索标题、标签、创作者或模型"
                @keyup.enter="submitSearch"
              >
            </label>

            <div
              v-if="searchSuggestions.length > 0"
              class="search-assist"
            >
              <div class="search-assist__label">
                搜索建议
              </div>
              <div class="search-assist__chips">
                <button
                  v-for="keyword in searchSuggestions"
                  :key="keyword"
                  class="search-assist__chip"
                  @click="applyKeyword(keyword)"
                >
                  {{ keyword }}
                </button>
              </div>
            </div>

            <div
              v-if="hotSearches.length > 0"
              class="search-assist"
            >
              <div class="search-assist__label">
                热门搜索
              </div>
              <div class="search-assist__chips">
                <button
                  v-for="keyword in hotSearches"
                  :key="keyword"
                  class="search-assist__chip"
                  @click="applyKeyword(keyword)"
                >
                  {{ keyword }}
                </button>
              </div>
            </div>

            <div
              v-if="searchHistory.length > 0"
              class="search-assist"
            >
              <div class="search-assist__head">
                <div class="search-assist__label">
                  搜索历史
                </div>
                <button
                  class="search-assist__clear"
                  @click="clearSearchHistory"
                >
                  清空
                </button>
              </div>
              <div class="search-assist__chips">
                <button
                  v-for="keyword in searchHistory"
                  :key="keyword"
                  class="search-assist__chip"
                  @click="applyKeyword(keyword)"
                >
                  {{ keyword }}
                </button>
              </div>
            </div>

            <label class="search-field">
              <span class="search-field__label">分类</span>
              <select
                v-model.number="filters.categoryId"
                class="field-input"
              >
                <option :value="0">
                  全部分类
                </option>
                <option
                  v-for="category in promptStore.categories"
                  :key="category.id"
                  :value="category.id"
                >
                  {{ category.name }}
                </option>
              </select>
            </label>

            <label class="search-field">
              <span class="search-field__label">模型</span>
              <select
                v-model="filters.model"
                class="field-input"
              >
                <option value="">
                  全部模型
                </option>
                <option
                  v-for="modelName in modelOptions"
                  :key="modelName"
                  :value="modelName"
                >
                  {{ modelName }}
                </option>
              </select>
            </label>

            <label class="search-field">
              <span class="search-field__label">排序</span>
              <select
                v-model="filters.sort"
                class="field-input"
              >
                <option value="latest">
                  最新
                </option>
                <option value="popular">
                  最热
                </option>
              </select>
            </label>

            <button
              class="btn-pill-primary search-submit"
              @click="submitSearch"
            >
              应用筛选
            </button>
          </div>
        </aside>

        <div class="search-main">
          <section class="search-summary panel-card">
            <div class="search-summary__head">
              <div>
                <div class="text-muted-sm">
                  结果
                </div>
                <div class="search-summary__count">
                  {{ total }} 条匹配
                </div>
              </div>
              <div class="search-summary__filters">
                {{ activeFilterCount }} 个筛选条件
              </div>
            </div>

            <div class="search-summary__chips">
              <span
                v-if="filters.keyword"
                class="tag-chip"
              >
                关键词：{{ filters.keyword }}
              </span>
              <span
                v-if="filters.categoryId > 0"
                class="tag-chip"
              >
                分类：{{ promptStore.categories.find((item) => item.id === filters.categoryId)?.name }}
              </span>
              <span
                v-if="filters.model"
                class="tag-chip"
              >
                模型：{{ filters.model }}
              </span>
              <span class="tag-chip">
                排序：{{ formatSortLabel(filters.sort) }}
              </span>
            </div>
          </section>

          <section
            v-if="loading"
            class="search-results"
          >
            <div
              v-for="index in 6"
              :key="index"
              class="search-skeleton"
            />
          </section>

          <section
            v-else-if="results.length > 0"
            class="search-results"
          >
            <RouterLink
              v-for="(prompt, index) in results"
              :key="prompt.id"
              :to="`/prompt/${prompt.id}`"
              class="result-card"
            >
              <div class="result-card__cover">
                <img
                  :src="resolveCover(prompt, index)"
                  :alt="prompt.title"
                  class="result-card__image"
                >
              </div>
              <div class="result-card__body">
                <div class="result-card__meta">
                  <span>{{ prompt.categoryName }}</span>
                  <span>{{ prompt.model }}</span>
                </div>
                <h2 class="result-card__title">
                  {{ prompt.title }}
                </h2>
                <p class="result-card__desc">
                  {{ prompt.description }}
                </p>
                <div class="result-card__tags">
                  <span
                    v-for="tag in prompt.tags.slice(0, 3)"
                    :key="tag"
                    class="result-card__tag"
                  >
                    {{ tag }}
                  </span>
                </div>
                <div class="result-card__footer">
                  <span>{{ prompt.user.username }}</span>
                  <div class="result-card__stats">
                    <span>{{ formatCount(prompt.likes) }} 赞</span>
                    <span>{{ formatCount(prompt.views) }} 浏览</span>
                  </div>
                </div>
              </div>
            </RouterLink>
          </section>

          <section
            v-else
            class="empty-state search-empty"
          >
            <div class="empty-state__title search-empty__title">
              暂无匹配的提示词
            </div>
            <p class="empty-state__desc search-empty__desc">
              试试更宽泛的关键词、切换分类，或清除部分筛选条件。
            </p>
          </section>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.search-page {
  @apply view-page;
}

.search-container {
  @apply view-container;
}

.search-header {
  @apply flex flex-wrap items-center justify-between gap-3;
}

.search-header__title {
  @apply mt-2 text-3xl font-semibold;
}

.search-layout {
  @apply mt-8 grid gap-6 lg:grid-cols-[320px_minmax(0,1fr)];
}

.search-sidebar {
  @apply self-start p-6;
}

.search-sidebar__head {
  @apply flex items-center justify-between gap-3;
}

.search-sidebar__title {
  @apply text-lg font-semibold;
}

.search-sidebar__reset {
  @apply text-sm text-[#777777] transition hover:text-black;
}

.search-sidebar__form {
  @apply mt-5 grid gap-4;
}

.search-field {
  @apply grid gap-2;
}

.search-field__label {
  @apply text-sm font-medium text-[#333333];
}

.search-assist {
  @apply grid gap-2 rounded-[16px] border border-black/10 bg-[#faf8f4] p-3;
}

.search-assist__head {
  @apply flex items-center justify-between gap-3;
}

.search-assist__label {
  @apply text-xs font-medium text-[#777777];
}

.search-assist__clear {
  @apply text-xs text-[#777777] transition hover:text-black;
}

.search-assist__chips {
  @apply flex flex-wrap gap-2;
}

.search-assist__chip {
  @apply rounded-full border border-black/10 bg-white px-3 py-1 text-xs text-[#555555] transition hover:border-black/20 hover:text-black;
}

.search-submit {
  @apply mt-2 w-full py-3;
}

.search-main {
  @apply min-w-0;
}

.search-summary {
  @apply p-6;
}

.search-summary__head {
  @apply flex flex-wrap items-center justify-between gap-3;
}

.search-summary__count {
  @apply mt-1 text-2xl font-semibold;
}

.search-summary__filters {
  @apply text-sm text-[#666666];
}

.search-summary__chips {
  @apply mt-4 flex flex-wrap gap-2;
}

.search-results {
  @apply mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-3;
}

.search-skeleton {
  @apply h-[320px] animate-pulse rounded-[24px] bg-black/5;
}

.result-card {
  @apply overflow-hidden rounded-[24px] border border-black/10 bg-white transition hover:-translate-y-1;
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.05);
}

.result-card__cover {
  @apply h-[220px] overflow-hidden bg-[#f6f4ef];
}

.result-card__image {
  @apply h-full w-full max-h-full max-w-full object-cover;
}

.result-card__body {
  @apply p-5;
}

.result-card__meta {
  @apply flex items-center justify-between gap-3 text-xs uppercase tracking-[0.14em] text-[#7c7c7c];
}

.result-card__title {
  @apply mt-3 line-clamp-2 text-xl font-semibold text-black;
}

.result-card__desc {
  @apply mt-2 line-clamp-3 text-sm leading-6 text-[#5f5f5f];
}

.result-card__tags {
  @apply mt-4 flex flex-wrap gap-2;
}

.result-card__tag {
  @apply rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1 text-xs text-[#555555];
}

.result-card__footer {
  @apply mt-5 flex items-center justify-between gap-3 text-sm text-[#777777];
}

.result-card__stats {
  @apply flex items-center gap-3;
}

.search-empty {
  @apply mt-6;
}

.search-empty__title {
  @apply text-xl;
}

.search-empty__desc {
  @apply mt-3;
}
</style>
