import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { userApi } from '@/api/userApi'
import { useUserStore } from '@/stores/user'

describe('user logout', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('revokes the server session before clearing local credentials', async () => {
    const logout = vi.spyOn(userApi, 'logout').mockResolvedValue({ code: 200, message: 'Success', data: null })
    const store = useUserStore()
    store.setToken('token-to-revoke')
    store.setUserInfo({
      id: 9,
      username: 'User',
      avatar: '',
      email: 'user@example.com',
      bio: '',
      level: 1,
      experience: 0,
      status: 1,
      createdAt: '2026-08-30'
    })

    await store.logoutServer()

    expect(logout).toHaveBeenCalledOnce()
    expect(store.token).toBe('')
    expect(store.userInfo).toBeNull()
    expect(localStorage.getItem('token')).toBeNull()
  })
})
