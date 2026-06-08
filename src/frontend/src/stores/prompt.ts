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

  const ensurePromptSeed = async () => {
    if (prompts.value.length > 0 && categories.value.length > 0) {
      return
    }

    await loadHomeFeed()
  }

  const loadHomeFeed = async () => {
    loading.value = true

    const enablePromptApi = import.meta.env.VITE_ENABLE_PROMPT_API === 'true'

    if (!enablePromptApi) {
      categories.value = mockCategories
      prompts.value = mockPrompts
      usingMockData.value = true
      loading.value = false
      return
    }

    try {
      const [categoryRes, promptRes] = await Promise.all([
        categoryApi.getCategoryList(),
        promptApi.getPromptList({ page: 1, pageSize: 12, sort: 'latest' })
      ])

      categories.value = categoryRes.data
      prompts.value = promptRes.data.list
      usingMockData.value = false
    } catch {
      categories.value = mockCategories
      prompts.value = mockPrompts
      usingMockData.value = true
    } finally {
      loading.value = false
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

  const loadPromptComments = async (id: number) => {
    commentsLoading.value = true

    const enablePromptApi = import.meta.env.VITE_ENABLE_PROMPT_API === 'true'

    if (!enablePromptApi) {
      comments.value = []
      commentsLoading.value = false
      return []
    }

    try {
      const response = await promptApi.getPromptComments(id)
      comments.value = response.data
      return response.data
    } catch {
      comments.value = []
      return []
    } finally {
      commentsLoading.value = false
    }
  }

  const createPromptComment = async (id: number, payload: CreateCommentRequest) => {
    const response = await promptApi.createPromptComment(id, payload)
    await loadPromptComments(id)
    return response.data
  }

  const likeComment = async (promptID: number, commentID: number) => {
    const response = await promptApi.likeComment(commentID)
    await loadPromptComments(promptID)
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
    loadPromptDetail,
    loadPromptComments,
    createPromptComment,
    likeComment,
    getRelatedPrompts
  }
})
