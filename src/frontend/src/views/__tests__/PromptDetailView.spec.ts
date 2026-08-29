import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import PromptDetailView from '../PromptDetailView.vue'
import { usePromptStore } from '@/stores/prompt'
import type { Prompt } from '@/types'

vi.mock('naive-ui', async (importOriginal) => {
  const actual = await importOriginal<typeof import('naive-ui')>()
  return {
    ...actual,
    useDialog: () => ({ create: vi.fn() }),
    useMessage: () => ({
      warning: vi.fn(),
      success: vi.fn(),
      error: vi.fn()
    })
  }
})

const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/prompt/:id', component: PromptDetailView },
    { path: '/profile/:id', component: { template: '<div />' } },
    { path: '/login', component: { template: '<div />' } }
  ]
})

describe('PromptDetailView', () => {
  it('does not throw when author and comment users are missing', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const promptStore = usePromptStore()
    const prompt = {
      id: 105,
      title: 'Detail fixture',
      description: 'Fixture description',
      cover: '',
      images: [],
      content: 'Fixture content',
      systemPrompt: 'Fixture system prompt',
      model: 'o3',
      params: { temperature: 0.4, topP: 0.9, maxTokens: 1700 },
      categoryId: 30,
      categoryName: '其他',
      tags: [],
      userId: 5,
      user: undefined,
      views: 0,
      likes: 0,
      favorites: 0,
      status: 1,
      createdAt: '2026-05-12',
      updatedAt: '2026-08-28'
    } as unknown as Prompt

    promptStore.setCurrentPrompt(prompt)
    promptStore.setComments([{
      id: 1,
      targetType: 'prompt',
      targetId: prompt.id,
      userId: 99,
      user: undefined,
      content: 'Fixture comment',
      likes: 0,
      createdAt: '2026-08-28',
      replies: undefined
    } as never])
    vi.spyOn(promptStore, 'ensurePromptSeed').mockResolvedValue()
    vi.spyOn(promptStore, 'loadPromptDetail').mockResolvedValue(prompt)
    vi.spyOn(promptStore, 'loadPromptComments').mockResolvedValue([])

    await router.push('/prompt/105')
    await router.isReady()
    const wrapper = mount(PromptDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          AppShell: { template: '<div><slot /></div>' },
          BackButton: { template: '<button />' },
          RouterLink: { template: '<a><slot /></a>' },
          PageLoading: { template: '<div />' },
          PageError: { template: '<div />' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('作者')
    expect(wrapper.text()).toContain('社区用户')
    expect(wrapper.text()).toContain('社区反馈')
  })
})
