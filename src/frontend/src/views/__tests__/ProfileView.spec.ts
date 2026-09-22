// 文件作用：个人中心冒烟测试。用户要求「统一身份只做基础验证、每个站点有自己的个人中心」，
// 这里固定「账号与安全」区块必须存在：登录方式、GitHub 绑定说明、统一资料入口与退出登录。
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import ProfileView from '../ProfileView.vue'
import { useUserStore } from '@/stores/user'
import { usePromptStore } from '@/stores/prompt'

vi.mock('naive-ui', async (importOriginal) => {
  const actual = await importOriginal<typeof import('naive-ui')>()
  return {
    ...actual,
    useDialog: () => ({ create: vi.fn() }),
    useMessage: () => ({ warning: vi.fn(), success: vi.fn(), error: vi.fn() })
  }
})

vi.mock('@/api/promptApi', () => ({
  promptApi: {
    getPromptList: vi.fn(async () => ({ data: { list: [], total: 0, page: 1, pageSize: 24 } })),
    getMyDraftPrompts: vi.fn(async () => ({ data: [] }))
  }
}))

vi.mock('@/api/userApi', () => ({
  userApi: {
    getFavoritePrompts: vi.fn(async () => ({ data: [] })),
    getLikedPrompts: vi.fn(async () => ({ data: [] })),
    getHistoryPrompts: vi.fn(async () => ({ data: { list: [], total: 0, page: 1 } })),
    getFollowingUsers: vi.fn(async () => ({ data: [] })),
    getFollowerUsers: vi.fn(async () => ({ data: [] })),
    getFollowStatus: vi.fn(async () => ({ data: null }))
  }
}))

const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/', component: { template: '<div />' } },
    { path: '/profile/:userId?', component: ProfileView },
    { path: '/admin', component: { template: '<div />' } },
    { path: '/login', component: { template: '<div />' } }
  ]
})

describe('ProfileView', () => {
  it('本人视角渲染账号与安全区块，并指向统一账号与退出登录', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const userStore = useUserStore()
    userStore.userInfo = { id: 7, username: 'e2e-member', email: 'e2e@example.com' } as never
    vi.spyOn(userStore, 'fetchUserInfo').mockResolvedValue(userStore.userInfo as never)
    const promptStore = usePromptStore()
    promptStore.loadHomeFeed = vi.fn(async () => undefined)
    await router.push('/profile/7')
    await router.isReady()

    const wrapper = mount(ProfileView, { global: { plugins: [pinia, router] } })
    await flushPromises()

    const security = wrapper.find('[data-testid="profile-security"]')
    expect(security.exists()).toBe(true)
    expect(security.text()).toContain('账号与安全')
    expect(security.text()).toContain('统一账号')
    expect(security.find('a[href*="/profile/"]').attributes('href')).toContain('/profile/')
    expect(security.text()).toContain('退出登录')
  })
})
