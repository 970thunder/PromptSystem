// 文件作用：注册页验证码冷却的组件测试。服务端对同一邮箱有 60s 冷却（429 +
// Retry-After），前端必须读取 Retry-After 启动倒计时并禁用按钮，避免用户反复
// 点击撞 429（生产实际发生过的体验问题）。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { NMessageProvider } from 'naive-ui'

vi.mock('@/api/userApi', () => ({
  userApi: {
    sendCaptcha: vi.fn()
  }
}))

import { userApi } from '@/api/userApi'
import RegisterView from '../RegisterView.vue'
import AppShell from '@/components/layout/AppShell.vue'

const createRouterForTest = () =>
  createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/register', component: RegisterView },
      { path: '/login', component: { template: '<div />' } }
    ]
  })

const mountRegister = async () => {
  const pinia = createPinia()
  const router = createRouterForTest()
  await router.push('/register')
  await router.isReady()
  const wrapper = mount(
    {
      template: '<n-message-provider><app-shell><router-view /></app-shell></n-message-provider>',
      components: { NMessageProvider, AppShell }
    },
    {
      global: { plugins: [pinia, router] },
      attachTo: document.body
    }
  )
  await flushPromises()
  return wrapper
}

describe('RegisterView captcha cooldown', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(userApi.sendCaptcha).mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('429 响应读取 Retry-After 启动倒计时并禁用按钮', async () => {
    vi.mocked(userApi.sendCaptcha).mockRejectedValueOnce({
      response: {
        status: 429,
        headers: { 'retry-after': '55' },
        data: { code: 429, message: 'Captcha was sent too frequently', errorCode: 'RATE_LIMITED' }
      }
    })

    const wrapper = await mountRegister()
    const emailInput = wrapper.find('input[placeholder="you@example.com"]')
    await emailInput.setValue('probe@example.com')

    const button = wrapper.find('.captcha-row__button')
    expect(button.attributes('disabled')).toBeUndefined()

    await button.trigger('click')
    await flushPromises()

    const countdownButton = wrapper.find('.captcha-row__button')
    expect(countdownButton.text()).toMatch(/^\d+s$/)
    expect(Number.parseInt(countdownButton.text(), 10)).toBeGreaterThan(0)
    expect(countdownButton.attributes('disabled')).toBeDefined()

    wrapper.unmount()
  })

  it('非 429 错误不触发冷却倒计时', async () => {
    vi.mocked(userApi.sendCaptcha).mockRejectedValueOnce({
      response: {
        status: 500,
        data: { code: 500, message: 'Internal error', errorCode: 'CAPTCHA_GENERATION_FAILED' }
      }
    })

    const wrapper = await mountRegister()
    await wrapper.find('input[placeholder="you@example.com"]').setValue('probe@example.com')

    await wrapper.find('.captcha-row__button').trigger('click')
    await flushPromises()

    expect(wrapper.find('.captcha-row__button').text()).toBe('获取验证码')

    wrapper.unmount()
  })
})
