import { expect, test } from '@playwright/test'

test.describe('PromptOS browser smoke', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/user/info', (route) => route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ code: 401, message: 'Unauthorized', errorCode: 'AUTH_TOKEN_MISSING', data: null })
    }))
    await page.route('**/api/v1/prompts/*/view', (route) => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 200, message: 'Success', data: { prompt: null, applied: true } })
    }))
  })

  test('home navigation, title barrage and prompt detail are reachable', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: '好提示词，马上复用。' })).toBeVisible()
    await expect(page.locator('.home-hero__title-barrage')).toBeVisible()
    await expect(page.locator('.home-hero__title-barrage')).not.toContainText('#')

    const firstPromptLink = page.locator('a[href^="/prompt/"]').first()
    await expect(firstPromptLink).toBeVisible()
    const href = await firstPromptLink.getAttribute('href')
    expect(href).toMatch(/^\/prompt\/\d+$/)
    await firstPromptLink.click()
    await expect(page).toHaveURL(/\/prompt\/\d+$/)
    await expect(page.locator('main')).toContainText('社区反馈')
  })

  test('search URL and mobile menu remain usable', async ({ page }) => {
    await page.goto('/search?sort=popular')
    await expect(page.getByRole('heading', { name: '按意图、模型或分类查找提示词' })).toBeVisible()
    await expect(page).toHaveURL(/sort=popular/)

    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto('/')
    const menuButton = page.getByRole('button', { name: /菜单|导航/ }).first()
    if (await menuButton.count()) {
      await menuButton.click()
      await expect(page.locator('nav').first()).toBeVisible()
    }
    await expect(page.locator('body')).toBeVisible()
  })
})
