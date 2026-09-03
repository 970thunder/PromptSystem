import axios, { isCancel, type AxiosInstance, type AxiosResponse } from 'axios'
import { useUserStore } from '@/stores/user'
import { createDiscreteApi } from 'naive-ui'
import router from '@/router'

const { message: messageApi } = createDiscreteApi(['message'])

const request: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json'
  }
})

const readCookie = (name: string) => {
  const prefix = `${encodeURIComponent(name)}=`
  const item = document.cookie.split('; ').find((entry) => entry.startsWith(prefix))
  return item ? decodeURIComponent(item.slice(prefix.length)) : ''
}

// Request interceptor
request.interceptors.request.use(
  (config) => {
    const userStore = useUserStore()
    if (userStore.token) {
      config.headers.Authorization = `Bearer ${userStore.token}`
    }
    if (config.method && !['get', 'head', 'options', 'trace'].includes(config.method.toLowerCase())) {
      const csrf = readCookie('promptos_csrf')
      if (csrf) {
        config.headers['X-CSRF-Token'] = csrf
      }
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interception
request.interceptors.response.use(
  (response: AxiosResponse) => {
    const res = response.data
    if (res.code !== 200) {
      messageApi.error(res.message || '请求失败')
      if (res.code === 401) {
        const userStore = useUserStore()
        userStore.logout()
        router.push('/login')
      }
      return Promise.reject(new Error(res.message || 'Error'))
    }
    return res
  },
  (error) => {
    if (isCancel(error)) {
      return Promise.reject(error)
    }
    const errorMessage = error.response?.data?.message || error.message || '网络错误'
    messageApi.error(errorMessage)
    return Promise.reject(error)
  }
)

export default request
