import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { promptApi } from '@/api/promptApi'
import { mockCategories, mockPrompts } from '@/mock/prompts'
import type { Prompt, Category } from '@/types'

export const usePromptStore = defineStore('prompt', () => {
  const prompts = ref<Prompt[]>([])
  const currentPrompt = ref<Prompt | null>(null)
  const categories = ref<Category[]>([])
  const loading = ref(false)
  const usingMockData = ref(false)

  const setPrompts = (list: Prompt[]) => {
    prompts.value = list
  }

  const setCurrentPrompt = (prompt: Prompt) => {
    currentPrompt.value = prompt
  }

  const setCategories = (list: Category[]) => {
    categories.value = list
  }

  const setLoading = (status: boolean) => {
    loading.value = status
  }

  const featuredPrompts = computed(() => prompts.value.slice(0, 3))
  const latestPrompts = computed(() => prompts.value.slice(3))

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
        promptApi.getCategories(),
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

  return {
    prompts,
    currentPrompt,
    categories,
    loading,
    usingMockData,
    featuredPrompts,
    latestPrompts,
    setPrompts,
    setCurrentPrompt,
    setCategories,
    setLoading
,
    loadHomeFeed
  }
})
