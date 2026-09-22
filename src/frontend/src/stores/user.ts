import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { userApi } from '@/api/userApi'
import type { User } from '@/types'

const bindPromptPendingKey = (userID: number) => `promptos:bind-github:pending:${userID}`
const bindPromptDailyKey = (userID: number) => `promptos:bind-github:last-date:${userID}`

const localDateKey = () => {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export const useUserStore = defineStore('user', () => {
  // JWTs are delivered through an HttpOnly cookie. `token` remains an
  // in-memory escape hatch for older API clients and is never persisted.
  const token = ref<string>('')
  const userInfo = ref<User | null>(
    localStorage.getItem('userInfo') ? JSON.parse(localStorage.getItem('userInfo') as string) : null
  )
  const loading = ref(false)
  const sessionReady = ref(false)
  const sessionActive = ref(false)
  localStorage.removeItem('token')

  const setToken = (newToken: string) => {
    token.value = newToken
    sessionActive.value = Boolean(newToken) || Boolean(userInfo.value)
  }

  const setUserInfo = (info: User | null) => {
    userInfo.value = info
    if (info) {
      localStorage.setItem('userInfo', JSON.stringify(info))
      sessionActive.value = true
      return
    }

    localStorage.removeItem('userInfo')
    sessionActive.value = false
  }

  const logout = () => {
    token.value = ''
    userInfo.value = null
    sessionActive.value = false
    sessionReady.value = true
    localStorage.removeItem('userInfo')
  }

  const logoutServer = async () => {
    if (sessionActive.value || token.value) {
      try {
        // 单点登出：后端返回身份中心结束会话地址，跳过去一起结束 IdP 会话。
        const response = await userApi.logout()
        const logoutUrl = (response as unknown as { data?: { logoutUrl?: string | null } }).data?.logoutUrl
        logout()
        if (logoutUrl) { window.location.href = logoutUrl; return }
      } catch {
        // Always clear local credentials even when the API is unavailable.
      }
    }
    logout()
  }

  const clearBindPromptPending = (userID: number) => {
    localStorage.removeItem(bindPromptPendingKey(userID))
  }

  const shouldPromptBindGitHub = () => {
    const user = userInfo.value
    if (!user || user.hasGitHubBound === true) {
      return false
    }

    const pending = localStorage.getItem(bindPromptPendingKey(user.id)) === '1'
    if (pending) {
      return true
    }

    const today = localDateKey()
    const lastPromptDate = localStorage.getItem(bindPromptDailyKey(user.id))
    return lastPromptDate !== today
  }

  const markBindPromptShown = () => {
    const user = userInfo.value
    if (!user) {
      return
    }

    const today = localDateKey()
    clearBindPromptPending(user.id)
    localStorage.setItem(bindPromptDailyKey(user.id), today)
  }

  const isLoggedIn = computed(() => sessionActive.value)

  const restoreSession = async () => {
    if (sessionReady.value) {
      return sessionActive.value
    }
    try {
      const response = await userApi.getUserInfo()
      userInfo.value = response.data
      localStorage.setItem('userInfo', JSON.stringify(response.data))
      sessionActive.value = true
    } catch {
      userInfo.value = null
      localStorage.removeItem('userInfo')
      sessionActive.value = false
    } finally {
      sessionReady.value = true
    }
    return sessionActive.value
  }

  const fetchUserInfo = async () => {
    try {
      const response = await userApi.getUserInfo()
      setUserInfo(response.data)
      return response.data
    } catch {
      logout()
      return null
    }
  }

  return {
    token,
    userInfo,
    loading,
    setToken,
    setUserInfo,
    logout,
    logoutServer,
    shouldPromptBindGitHub,
    markBindPromptShown,
    isLoggedIn,
    sessionReady,
    restoreSession,
    fetchUserInfo
  }
})
