import { describe, expect, it } from 'vitest'
import router from '@/router'

describe('profile route access', () => {
  it('keeps the owner workspace private and author profiles public', () => {
    const owner = router.getRoutes().find((route) => route.name === 'Profile')
    const publicProfile = router.getRoutes().find((route) => route.name === 'PublicProfile')

    expect(owner?.path).toBe('/profile')
    expect(owner?.meta.requiresAuth).toBe(true)
    expect(publicProfile?.path).toBe('/profile/:userId')
    expect(publicProfile?.meta.requiresAuth).toBeUndefined()
  })

  it('exposes public terms and privacy pages', () => {
    const terms = router.getRoutes().find((route) => route.name === 'Terms')
    const privacy = router.getRoutes().find((route) => route.name === 'Privacy')

    expect(terms?.path).toBe('/terms')
    expect(terms?.meta.requiresAuth).toBeUndefined()
    expect(privacy?.path).toBe('/privacy')
    expect(privacy?.meta.requiresAuth).toBeUndefined()
  })

  it('keeps the admin console authenticated and registers the setup wizard', () => {
    const admin = router.getRoutes().find((route) => route.name === 'Admin')
    const setup = router.getRoutes().find((route) => route.name === 'Setup')

    expect(admin?.path).toBe('/admin')
    expect(admin?.meta.requiresAuth).toBe(true)
    // 初始化向导必须匿名可达（尚未有管理员时这是唯一的引导入口）。
    expect(setup?.path).toBe('/setup')
    expect(setup?.meta.requiresAuth).toBeUndefined()
  })
})
