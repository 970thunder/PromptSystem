import { test, expect } from '@playwright/test'

// 首次部署初始化向导的端到端流程：状态 → 五步向导 → 一次性初始化 → 管理员登录。
// 与其余 e2e 不同，本流程不 mock 网络，直接打真实后端（经 vite 代理）。
//
// 运行前提：start-dev.bat 已启动后端（dev 无 MySQL/Redis 时自动落到内存存储，
// 天然处于未初始化状态）。以下情况自动跳过，不阻塞常规 e2e 套件：
//   - 后端未运行（仅 npm run dev 的纯前端环境）；
//   - 后端已完成初始化（初始化是一次性的，重启后端可重跑一次）；
//   - 移动端项目（向导只需在桌面验证一次）。

interface SetupStatus {
  initialized: boolean
}

test('first-run setup wizard bootstraps an admin and signs in', async ({ page, request }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'wizard exercised once on desktop')

  let initialized: boolean | null = null
  try {
    const response = await request.get('/api/v1/setup/status', { timeout: 3000 })
    if (!response.ok()) {
      test.skip(true, `backend not reachable via proxy (status ${response.status()}); run start-dev.bat for the full-stack e2e`)
    }
    const envelope = (await response.json()) as { data?: SetupStatus }
    initialized = envelope.data?.initialized ?? null
  } catch {
    test.skip(true, 'backend not reachable; run start-dev.bat for the full-stack e2e')
  }
  if (initialized) {
    test.skip(true, 'deployment already initialized; bootstrap is one-shot (restart backend to rerun)')
  }

  // 步骤 0：环境检查（内存/MySQL 均可继续，向导只提示不阻断）。
  await page.goto('/setup')
  await expect(page.getByRole('heading', { name: '系统初始化' })).toBeVisible()
  await expect(page.getByText('环境检查')).toBeVisible()

  await page.getByRole('button', { name: '下一步' }).click()

  // 步骤 1：管理员账号。
  await expect(page.getByText('管理员用户名')).toBeVisible()
  await page.getByPlaceholder('例如 founder').fill('setup-e2e-admin')
  await page.getByPlaceholder('admin@example.com').fill('setup-e2e@example.com')
  await page.getByPlaceholder('设置强密码').fill('SetupE2E123!')
  await page.getByPlaceholder('再次输入密码').fill('SetupE2E123!')

  await page.getByRole('button', { name: '下一步' }).click()

  // 步骤 2：站点信息（保持默认，关闭注册开关一并验证写入）。
  await expect(page.getByText('站点名称')).toBeVisible()
  await page.getByRole('checkbox').uncheck()
  await page.getByRole('button', { name: '下一步' }).click()

  // 步骤 3：邮件配置——跳过（沿用环境变量/未配置）。
  await expect(page.getByText('SMTP 服务器')).toBeVisible()
  await page.getByRole('button', { name: '跳过邮件配置' }).click()

  // 步骤 4：确认并提交。
  await expect(page.getByText('确认初始化')).toBeVisible()
  await expect(page.getByText('setup-e2e@example.com')).toBeVisible()
  await page.getByRole('button', { name: '完成初始化' }).click()

  await expect(page.getByRole('heading', { name: '初始化完成' })).toBeVisible({ timeout: 10_000 })

  // 初始化后的部署再次打开 /setup 直接进入终态。
  await page.goto('/setup')
  await expect(page.getByText('系统已完成初始化')).toBeVisible()

  // 管理员账号可真实登录并进入工作台。
  await page.goto('/login')
  await page.getByPlaceholder('you@example.com').fill('setup-e2e@example.com')
  await page.getByPlaceholder('请输入密码').fill('SetupE2E123!')
  await page.locator('button[type="submit"]').click()

  await page.waitForURL('/', { timeout: 10_000 })
  await page.goto('/profile')
  await expect(page.getByText('账户数据')).toBeVisible({ timeout: 10_000 })
})
