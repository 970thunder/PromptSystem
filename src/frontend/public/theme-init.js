(function () {
  try {
    var stored = localStorage.getItem('promptos-theme-mode')
    var mode = stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system'
    var resolved = mode === 'system'
      ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
      : mode
    document.documentElement.dataset.mode = resolved
    document.documentElement.dataset.themeReady = 'true'
  } catch (e) {
    document.documentElement.dataset.mode = 'light'
    document.documentElement.dataset.themeReady = 'true'
  }
})()
