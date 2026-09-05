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
})
