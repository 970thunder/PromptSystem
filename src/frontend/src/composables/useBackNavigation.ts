import { useRouter } from 'vue-router'
import { useNavigationHistory } from '@/navigation/navigationHistory'

const DEFAULT_FALLBACK = '/'

export const isSafeInternalPath = (value: unknown, fallback = DEFAULT_FALLBACK) => {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//')) {
    return fallback
  }

  let path = value
  try {
    path = decodeURIComponent(value)
  } catch {
    return fallback
  }

  if (!path.startsWith('/') || path.startsWith('//')) {
    return fallback
  }

  const pathname = path.split(/[?#]/, 1)[0]
  if (['/login', '/register', '/forgot-password', '/auth/callback'].includes(pathname)) {
    return fallback
  }

  return path
}

export const useBackNavigation = (fallback = DEFAULT_FALLBACK) => {
  const router = useRouter()
  const { canGoBack } = useNavigationHistory(router)
  const safeFallback = isSafeInternalPath(fallback)

  const goBack = async () => {
    if (canGoBack.value) {
      await router.back()
      return
    }

    await router.replace(safeFallback)
  }

  return {
    canGoBack,
    goBack,
    fallback: safeFallback
  }
}
