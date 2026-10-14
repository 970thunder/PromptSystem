// 文件作用：site store 测试。ensureLoaded 单次请求、成功应用站点信息、
// 失败回退默认值且不阻塞（loaded 置位、不抛错）。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/setupApi', () => ({
  setupApi: {
    getStatus: vi.fn(),
    initialize: vi.fn(),
    testSMTP: vi.fn(),
    getSiteInfo: vi.fn(),
    getAdminSettings: vi.fn(),
    updateAdminSettings: vi.fn(),
    adminTestSMTP: vi.fn()
  }
}))

import { setupApi } from '@/api/setupApi'
import { useSiteStore } from '@/stores/site'

describe('site store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(setupApi.getSiteInfo).mockReset()
  })

  it('加载站点信息并暴露默认站点名回退', async () => {
    vi.mocked(setupApi.getSiteInfo).mockResolvedValue({
      code: 200,
      message: 'Success',
      data: { name: 'My Hub', description: 'd', registrationOpen: false, emailAuthEnabled: true, githubOAuthEnabled: false }
    })

    const store = useSiteStore()
    expect(store.siteName).toBe('PromptOS')

    await store.ensureLoaded()
    expect(store.siteName).toBe('My Hub')
    expect(store.siteInfo.registrationOpen).toBe(false)

    // 第二次调用不重复请求
    await store.ensureLoaded()
    expect(setupApi.getSiteInfo).toHaveBeenCalledTimes(1)
  })

  it('接口失败时保持默认值并标记完成（不阻塞调用方）', async () => {
    vi.mocked(setupApi.getSiteInfo).mockRejectedValue(new Error('network down'))

    const store = useSiteStore()
    await expect(store.ensureLoaded()).resolves.toBeUndefined()

    expect(store.loaded).toBe(true)
    expect(store.siteName).toBe('PromptOS')
    expect(store.siteInfo.registrationOpen).toBe(true)
  })
})
