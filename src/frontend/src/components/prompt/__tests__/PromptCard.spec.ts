import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import PromptCard from '../PromptCard.vue'
import { mockPrompts } from '@/mock/prompts'

const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/search', component: { template: '<div />' } },
    { path: '/prompt/:id', component: { template: '<div />' } }
  ]
})

describe('PromptCard', () => {
  it('links the whole card to its prompt detail by default', async () => {
    await router.push('/search')
    await router.isReady()

    const prompt = mockPrompts[0]
    const wrapper = mount(PromptCard, {
      props: { prompt },
      global: { plugins: [router] }
    })

    const link = wrapper.get('.prompt-card__link')
    expect(link.attributes('href')).toBe(`/prompt/${prompt.id}`)

    expect(link.element.tagName).toBe('A')
  })

  it('uses an explicit target when one is provided', () => {
    const wrapper = mount(PromptCard, {
      props: {
        prompt: mockPrompts[0],
        target: '/search?tag=workflow'
      },
      global: { plugins: [router] }
    })

    expect(wrapper.get('.prompt-card__link').attributes('href')).toBe('/search?tag=workflow')
  })
})
