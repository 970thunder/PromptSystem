<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { promptApi } from '@/api/promptApi'
import { mockPrompts } from '@/mock/prompts'
import { usePromptStore } from '@/stores/prompt'
import type { Prompt } from '@/types'
import AppShell from '@/components/layout/AppShell.vue'
import BackButton from '@/components/navigation/BackButton.vue'
import PromptCard from '@/components/prompt/PromptCard.vue'
import PageLoading from '@/components/feedback/PageLoading.vue'
import PageError from '@/components/feedback/PageError.vue'

const message = useMessage()

const route = useRoute()
const router = useRouter()
const promptStore = usePromptStore()

const loading = ref(false)
const hasLoaded = ref(false)
const syncingRoute = ref(false)
const total = ref(0)
const results = ref<Prompt[]>([])
const searchHistory = ref<string[]>([])
const page = ref(1)
const pageSize = 24
const errorMessage = ref('')
let latestRequestId = 0
let skipNextRouteLoad = false

const searchHistoryKey = 'promptos:search-history'
const maxSearchHistory = 8

const filters = reactive({
  keyword: '',
  categoryId: 0,
  model: '',
  tag: '',
  sort: 'latest'
})

const modelOptions = computed(() => {
  const set = new Set<string>()
  promptStore.prompts.forEach((prompt) => set.add(prompt.model))
  results.value.forEach((prompt) => set.add(prompt.model))
  return Array.from(set).sort((a, b) => a.localeCompare(b))
})

const sortLabels: Record<string, string> = {
  latest: '最新',
  popular: '最热',
  hot: '最热'
}

const formatSortLabel = (sort: string) => sortLabels[sort] ?? sort

const activeFilterCount = computed(() => {
  let count = 0
  if (filters.keyword.trim()) count++
  if (filters.categoryId > 0) count++
  if (filters.model.trim()) count++
  if (filters.tag.trim()) count++
  if (filters.sort !== 'latest') count++
  return count
})

const hasMore = computed(() => page.value * pageSize < total.value)

// Movie-style category tabs. Each maps to an existing query param the backend
// already understands (keyword for topic categories, tag for workflow/agent).
const searchTabs = [
  { label: '全部', keyword: '', tag: '' },
  { label: '图像', keyword: '图像', tag: '' },
  { label: '文案', keyword: '文案', tag: '' },
  { label: '代码', keyword: '代码', tag: '' },
  { label: '工作流', keyword: '', tag: '流程' },
  { label: '智能体', keyword: '', tag: '智能体' }
] as const

const activeTab = computed(() => {
  if (filters.tag === '流程') return '工作流'
  if (filters.tag === '智能体') return '智能体'
  if (filters.keyword === '图像') return '图像'
  if (filters.keyword === '文案') return '文案'
  if (filters.keyword === '代码') return '代码'
  return '全部'
})

const selectTab = async (tab: typeof searchTabs[number]) => {
  filters.keyword = tab.keyword
  filters.tag = tab.tag
  await submitSearch()
}

const parseRouteQuery = () => {
  filters.keyword = typeof route.query.keyword === 'string' ? route.query.keyword : ''
  filters.categoryId = typeof route.query.categoryId === 'string' ? Number(route.query.categoryId) || 0 : 0
  filters.model = typeof route.query.model === 'string' ? route.query.model : ''
  const legacyTab = typeof route.query.tab === 'string' ? route.query.tab : ''
  filters.tag = typeof route.query.tag === 'string' ? route.query.tag : ''
  if (!filters.tag && !filters.keyword && legacyTab === 'workflow') filters.tag = '流程'
  if (!filters.tag && !filters.keyword && legacyTab === 'agent') filters.tag = '智能体'
  const routeSort = typeof route.query.sort === 'string' ? route.query.sort : 'latest'
  filters.sort = routeSort === 'hot' ? 'popular' : routeSort === 'popular' ? 'popular' : 'latest'
  page.value = Math.max(1, Number(route.query.page) || 1)
}

