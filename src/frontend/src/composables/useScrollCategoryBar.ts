import { onMounted, onUnmounted, ref } from 'vue'

const DEFAULT_SCROLL_THRESHOLD = 80
const DEFAULT_IDLE_HIDE_MS = 1000
const TOP_OFFSET = 10

export function useScrollCategoryBar(
  scrollThreshold = DEFAULT_SCROLL_THRESHOLD,
  idleHideMs = DEFAULT_IDLE_HIDE_MS
) {
  const visible = ref(true)

  let lastScrollY = 0
  let scrollTicking = false
  let idleTimer: ReturnType<typeof setTimeout> | null = null

  const clearIdleTimer = () => {
    if (idleTimer) {
      clearTimeout(idleTimer)
      idleTimer = null
    }
  }

  const resetIdleTimer = () => {
    clearIdleTimer()

    if (window.scrollY <= TOP_OFFSET) {
      return
    }

    idleTimer = setTimeout(() => {
      visible.value = false
    }, idleHideMs)
  }

  const updateVisibility = () => {
    const currentY = window.scrollY

    if (currentY <= TOP_OFFSET) {
      visible.value = true
      lastScrollY = currentY
      resetIdleTimer()
      return
    }

    if (currentY > lastScrollY && currentY > scrollThreshold) {
      visible.value = false
    } else if (currentY < lastScrollY) {
      visible.value = true
    }

    lastScrollY = currentY
    resetIdleTimer()
  }

  const handleScroll = () => {
    if (scrollTicking) {
      return
    }

    scrollTicking = true
    requestAnimationFrame(() => {
      updateVisibility()
      scrollTicking = false
    })
  }

  onMounted(() => {
    window.addEventListener('scroll', handleScroll, { passive: true })
  })

  onUnmounted(() => {
    window.removeEventListener('scroll', handleScroll)
    clearIdleTimer()
  })

  return { visible }
}
