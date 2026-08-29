import { useRouter } from 'vue-router'
import { useNavigationHistory } from '@/navigation/navigationHistory'

const DEFAULT_FALLBACK = '/'

export const isSafeInternalPath = (value: unknown, fallback = DEFAULT_FALLBACK) => {
  if (typeof value !== 'string' || value.length > 2048 || !value.startsWith('/') || value.startsWith('//')) {
    return fallback
  }

  let path = value
  try {
    path = decodeURIComponent(value)
  } catch {
    return fallback
  }

  if (!path.startsWith('/') || path.startsWith('//') || path.includes('\\') || hasControlCharacter(path)) {
    return fallback
  }

  const pathname = path.split(/[?#]/, 1)[0]
  if (['/login', '/register', '/forgot-password', '/auth/callback'].includes(pathname)) {
    return fallback
  }

  return path
}

const hasControlCharacter = (value: string) => {
  for (const character of value) {
    const code = character.charCodeAt(0)
    if (code <= 0x1f || code === 0x7f) return true
  }
  return false
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
