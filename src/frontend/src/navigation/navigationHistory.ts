import { computed, ref, type ComputedRef } from 'vue'
import type { Router, RouteLocationNormalized } from 'vue-router'

export type NavigationDirection = 'forward' | 'back' | 'replace'

type NavigationEntry = {
  fullPath: string
  position: number | null
}

const entries = ref<NavigationEntry[]>([])
const direction = ref<NavigationDirection>('forward')
let installed = false
let currentPosition: number | null = null

const getHistoryPosition = (): number | null => {
  if (typeof window === 'undefined') {
    return null
  }

  const position = window.history.state?.position
  return typeof position === 'number' ? position : null
}

const markCurrentHistoryEntry = () => {
  if (typeof window === 'undefined' || !window.history.replaceState) {
    return
  }

  const state = window.history.state
  if (state?.promptos === true) {
    return
  }

  window.history.replaceState({ ...state, promptos: true }, '', window.location.href)
}

const isQueryOnlyChange = (to: RouteLocationNormalized, from: RouteLocationNormalized) => {
  return to.path === from.path && to.hash === from.hash
}

const installNavigationHistory = (router: Router) => {
  if (installed) {
    return
  }

  installed = true
  entries.value = []
  currentPosition = null

  router.afterEach((to, from, failure) => {
    if (failure) {
      return
    }

    const nextPosition = getHistoryPosition()
    const queryOnly = isQueryOnlyChange(to, from)

    if (entries.value.length === 0) {
      entries.value = [{ fullPath: to.fullPath, position: nextPosition }]
      currentPosition = nextPosition
      direction.value = 'replace'
      markCurrentHistoryEntry()
      return
    }

    const movingBack = nextPosition !== null
      && currentPosition !== null
      && nextPosition < currentPosition
    const replacing = nextPosition !== null
      && currentPosition !== null
      && nextPosition === currentPosition

    if (queryOnly || replacing) {
      entries.value = [
        ...entries.value.slice(0, -1),
        { fullPath: to.fullPath, position: nextPosition }
      ]
      direction.value = 'replace'
    } else if (movingBack) {
      const previousIndex = entries.value.findIndex((entry) => entry.position === nextPosition)
      entries.value = previousIndex >= 0
        ? entries.value.slice(0, previousIndex + 1)
        : [{ fullPath: to.fullPath, position: nextPosition }]
      direction.value = 'back'
    } else {
      entries.value = [...entries.value, { fullPath: to.fullPath, position: nextPosition }]
      direction.value = 'forward'
    }

    currentPosition = nextPosition
    markCurrentHistoryEntry()
  })
}

export const setupNavigationHistory = installNavigationHistory

export const useNavigationHistory = (router: Router) => {
  installNavigationHistory(router)

  const canGoBack: ComputedRef<boolean> = computed(() => entries.value.length > 1)

  return {
    canGoBack,
    direction,
    entries
  }
}
