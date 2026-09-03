import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { promptApi } from '@/api/promptApi'
import { usePromptStore } from '@/stores/prompt'

const comment = (id: number) => ({
  id,
  targetType: 'prompt' as const,
  targetId: 101,
  userId: 1,
  user: { id: 1, username: 'Author', avatar: '', email: '', bio: '', level: 1, experience: 0, status: 1, createdAt: '' },
  content: `comment-${id}`,
  likes: 0,
  parentId: null,
  replies: [],
  createdAt: '2026-08-30T00:00:00Z'
})

const prompt = (id: number) => ({
  id,
  title: `prompt-${id}`,
  description: 'description',
  cover: '',
  images: [],
  content: 'content',
  systemPrompt: '',
  model: 'gpt-4o',
  params: {},
  categoryId: 1,
  categoryName: '测试',
  tags: ['测试'],
  userId: 1,
  user: { id: 1, username: 'Author', avatar: '', email: '', bio: '', level: 1, experience: 0, status: 1, createdAt: '' },
  views: 0,
  likes: 0,
  favorites: 0,
  status: 1,
  createdAt: '',
  updatedAt: ''
})

describe('prompt store comments pagination', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.unstubAllEnvs()
    vi.stubEnv('VITE_ENABLE_PROMPT_API', 'true')
    vi.restoreAllMocks()
  })

  it('reads page envelopes and appends unique comments', async () => {
    vi.spyOn(promptApi, 'getPromptComments')
      .mockResolvedValueOnce({ code: 200, message: 'Success', data: { list: [comment(1)], total: 2, page: 1, pageSize: 20 } })
      .mockResolvedValueOnce({ code: 200, message: 'Success', data: { list: [comment(1), comment(2)], total: 2, page: 2, pageSize: 20 } })

    const store = usePromptStore()
    await store.loadPromptComments(101, 'latest')
    expect(store.comments.map((item) => item.id)).toEqual([1])
    expect(store.commentsTotal).toBe(2)

    await store.loadMoreComments(101, 'latest')
    expect(store.comments.map((item) => item.id)).toEqual([1, 2])
    expect(store.commentsPage).toBe(2)
  })

  it('cancels stale detail requests and keeps the newest response', async () => {
    vi.stubEnv('MODE', 'development')
    const pending: Array<{
      signal: AbortSignal | undefined
      resolve: (value: { code: number; message: string; data: ReturnType<typeof prompt> }) => void
    }> = []
    vi.spyOn(promptApi, 'getPromptDetail').mockImplementation((_id, signal) => new Promise((resolve) => {
      pending.push({ signal, resolve })
    }))

    const store = usePromptStore()
    const first = store.loadPromptDetail(1)
    const second = store.loadPromptDetail(2)
    expect(pending[0].signal?.aborted).toBe(true)

    pending[1].resolve({ code: 200, message: 'Success', data: prompt(2) })
    await second
    pending[0].resolve({ code: 200, message: 'Success', data: prompt(1) })
    await first

    expect(store.currentPrompt?.id).toBe(2)
  })

  it('cancels stale comment requests without surfacing an error state', async () => {
    vi.stubEnv('MODE', 'development')
    const pending: Array<{
      signal: AbortSignal | undefined
      resolve: (value: { code: number; message: string; data: { list: ReturnType<typeof comment>[]; total: number; page: number; pageSize: number } }) => void
    }> = []
    vi.spyOn(promptApi, 'getPromptComments').mockImplementation((_id, _sort, _page, _pageSize, signal) => new Promise((resolve) => {
      pending.push({ signal, resolve })
    }))

    const store = usePromptStore()
    const first = store.loadPromptComments(1)
    const second = store.loadPromptComments(2)
    expect(pending[0].signal?.aborted).toBe(true)

    pending[1].resolve({ code: 200, message: 'Success', data: { list: [comment(2)], total: 1, page: 1, pageSize: 20 } })
    await second
    pending[0].resolve({ code: 200, message: 'Success', data: { list: [comment(1)], total: 1, page: 1, pageSize: 20 } })
    await first

    expect(store.comments.map((item) => item.id)).toEqual([2])
    expect(store.commentsError).toBe(false)
  })
})
