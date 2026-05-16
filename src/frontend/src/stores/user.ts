import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { userApi } from '@/api/userApi'
import type { User } from '@/types'

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(localStorage.getItem('token') || '')
  const userInfo = ref<User | null>(
    localStorage.getItem('userInfo') ? JSON.parse(localStorage.getItem('userInfo') as string) : null
  )
  const loading = ref(false)

  const setToken = (newToken: string) => {
    token.value = newToken
    localStorage.setItem('token', newToken)
  }

  const setUserInfo = (info: User | null) => {
    userInfo.value = info
    if (info) {
      localStorage.setItem('userInfo', JSON.stringify(info))
      return
    }

    localStorage.removeItem('userInfo')
  }

  const logout = () => {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('userInfo')
  }

  const isLoggedIn = computed(() => !!token.value)

  const login = async (payload: { email: string; password: string }) => {
    loading.value = true
    try {
      const response = await userApi.login(payload)
      setToken(response.data.token)
      setUserInfo(response.data.user)
      return response.data.user
    } finally {
      loading.value = false
    }
  }

  const register = async (payload: { username: string; email: string; password: string; captcha: string }) => {
    loading.value = true
    try {
      const response = await userApi.register(payload)
      setToken(response.data.token)
      setUserInfo(response.data.user)
      return response.data.user
    } finally {
      loading.value = false
    }
  }

  const fetchUserInfo = async () => {
    if (!token.value) {
      return null
    }

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
    isLoggedIn,
    login,
    register,
    fetchUserInfo
  }
})