const updateRouteQuery = async (replace = false) => {
  const query: Record<string, string> = {}
  if (filters.keyword.trim()) query.keyword = filters.keyword.trim()
  if (filters.categoryId > 0) query.categoryId = String(filters.categoryId)
  if (filters.model.trim()) query.model = filters.model.trim()
  if (filters.tag.trim()) query.tag = filters.tag.trim()
  if (filters.sort !== 'latest') query.sort = filters.sort
  if (page.value > 1) query.page = String(page.value)

  const target = router.resolve({ path: '/search', query })
  if (target.fullPath === route.fullPath) {
    return
  }

  syncingRoute.value = true
  skipNextRouteLoad = true
  try {
    if (replace) {
      await router.replace({ path: '/search', query })
    } else {
      await router.push({ path: '/search', query })
    }
  } finally {
    syncingRoute.value = false
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

const filterMockPrompts = () => {
  const keyword = filters.keyword.trim().toLowerCase()
  const model = filters.model.trim().toLowerCase()
  const tag = filters.tag.trim().toLowerCase()

  let list = mockPrompts.filter((prompt) => {
    if (filters.categoryId > 0 && prompt.categoryId !== filters.categoryId) {
      return false
    }
    if (model && !prompt.model.toLowerCase().includes(model)) {
      return false
    }
    if (tag && !prompt.tags.some((item) => item.toLowerCase().includes(tag))) {
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

const loadResults = async (append = false) => {
  const requestId = ++latestRequestId
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await promptApi.searchPrompts({
      keyword: filters.keyword.trim(),
      categoryId: filters.categoryId || undefined,
      model: filters.model.trim() || undefined,
      tag: filters.tag.trim() || undefined,
      sort: filters.sort,
      page: page.value,
      pageSize
    })

    if (requestId !== latestRequestId) return
    if (append) {
      const existingIds = new Set(results.value.map((item) => item.id))
      results.value = [
        ...results.value,
        ...response.data.list.filter((item) => !existingIds.has(item.id))
      ]
    } else {
      results.value = response.data.list
    }
    total.value = response.data.total
    page.value = response.data.page
  } catch {
    if (requestId !== latestRequestId) return
    // API unavailable: show labeled demo data and surface the failure, never silently imply live results.
    const fallback = filterMockPrompts()
    const start = append ? results.value.length : (page.value - 1) * pageSize
    const nextPage = fallback.slice(start, start + pageSize)
    results.value = append ? [...results.value, ...nextPage] : nextPage
    total.value = fallback.length
    errorMessage.value = '暂时无法连接服务，以下为演示数据。你可以稍后重试。'
    message.warning('服务暂不可用，已切换到演示数据')
  } finally {
    if (requestId === latestRequestId) {
      hasLoaded.value = true
      loading.value = false
    }
  }
}

const submitSearch = async () => {
  saveSearchHistory(filters.keyword)
  page.value = 1
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
  filters.tag = ''
  filters.sort = 'latest'
  await submitSearch()
}

const retrySearch = async () => {
  await loadResults()
}

const loadMore = async () => {
  if (loading.value || !hasMore.value) return
  page.value += 1
  await updateRouteQuery(true)
  await loadResults(true)
}

watch(
  () => route.fullPath,
  async () => {
    if (skipNextRouteLoad) {
      skipNextRouteLoad = false
      return
    }
    if (syncingRoute.value) {
      return
    }
    parseRouteQuery()
    const legacyTab = typeof route.query.tab === 'string' ? route.query.tab : ''
    const legacySort = route.query.sort === 'hot'
    if (legacyTab === 'workflow' || legacyTab === 'agent' || legacySort) {
      await updateRouteQuery(true)
    }
    if (hasLoaded.value) {
      await loadResults()
    }
  }
)

onMounted(async () => {
  if (promptStore.categories.length === 0 || promptStore.prompts.length === 0) {
    await promptStore.loadHomeFeed()
  }

  parseRouteQuery()
  const legacyTab = typeof route.query.tab === 'string' ? route.query.tab : ''
  const legacySort = route.query.sort === 'hot'
  if (legacyTab === 'workflow' || legacyTab === 'agent' || legacySort) {
    await updateRouteQuery(true)
  }
  await loadResults()
})
</script>

<template>
  <AppShell>
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
          <BackButton
            fallback="/"
            label="返回"
            aria-label="返回上一页或首页"
          />
        </header>

        <!-- Movie-style category tabs -->
        <nav
          class="search-tabs"
          aria-label="分类筛选"
        >
          <button
            v-for="tab in searchTabs"
            :key="tab.label"
            type="button"
            class="search-tab"
            :class="{ 'search-tab--active': activeTab === tab.label }"
            :aria-pressed="activeTab === tab.label"
            @click="selectTab(tab)"
          >
            {{ tab.label }}
          </button>
        </nav>

        <!-- Compact filter row: wraps on mobile -->
        <div class="search-filters panel-card">
          <label class="search-field search-field--grow">
            <span class="search-field__label">关键词</span>
            <input
              v-model="filters.keyword"
              type="text"
              class="field-input"
              placeholder="搜索标题、标签、创作者或模型"
              @keyup.enter="submitSearch"
            >
          </label>

          <label class="search-field">
            <span class="search-field__label">排序</span>
            <select
              v-model="filters.sort"
              class="field-input"
              @change="submitSearch"
            >
              <option value="latest">
                最新
              </option>
              <option value="popular">
                热门
              </option>
            </select>
          </label>

          <label class="search-field">
            <span class="search-field__label">模型</span>
            <select
              v-model="filters.model"
              class="field-input"
              @change="submitSearch"
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

          <button
            class="btn-pill-primary search-submit"
            type="button"
            @click="submitSearch"
          >
            应用筛选
          </button>
        </div>

        <!-- Tag chips -->
        <div
          v-if="promptStore.hotTags.length > 0"
          class="search-tags"
        >
          <button
            v-for="tag in promptStore.hotTags"
            :key="tag.name"
            type="button"
            class="search-tag"
            :class="{ 'search-tag--active': filters.tag === tag.name }"
            @click="applyKeyword(tag.name)"
          >
            #{{ tag.name }}
            <span>{{ tag.count }}</span>
          </button>
        </div>

        <!-- Active filter summary -->
        <div
          v-if="activeFilterCount > 0"
          class="search-summary"
        >
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
            <span
              v-if="filters.tag"
              class="tag-chip"
            >
              标签：{{ filters.tag }}
            </span>
            <span class="tag-chip">
              排序：{{ formatSortLabel(filters.sort) }}
            </span>
          </div>
          <button
            type="button"
            class="search-summary__reset"
            @click="clearFilters"
          >
            清除筛选
          </button>
        </div>

        <section
          v-if="errorMessage"
          class="search-status search-status--error"
          role="alert"
        >
          <span>{{ errorMessage }}</span>
          <button
            type="button"
            class="search-status__action"
            :disabled="loading"
            @click="retrySearch"
          >
            {{ loading ? '重试中...' : '重试' }}
          </button>
        </section>

        <PageLoading
          v-if="loading && !hasLoaded"
          variant="grid"
          :rows="6"
          label="正在加载搜索结果"
        />

        <section
          v-else-if="results.length > 0"
          class="search-results"
        >
          <PromptCard
            v-for="(prompt, index) in results"
            :key="prompt.id"
            :prompt="prompt"
            :index="index"
            variant="result"
          />
        </section>

        <section
          v-if="results.length > 0"
          class="search-pagination"
        >
          <button
            v-if="hasMore"
            type="button"
            class="search-pagination__button"
            :disabled="loading"
            :aria-busy="loading"
            @click="loadMore"
          >
            {{ loading ? '加载中...' : '加载更多' }}
          </button>
          <span
            v-else
            class="search-pagination__end"
          >
            已显示全部结果
          </span>
        </section>

        <PageError
          v-else-if="!loading && results.length === 0"
          kind="empty"
          title="暂无匹配的提示词"
          description="试试更宽泛的关键词、切换分类，或清除部分筛选条件。"
          :action-label="activeFilterCount > 0 ? '清除筛选' : ''"
          @action="clearFilters"
        />
      </div>
    </div>
  </AppShell>
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
  color: var(--prompt-text);
}

.search-tabs {
  @apply mt-6 flex gap-2 overflow-x-auto pb-2;
  scrollbar-width: none;
}

.search-tabs::-webkit-scrollbar {
  display: none;
}

.search-tab {
  @apply shrink-0 rounded-full px-4 py-2 text-sm font-medium transition;
  border: 1px solid var(--prompt-border);
  background-color: var(--prompt-surface);
  color: var(--prompt-text-muted);
}

.search-tab:hover {
  border-color: var(--prompt-border-strong);
  color: var(--prompt-text);
}

.search-tab--active {
  border-color: transparent;
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
}

.search-filters {
  @apply mt-4 flex flex-wrap items-end gap-3 p-4;
}

.search-field {
  @apply grid gap-2;
  min-width: 160px;
}

.search-field--grow {
  @apply flex-1;
  min-width: 220px;
}

.search-field__label {
  @apply text-sm font-medium;
  color: var(--prompt-text-muted);
}

.search-submit {
  @apply py-3;
}

.search-tags {
  @apply mt-4 flex flex-wrap gap-2;
}

.search-tag {
  @apply rounded-full border px-3 py-1.5 text-sm transition;
  border-color: var(--prompt-border);
  background-color: var(--prompt-surface-muted);
  color: var(--prompt-text-muted);
}

.search-tag:hover {
  border-color: var(--prompt-border-strong);
  color: var(--prompt-text);
}

.search-tag span {
  @apply ml-1 text-xs;
  color: var(--prompt-text-faint);
}

.search-tag--active {
  border-color: transparent;
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
}

.search-tag--active span {
  color: color-mix(in srgb, var(--prompt-primary-contrast) 60%, transparent);
}

.search-summary {
  @apply mt-4 flex flex-wrap items-center justify-between gap-3;
}

.search-summary__chips {
  @apply flex flex-wrap gap-2;
}

.search-summary__reset {
  @apply shrink-0 rounded-full px-3 py-1 text-sm transition;
  color: var(--prompt-text-faint);
}

.search-summary__reset:hover {
  background-color: var(--prompt-surface-muted);
  color: var(--prompt-text);
}

.search-status {
  @apply mt-4 flex flex-wrap items-center justify-between gap-3 rounded-[16px] border px-4 py-3 text-sm;
}

.search-status--error {
  border-color: color-mix(in srgb, var(--prompt-warning) 35%, transparent);
  background-color: color-mix(in srgb, var(--prompt-warning) 12%, transparent);
  color: var(--prompt-text);
}

.search-status__action {
  @apply rounded-full border border-current px-3 py-1 text-xs font-medium transition disabled:cursor-wait disabled:opacity-60;
}

.search-status__action:hover {
  background-color: color-mix(in srgb, var(--prompt-text) 6%, transparent);
}

.search-results {
  @apply mt-6 grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6;
}

.search-pagination {
  @apply flex min-h-[52px] items-center justify-center pb-10 pt-6;
}

.search-pagination__button {
  @apply rounded-full border px-6 py-3 text-sm font-medium transition disabled:cursor-wait disabled:opacity-60;
  border-color: var(--prompt-border);
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
}

.search-pagination__button:hover:not(:disabled) {
  background-color: var(--prompt-primary-hover);
}

.search-pagination__end {
  @apply rounded-full border px-5 py-2 text-sm;
  border-color: var(--prompt-border);
  color: var(--prompt-text-faint);
}

@media (prefers-reduced-motion: reduce) {
  .search-tab,
  .search-tag {
    transition-duration: 1ms;
  }
}
</style>
