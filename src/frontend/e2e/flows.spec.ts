import { expect, test, type Page, type Route } from '@playwright/test'

// 文件作用：F-10 浏览器 E2E 流程覆盖。在 e2e 模式（VITE_ENABLE_PROMPT_API=false，
// 内容数据走内置 mock）下，通过网络路由拦截验证认证、评论、点赞、发布、工作台、
// 主题与完整导航的前端流程与 API 契约。

const ok = (data: unknown) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ code: 200, message: 'Success', data })
})

const unauthorized = {
  status: 401,
  contentType: 'application/json',
  body: JSON.stringify({ code: 401, message: 'Unauthorized', errorCode: 'AUTH_TOKEN_MISSING', data: null })
}

const testUser = {
  id: 9,
  username: 'E2E Tester',
  avatar: '',
  email: 'e2e@example.com',
  bio: 'e2e',
  level: 1,
  experience: 0,
  status: 1,
  createdAt: '2026-01-01',
  hasGitHubBound: true
}

// 与 src/mock/prompts.ts 中 id=101 的记录保持一致的关键字段。
const basePrompt = {
  id: 101,
  title: 'Brand Poster Prompt Builder',
  description: 'Turn a short slogan into polished prompt variants.',
  cover: 'Aurora campaign board',
  images: [] as string[],
  content: 'You are a senior visual director.',
  systemPrompt: 'Act as an art director.',
  model: 'Midjourney v6',
  params: { temperature: 0.7, topP: 0.9, maxTokens: 1200 },
  categoryId: 1,
  categoryName: '摄影',
  tags: ['品牌', '海报', '电商'],
  userId: 1,
  user: {
    id: 1,
    username: 'Astra Lab',
    avatar: '',
    email: 'astra@example.com',
    bio: '',
    level: 8,
    experience: 1320,
    status: 1,
    createdAt: '2026-05-01'
  },
  views: 12430,
  likes: 893,
  favorites: 516,
  status: 1,
  createdAt: '2026-05-10',
  updatedAt: '2026-05-15'
}

async function mockAnonymous(page: Page) {
  await page.route('**/api/v1/user/info', (route) => route.fulfill(unauthorized))
}

async function mockSignedIn(page: Page) {
  await page.route('**/api/v1/user/info', (route) => route.fulfill(ok(testUser)))
  // 带 query 的端点必须以 * 结尾，否则 glob 匹配不到真实请求，会穿透到本地代理。
  await page.route('**/api/v1/user/history*', (route) => route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 24 })))
  await page.route('**/api/v1/user/favorites', (route) => route.fulfill(ok([basePrompt])))
  await page.route('**/api/v1/user/likes', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/drafts', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/following', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/followers', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/prompts**', (route) => route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 24 })))
  await page.route('**/api/v1/prompts/*/view', (route) => route.fulfill(ok({ prompt: null, applied: true })))
  await page.route('**/api/v1/users/*/follow-status', (route) => route.fulfill(ok({ userId: 1, following: false, followerCount: 0, followingCount: 0 })))
  await page.route('**/uploads/**', (route) => route.fulfill({ status: 200, contentType: 'image/png', body: coverPng }))
}

async function mockAuthGuardedLibrary(page: Page) {
  await page.route('**/api/v1/prompts**', (route) => route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 24 })))
  await page.route('**/api/v1/user/history*', (route) => route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 24 })))
  await page.route('**/api/v1/user/favorites', (route) => route.fulfill(ok([basePrompt])))
  await page.route('**/api/v1/user/likes', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/drafts', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/following', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/followers', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/users/*/follow-status', (route) => route.fulfill(ok({ userId: 9, following: false, followerCount: 0, followingCount: 0 })))
}

// 1x1 透明 PNG，用于发布向导的封面上传。
const coverPng = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
  'base64'
)

