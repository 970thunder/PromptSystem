import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { promptApi } from '@/api/promptApi'
import { categoryApi } from '@/api/categoryApi'
import { mockCategories, mockPrompts } from '@/mock/prompts'
import type { Prompt, Category, Comment, CreateCommentRequest } from '@/types'

export const usePromptStore = defineStore('prompt', () => {
  const prompts = ref<Prompt[]>([])
  const currentPrompt = ref<Prompt | null>(null)
  const comments = ref<Comment[]>([])
  const categories = ref<Category[]>([])
  const loading = ref(false)
  const detailLoading = ref(false)
  const commentsLoading = ref(false)
  const usingMockData = ref(false)
  const page = ref(1)
  const pageSize = ref(12)
  const total = ref(0)
  const currentCategoryId = ref<number | undefined>(undefined)
  const currentTag = ref<string | undefined>(undefined)
  const loadingMore = ref(false)

  const setPrompts = (list: Prompt[]) => {
    prompts.value = list
  }

  const setCurrentPrompt = (prompt: Prompt | null) => {
    currentPrompt.value = prompt
  }

  const mergePrompt = (prompt: Prompt) => {
    if (currentPrompt.value?.id === prompt.id) {
      currentPrompt.value = prompt
    }

    const existingIndex = prompts.value.findIndex((item) => item.id === prompt.id)
    if (existingIndex === -1) {
      prompts.value = [prompt, ...prompts.value]
      return
    }

    const next = [...prompts.value]
    next.splice(existingIndex, 1, prompt)
    prompts.value = next
  }

  const setCategories = (list: Category[]) => {
    categories.value = list
  }

  const setComments = (list: Comment[]) => {
    comments.value = list
  }

  const setLoading = (status: boolean) => {
    loading.value = status
  }

  const prependPrompt = (prompt: Prompt) => {
    prompts.value = [prompt, ...prompts.value.filter((item) => item.id !== prompt.id)]
  }

  const upsertPrompt = (prompt: Prompt) => {
    const existingIndex = prompts.value.findIndex((item) => item.id === prompt.id)
    if (existingIndex === -1) {
      prompts.value = [prompt, ...prompts.value]
      return
    }

    const next = [...prompts.value]
    next.splice(existingIndex, 1)
    prompts.value = [prompt, ...next]
  }

  const removePrompt = (id: number) => {
    prompts.value = prompts.value.filter((item) => item.id !== id)
    if (currentPrompt.value?.id === id) {
      currentPrompt.value = null
    }
  }

  const featuredPrompts = computed(() => prompts.value.slice(0, 3))
  const latestPrompts = computed(() => prompts.value.slice(3))
  const hasMore = computed(() => prompts.value.length < total.value)
  const hotTags = computed(() => {
    const counts = new Map<string, number>()
    prompts.value.forEach((prompt) => {
      prompt.tags.forEach((tag) => {
        const trimmed = tag.trim()
        if (!trimmed) {
          return
        }
        counts.set(trimmed, (counts.get(trimmed) ?? 0) + 1)
      })
    })

    return Array.from(counts.entries())
      .sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0], 'zh-Hans-CN'))
      .slice(0, 12)
      .map(([name, count]) => ({ name, count }))
  })

  const ensurePromptSeed = async () => {
    if (prompts.value.length > 0 && categories.value.length > 0) {
      return
    }

    await loadHomeFeed()
  }

  const filterMockPrompts = (categoryId?: number, tag?: string) => {
    const normalizedTag = tag?.trim().toLowerCase()
    return mockPrompts.filter((prompt) => {
      if (categoryId && prompt.categoryId !== categoryId) {
        return false
      }
      if (normalizedTag && !prompt.tags.some((item) => item.trim().toLowerCase() === normalizedTag)) {
        return false
      }
      return true
    })
  }

  const loadHomeFeed = async (categoryId?: number, tag?: string) => {
    loading.value = true
    page.value = 1
    currentCategoryId.value = categoryId
    currentTag.value = tag?.trim() || undefined

    const enablePromptApi = import.meta.env.VITE_ENABLE_PROMPT_API === 'true'

    if (!enablePromptApi) {
      categories.value = mockCategories
      const filtered = filterMockPrompts(categoryId, currentTag.value)
      prompts.value = filtered
      total.value = filtered.length
      usingMockData.value = true
      loading.value = false
      return
    }

    try {
      const [categoryRes, promptRes] = await Promise.all([
        categoryApi.getCategoryList(),
        promptApi.getPromptList({
          page: page.value,
          pageSize: pageSize.value,
          sort: 'latest',
          categoryId,
          tag: currentTag.value
        })
      ])

      categories.value = categoryRes.data
      prompts.value = promptRes.data.list
      total.value = promptRes.data.total
      usingMockData.value = false
    } catch {
      categories.value = mockCategories
      const filtered = filterMockPrompts(categoryId, currentTag.value)
      prompts.value = filtered
      total.value = filtered.length
      usingMockData.value = true
    } finally {
      loading.value = false
    }
  }

  const loadMorePrompts = async () => {
    if (loadingMore.value || loading.value || !hasMore.value) {
      return
    }

    const enablePromptApi = import.meta.env.VITE_ENABLE_PROMPT_API === 'true'
    if (!enablePromptApi) {
      return
    }

    loadingMore.value = true
    try {
      const nextPage = page.value + 1
      const response = await promptApi.getPromptList({
        page: nextPage,
        pageSize: pageSize.value,
        sort: 'latest',
        categoryId: currentCategoryId.value,
        tag: currentTag.value
      })

      page.value = response.data.page
      total.value = response.data.total
      const existing = new Set(prompts.value.map((item) => item.id))
      prompts.value = [
        ...prompts.value,
        ...response.data.list.filter((item) => !existing.has(item.id))
      ]
      usingMockData.value = false
    } finally {
      loadingMore.value = false
    }
  }

  const loadPromptDetail = async (id: number) => {
    detailLoading.value = true

    const enablePromptApi = import.meta.env.VITE_ENABLE_PROMPT_API === 'true'

    if (!enablePromptApi) {
      const prompt = mockPrompts.find((item) => item.id === id) ?? null
      currentPrompt.value = prompt
      usingMockData.value = true
      detailLoading.value = false
      return prompt
    }

    try {
      const response = await promptApi.getPromptDetail(id)
      currentPrompt.value = response.data
      usingMockData.value = false

      if (!prompts.value.some((item) => item.id === response.data.id)) {
        prompts.value = [response.data, ...prompts.value]
      }

      return response.data
    } catch {
      const prompt = mockPrompts.find((item) => item.id === id) ?? null
      currentPrompt.value = prompt
      usingMockData.value = true
      return prompt
    } finally {
      detailLoading.value = false
    }
  }

  const loadPromptComments = async (id: number, sort = 'latest') => {
    commentsLoading.value = true

    const enablePromptApi = import.meta.env.VITE_ENABLE_PROMPT_API === 'true'

    if (!enablePromptApi) {
      comments.value = []
      commentsLoading.value = false
      return []
    }

    try {
      const response = await promptApi.getPromptComments(id, sort)
      comments.value = response.data
      return response.data
    } catch {
      comments.value = []
      return []
    } finally {
      commentsLoading.value = false
    }
  }

  const createPromptComment = async (id: number, payload: CreateCommentRequest, sort = 'latest') => {
    const response = await promptApi.createPromptComment(id, payload)
    await loadPromptComments(id, sort)
    return response.data
  }

  const likeComment = async (promptID: number, commentID: number, sort = 'latest') => {
    const response = await promptApi.likeComment(commentID)
    await loadPromptComments(promptID, sort)
    return response.data
  }

  const reportComment = async (commentID: number, payload: { reason: string; detail?: string }) => {
    const response = await promptApi.reportComment(commentID, payload)
    return response.data
  }

  const reportPrompt = async (promptID: number, payload: { reason: string; detail?: string }) => {
    const response = await promptApi.reportPrompt(promptID, payload)
    return response.data
  }

  const getRelatedPrompts = (promptId: number, categoryId: number) => {
    return prompts.value
      .filter((item) => item.id !== promptId && item.categoryId === categoryId)
      .slice(0, 3)
  }

  return {
    prompts,
    currentPrompt,
    comments,
    categories,
    loading,
    detailLoading,
    commentsLoading,
    usingMockData,
    page,
    pageSize,
    total,
    currentTag,
    hasMore,
    loadingMore,
    hotTags,
    featuredPrompts,
    latestPrompts,
    setPrompts,
    setCurrentPrompt,
    mergePrompt,
    setCategories,
    setComments,
    setLoading,
    prependPrompt,
    upsertPrompt,
    removePrompt,
    ensurePromptSeed,
    loadHomeFeed,
    loadMorePrompts,
    loadPromptDetail,
    loadPromptComments,
    createPromptComment,
    likeComment,
    reportComment,
    reportPrompt,
    getRelatedPrompts
  }
})
