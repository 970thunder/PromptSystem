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
})
