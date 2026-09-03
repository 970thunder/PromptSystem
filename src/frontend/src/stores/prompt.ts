import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { isCancel } from 'axios'
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
  const commentsTotal = ref(0)
  const commentsPage = ref(1)
  const commentsPageSize = ref(20)
  const commentsLoadingMore = ref(false)
  const commentsError = ref(false)
  const usingMockData = ref(false)
  const feedError = ref('')
  const page = ref(1)
  const pageSize = ref(12)
  const total = ref(0)
  const currentCategoryId = ref<number | undefined>(undefined)
  const currentTag = ref<string | undefined>(undefined)
  const loadingMore = ref(false)
  const searchResults = ref<Prompt[]>([])
  const searchLoading = ref(false)
  const searchError = ref('')
  const searchPage = ref(1)
  const searchTotal = ref(0)
  let feedRequestID = 0
  let feedController: AbortController | null = null
  let detailRequestID = 0
  let detailController: AbortController | null = null
  let commentsRequestID = 0
  let commentsController: AbortController | null = null
  let searchRequestID = 0
  let searchController: AbortController | null = null

  const cancelPendingRequests = () => {
    feedController?.abort()
    detailController?.abort()
    commentsController?.abort()
    searchController?.abort()
    feedController = null
    detailController = null
    commentsController = null
    searchController = null
  }

  const cancelSearch = () => {
    searchController?.abort()
    searchController = null
  }

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

  const loadHomeFeed = async (categoryId?: number, tag?: string) => {
    feedController?.abort()
    const controller = new AbortController()
    feedController = controller
    const requestID = ++feedRequestID
    loading.value = true
    feedError.value = ''
    page.value = 1
    currentCategoryId.value = categoryId
    currentTag.value = tag?.trim() || undefined

    // Unit tests use deterministic fixtures; no runtime build uses this branch.
    if (import.meta.env.MODE === 'test') {
      categories.value = mockCategories
      const normalizedTag = currentTag.value?.toLowerCase()
      const filtered = mockPrompts.filter((prompt) => {
        if (categoryId && prompt.categoryId !== categoryId) return false
        if (normalizedTag && !prompt.tags.some((tag) => tag.toLowerCase() === normalizedTag)) return false
        return true
      })
      prompts.value = filtered
      total.value = filtered.length
      usingMockData.value = true
      loading.value = false
      return
    }

    try {
      const [categoryRes, promptRes] = await Promise.all([
        categoryApi.getCategoryList(controller.signal),
        promptApi.getPromptList({
          page: page.value,
          pageSize: pageSize.value,
          sort: 'latest',
          categoryId,
          tag: currentTag.value
        }, controller.signal)
      ])

      if (controller.signal.aborted || requestID !== feedRequestID) return

      categories.value = categoryRes.data
      prompts.value = promptRes.data.list
      total.value = promptRes.data.total
      usingMockData.value = false
    } catch (error) {
      if (controller.signal.aborted || requestID !== feedRequestID || isCancel(error)) return
      categories.value = []
      prompts.value = []
      total.value = 0
      usingMockData.value = false
      feedError.value = '暂时无法加载内容，请检查服务连接后重试。'
    } finally {
      if (requestID === feedRequestID) loading.value = false
    }
  }

  const loadMorePrompts = async () => {
    if (loadingMore.value || loading.value || !hasMore.value) {
      return
    }

    feedController?.abort()
    const controller = new AbortController()
    feedController = controller
    const requestID = ++feedRequestID
    loadingMore.value = true
    try {
      const nextPage = page.value + 1
      const response = await promptApi.getPromptList({
        page: nextPage,
        pageSize: pageSize.value,
        sort: 'latest',
        categoryId: currentCategoryId.value,
        tag: currentTag.value
      }, controller.signal)

      if (controller.signal.aborted || requestID !== feedRequestID) return

      page.value = response.data.page
      total.value = response.data.total
      const existing = new Set(prompts.value.map((item) => item.id))
      prompts.value = [
        ...prompts.value,
        ...response.data.list.filter((item) => !existing.has(item.id))
      ]
      usingMockData.value = false
      feedError.value = ''
    } catch (error) {
      if (controller.signal.aborted || requestID !== feedRequestID || isCancel(error)) return
      feedError.value = '暂时无法加载内容，请检查服务连接后重试。'
    } finally {
      loadingMore.value = false
    }
  }

  const loadPromptDetail = async (id: number) => {
    detailController?.abort()
    const controller = new AbortController()
    detailController = controller
    const requestID = ++detailRequestID
    detailLoading.value = true

    if (import.meta.env.MODE === 'test') {
      const prompt = mockPrompts.find((item) => item.id === id) ?? null
      currentPrompt.value = prompt
      usingMockData.value = true
      detailLoading.value = false
      return prompt
    }

    try {
      const response = await promptApi.getPromptDetail(id, controller.signal)
      if (controller.signal.aborted || requestID !== detailRequestID) return null
      currentPrompt.value = response.data
      usingMockData.value = false

      if (!prompts.value.some((item) => item.id === response.data.id)) {
        prompts.value = [response.data, ...prompts.value]
      }

      return response.data
    } catch (error) {
      if (controller.signal.aborted || requestID !== detailRequestID || isCancel(error)) return null
      currentPrompt.value = null
      usingMockData.value = false
      return null
    } finally {
      if (requestID === detailRequestID) detailLoading.value = false
    }
  }

  // Search state lives in the store so every view gets the same cancellation,
  // request-order protection, pagination and error semantics.
  const searchPrompts = async (params: {
    page: number
    pageSize: number
    categoryId?: number
    model?: string
    sort?: string
    tag?: string
    keyword?: string
  }, append = false) => {
    searchController?.abort()
    const controller = new AbortController()
    searchController = controller
    const requestID = ++searchRequestID
    searchLoading.value = true
    if (!append) {
      searchError.value = ''
    }

    try {
      const response = await promptApi.searchPrompts(params, controller.signal)
      if (controller.signal.aborted || requestID !== searchRequestID) return null

      if (append) {
        const existing = new Set(searchResults.value.map((item) => item.id))
        searchResults.value = [
          ...searchResults.value,
          ...response.data.list.filter((item) => !existing.has(item.id))
        ]
      } else {
        searchResults.value = response.data.list
      }
      searchTotal.value = response.data.total
      searchPage.value = response.data.page
      searchError.value = ''
      return response.data
    } catch (error) {
      if (controller.signal.aborted || requestID !== searchRequestID || isCancel(error)) return null
      if (!append) {
        searchResults.value = []
        searchTotal.value = 0
        searchPage.value = 1
      }
      searchError.value = '暂时无法连接服务，请检查网络或稍后重试。'
      return null
    } finally {
      if (requestID === searchRequestID) searchLoading.value = false
    }
  }

  const loadPromptComments = async (id: number, sort = 'latest') => {
    commentsController?.abort()
    const controller = new AbortController()
    commentsController = controller
    const requestID = ++commentsRequestID
    commentsLoading.value = true
    commentsError.value = false

    try {
      const response = await promptApi.getPromptComments(id, sort, 1, commentsPageSize.value, controller.signal)
      if (controller.signal.aborted || requestID !== commentsRequestID) return comments.value
      comments.value = response.data.list
      commentsTotal.value = response.data.total
      commentsPage.value = response.data.page
      return response.data.list
    } catch (error) {
      if (controller.signal.aborted || requestID !== commentsRequestID || isCancel(error)) return comments.value
      comments.value = []
      commentsTotal.value = 0
      commentsPage.value = 1
      commentsError.value = true
      return []
    } finally {
      if (requestID === commentsRequestID) commentsLoading.value = false
    }
  }

  const loadMoreComments = async (id: number, sort = 'latest') => {
    if (commentsLoadingMore.value || comments.value.length >= commentsTotal.value) {
      return comments.value
    }

    commentsController?.abort()
    const controller = new AbortController()
    commentsController = controller
    const requestID = ++commentsRequestID
    commentsLoadingMore.value = true
    try {
      const nextPage = commentsPage.value + 1
      const response = await promptApi.getPromptComments(id, sort, nextPage, commentsPageSize.value, controller.signal)
      if (controller.signal.aborted || requestID !== commentsRequestID) return comments.value
      const existing = new Set(comments.value.map((item) => item.id))
      comments.value = [
        ...comments.value,
        ...response.data.list.filter((item) => !existing.has(item.id))
      ]
      commentsTotal.value = response.data.total
      commentsPage.value = response.data.page
      return comments.value
    } catch (error) {
      if (controller.signal.aborted || requestID !== commentsRequestID || isCancel(error)) return comments.value
      throw error
    } finally {
      if (requestID === commentsRequestID) commentsLoadingMore.value = false
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
    commentsTotal,
    commentsPage,
    commentsLoadingMore,
    commentsError,
    usingMockData,
    feedError,
    page,
    pageSize,
    total,
    currentTag,
    hasMore,
    loadingMore,
    searchResults,
    searchLoading,
    searchError,
    searchPage,
    searchTotal,
    hotTags,
    featuredPrompts,
    latestPrompts,
    cancelPendingRequests,
    cancelSearch,
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
    searchPrompts,
    loadPromptDetail,
    loadPromptComments,
    loadMoreComments,
    createPromptComment,
    likeComment,
    reportComment,
    reportPrompt,
    getRelatedPrompts
  }
})
