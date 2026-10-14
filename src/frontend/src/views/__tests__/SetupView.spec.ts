// 文件作用：初始化向导组件测试。覆盖两条关键路径：已初始化部署直接进入
// 终态（不再暴露表单）；未初始化部署走完五步向导并以正确 payload 调用
// /setup/initialize。跳过邮件配置时 payload 不携带 smtp。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { NMessageProvider } from 'naive-ui'

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

import { setupApi, type SetupStatus } from '@/api/setupApi'
import SetupView from '../SetupView.vue'

const notInitialized: SetupStatus = {
  initialized: false,
  storageMode: 'mysql',
  environment: 'test',
  smtpConfigured: false,
  emailAuthEnabled: true,
  registrationOpen: true,
  siteName: 'PromptOS',
  siteDescription: ''
}

const createRouterForTest = () =>
  createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/setup', component: SetupView },
      { path: '/login', component: { template: '<div />' } }
    ]
  })

const mountSetup = async () => {
  const pinia = createPinia()
  const router = createRouterForTest()
  await router.push('/setup')
  await router.isReady()
  const wrapper = mount(
    {
      template: '<n-message-provider><router-view /></n-message-provider>',
      components: { NMessageProvider }
    },
    { global: { plugins: [pinia, router] }, attachTo: document.body }
  )
  await flushPromises()
  return wrapper
}

const findButton = (wrapper: ReturnType<typeof mount>, text: string) =>
  wrapper.findAll('button').find((button) => button.text().includes(text))

describe('SetupView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(setupApi.getStatus).mockReset()
    vi.mocked(setupApi.initialize).mockReset()
    // router.afterEach 会触发 site store 的 ensureLoaded，给出稳定解析值。
    vi.mocked(setupApi.getSiteInfo).mockReset().mockResolvedValue({
      code: 200,
      message: 'Success',
      data: { name: 'PromptOS', description: '', registrationOpen: true, emailAuthEnabled: true, githubOAuthEnabled: false }
    })
  })

  it('已初始化的部署直接展示终态，不渲染向导表单', async () => {
    vi.mocked(setupApi.getStatus).mockResolvedValue({
      code: 200,
      message: 'Success',
      data: { ...notInitialized, initialized: true }
    })

    const wrapper = await mountSetup()

    expect(wrapper.text()).toContain('系统已完成初始化')
    expect(wrapper.find('input').exists()).toBe(false)
    expect(setupApi.initialize).not.toHaveBeenCalled()
  })

  it('五步向导完成初始化并携带正确 payload（跳过邮件配置）', async () => {
    vi.mocked(setupApi.getStatus).mockResolvedValue({ code: 200, message: 'Success', data: notInitialized })
    vi.mocked(setupApi.initialize).mockResolvedValue({ code: 200, message: 'Success', data: { adminId: 42 } })

    const wrapper = await mountSetup()
    expect(wrapper.text()).toContain('环境检查')

    // 步骤 0 → 1
    await findButton(wrapper, '下一步')!.trigger('click')
    expect(wrapper.text()).toContain('管理员用户名')

    await wrapper.find('input[placeholder="例如 founder"]').setValue('founder')
    await wrapper.find('input[placeholder="admin@example.com"]').setValue('founder@example.com')
    await wrapper.find('input[placeholder="设置强密码"]').setValue('SetupPass123!')
    await wrapper.find('input[placeholder="再次输入密码"]').setValue('SetupPass123!')

    // 步骤 1 → 2（站点信息保持默认）
    await findButton(wrapper, '下一步')!.trigger('click')
    expect(wrapper.text()).toContain('站点名称')

    // 步骤 2 → 3（邮件配置）
    await findButton(wrapper, '下一步')!.trigger('click')
    expect(wrapper.text()).toContain('SMTP 服务器')

    // 跳过邮件配置 → 步骤 4
    await findButton(wrapper, '跳过邮件配置')!.trigger('click')
    expect(wrapper.text()).toContain('确认初始化')
    expect(wrapper.text()).toContain('founder@example.com')

    await findButton(wrapper, '完成初始化')!.trigger('click')
    await flushPromises()

    expect(setupApi.initialize).toHaveBeenCalledTimes(1)
    expect(setupApi.initialize).toHaveBeenCalledWith({
      username: 'founder',
      email: 'founder@example.com',
      password: 'SetupPass123!',
      siteName: 'PromptOS',
      siteDescription: '',
      smtp: undefined,
      registrationEnabled: true
    })
    expect(wrapper.text()).toContain('初始化完成')
  })

  it('管理员信息不完整时阻断进入下一步', async () => {
    vi.mocked(setupApi.getStatus).mockResolvedValue({ code: 200, message: 'Success', data: notInitialized })

    const wrapper = await mountSetup()
    await findButton(wrapper, '下一步')!.trigger('click')

    // 未填任何管理员信息，仍停留在第一步
    await findButton(wrapper, '下一步')!.trigger('click')
    expect(wrapper.text()).toContain('管理员用户名')
    expect(setupApi.initialize).not.toHaveBeenCalled()
  })
})
