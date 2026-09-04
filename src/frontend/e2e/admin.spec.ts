import { expect, test, type Page } from '@playwright/test'

// 文件作用：S-14 治理闭环的浏览器 E2E——管理审核控制台。覆盖管理员查看举报、
// 下架内容并办结、审计链查询，以及非管理员访问得到明确权限提示。

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

const adminUser = {
  id: 8,
  username: 'PromptOS Admin',
  avatar: '',
  email: 'admin@example.com',
  bio: '',
  level: 9,
  experience: 0,
  status: 1,
  createdAt: '2026-01-01',
  hasGitHubBound: true
}

const pendingReport = {
  id: 7,
  userId: 3,
  targetType: 'prompt',
  targetId: 105,
  reason: 'spam',
  detail: '疑似营销灌水内容',
  status: 'pending' as const,
  createdAt: '2026-09-05T08:00:00Z'
}

const reviewedReport = {
  ...pendingReport,
  status: 'reviewed' as const,
  reviewedBy: 8,
  reviewNote: '已下架'
}

async function mockSignedIn(page: Page) {
  await page.route('**/api/v1/user/info', (route) => route.fulfill(ok(adminUser)))
  await page.route('**/api/v1/user/history*', (route) => route.fulfill(ok({ list: [], total: 0, page: 1, pageSize: 24 })))
  await page.route('**/api/v1/user/favorites', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/likes', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/drafts', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/following', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/user/followers', (route) => route.fulfill(ok([])))
  await page.route('**/api/v1/users/*/follow-status', (route) => route.fulfill(ok({ userId: 8, following: false, followerCount: 0, followingCount: 0 })))
}

test.describe('PromptOS admin console', () => {
  test('admin reviews a report and inspects the audit chain', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin console exercised on desktop')
    await mockSignedIn(page)

    let reviewPayload: Record<string, unknown> | null = null
    let pending = true
    await page.route('**/api/v1/admin/reports**', async (route) => {
      if (route.request().method() === 'PATCH') {
        reviewPayload = route.request().postDataJSON()
        pending = false
        return route.fulfill(ok(reviewedReport))
      }
      return route.fulfill(ok({
        list: pending ? [pendingReport] : [reviewedReport],
        total: 1,
        page: 1,
        pageSize: 20
      }))
    })
    await page.route('**/api/v1/admin/audit**', (route) => route.fulfill(ok({
      list: [{
        id: 1,
        actorId: 8,
        action: 'report.review',
        targetType: 'prompt',
        targetId: 105,
        metadata: '{"status":"reviewed","action":"remove"}',
        requestId: 'e2e',
        prevHash: '',
        eventHash: 'a'.repeat(64),
        createdAt: '2026-09-05T08:10:00Z'
      }],
      total: 1,
      page: 1,
      pageSize: 10
    })))

    await page.goto('/admin')
    await expect(page.getByRole('heading', { name: '审核控制台' })).toBeVisible()
    await expect(page.getByText('疑似营销灌水内容')).toBeVisible()

    await page.getByPlaceholder('处理备注（可选）').fill('E2E 审核备注')
    await page.getByRole('button', { name: '下架内容并办结' }).click()

    // 等 PATCH 完成（列表刷新后行会变成已处理态），再断言请求体
    await expect(page.getByText('已下架')).toBeVisible({ timeout: 10_000 })
    expect(reviewPayload).toMatchObject({
      status: 'reviewed',
      action: 'remove',
      note: 'E2E 审核备注'
    })

    await page.getByRole('button', { name: '查看审计链' }).click()
    await expect(page.getByText('report.review')).toBeVisible()
    await expect(page.getByText(/hash aaaaaaaaaaa/)).toBeVisible()
  })

  test('non-admin sees a permission notice instead of data', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin console exercised on desktop')
    await page.route('**/api/v1/user/info', (route) => route.fulfill(ok(adminUser)))
    await page.route('**/api/v1/admin/reports**', (route) => route.fulfill({
      status: 403,
      contentType: 'application/json',
      body: JSON.stringify({ code: 403, message: 'Administrator role required', errorCode: 'ADMIN_REQUIRED', data: null })
    }))

    await page.goto('/admin')
    await expect(page.getByText('需要管理员角色')).toBeVisible()
    await expect(page.getByText('疑似营销灌水内容')).not.toBeVisible()
  })
})
