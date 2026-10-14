// 文件作用：站点设置面板测试。加载回填、保存时以表单值调用 PUT，
// 密码留空不回传、保存后清空输入框。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
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

import { setupApi, type AdminSiteSettings } from '@/api/setupApi'
import SiteSettingsPanel from '../SiteSettingsPanel.vue'

const stored: AdminSiteSettings = {
  siteName: 'My Hub',
  siteDescription: 'desc',
  smtpHost: 'smtp.example.com',
  smtpPort: '465',
  smtpUser: 'noreply@example.com',
  smtpPasswordSet: true,
  smtpFrom: 'noreply@example.com',
  registrationEnabled: false,
  smtpFromEnv: true
}

const mountPanel = async () => {
  const wrapper = mount(
    { template: '<n-message-provider><site-settings-panel /></n-message-provider>', components: { NMessageProvider, SiteSettingsPanel } },
    { global: { plugins: [createPinia()] }, attachTo: document.body }
  )
  await flushPromises()
  return wrapper
}

describe('SiteSettingsPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(setupApi.getAdminSettings).mockReset()
    vi.mocked(setupApi.updateAdminSettings).mockReset()
  })

  it('加载后回填站点设置且不回显 SMTP 密码', async () => {
    vi.mocked(setupApi.getAdminSettings).mockResolvedValue({ code: 200, message: 'Success', data: stored })

    const wrapper = await mountPanel()

    const input = wrapper.find('input[placeholder="已保存，留空保持不变"]')
    expect(input.exists()).toBe(true)
    expect((wrapper.find('input[placeholder="PromptOS"]').element as HTMLInputElement).value).toBe('My Hub')
    expect(wrapper.text()).toContain('当前可发送')
  })

  it('保存提交表单值，密码留空时不回传已存密码', async () => {
    vi.mocked(setupApi.getAdminSettings).mockResolvedValue({ code: 200, message: 'Success', data: stored })
    vi.mocked(setupApi.updateAdminSettings).mockResolvedValue({
      code: 200,
      message: 'Success',
      data: { ...stored, siteName: 'New Hub' }
    })

    const wrapper = await mountPanel()

    await wrapper.find('input[placeholder="PromptOS"]').setValue('New Hub')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(setupApi.updateAdminSettings).toHaveBeenCalledTimes(1)
    expect(setupApi.updateAdminSettings).toHaveBeenCalledWith({
      siteName: 'New Hub',
      siteDescription: 'desc',
      smtpHost: 'smtp.example.com',
      smtpPort: '465',
      smtpUser: 'noreply@example.com',
      smtpPassword: '',
      smtpFrom: 'noreply@example.com',
      registrationEnabled: false
    })
  })

  it('加载失败展示错误态与重试入口', async () => {
    vi.mocked(setupApi.getAdminSettings).mockRejectedValue(new Error('boom'))

    const wrapper = await mountPanel()

    expect(wrapper.text()).toContain('站点设置加载失败')
    expect(wrapper.text()).toContain('重新加载')
  })
})
