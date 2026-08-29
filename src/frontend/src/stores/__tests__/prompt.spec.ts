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

describe('prompt store comments pagination', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
})
