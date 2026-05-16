import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Prompt, Category } from '@/types'

export const usePromptStore = defineStore('prompt', () => {
  const prompts = ref<Prompt[]>([])
  const currentPrompt = ref<Prompt | null>(null)
  const categories = ref<Category[]>([])
  const loading = ref(false)

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

  return {
    prompts,
    currentPrompt,
    categories,
    loading,
    setPrompts,
    setCurrentPrompt,
    setCategories,
    setLoading
  }
})
