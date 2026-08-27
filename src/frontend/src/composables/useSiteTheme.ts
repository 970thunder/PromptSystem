import { computed, ref, type Ref } from 'vue'

export type ThemeMode = 'system' | 'light' | 'dark'
export type ResolvedMode = 'light' | 'dark'

const STORAGE_KEY = 'promptos-theme-mode'

type ThemeDocument = Document & {
  startViewTransition?: (update: () => void | Promise<void>) => {
    ready: Promise<void>
    finished: Promise<void>
  }
}

function systemResolved(): ResolvedMode {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function resolve(mode: ThemeMode): ResolvedMode {
  return mode === 'system' ? systemResolved() : mode
}

function readStoredMode(): ThemeMode {
  const stored = window.localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark' || stored === 'system') {
    return stored
  }
  return 'system'
}

/**
 * Site-wide theme state. Mirrors the NebulaSite mechanism: html[data-mode] is
 * the single source of truth, and switching uses a clip-path circle reveal via
 * the View Transitions API (no DOM overlay/mask node).
 */
const mode: Ref<ThemeMode> = ref(readStoredMode())
const resolvedMode = computed<ResolvedMode>(() => resolve(mode.value))

function apply(resolved: ResolvedMode) {
  const root = document.documentElement
  root.dataset.mode = resolved
  root.dataset.themeReady = 'true'
}

function switchTheme(next: ThemeMode, event?: MouseEvent | KeyboardEvent) {
  mode.value = next
  window.localStorage.setItem(STORAGE_KEY, next)

  const doc = document as ThemeDocument
  const root = document.documentElement
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const target = resolve(next)

  if (!doc.startViewTransition || reduceMotion || !event) {
    apply(target)
    return
  }

  const clientX = event instanceof MouseEvent ? event.clientX : window.innerWidth / 2
  const clientY = event instanceof MouseEvent ? event.clientY : window.innerHeight / 2
  const radius = Math.hypot(
    Math.max(clientX, window.innerWidth - clientX),
    Math.max(clientY, window.innerHeight - clientY),
  )

  const transition = doc.startViewTransition(() => {
    apply(target)
    return Promise.resolve()
  })

  void transition.ready.then(() => {
    root.animate(
      {
        clipPath: [
          `circle(0px at ${clientX}px ${clientY}px)`,
          `circle(${radius}px at ${clientX}px ${clientY}px)`,
        ],
      },
      {
        duration: 460,
        easing: 'cubic-bezier(.22, 1, .36, 1)',
        pseudoElement: '::view-transition-new(root)',
      },
    )
  })
}

window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
  if (mode.value === 'system') {
    apply(systemResolved())
  }
})

export function useSiteTheme() {
  return {
    mode,
    resolvedMode,
    apply,
    switchTheme,
  }
}
