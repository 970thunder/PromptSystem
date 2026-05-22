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
  await updateRouteQuery()
  await loadResults()
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
  if (promptStore.categories.length === 0 || promptStore.prompts.length === 0) {
    await promptStore.loadHomeFeed()
  }

  parseRouteQuery()
  await loadResults()
})
</script>

<template>
  <div class="min-h-screen bg-[#f5f3ee] text-[#111111]">
    <div class="mx-auto max-w-[1160px] px-4 pb-16 pt-6 sm:px-6 lg:px-8">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="text-sm uppercase tracking-[0.2em] text-[#7c7c7c]">
            搜索
          </div>
          <h1 class="mt-2 text-3xl font-semibold">
            按意图、模型或分类查找提示词
          </h1>
        </div>
        <RouterLink
          to="/"
          class="rounded-full border border-black/10 bg-white px-4 py-2 text-sm text-[#555555] transition hover:border-black/20 hover:text-black"
        >
          返回首页
        </RouterLink>
      </header>

      <section class="mt-8 grid gap-6 lg:grid-cols-[320px_minmax(0,1fr)]">
        <aside class="self-start rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
          <div class="flex items-center justify-between gap-3">
            <div class="text-lg font-semibold">
              筛选
            </div>
            <button
              class="text-sm text-[#777777] transition hover:text-black"
              @click="clearFilters"
            >
              重置
            </button>
          </div>

          <div class="mt-5 grid gap-4">
            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">关键词</span>
              <input
                v-model="filters.keyword"
                type="text"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
                placeholder="搜索标题、标签、创作者或模型"
                @keyup.enter="submitSearch"
              >
            </label>

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">分类</span>
              <select
                v-model.number="filters.categoryId"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
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

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">模型</span>
              <select
                v-model="filters.model"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
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

            <label class="grid gap-2">
              <span class="text-sm font-medium text-[#333333]">排序</span>
              <select
                v-model="filters.sort"
                class="rounded-[16px] border border-black/10 bg-[#faf8f4] px-4 py-3 text-sm outline-none transition focus:border-black/30"
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
              class="mt-2 rounded-full bg-black px-4 py-3 text-sm font-medium text-white transition hover:bg-black/85"
              @click="submitSearch"
            >
              应用筛选
            </button>
          </div>
        </aside>

        <div class="min-w-0">
          <section class="rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div class="text-sm text-[#777777]">
                  结果
                </div>
                <div class="mt-1 text-2xl font-semibold">
                  {{ total }} 条匹配
                </div>
              </div>
              <div class="text-sm text-[#666666]">
                {{ activeFilterCount }} 个筛选条件
              </div>
            </div>

            <div class="mt-4 flex flex-wrap gap-2">
              <span
                v-if="filters.keyword"
                class="rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1 text-sm text-[#444444]"
              >
                关键词：{{ filters.keyword }}
              </span>
              <span
                v-if="filters.categoryId > 0"
                class="rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1 text-sm text-[#444444]"
              >
                分类：{{ promptStore.categories.find((item) => item.id === filters.categoryId)?.name }}
              </span>
              <span
                v-if="filters.model"
                class="rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1 text-sm text-[#444444]"
              >
                模型：{{ filters.model }}
              </span>
              <span class="rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1 text-sm text-[#444444]">
                排序：{{ formatSortLabel(filters.sort) }}
              </span>
            </div>
          </section>

          <section
            v-if="loading"
            class="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-3"
          >
            <div
              v-for="index in 6"
              :key="index"
              class="h-[320px] animate-pulse rounded-[24px] bg-black/6"
            />
          </section>

          <section
            v-else-if="results.length > 0"
            class="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-3"
          >
            <RouterLink
              v-for="(prompt, index) in results"
              :key="prompt.id"
              :to="`/prompt/${prompt.id}`"
              class="overflow-hidden rounded-[24px] border border-black/8 bg-white shadow-[0_16px_40px_rgba(15,23,42,0.05)] transition hover:-translate-y-1"
            >
              <img
                :src="resolveCover(prompt, index)"
                :alt="prompt.title"
                class="h-[220px] w-full object-cover"
              >
              <div class="p-5">
                <div class="flex items-center justify-between gap-3 text-xs uppercase tracking-[0.14em] text-[#7c7c7c]">
                  <span>{{ prompt.categoryName }}</span>
                  <span>{{ prompt.model }}</span>
                </div>
                <h2 class="mt-3 line-clamp-2 text-xl font-semibold text-black">
                  {{ prompt.title }}
                </h2>
                <p class="mt-2 line-clamp-3 text-sm leading-6 text-[#5f5f5f]">
                  {{ prompt.description }}
                </p>
                <div class="mt-4 flex flex-wrap gap-2">
                  <span
                    v-for="tag in prompt.tags.slice(0, 3)"
                    :key="tag"
                    class="rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1 text-xs text-[#555555]"
                  >
                    {{ tag }}
                  </span>
                </div>
                <div class="mt-5 flex items-center justify-between gap-3 text-sm text-[#777777]">
                  <span>{{ prompt.user.username }}</span>
                  <div class="flex items-center gap-3">
                    <span>{{ formatCount(prompt.likes) }} 赞</span>
                    <span>{{ formatCount(prompt.views) }} 浏览</span>
                  </div>
                </div>
              </div>
            </RouterLink>
          </section>

          <section
            v-else
            class="mt-6 rounded-[28px] border border-dashed border-black/12 bg-white px-6 py-16 text-center"
          >
            <div class="text-xl font-semibold text-black">
              暂无匹配的提示词
            </div>
            <p class="mt-3 text-sm text-[#777777]">
              试试更宽泛的关键词、切换分类，或清除部分筛选条件。
            </p>
          </section>
        </div>
      </section>
    </div>
  </div>
</template>
