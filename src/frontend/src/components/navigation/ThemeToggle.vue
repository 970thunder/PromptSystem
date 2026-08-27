<!-- 文件作用：主题切换按钮，触发 useSiteTheme 的圆环扩散切换。 -->
<script setup lang="ts">
import { useSiteTheme } from '@/composables/useSiteTheme'

const { mode, switchTheme } = useSiteTheme()

function toggle(event: MouseEvent) {
  const next = mode.value === 'dark' ? 'light' : 'dark'
  switchTheme(next, event)
}
</script>

<template>
  <button
    type="button"
    class="btn-pill btn-pill-secondary"
    :aria-label="mode === 'dark' ? '切换到浅色主题' : '切换到深色主题'"
    :title="mode === 'dark' ? '切换到浅色主题' : '切换到深色主题'"
    @click="toggle"
  >
    {{ mode === 'dark' ? '☀' : '☾' }}
  </button>
</template>