test.describe('PromptOS F-10 flows', () => {
  test('register with email captcha signs the user in', async ({ page }) => {
    await mockAnonymous(page)
    let captchaPayload: Record<string, unknown> | null = null
    let registerPayload: Record<string, unknown> | null = null
    await page.route('**/api/v1/user/captcha', (route) => {
      captchaPayload = route.request().postDataJSON()
      return route.fulfill(ok({ expiresInSeconds: 600, devCode: '123456' }))
    })
    await page.route('**/api/v1/user/register', (route) => {
      registerPayload = route.request().postDataJSON()
      return route.fulfill(ok({ token: 'e2e-token', user: testUser }))
    })

    await page.goto('/register')
    await page.getByPlaceholder('输入你的展示名称').fill('E2E Tester')
    await page.getByPlaceholder('you@example.com').fill('e2e@example.com')
    const captchaButton = page.getByRole('button', { name: '获取验证码' })
    await expect(captchaButton).toBeEnabled()
    await captchaButton.click()
    await page.getByPlaceholder('输入 6 位验证码').fill('123456')
    await page.getByPlaceholder('至少 8 个字符').fill('password-e2e-123')
    await page.getByPlaceholder('再次输入密码').fill('password-e2e-123')
    await page.locator('button[type="submit"]').click()

    await expect(page.locator('.header-avatar')).toBeVisible({ timeout: 10_000 })
    expect(captchaPayload).toMatchObject({ email: 'e2e@example.com' })
    expect(registerPayload).toMatchObject({ username: 'E2E Tester', email: 'e2e@example.com', captcha: '123456' })
  })

  test('login honors redirect and reaches the workspace', async ({ page }) => {
    // /user/info 必须在登录前返回 401（否则 /login 会因已登录被重定向走），
    // 登录成功后再返回用户，供工作台页面使用。
    let signedIn = false
    await page.route('**/api/v1/user/info', (route) => route.fulfill(signedIn ? ok(testUser) : unauthorized))
    await page.route('**/api/v1/user/login', (route) => {
      signedIn = true
      return route.fulfill(ok({ token: 'e2e-token', user: testUser }))
    })
    await mockAuthGuardedLibrary(page)

    await page.goto('/login?redirect=%2Fprofile')
    await page.getByPlaceholder('you@example.com').fill('e2e@example.com')
    await page.getByPlaceholder('请输入密码').fill('password-e2e-123')
    await page.locator('button[type="submit"]').click()

    await expect(page).toHaveURL(/\/profile$/, { timeout: 10_000 })
    await expect(page.getByText('E2E Tester').first()).toBeVisible()
    await expect(page.getByRole("button", { name: /收藏/ })).toBeVisible()
  })

  test('prompt detail supports comment and like interactions', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'interaction flow exercised on desktop')
    await mockSignedIn(page)

    let commentPayload: { content?: string } | null = null
    await page.route('**/api/v1/prompts/*/comments', async (route: Route) => {
      if (route.request().method() === 'POST') {
        commentPayload = route.request().postDataJSON()
        return route.fulfill(ok({
          id: 501,
          targetType: 'prompt',
          targetId: 101,
          userId: 9,
          user: testUser,
          content: commentPayload?.content ?? '',
          likes: 0,
          parentId: null,
          replies: [],
          createdAt: new Date().toISOString()
        }))
      }
      return route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 20 }))
    })
    await page.route('**/api/v1/prompts/*/like', (route) => route.fulfill(ok({ prompt: { ...basePrompt, likes: 894 }, applied: true })))
    await page.route('**/api/v1/prompts/*/interaction', (route) => route.fulfill(ok({ liked: false, favorited: false })))
    await page.route('**/api/v1/prompts/*/related', (route) => route.fulfill(ok([])))

    await page.goto('/')
    const firstPromptLink = page.locator('a[href="/prompt/101"]').first()
    await firstPromptLink.click()
    await expect(page).toHaveURL(/\/prompt\/101$/)

    await page.getByLabel('写下你的评论').fill('E2E 评论：实测可用，复制即走。')
    await page.getByRole('button', { name: '发布评论' }).click()
    await expect(page.getByText('评论已发布')).toBeVisible()
    expect(commentPayload).toMatchObject({ content: 'E2E 评论：实测可用，复制即走。' })

    await page.getByRole('button', { name: /点赞 · 893/ }).click()
    await expect(page.getByRole('button', { name: '点赞 · 894' })).toBeVisible()
  })

  test('publish wizard uploads a cover and creates a prompt', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'publish wizard exercised on desktop')
    await mockSignedIn(page)

    let createPayload: Record<string, unknown> | null = null
    await page.route('**/api/v1/uploads/images', (route) => route.fulfill(ok({ url: '/uploads/e2e-cover.png' })))
    await page.route('**/api/v1/prompts', async (route: Route) => {
      if (route.request().method() === 'POST') {
        createPayload = route.request().postDataJSON()
        return route.fulfill(ok({ ...basePrompt, id: 999, title: (createPayload as { title?: string } | null)?.title }))
      }
      return route.continue()
    })

    await page.goto('/publish')
    await page.locator('.publish-cover-pane input[type="file"]').first().setInputFiles({
      name: 'cover.png',
      mimeType: 'image/png',
      buffer: coverPng
    })
    await expect(page.getByText('封面已就绪，可进入下一步')).toBeVisible()
    await page.getByRole('button', { name: '下一步' }).click()

    await page.getByPlaceholder('例如：电影感产品海报生成器').fill('E2E 发布测试提示词')
    await page.getByPlaceholder('说明使用场景、风格与预期输出').fill('验证发布向导的端到端流程。')
    await page.locator('.n-select').first().click()
    await page.locator('.n-base-select-menu').getByText('摄影', { exact: true }).click()
    await page.getByPlaceholder('Midjourney v6 / SDXL / DALL·E 3').fill('GPT-4.1')
    await page.getByRole('button', { name: '下一步' }).click()

    await page.getByPlaceholder('输入主提示词；JSON 模式可一键格式化').fill('You are an e2e smoke test prompt.')
    await page.getByRole('button', { name: '下一步' }).click()
    await page.getByRole('button', { name: '下一步' }).click()

    await page.getByRole('button', { name: '发布提示词' }).click()
    await expect(page.getByText('提示词已发布')).toBeVisible({ timeout: 10_000 })
    expect(createPayload).toMatchObject({ title: 'E2E 发布测试提示词', categoryId: 1, model: 'GPT-4.1' })
    expect((createPayload as { cover?: string } | null)?.cover).toBe('/uploads/e2e-cover.png')
  })

  test('workspace switches between library tabs', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'workspace flow exercised on desktop')
    await mockSignedIn(page)
    await mockAuthGuardedLibrary(page)

    await page.goto('/profile')
    await expect(page.getByText('E2E Tester').first()).toBeVisible()

    await page.getByRole("button", { name: /收藏/ }).click()
    await expect(page.getByText('Brand Poster Prompt Builder')).toBeVisible()
  })

  test('theme toggle persists across reloads', async ({ page }) => {
    await mockAnonymous(page)
    await page.goto('/')
    const root = page.locator('html')
    const initialMode = await root.getAttribute('data-mode')

    await page.getByRole('button', { name: /切换到(深色|浅色)主题/ }).click()
    const expected = initialMode === 'dark' ? 'light' : 'dark'
    await expect(root).toHaveAttribute('data-mode', expected)

    await page.reload()
    await expect(root).toHaveAttribute('data-mode', expected)
  })

  test('mega menu navigates to tagged discovery', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'hover mega menu exercised on desktop')
    await mockAnonymous(page)

    await page.goto('/')
    const discover = page.getByRole('link', { name: '发现' }).first()
    await discover.hover()
    const workflowLink = page.getByRole('link', { name: '工作流' }).first()
    await expect(workflowLink).toBeVisible()
    await workflowLink.click()
    await expect(page).toHaveURL(/\/search\?tag=%E6%B5%81%E7%A8%8B|\/search\?tag=流程/)
  })
})
