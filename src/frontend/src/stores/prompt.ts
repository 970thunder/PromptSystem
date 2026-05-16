import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { promptApi } from '@/api/promptApi'
import { categoryApi } from '@/api/categoryApi'
import { mockCategories, mockPrompts } from '@/mock/prompts'
import type { Prompt, Category } from '@/types'

export const usePromptStore = defineStore('prompt', () => {
  const prompts = ref<Prompt[]>([])
  const currentPrompt = ref<Prompt | null>(null)
  const categories = ref<Category[]>([])
  const loading = ref(false)
  const detailLoading = ref(false)
  const usingMockData = ref(false)

  const setPrompts = (list: Prompt[]) => {
    prompts.value = list
  }

  const setCurrentPrompt = (prompt: Prompt | null) => {
    currentPrompt.value = prompt
  }

  const setCategories = (list: Category[]) => {
    categories.value = list
  }

  const setLoading = (status: boolean) => {
    loading.value = status
  }

  const prependPrompt = (prompt: Prompt) => {
    prompts.value = [prompt, ...prompts.value.filter((item) => item.id !== prompt.id)]
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

  const getRelatedPrompts = (promptId: number, categoryId: number) => {
    return prompts.value
      .filter((item) => item.id !== promptId && item.categoryId === categoryId)
      .slice(0, 3)
  }

  return {
    prompts,
    currentPrompt,
    categories,
    loading,
    detailLoading,
    usingMockData,
    featuredPrompts,
    latestPrompts,
    setPrompts,
    setCurrentPrompt,
    setCategories,
    setLoading,
    prependPrompt,
    ensurePromptSeed,
    loadHomeFeed,
    loadPromptDetail,
    getRelatedPrompts
  }
})
