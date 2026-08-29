// 文件作用：首页冒烟测试。挂载 HomeView（mock 数据路径），保证首屏关键分区
// ——品牌 hero + 搜索、今日精选、最新发布网格、分类入口——始终完整渲染。
import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import HomeView from '../HomeView.vue'

const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/', component: HomeView },
    { path: '/search', component: { template: '<div />' } },
    { path: '/prompt/:id', component: { template: '<div />' } },
    { path: '/publish', component: { template: '<div />' } },
    { path: '/login', component: { template: '<div />' } },
    { path: '/community', component: { template: '<div />' } },
    { path: '/profile/:id?', component: { template: '<div />' } }
  ]
})

const mountHome = async () => {
  const pinia = createPinia()
  setActivePinia(pinia)
  await router.push('/')
  await router.isReady()
  const wrapper = mount(HomeView, {
    global: {
      plugins: [pinia, router]
    }
  })
  await flushPromises()
  return wrapper
}

describe('HomeView', () => {
  it('渲染品牌 hero 与搜索入口', async () => {
    const wrapper = await mountHome()

    expect(wrapper.find('h1').text()).toContain('好提示词，马上复用。')
    expect(wrapper.find('form[role="search"] input').exists()).toBe(true)
    expect(wrapper.find('form[role="search"] button[type="submit"]').text()).toBe('搜索')
  })

  it('用 mock 信息流渲染今日精选、最新发布与分类入口', async () => {
    const wrapper = await mountHome()

    // 今日精选主卡取信息流第一条
    expect(wrapper.text()).toContain('今日精选')
    expect(wrapper.text()).toContain('Brand Poster Prompt Builder')

    // 最新发布网格展示第三条之后的内容
    expect(wrapper.text()).toContain('刚刚更新')
    expect(wrapper.text()).toContain('Short Video Script Factory')

    // 分类入口来自真实 mock 数据，首屏改为标题飘带而非统计面板
    expect(wrapper.text()).toContain('探索分类')
    expect(wrapper.find('.home-hero__title-barrage').exists()).toBe(true)
    expect(wrapper.find('[aria-label="社区数据"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('收录提示词')
    expect(wrapper.text()).not.toContain('内容分类')
    expect(wrapper.text()).not.toContain('热门标签')
    expect(wrapper.findAll('.home-hero__title-item').length).toBeGreaterThan(0)
    expect(wrapper.findAll('.home-hero__title-cover').length).toBeGreaterThan(0)
    expect(wrapper.find('.home-hero__title-barrage').text()).not.toContain('#')
  })
})
